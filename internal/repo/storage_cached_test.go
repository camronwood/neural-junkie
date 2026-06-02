package repo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetAllCachedReposUsesStoredAgentNames(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	home := dir
	_ = os.MkdirAll(filepath.Join(home, ".neural-junkie", "repos"), 0755)

	storage, err := NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	repoPath := filepath.Join(dir, "sample-app")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatal(err)
	}
	key, err := storage.GetCacheKeyForPath(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveMetadata(key, &RepoMetadata{
		Path:       repoPath,
		CacheKey:   key,
		AgentNames: []string{"my-custom-expert"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveIndex(key, &RepositoryIndex{
		Path:        repoPath,
		Name:        "sample-app",
		LastIndexed: time.Now(),
		FileCount:   3,
		TotalSize:   1024,
	}); err != nil {
		t.Fatal(err)
	}

	agents, err := storage.GetAllCachedRepos()
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 cached agent, got %d", len(agents))
	}
	if agents[0]["name"] != "my-custom-expert" {
		t.Fatalf("name = %v, want my-custom-expert", agents[0]["name"])
	}
}
