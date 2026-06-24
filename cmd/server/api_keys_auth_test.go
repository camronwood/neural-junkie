package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/hub/authstore"
)

func TestAPIKeyViewerBlockedFromMutation(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "1")
	dir := t.TempDir()
	store, err := authstore.Open(filepath.Join(dir, "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	authKeyStore = store

	rawViewer, _, err := store.CreateAPIKey("readonly", authstore.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/send", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer "+rawViewer)
	rec := httptest.NewRecorder()
	sess, ok := ensureMutationAccess(rec, req, "general")
	if ok {
		t.Fatalf("viewer key should be blocked, got session %+v", sess)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestAPIKeyMemberAllowedToMutate(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "1")
	dir := t.TempDir()
	store, err := authstore.Open(filepath.Join(dir, "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	authKeyStore = store

	rawMember, _, err := store.CreateAPIKey("ci", authstore.RoleMember)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/send", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer "+rawMember)
	rec := httptest.NewRecorder()
	sess, ok := ensureMutationAccess(rec, req, "general")
	if !ok {
		t.Fatalf("member key should mutate, body=%s", rec.Body.String())
	}
	if sess == nil || sess.Role != "member" {
		t.Fatalf("expected member session, got %+v", sess)
	}
}

func TestSessionViewerCannotMutate(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	hubSessions = hub.NewSessionManager()
	sess := hubSessions.CreateSession("viewer-user", "viewer")

	req := httptest.NewRequest(http.MethodPost, "/api/send", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-NJ-Session", sess.Token)
	rec := httptest.NewRecorder()
	got, ok := ensureMutationAccess(rec, req, "general")
	if ok {
		t.Fatalf("viewer session should be blocked, got %+v", got)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
