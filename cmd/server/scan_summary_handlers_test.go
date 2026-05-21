package main

import (
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
	ws, err := workspaceManager.AddWorkspace("scan-test", dir)
	if err != nil {
		t.Fatal(err)
	}

	appConfig = config.DefaultConfig()
	appConfig.Packs = config.DefaultPacksConfig()
	appConfig.Packs.Enabled[config.PackLifeSciences] = true

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
