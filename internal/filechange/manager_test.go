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

func TestProposeAndApproveFileChangeRequest(t *testing.T) {
	mgr, root := newTestManager(t)
	a := filepath.Join(root, "a.txt")
	b := filepath.Join(root, "b.txt")
	if err := os.WriteFile(a, []byte("old-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("old-b"), 0o644); err != nil {
		t.Fatal(err)
	}

	req, err := mgr.ProposeFileChangeRequest([]*FileChange{
		{Operation: FileOperationEdit, FilePath: a, OldContent: "old-a", NewContent: "new-a"},
		{Operation: FileOperationEdit, FilePath: b, OldContent: "old-b", NewContent: "new-b"},
	}, testAgent(), "general")
	if err != nil {
		t.Fatal(err)
	}
	if req.ID == "" || len(req.Changes) != 2 {
		t.Fatalf("unexpected request: %+v", req)
	}
	for _, c := range req.Changes {
		if c.Metadata["request_id"] != req.ID {
			t.Fatalf("change missing request_id: %+v", c.Metadata)
		}
	}
	if mgr.GetPendingCount() != 2 {
		t.Fatalf("expected 2 pending, got %d", mgr.GetPendingCount())
	}

	approved, err := mgr.ApproveFileChangeRequest(req.ID, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != FileChangeStatusApproved {
		t.Fatalf("expected approved request, got %s", approved.Status)
	}
	for _, path := range []string{a, b} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(data), "new-") {
			t.Fatalf("file %s not updated: %q", path, data)
		}
	}
	if mgr.GetPendingCount() != 0 {
		t.Fatal("expected no pending after batch approve")
	}
}

func TestRejectFileChangeRequest(t *testing.T) {
	mgr, root := newTestManager(t)
	target := filepath.Join(root, "batch-reject.txt")
	req, err := mgr.ProposeFileChangeRequest([]*FileChange{
		{Operation: FileOperationCreate, FilePath: target, NewContent: "x"},
	}, testAgent(), "general")
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := mgr.RejectFileChangeRequest(req.ID, "user-1", "nope")
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != FileChangeStatusRejected {
		t.Fatalf("expected rejected, got %s", rejected.Status)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("rejected create should not write file")
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

func TestApproveFileChangeRequestRollsBackOnMidBatchFailure(t *testing.T) {
	mgr, root := newTestManager(t)
	a := filepath.Join(root, "a.txt")
	b := filepath.Join(root, "b.txt")
	if err := os.WriteFile(a, []byte("old-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("old-b"), 0o644); err != nil {
		t.Fatal(err)
	}

	req, err := mgr.ProposeFileChangeRequest([]*FileChange{
		{Operation: FileOperationEdit, FilePath: a, OldContent: "old-a", NewContent: "new-a"},
		{Operation: FileOperationEdit, FilePath: b, OldContent: "old-b", NewContent: "new-b"},
	}, testAgent(), "general")
	if err != nil {
		t.Fatal(err)
	}

	// Make the second edit fail by changing the file after proposal (stale).
	if err := os.WriteFile(b, []byte("user-changed-b"), 0o644); err != nil {
		t.Fatal(err)
	}

	failed, err := mgr.ApproveFileChangeRequest(req.ID, "user-1")
	if err == nil {
		t.Fatal("expected mid-batch failure")
	}
	if failed == nil || failed.Status != FileChangeStatusFailed {
		t.Fatalf("expected failed request, got %+v", failed)
	}

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotA) != "old-a" {
		t.Fatalf("first edit should be rolled back, got %q", gotA)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotB) != "user-changed-b" {
		t.Fatalf("failed member path should remain user content, got %q", gotB)
	}

	if len(failed.Changes) != 2 {
		t.Fatalf("expected 2 members, got %d", len(failed.Changes))
	}
	if failed.Changes[0].Status != FileChangeStatusRolledBack {
		t.Fatalf("first member status = %q, want rolled_back (reason=%q)", failed.Changes[0].Status, failed.Changes[0].Reason)
	}
	if failed.Changes[1].Status != FileChangeStatusStale && failed.Changes[1].Status != FileChangeStatusFailed {
		t.Fatalf("second member status = %q, want stale or failed", failed.Changes[1].Status)
	}
	if !strings.Contains(failed.Changes[0].Reason, "rolled back") {
		t.Fatalf("rolled_back reason missing: %q", failed.Changes[0].Reason)
	}
}

func TestApproveFileChangeRequestRollsBackCreateAndDelete(t *testing.T) {
	mgr, root := newTestManager(t)
	existing := filepath.Join(root, "keep.txt")
	created := filepath.Join(root, "created.txt")
	missingEdit := filepath.Join(root, "missing-edit.txt")
	if err := os.WriteFile(existing, []byte("keep-me"), 0o644); err != nil {
		t.Fatal(err)
	}

	req, err := mgr.ProposeFileChangeRequest([]*FileChange{
		{Operation: FileOperationCreate, FilePath: created, NewContent: "brand-new"},
		{Operation: FileOperationDelete, FilePath: existing},
		// Third member fails: edit of a file that does not exist.
		{Operation: FileOperationEdit, FilePath: missingEdit, OldContent: "x", NewContent: "y"},
	}, testAgent(), "general")
	if err != nil {
		t.Fatal(err)
	}

	failed, err := mgr.ApproveFileChangeRequest(req.ID, "user-1")
	if err == nil {
		t.Fatal("expected mid-batch failure")
	}
	if failed == nil || failed.Status != FileChangeStatusFailed {
		t.Fatalf("expected failed request, got %+v", failed)
	}

	if _, err := os.Stat(created); !os.IsNotExist(err) {
		t.Fatalf("created file should be removed on rollback, stat err=%v", err)
	}
	got, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "keep-me" {
		t.Fatalf("deleted file should be restored, got %q", got)
	}

	statuses := make([]FileChangeStatus, 0, len(failed.Changes))
	for _, c := range failed.Changes {
		statuses = append(statuses, c.Status)
	}
	if statuses[0] != FileChangeStatusRolledBack || statuses[1] != FileChangeStatusRolledBack {
		t.Fatalf("expected first two rolled_back, got %v", statuses)
	}
	if statuses[2] != FileChangeStatusFailed {
		t.Fatalf("expected third failed, got %v", statuses)
	}
}

