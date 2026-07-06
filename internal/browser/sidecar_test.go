package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSidecarClientScreenshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/browser/screenshot" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"png_b64": "abc",
			"width":   1280,
			"height":  800,
			"url":     "http://localhost/",
		})
	}))
	defer srv.Close()

	client := NewSidecarClient(func() string { return srv.URL })
	out, err := client.Screenshot(context.Background(), map[string]any{"url": "http://localhost/"})
	if err != nil {
		t.Fatal(err)
	}
	if out["png_b64"] != "abc" {
		t.Fatalf("unexpected response: %#v", out)
	}
}

func TestSidecarClientNotRunning(t *testing.T) {
	client := NewSidecarClient(func() string { return "" })
	_, err := client.Screenshot(context.Background(), map[string]any{"url": "http://localhost/"})
	if err == nil {
		t.Fatal("expected error when sidecar not running")
	}
}
