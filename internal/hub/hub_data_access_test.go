package hub

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveHubDataPathBlocksTraversal(t *testing.T) {
	_, _, err := resolveHubDataPath("../etc/passwd")
	if err == nil {
		t.Fatal("expected error for traversal")
	}
}

func TestReadHubDataForAgent_FileAndDir(t *testing.T) {
	root := t.TempDir()
	home := root
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", home)
	_ = oldHome

	dataDir := filepath.Join(home, ".neural-junkie")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "config.json"), []byte(`{"x":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "last-session.json"), []byte(`{"saved_at":"x","channels":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadHubDataForAgent([]HubDataReadTarget{
		{Kind: "file", RelativePath: "last-session.json"},
		{Kind: "directory", RelativePath: ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) < 2 {
		t.Fatalf("expected multiple entries, got %d", len(got.Entries))
	}
}
