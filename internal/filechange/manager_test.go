package filechange

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func testAgent() protocol.AgentInfo {
	return protocol.AgentInfo{ID: "a1", Name: "TestAgent", Type: protocol.AgentTypeBackend}
}

func newTestManager(t *testing.T) (*FileChangeManager, string) {
	t.Helper()
	root := t.TempDir()
	exec := NewFileChangeExecutor(root)
	mgr := NewFileChangeManager(exec)
	t.Cleanup(func() { mgr.Stop() })
	return mgr, root
}

func TestProposeAndApproveCreate(t *testing.T) {
	mgr, root := newTestManager(t)
	target := filepath.Join(root, "new.txt")

	change, err := mgr.ProposeFileChange(FileOperationCreate, target, "", "", "", "hello", testAgent(), "general")
	if err != nil {
		t.Fatal(err)
	}
	if change.Status != FileChangeStatusPending {
		t.Fatalf("expected pending, got %s", change.Status)
	}

	approved, err := mgr.ApproveFileChange(change.ID, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != FileChangeStatusApproved {
		t.Fatalf("expected approved, got %s", approved.Status)
	}
	if mgr.GetPendingCount() != 0 {
		t.Fatal("expected no pending after approve")
	}
}

func TestRejectFileChange(t *testing.T) {
	mgr, root := newTestManager(t)
	target := filepath.Join(root, "reject-me.txt")

	change, err := mgr.ProposeFileChange(FileOperationCreate, target, "", "", "", "x", testAgent(), "general")
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := mgr.RejectFileChange(change.ID, "user-1", "not needed")
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != FileChangeStatusRejected || rejected.Reason != "not needed" {
		t.Fatalf("unexpected reject state: %+v", rejected)
	}
}

func TestProposeMoveRequiresPaths(t *testing.T) {
	mgr, _ := newTestManager(t)
	_, err := mgr.ProposeFileChange(FileOperationMove, "ignored", "", "", "", "", testAgent(), "general")
	if err == nil {
		t.Fatal("expected validation error for move without old/new paths")
	}
}

func TestExpiredChangeRejected(t *testing.T) {
	mgr, root := newTestManager(t)
	target := filepath.Join(root, "expired.txt")
	change, err := mgr.ProposeFileChange(FileOperationCreate, target, "", "", "", "x", testAgent(), "general")
	if err != nil {
		t.Fatal(err)
	}
	mgr.mu.Lock()
	change.ExpiresAt = time.Now().Add(-time.Minute)
	mgr.mu.Unlock()

	_, err = mgr.ApproveFileChange(change.ID, "user-1")
	if err == nil {
		t.Fatal("expected expired error")
	}
}

func TestListPendingFileChanges(t *testing.T) {
	mgr, root := newTestManager(t)
	p1 := filepath.Join(root, "a.txt")
	p2 := filepath.Join(root, "b.txt")
	if _, err := mgr.ProposeFileChange(FileOperationCreate, p1, "", "", "", "1", testAgent(), "general"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.ProposeFileChange(FileOperationCreate, p2, "", "", "", "2", testAgent(), "general"); err != nil {
		t.Fatal(err)
	}
	if len(mgr.ListPendingFileChanges("user")) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(mgr.ListPendingFileChanges("user")))
	}
}

func TestApproveEditRejectsStaleBase(t *testing.T) {
	mgr, root := newTestManager(t)
	target := filepath.Join(root, "app.txt")
	if err := os.WriteFile(target, []byte("version one"), 0o644); err != nil {
		t.Fatal(err)
	}
	change, err := mgr.ProposeFileChange(
		FileOperationEdit, target, "", "", "version one", "agent version", testAgent(), "general",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("user version"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.ApproveFileChange(change.ID, "user-1"); err == nil || !strings.Contains(err.Error(), "stale edit") {
		t.Fatalf("expected stale edit rejection, got %v", err)
	}
	if change.Status != FileChangeStatusStale {
		t.Fatalf("status = %q, want stale", change.Status)
	}
	if _, err := mgr.ApproveFileChange(change.ID, "user-1"); err == nil || !strings.Contains(err.Error(), "already processed") {
		t.Fatalf("expected terminal stale proposal to reject another action, got %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "user version" {
		t.Fatalf("stale approval clobbered user content: %q", got)
	}
}

func TestBoundExecutionContextCannotBeRetargeted(t *testing.T) {
	mgr, rootOne := newTestManager(t)
	rootTwo := t.TempDir()
	target := filepath.Join(rootOne, "app.txt")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	change, err := mgr.ProposeFileChange(
		FileOperationEdit, target, "", "", "old", "new", testAgent(), "general",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.BindExecutionContext(change.ID, rootOne, nil); err != nil {
		t.Fatal(err)
	}
	// Simulate another workspace registration mutating the legacy shared executor.
	mgr.GetExecutor().SetWorkspaceRoot(rootTwo)
	if _, err := mgr.ApproveFileChange(change.ID, "user-1"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new" {
		t.Fatalf("bound executor wrote wrong workspace: %q", got)
	}
}
