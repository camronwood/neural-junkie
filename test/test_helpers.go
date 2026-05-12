package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/repo"
)

// useIsolatedRepoStorage redirects repo storage to a temp directory for the
// duration of the test, preventing test artifacts from polluting ~/.neural-junkie/repos.
// Call this at the start of any test that creates repo agents or uses repo.NewStorage().
func useIsolatedRepoStorage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("NEURAL_JUNKIE_REPO_DIR", dir)
	return dir
}

// cleanupRepoAgentCache registers a t.Cleanup to remove the repo agent cache for repoPath.
func cleanupRepoAgentCache(t *testing.T, repoPath string) {
	t.Helper()
	t.Cleanup(func() {
		cleanupRepoAgentCacheImmediate(t, repoPath)
	})
}

// cleanupRepoAgentCacheImmediate immediately cleans up repository agent cache entries
func cleanupRepoAgentCacheImmediate(t *testing.T, repoPath string) {
	t.Helper()

	storage, err := repo.NewStorage()
	if err != nil {
		t.Logf("Warning: Failed to create storage for cleanup: %v", err)
		return
	}

	cacheKey, err := storage.GetCacheKeyForPath(repoPath)
	if err != nil {
		t.Logf("Warning: Failed to get cache key for cleanup: %v", err)
		return
	}

	if err := storage.DeleteIndex(cacheKey); err != nil {
		t.Logf("Warning: Failed to delete cache entry for %s: %v", repoPath, err)
	} else {
		t.Logf("Cleaned up repo agent cache for: %s", repoPath)
	}
}

// cleanupAllTestCaches removes all cache entries that appear to be from tests
// This is a nuclear option for comprehensive cleanup
func cleanupAllTestCaches(t *testing.T) {
	t.Helper()

	t.Cleanup(func() {
		repoStorage, err := repo.NewStorage()
		if err == nil {
			cacheKeys, err := repoStorage.ListIndexes()
			if err == nil {
				for _, cacheKey := range cacheKeys {
					metadata, err := repoStorage.LoadMetadata(cacheKey)
					if err == nil && isTestPath(metadata.Path) {
						if err := repoStorage.DeleteIndex(cacheKey); err != nil {
							t.Logf("Warning: Failed to delete test repo cache %s: %v", cacheKey, err)
						} else {
							t.Logf("Cleaned up test repo cache: %s", metadata.Path)
						}
					}
				}
			}
		}
	})
}

// isTestPath checks if a path appears to be from a test (temporary directory)
func isTestPath(path string) bool {
	return strings.Contains(path, "/tmp/") ||
		strings.Contains(path, "/T/") ||
		strings.Contains(path, "TestRepoAgent") ||
		strings.Contains(path, "TestRepositoryAgent") ||
		strings.Contains(path, "TestHelperAgent")
}

// verifyNoTestCachesRemain checks that no test-related caches remain after test execution
// This can be called at the end of test suites to verify cleanup worked
func verifyNoTestCachesRemain(t *testing.T) {
	t.Helper()

	repoStorage, err := repo.NewStorage()
	if err == nil {
		cacheKeys, err := repoStorage.ListIndexes()
		if err == nil {
			for _, cacheKey := range cacheKeys {
				metadata, err := repoStorage.LoadMetadata(cacheKey)
				if err == nil && isTestPath(metadata.Path) {
					t.Errorf("Test repository cache still exists: %s (cache key: %s)", metadata.Path, cacheKey)
				}
			}
		}
	}
}

// createTestRepoPath creates a test repository path in a temporary directory
// and registers cleanup for it
func createTestRepoPath(t *testing.T, repoName string) string {
	t.Helper()

	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, repoName)

	if err := os.MkdirAll(testRepoPath, 0755); err != nil {
		t.Fatalf("Failed to create test repository directory: %v", err)
	}

	cleanupRepoAgentCache(t, testRepoPath)

	return testRepoPath
}
