package filechange

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecutorCreateEditDeleteMove(t *testing.T) {
	root := t.TempDir()
	exec := NewFileChangeExecutor(root)

	createPath := filepath.Join(root, "subdir", "file.txt")
	if err := exec.ExecuteFileChange(&FileChange{
		Operation:  FileOperationCreate,
		FilePath:   createPath,
		NewContent: "version 1",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if content, err := exec.GetFileContent(createPath); err != nil || content != "version 1" {
		t.Fatalf("read after create: %q %v", content, err)
	}

	if err := exec.ExecuteFileChange(&FileChange{
		Operation:  FileOperationEdit,
		FilePath:   createPath,
		NewContent: "version 2",
	}); err != nil {
		t.Fatalf("edit: %v", err)
	}

	moveSrc := filepath.Join(root, "move-me.txt")
	if err := os.WriteFile(moveSrc, []byte("mv"), 0644); err != nil {
		t.Fatal(err)
	}
	moveDst := filepath.Join(root, "moved", "here.txt")
	if err := exec.ExecuteFileChange(&FileChange{
		Operation: FileOperationMove,
		FilePath:  moveSrc,
		OldPath:   moveSrc,
		NewPath:   moveDst,
	}); err != nil {
		t.Fatalf("move: %v", err)
	}
	if _, err := os.Stat(moveSrc); !os.IsNotExist(err) {
		t.Fatal("source should be gone after move")
	}

	if err := exec.ExecuteFileChange(&FileChange{
		Operation: FileOperationDelete,
		FilePath:  moveDst,
	}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(moveDst); !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}
}

func TestExecutorRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	exec := NewFileChangeExecutor(root)
	outside := filepath.Join(root, "..", "outside.txt")
	err := exec.ExecuteFileChange(&FileChange{
		Operation:  FileOperationCreate,
		FilePath:   outside,
		NewContent: "nope",
	})
	if err == nil {
		t.Fatal("expected path validation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "directory traversal") && !strings.Contains(msg, "path outside workspace") {
		t.Fatalf("expected traversal/outside error, got %v", err)
	}
}

func TestExecutorRejectsPrefixSibling(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws")
	sibling := filepath.Join(t.TempDir(), "ws_evil")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sibling, 0755); err != nil {
		t.Fatal(err)
	}
	exec := NewFileChangeExecutor(root)
	outside := filepath.Join(sibling, "oops.txt")
	err := exec.ExecuteFileChange(&FileChange{
		Operation:  FileOperationCreate,
		FilePath:   outside,
		NewContent: "nope",
	})
	if err == nil {
		t.Fatal("expected prefix sibling rejection")
	}
	if !strings.Contains(err.Error(), "path outside workspace") {
		t.Fatalf("expected outside workspace error, got %v", err)
	}
}

func TestExecutorGetFileDiff(t *testing.T) {
	root := t.TempDir()
	exec := NewFileChangeExecutor(root)
	diff, err := exec.GetFileDiff("line one\n", "line one\nline two\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "+line two") {
		t.Fatalf("expected diff hunk, got %q", diff)
	}
	same, err := exec.GetFileDiff("same", "same")
	if err != nil || same != "No changes" {
		t.Fatalf("same content diff: %q %v", same, err)
	}
}

func TestFileChangeHelpers(t *testing.T) {
	fc := &FileChange{
		Operation: FileOperationMove,
		OldPath:   "/a",
		NewPath:   "/b",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if !fc.IsMoveOperation() || fc.IsDeleteOperation() {
		t.Fatal("move/delete helpers")
	}
	if fc.GetDisplayPath() != "/a → /b" {
		t.Fatal(fc.GetDisplayPath())
	}
	if fc.GetTimeRemaining() <= 0 {
		t.Fatal("expected time remaining")
	}
	fc.ExpiresAt = time.Now().Add(-time.Hour)
	if !fc.IsExpired() || fc.GetTimeRemaining() != 0 {
		t.Fatal("expired helpers")
	}
}
