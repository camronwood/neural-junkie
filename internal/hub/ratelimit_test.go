package hub

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiterExemptsModelLibraryCatalog(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_RATE_LIMIT", "1")
	t.Setenv("NEURAL_JUNKIE_RATE_READ", "1")
	rl := NewRateLimiter()

	paths := []string{
		"/api/ollama/catalog",
		"/api/hf/catalog",
		"/api/ollama/install-status",
		"/api/ollama/library/search",
		"/api/ollama/library/tags",
	}
	for _, path := range paths {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.RemoteAddr = "127.0.0.1:1234"
			if !rl.Allow(req) {
				t.Fatalf("expected exempt path %s to bypass rate limit (attempt %d)", path, i+1)
			}
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	if !rl.Allow(req) {
		t.Fatal("first non-exempt request should be allowed")
	}
	if rl.Allow(req) {
		t.Fatal("second non-exempt request should be rate limited")
	}
}
