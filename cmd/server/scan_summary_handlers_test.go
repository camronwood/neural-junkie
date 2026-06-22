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

func installCustomerLabPack(t *testing.T, cfg *config.Config) {
	t.Helper()
	config.SetupTestOfficialPackCatalog(t)
	t.Setenv("HOME", t.TempDir())
	if err := cfg.InstallPack(config.PackLifeSciences); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(config.PackLifeSciences, true); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("..", "..", "internal", "packs", "testdata", "customer-lab-pack")
	if _, err := os.Stat(filepath.Join(src, "pack.yaml")); err != nil {
		t.Skip("customer-lab-pack fixture missing")
	}
	zipPath := filepath.Join(t.TempDir(), "customer-lab-pack.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(out)
	walkErr := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, path)
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
	if walkErr != nil {
		out.Close()
		t.Fatal(walkErr)
	}
	if err := w.Close(); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.InstallPackFromZip(data); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled("customer-lab-pack", true); err != nil {
		t.Fatal(err)
	}
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
	installCustomerLabPack(t, appConfig)

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
