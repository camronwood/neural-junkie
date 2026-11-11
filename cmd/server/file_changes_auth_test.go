package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/hub/authstore"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func setupFileChangeAuthHub(t *testing.T) (*filechange.FileChangeManager, string) {
	t.Helper()
	t.Setenv("NEURAL_JUNKIE_HUB_TOKEN", "")
	chatHub = hub.NewHub()
	hubSessions = hub.NewSessionManager()
	root := t.TempDir()
	mgr := chatHub.GetFileChangeManager()
	mgr.GetExecutor().SetWorkspaceRoot(root)
	return mgr, root
}

func proposeCreateChange(t *testing.T, mgr *filechange.FileChangeManager, root, name string) *filechange.FileChange {
	t.Helper()
	target := filepath.Join(root, name)
	change, err := mgr.ProposeFileChange(
		filechange.FileOperationCreate,
		target,
		"",
		"",
		"",
		"hello-"+name,
		protocol.AgentInfo{ID: "a1", Name: "Agent", Type: protocol.AgentTypeBackend},
		"general",
	)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func TestFileChangeApproveRequiresSessionWhenAuthRequired(t *testing.T) {
	mgr, root := setupFileChangeAuthHub(t)
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "")
	change := proposeCreateChange(t, mgr, root, "auth-required.txt")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/file-changes/approve/"+change.ID, nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handleApproveFileChange(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestFileChangeApproveUsesSessionUsername(t *testing.T) {
	mgr, root := setupFileChangeAuthHub(t)
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "")
	sess := hubSessions.CreateSession("Camron", "member")
	change := proposeCreateChange(t, mgr, root, "session-user.txt")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/file-changes/approve/"+change.ID+"?user_id=default", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-NJ-Session", sess.Token)
	handleApproveFileChange(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out filechange.FileChange
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Status != filechange.FileChangeStatusApproved {
		t.Fatalf("expected approved, got %s", out.Status)
	}
	data, err := os.ReadFile(filepath.Join(root, "session-user.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello-session-user.txt" {
		t.Fatalf("unexpected file content %q", data)
	}
}

func TestFileChangeApproveViewerForbidden(t *testing.T) {
	mgr, root := setupFileChangeAuthHub(t)
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "")
	sess := hubSessions.CreateSession("readonly", "viewer")
	change := proposeCreateChange(t, mgr, root, "viewer-block.txt")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/file-changes/approve/"+change.ID, nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-NJ-Session", sess.Token)
	handleApproveFileChange(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestFileChangeApproveAPIKeyMember(t *testing.T) {
	mgr, root := setupFileChangeAuthHub(t)
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "")
	dir := t.TempDir()
	store, err := authstore.Open(filepath.Join(dir, "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	authKeyStore = store
	raw, _, err := store.CreateAPIKey("ci", authstore.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	change := proposeCreateChange(t, mgr, root, "api-key.txt")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/file-changes/approve/"+change.ID, nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer "+raw)
	handleApproveFileChange(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestFileChangeRejectRelaxedLocal(t *testing.T) {
	mgr, root := setupFileChangeAuthHub(t)
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "1")
	change := proposeCreateChange(t, mgr, root, "relaxed-reject.txt")

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"reason":"nope"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/file-changes/reject/"+change.ID, body)
	req.RemoteAddr = "127.0.0.1:1234"
	handleRejectFileChange(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 under relaxed local, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out filechange.FileChange
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Status != filechange.FileChangeStatusRejected {
		t.Fatalf("expected rejected, got %s", out.Status)
	}
}

func TestFileChangeRejectRequiresSessionWhenAuthRequired(t *testing.T) {
	mgr, root := setupFileChangeAuthHub(t)
	t.Setenv("NEURAL_JUNKIE_AUTH_REQUIRED", "1")
	t.Setenv("NEURAL_JUNKIE_RELAXED_LOCAL", "")
	change := proposeCreateChange(t, mgr, root, "reject-auth.txt")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/file-changes/reject/"+change.ID, strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:1234"
	handleRejectFileChange(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSessionActorID(t *testing.T) {
	if got := sessionActorID(nil); got != "" {
		t.Fatalf("nil => empty, got %q", got)
	}
	if got := sessionActorID(&hub.HubSession{Username: "Camron", UserID: "camron"}); got != "Camron" {
		t.Fatalf("prefer username, got %q", got)
	}
	if got := sessionActorID(&hub.HubSession{UserID: "local"}); got != "local" {
		t.Fatalf("fallback user id, got %q", got)
	}
}
