package hub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/repo"
)

func TestDeleteCachedRepoAgentByPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEURAL_JUNKIE_REPO_DIR", dir)

	repoPath := filepath.Join(dir, "sample-repo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatal(err)
	}

	storage, err := repo.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	key, err := storage.GetCacheKeyForPath(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveMetadata(key, &repo.RepoMetadata{
		Path:       repoPath,
		CacheKey:   key,
		AgentNames: []string{"SampleExpert"},
	}); err != nil {
		t.Fatal(err)
	}

	deleted, err := DeleteCachedAgent("repo", "", repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected cache delete")
	}
	if storage.IndexExists(key) {
		t.Fatal("index should be removed")
	}
}

func TestDeleteCachedCLIAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	storage, err := agent.NewCLIAgentStorage()
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Save(agent.CLIAgentRecord{
		Type:    "cursor",
		Name:    "CursorTest",
		WorkDir: home,
	}); err != nil {
		t.Fatal(err)
	}

	deleted, err := DeleteCachedAgent("cli", "CursorTest", "")
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected CLI record delete")
	}
	records, _ := storage.ListWithMetadata()
	for _, r := range records {
		if r["name"] == "CursorTest" {
			t.Fatal("record should be gone")
		}
	}
}
