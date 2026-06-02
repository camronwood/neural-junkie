package filechange

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFileChangeContent_stripsTaskStatusLines(t *testing.T) {
	in := "# findings.md\n\nSummary of README.\n\n[TASK_STATUS: completed]\n"
	got := SanitizeFileChangeContent(in)
	if strings.Contains(strings.ToLower(got), "task_status") {
		t.Fatalf("expected TASK_STATUS stripped, got %q", got)
	}
	if !strings.Contains(got, "Summary of README") {
		t.Fatalf("expected body preserved, got %q", got)
	}
}

func TestSanitizeFileChangeContent_stripsPlainTaskStatusLine(t *testing.T) {
	in := "# Findings\n\n- one\n\nTASK_STATUS: completed\n"
	got := SanitizeFileChangeContent(in)
	if strings.Contains(strings.ToLower(got), "task_status") {
		t.Fatalf("expected TASK_STATUS stripped, got %q", got)
	}
}

func TestExecutorCreate_stripsTaskStatusBeforeWrite(t *testing.T) {
	root := t.TempDir()
	exec := NewFileChangeExecutor(root)
	path := filepath.Join(root, "collabs", "abc", "findings.md")
	body := "operation: create\n\n# findings.md\n\nGrounded notes.\n\n[TASK_STATUS: completed]\n"
	if err := exec.ExecuteFileChange(&FileChange{
		Operation:  FileOperationCreate,
		FilePath:   path,
		NewContent: body,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(strings.ToLower(text), "task_status") {
		t.Fatalf("file on disk contains TASK_STATUS: %q", text)
	}
	if !strings.Contains(text, "Grounded notes") {
		t.Fatalf("expected content preserved: %q", text)
	}
}
