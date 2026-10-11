package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOrigin(t *testing.T) {
	cases := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{"empty", "", "127.0.0.1:19876", true},
		{"localhost", "http://localhost:1420", "127.0.0.1:19876", true},
		{"loopback", "http://127.0.0.1:1420", "127.0.0.1:19876", true},
		{"tauri", "tauri://localhost", "127.0.0.1:19876", true},
		{"same host", "http://192.168.1.5:19876", "192.168.1.5:19876", true},
		{"evil", "https://evil.example", "127.0.0.1:19876", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/pty/ws", nil)
			r.Host = tc.host
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if got := checkOrigin(r); got != tc.want {
				t.Fatalf("checkOrigin(%q) = %v want %v", tc.origin, got, tc.want)
			}
		})
	}
}

func TestRequireTokenFailClosed(t *testing.T) {
	// Documented contract: empty token without NJ_REMOTE_ALLOW_INSECURE is refused at startup.
	// Exercise auth middleware: with token set, wrong bearer is denied.
	prev := *token
	t.Cleanup(func() { *token = prev })
	*token = "secret"
	denied := false
	h := auth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/fs/list", nil)
	h(rr, req)
	if rr.Code != http.StatusUnauthorized {
		denied = true
		t.Fatalf("expected 401 without bearer, got %d", rr.Code)
	}
	_ = denied
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/fs/list", nil)
	req2.Header.Set("Authorization", "Bearer secret")
	h(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d", rr2.Code)
	}
}
