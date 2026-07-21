package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/artifacts"
)

func TestExpectedArtifactRevision(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/artifacts/a", nil)
	req.Header.Set("If-Match", `"7"`)
	rec := httptest.NewRecorder()
	revision, ok := expectedArtifactRevision(rec, req, 0)
	if !ok || revision != 7 {
		t.Fatalf("revision=%d ok=%v status=%d", revision, ok, rec.Code)
	}
}

func TestArtifactExportContentPrefersTextFallback(t *testing.T) {
	fallback, _ := json.Marshal("# Report")
	content, err := artifactExportContent(&artifacts.Artifact{
		ID:      "a",
		Payload: json.RawMessage(`{"value":1}`),
		Fallback: &artifacts.Fallback{
			MediaType: "text/markdown",
			Data:      fallback,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Report" {
		t.Fatalf("content=%q", content)
	}
}
