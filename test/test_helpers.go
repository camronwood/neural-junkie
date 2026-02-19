package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
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

// cleanupHelperAgentCache cleans up helper agent cache entries for test agents
func cleanupHelperAgentCache(t *testing.T, agentName string) {
	t.Helper()

	// Register cleanup
	t.Cleanup(func() {
		// Get storage instance
		storage, err := agent.NewHelperAgentStorage()
		if err != nil {
			t.Logf("Warning: Failed to create helper storage for cleanup: %v", err)
			return
		}

		// Delete the helper agent config
		if err := storage.DeleteConfig(agentName); err != nil {
			t.Logf("Warning: Failed to delete helper agent config for %s: %v", agentName, err)
		} else {
			t.Logf("Cleaned up helper agent cache for: %s", agentName)
		}
	})
}

// cleanupAllTestCaches removes all cache entries that appear to be from tests
// This is a nuclear option for comprehensive cleanup
func cleanupAllTestCaches(t *testing.T) {
	t.Helper()

	t.Cleanup(func() {
		// Clean up repository agent caches
		repoStorage, err := repo.NewStorage()
		if err == nil {
			cacheKeys, err := repoStorage.ListIndexes()
			if err == nil {
				for _, cacheKey := range cacheKeys {
					// Load metadata to check if it's a test path
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

		// Clean up helper agent caches
		helperStorage, err := agent.NewHelperAgentStorage()
		if err == nil {
			configs, err := helperStorage.ListConfigs()
			if err == nil {
				for _, configName := range configs {
					// Check if this looks like a test agent
					if isTestAgentName(configName) {
						if err := helperStorage.DeleteConfig(configName); err != nil {
							t.Logf("Warning: Failed to delete test helper agent %s: %v", configName, err)
						} else {
							t.Logf("Cleaned up test helper agent: %s", configName)
						}
					}
				}
			}
		}
	})
}

// isTestPath checks if a path appears to be from a test (temporary directory)
func isTestPath(path string) bool {
	// Check for common test temporary directory patterns
	return strings.Contains(path, "/tmp/") ||
		strings.Contains(path, "/T/") ||
		strings.Contains(path, "TestRepoAgent") ||
		strings.Contains(path, "TestRepositoryAgent") ||
		strings.Contains(path, "TestHelperAgent")
}

// isTestAgentName checks if an agent name appears to be from a test
func isTestAgentName(name string) bool {
	// Check for common test agent name patterns
	return strings.HasPrefix(name, "Test") ||
		strings.Contains(name, "TestAgent") ||
		strings.Contains(name, "test-") ||
		strings.Contains(name, "TestRepo") ||
		strings.Contains(name, "TestHelper")
}

// verifyNoTestCachesRemain checks that no test-related caches remain after test execution
// This can be called at the end of test suites to verify cleanup worked
func verifyNoTestCachesRemain(t *testing.T) {
	t.Helper()

	// Check repository caches
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

	// Check helper agent caches
	helperStorage, err := agent.NewHelperAgentStorage()
	if err == nil {
		configs, err := helperStorage.ListConfigs()
		if err == nil {
			for _, configName := range configs {
				if isTestAgentName(configName) {
					t.Errorf("Test helper agent cache still exists: %s", configName)
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

	// Create the repository directory
	if err := os.MkdirAll(testRepoPath, 0755); err != nil {
		t.Fatalf("Failed to create test repository directory: %v", err)
	}

	// Register cleanup for the cache
	cleanupRepoAgentCache(t, testRepoPath)

	return testRepoPath
}

// createTestHelperAgent creates a test helper agent name and registers cleanup
func createTestHelperAgent(t *testing.T, baseName string) string {
	t.Helper()

	agentName := baseName + "-" + t.Name()

	// Register cleanup for the helper agent
	cleanupHelperAgentCache(t, agentName)

	return agentName
}
