package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
)

func TestResolveListenAddr_defaultsLoopback(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_LISTEN_ALL", "")
	addr := resolveListenAddr(":18765", config.DefaultConfig())
	if addr != "127.0.0.1:18765" {
		t.Fatalf("expected 127.0.0.1:18765, got %q", addr)
	}
}

func TestResolveListenAddr_listenAll(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_LISTEN_ALL", "1")
	addr := resolveListenAddr(":18765", config.DefaultConfig())
	if addr != "0.0.0.0:18765" {
		t.Fatalf("expected 0.0.0.0:18765, got %q", addr)
	}
}

func TestCorsAllowsOrigin_localDev(t *testing.T) {
	if !corsAllowsOrigin("http://localhost:1420") {
		t.Fatal("expected localhost:1420")
	}
	if corsAllowsOrigin("https://evil.example") {
		t.Fatal("expected evil origin denied")
	}
}

func TestCorsAllowsOrigin_wildcardEnv(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_CORS_ANY", "1")
	if !corsAllowsOrigin("https://evil.example") {
		t.Fatal("expected any origin with CORS_ANY")
	}
}

func TestCorsMiddlewareAddsHeadersOnRateLimit(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_RATE_LIMIT", "1")
	t.Setenv("NEURAL_JUNKIE_RATE_MUTATE", "1")
	oldLimiter := apiRateLimiter
	apiRateLimiter = hub.NewRateLimiter()
	t.Cleanup(func() { apiRateLimiter = oldLimiter })

	handler := corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/runbooks", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("Origin", "http://localhost:1420")
		rec := httptest.NewRecorder()
		handler(rec, req)
		if i == 1 {
			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("expected 429, got %d", rec.Code)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:1420" {
				t.Fatalf("CORS origin header = %q", got)
			}
		}
	}
}

func TestLocalOnlyMiddleware(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	called := false
	h := localOnly(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/send", nil)
	req.RemoteAddr = "203.0.113.8:1234"
	h(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if called {
		t.Fatal("handler should not run")
	}
}
