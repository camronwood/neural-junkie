package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"golang.org/x/image/tiff"
)

func installScanSummaryLegacyTestPack(t *testing.T, cfg *config.Config) {
	t.Helper()
	config.SetupTestOfficialPackCatalog(t)
	t.Setenv("HOME", t.TempDir())
	if err := cfg.InstallPack(config.PackLifeSciences); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(config.PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}

	packDir := t.TempDir()
	packYAML := `id: scan-summary-legacy-test
version: "1.0.0"
title: Scan summary legacy test
description: Capability-only fixture for in-hub scan summary handler tests.
pack_kind: customer
capabilities:
  - scan-summary-api
requires_packs:
  - life-sciences
`
	if err := os.WriteFile(filepath.Join(packDir, "pack.yaml"), []byte(packYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "scan-summary-legacy-test.zip")
	if err := writeDirZip(zipPath, packDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.InstallPackFromZip(data); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled("scan-summary-legacy-test", true); err != nil {
		t.Fatal(err)
	}
	if cfg.RouteOwnerPackID("/api/scan-summary") != "" {
		t.Fatal("expected legacy scan-summary handler without pack sidecar route owner")
	}
}

func writeDirZip(destZip, srcDir string) error {
	out, err := os.Create(destZip)
	if err != nil {
		return err
	}
	w := zip.NewWriter(out)
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		fw, err := w.Create(rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = fw.Write(data)
		return err
	})
	if err != nil {
		out.Close()
		return err
	}
	if err := w.Close(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func setupScanSummaryHandlerTest(t *testing.T) (workspaceID string, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	runDir := filepath.Join(dir, "HIF1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gray := image.NewGray16(image.Rect(0, 0, 2, 2))
	gray.SetGray16(0, 0, color.Gray16{Y: 1000})
	var tiffBuf bytes.Buffer
	if err := tiff.Encode(&tiffBuf, gray, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "A1"), tiffBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	var err error
	workspaceManager, err = hub.NewWorkspaceManager()
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspaceManager.AddWorkspace("scan-test", dir, hub.AddWorkspaceOptions{Create: false})
	if err != nil {
		t.Fatal(err)
	}

	appConfig = config.DefaultConfig()
	installScanSummaryLegacyTestPack(t, appConfig)

	return ws.ID, func() {
		appConfig = nil
		workspaceManager = nil
	}
}

func TestHandleScanSummaryWellImage_methodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/scan-summary/well-image", nil)
	rec := httptest.NewRecorder()
	handleScanSummaryWellImage(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleScanSummaryWellImage_packDisabled(t *testing.T) {
	appConfig = config.DefaultConfig()
	appConfig.Packs = config.DefaultPacksConfig()
	req := httptest.NewRequest(http.MethodGet, "/api/scan-summary/well-image?workspace=x&well=A1", nil)
	rec := httptest.NewRecorder()
	handleScanSummaryWellImage(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	appConfig = nil
}

func TestHandleScanSummaryWellImage_success(t *testing.T) {
	wsID, cleanup := setupScanSummaryHandlerTest(t)
	defer cleanup()

	q := "?workspace=" + wsID + "&dir=HIF1&well=A1"
	req := httptest.NewRequest(http.MethodGet, "/api/scan-summary/well-image"+q, nil)
	rec := httptest.NewRecorder()
	handleScanSummaryWellImage(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Mime           string `json:"mime"`
		ContentBase64  string `json:"content_base64"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Mime != "image/png" || body.ContentBase64 == "" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestHandleScanSummaryWellImage_invalidWell(t *testing.T) {
	wsID, cleanup := setupScanSummaryHandlerTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/scan-summary/well-image?workspace="+wsID+"&dir=HIF1&well=Z99", nil)
	rec := httptest.NewRecorder()
	handleScanSummaryWellImage(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleScanSummaryWellImage_pathOutsideWorkspace(t *testing.T) {
	wsID, cleanup := setupScanSummaryHandlerTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/scan-summary/well-image?workspace="+wsID+"&dir=..&well=A1", nil)
	rec := httptest.NewRecorder()
	handleScanSummaryWellImage(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
