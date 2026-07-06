package awssidecar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSidecarClientPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/aws/get-caller-identity" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"Account": "123456789012"})
	}))
	defer srv.Close()

	client := NewSidecarClient(func() string { return srv.URL })
	out, err := client.PostJSON(context.Background(), "/api/aws/get-caller-identity", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || !contains(out, "123456789012") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestSidecarClientNotRunning(t *testing.T) {
	client := NewSidecarClient(func() string { return "" })
	_, err := client.PostJSON(context.Background(), "/api/aws/list-s3-buckets", nil)
	if err == nil {
		t.Fatal("expected error when sidecar not running")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
