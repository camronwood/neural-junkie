package filechange

import (
	"path/filepath"
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
