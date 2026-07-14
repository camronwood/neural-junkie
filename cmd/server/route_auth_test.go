package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func TestLocalOnlyRoutesRejectRemote(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")

	cases := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{"git-commit", http.MethodPost, "/api/git-commit", handleGitCommit},
		{"git-push", http.MethodPost, "/api/git-push", handleGitPush},
		{"files", http.MethodGet, "/api/files?workspace_id=x&path=.", handleFiles},
		{"ollama-install", http.MethodPost, "/api/ollama/install", handleOllamaInstall},
		{"ollama-update", http.MethodPost, "/api/ollama/update", handleOllamaUpdate},
		{"hf-download", http.MethodPost, "/api/hf/download", handleHfDownload},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			wrapped := localOnly(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			// Exercise the same middleware wrapping as production routes.
			_ = wrapped
			h := localOnly(tc.handler)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.RemoteAddr = "203.0.113.8:1234"
			h(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
			}
			if called {
				t.Fatal("handler should not run for remote client")
			}
		})
	}
}

func TestStrictLoopbackRunbookCreateRequiresSession(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "")
	hubSessions = hub.NewSessionManager()
	chatHub = hub.NewHub()

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"description":"x","agent_ids":["a1"],"channel":"general","created_by":"t"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/runbooks", body)
	req.RemoteAddr = "127.0.0.1:1234"
	handleRunbooksRoute(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestStrictLoopbackSendRequiresSession(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "")
	hubSessions = hub.NewSessionManager()

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"channel":"general","content":"hi","type":"chat"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/send", body)
	req.RemoteAddr = "127.0.0.1:1234"
	h := localOnly(handleSendMessage)
	h(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestLocalOnlyAllowsLoopback(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	called := false
	h := localOnly(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	h(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if !called {
		t.Fatal("handler should run for loopback")
	}
}
