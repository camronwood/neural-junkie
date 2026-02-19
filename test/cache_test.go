package test

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/repo"
)

func TestCacheKeyGeneration(t *testing.T) {
	useIsolatedRepoStorage(t)
	storage, err := repo.NewStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Test that same path generates same cache key
	path1 := "/Users/test/my-repo"
	key1, err1 := storage.GetCacheKeyForPath(path1)
	if err1 != nil {
		t.Fatalf("Failed to generate cache key for path1: %v", err1)
	}

	key2, err2 := storage.GetCacheKeyForPath(path1)
	if err2 != nil {
		t.Fatalf("Failed to generate cache key for path1 (second call): %v", err2)
	}

	if key1 != key2 {
		t.Errorf("Cache keys should be identical for same path. Got %s and %s", key1, key2)
	}

	// Test that different paths generate different cache keys
	path2 := "/Users/test/another-repo"
	key3, err3 := storage.GetCacheKeyForPath(path2)
	if err3 != nil {
		t.Fatalf("Failed to generate cache key for path2: %v", err3)
	}

	if key1 == key3 {
		t.Errorf("Cache keys should be different for different paths. Got %s for both", key1)
	}

	// Verify cache key is a valid hex string
	if len(key1) != 64 { // SHA256 produces 64 hex characters
		t.Errorf("Cache key should be 64 characters (SHA256 hex), got %d", len(key1))
	}

	t.Logf("✅ Cache key for '%s': %s", path1, key1)
	t.Logf("✅ Cache key for '%s': %s", path2, key3)
}

func TestMetadataSaveLoad(t *testing.T) {
	useIsolatedRepoStorage(t)
	storage, err := repo.NewStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Generate a cache key
	testPath := "/Users/test/my-test-repo"
	cacheKey, err := storage.GetCacheKeyForPath(testPath)
	if err != nil {
		t.Fatalf("Failed to generate cache key: %v", err)
	}

	// Create and save metadata
	metadata := &repo.RepoMetadata{
		Path:       testPath,
		CacheKey:   cacheKey,
		AgentNames: []string{"TestAgent1", "TestAgent2"},
	}

	err = storage.SaveMetadata(cacheKey, metadata)
	if err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Load metadata back
	loadedMetadata, err := storage.LoadMetadata(cacheKey)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Verify metadata
	if loadedMetadata.Path != testPath {
		t.Errorf("Path mismatch: expected %s, got %s", testPath, loadedMetadata.Path)
	}

	if loadedMetadata.CacheKey != cacheKey {
		t.Errorf("CacheKey mismatch: expected %s, got %s", cacheKey, loadedMetadata.CacheKey)
	}

	if len(loadedMetadata.AgentNames) != 2 {
		t.Errorf("AgentNames length mismatch: expected 2, got %d", len(loadedMetadata.AgentNames))
	}

	// Clean up
	t.Cleanup(func() {
		if err := storage.DeleteIndex(cacheKey); err != nil {
			t.Logf("Warning: Failed to delete test cache: %v", err)
		}
	})

	t.Logf("✅ Metadata saved and loaded successfully")
}
