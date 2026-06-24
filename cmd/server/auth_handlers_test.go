package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func TestResolveSessionRoleCapsAdminWithoutBootstrap(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN", "")
	hubSessions = hub.NewSessionManager()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/session", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	role := resolveSessionRole(r, "admin")
	if role != "member" {
		t.Fatalf("expected member, got %q", role)
	}
}

func TestResolveSessionRoleAllowsAdminWithBootstrap(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN", "bootstrap-test-secret")
	hubSessions = hub.NewSessionManager()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/session", nil)
	r.Header.Set("X-NJ-Bootstrap", "bootstrap-test-secret")
	role := resolveSessionRole(r, "admin")
	if role != "admin" {
		t.Fatalf("expected admin, got %q", role)
	}
}

func TestHandleAuthSessionCapsAdminRole(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN", "")
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	hubSessions = hub.NewSessionManager()

	body, _ := json.Marshal(map[string]string{"username": "Eve", "role": "admin"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/session", bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	handleAuthSession(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var sess hub.HubSession
	if err := json.NewDecoder(rec.Body).Decode(&sess); err != nil {
		t.Fatal(err)
	}
	if sess.Role != "member" {
		t.Fatalf("expected capped member role, got %q", sess.Role)
	}
}

func TestHandleAuthSessionAdminWithBootstrap(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_BOOTSTRAP_TOKEN", "bootstrap-test-secret")
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	hubSessions = hub.NewSessionManager()

	body, _ := json.Marshal(map[string]string{"username": "Admin", "role": "admin"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/session", bytes.NewReader(body))
	req.Header.Set("X-NJ-Bootstrap", "bootstrap-test-secret")
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	handleAuthSession(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var sess hub.HubSession
	if err := json.NewDecoder(rec.Body).Decode(&sess); err != nil {
		t.Fatal(err)
	}
	if sess.Role != "admin" {
		t.Fatalf("expected admin role, got %q", sess.Role)
	}
}
