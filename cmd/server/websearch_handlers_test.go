package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestHandleWebSearchConfigGet(t *testing.T) {
	appConfig = &config.Config{
		WebSearch: config.WebSearchConfig{
			Enabled:  true,
			Provider: "brave",
			APIKey:   "secret-key",
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/web-search/config", nil)
	rec := httptest.NewRecorder()
	handleWebSearchConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["ready"] != true {
		t.Fatalf("ready = %v", resp["ready"])
	}
}

func TestHandleWebSearchConfigPut(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	appConfig = config.DefaultConfig()
	appConfig.WebSearch.Enabled = false

	body, _ := json.Marshal(map[string]interface{}{
		"enabled": true,
		"api_key": "new-key",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/web-search/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleWebSearchConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !appConfig.WebSearch.Enabled || appConfig.WebSearch.APIKey != "new-key" {
		t.Fatalf("config not updated: %+v", appConfig.WebSearch)
	}
}
