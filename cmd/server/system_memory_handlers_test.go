package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleSystemMemory(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/system/memory", nil)
	rec := httptest.NewRecorder()
	handleSystemMemory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var snap struct {
		TotalBytes  uint64  `json:"total_bytes"`
		UsedPercent float64 `json:"used_percent"`
		Tier        string `json:"tier"`
		Ollama      struct {
			Running      bool   `json:"running"`
			Endpoint     string `json:"endpoint"`
			LoadedModels []any  `json:"loaded_models"`
		} `json:"ollama"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.TotalBytes < 1 {
		t.Fatalf("total_bytes = %d", snap.TotalBytes)
	}
	if snap.UsedPercent < 0 || snap.UsedPercent > 100 {
		t.Fatalf("used_percent = %v", snap.UsedPercent)
	}
	if snap.Tier == "" {
		t.Fatal("expected tier")
	}
	if snap.Ollama.Endpoint == "" {
		t.Fatal("expected ollama endpoint")
	}
}

func TestHandleSystemMemoryMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/system/memory", nil)
	rec := httptest.NewRecorder()
	handleSystemMemory(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rec.Code)
	}
}
