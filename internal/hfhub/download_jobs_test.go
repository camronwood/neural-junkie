package hfhub

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureDownloadStartedAndWatch(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	repoID := "aaditya/OpenBioLLM-Llama3-8B-GGUF"
	filename := "openbiollm-llama3-8b.Q4_K_M.gguf"
	dest := m.filePath(repoID, filename)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var last DownloadProgress
	err = m.WatchDownload(ctx, repoID, filename, func(p DownloadProgress) {
		last = p
	})
	if err != nil {
		t.Fatal(err)
	}
	if last.Status != "success" {
		t.Fatalf("status=%s", last.Status)
	}
}

func TestWatchReturnsOnClientCancelJobContinues(t *testing.T) {
	// FileReady false + no real HF call: EnsureDownloadStarted spawns job that will fail quickly without network in test - skip heavy test
	t.Skip("integration: requires network for full download job")
}
