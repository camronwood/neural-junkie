package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleSystemHardware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/system/hardware", nil)
	rec := httptest.NewRecorder()
	handleSystemHardware(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var snap struct {
		TotalMemoryGB int    `json:"total_memory_gb"`
		Tier          string `json:"tier"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.TotalMemoryGB < 1 {
		t.Fatalf("total_memory_gb = %d", snap.TotalMemoryGB)
	}
	if snap.Tier == "" {
		t.Fatal("expected tier")
	}
}

func TestHandleOllamaLibraryLookup(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/ollama/library/lookup?name=qwen2.5-coder:14b", nil)
	rec := httptest.NewRecorder()
	handleOllamaLibraryLookup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var row struct {
		Name            string  `json:"name"`
		EstimatedRAMGB  int     `json:"estimated_ram_gb"`
		EstimatedDiskGB float64 `json:"estimated_disk_gb"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &row); err != nil {
		t.Fatal(err)
	}
	if row.Name != "qwen2.5-coder:14b" {
		t.Fatalf("name %q", row.Name)
	}
	if row.EstimatedRAMGB != 16 {
		t.Fatalf("ram %d", row.EstimatedRAMGB)
	}

	missing := httptest.NewRequest(http.MethodGet, "/api/ollama/library/lookup?name=not-real", nil)
	mrec := httptest.NewRecorder()
	handleOllamaLibraryLookup(mrec, missing)
	if mrec.Body.String() != "null\n" && mrec.Body.String() != "null" {
		// json encoder may append newline
		if mrec.Body.String() != "null\n" {
			t.Fatalf("missing body %q", mrec.Body.String())
		}
	}
}

func TestHandleOllamaLibraryLookupMissingName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/ollama/library/lookup", nil)
	rec := httptest.NewRecorder()
	handleOllamaLibraryLookup(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}
