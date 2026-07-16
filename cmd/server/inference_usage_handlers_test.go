package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/inference"
)

func TestHandleInferenceUsageGetReset(t *testing.T) {
	dir := t.TempDir()
	store, err := inference.NewStatsStore(filepath.Join(dir, "stats.json"))
	if err != nil {
		t.Fatal(err)
	}
	inference.SetDefaultStore(store)
	t.Cleanup(func() { inference.SetDefaultStore(nil) })

	store.Record(inference.TurnRecord{PromptTokens: 100, CompletionTokens: 50, Calls: 1})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/inference/usage", nil)
	handleInferenceUsage(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", rec.Code)
	}
	var sum inference.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &sum); err != nil {
		t.Fatal(err)
	}
	if sum.Totals.PromptTokens != 100 {
		t.Fatalf("prompt tokens = %d", sum.Totals.PromptTokens)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/inference/usage", nil)
	handleInferenceUsage(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d", rec.Code)
	}
	sum = store.Summary()
	if sum.Totals.PromptTokens != 0 {
		t.Fatalf("after reset prompt tokens = %d", sum.Totals.PromptTokens)
	}
}
