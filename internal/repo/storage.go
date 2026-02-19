package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	StorageDir       = ".neural-junkie/repos"
	ConfigFileName   = "config.json"
	MetadataFileName = "metadata.json"
)

// RepoMetadata stores information about a cached repository
type RepoMetadata struct {
	Path       string   `json:"path"`        // Absolute path to repository
	CacheKey   string   `json:"cache_key"`   // SHA256 hash of path
	AgentNames []string `json:"agent_names"` // Agents using this cache
}

// Storage handles persistent storage of repository indexes
type Storage struct {
	baseDir string
}

// NewStorage creates a new storage instance. If NEURAL_JUNKIE_REPO_DIR is set
// (e.g. by tests), that directory is used; otherwise falls back to ~/.neural-junkie/repos.
func NewStorage() (*Storage, error) {
	if override := os.Getenv("NEURAL_JUNKIE_REPO_DIR"); override != "" {
		return NewStorageWithDir(override)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return NewStorageWithDir(filepath.Join(homeDir, StorageDir))
}

// NewStorageWithDir creates a storage instance rooted at the given directory.
func NewStorageWithDir(baseDir string) (*Storage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &Storage{
		baseDir: baseDir,
	}, nil
}

// GetCacheKeyForPath generates a cache key from a repository path
func (s *Storage) GetCacheKeyForPath(repoPath string) (string, error) {
	// Get absolute path to ensure consistency
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Generate SHA256 hash of the path
	hash := sha256.Sum256([]byte(absPath))
	return hex.EncodeToString(hash[:]), nil
}

// SaveIndex saves a repository index to disk using cache key
func (s *Storage) SaveIndex(cacheKey string, index *RepositoryIndex) error {
	repoDir := filepath.Join(s.baseDir, cacheKey)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}

	indexPath := filepath.Join(repoDir, "index.json")
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// LoadIndex loads a repository index from disk using cache key
func (s *Storage) LoadIndex(cacheKey string) (*RepositoryIndex, error) {
	indexPath := filepath.Join(s.baseDir, cacheKey, "index.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("index not found for cache key: %s", cacheKey)
		}
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index RepositoryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return &index, nil
}

// DeleteIndex removes a repository index from disk using cache key
func (s *Storage) DeleteIndex(cacheKey string) error {
	repoDir := filepath.Join(s.baseDir, cacheKey)

	if err := os.RemoveAll(repoDir); err != nil {
		return fmt.Errorf("failed to delete repo directory: %w", err)
	}

	return nil
}

// IndexExists checks if an index exists for a cache key
func (s *Storage) IndexExists(cacheKey string) bool {
	indexPath := filepath.Join(s.baseDir, cacheKey, "index.json")
	_, err := os.Stat(indexPath)
	return err == nil
}

// ListIndexes returns a list of all stored repository names
func (s *Storage) ListIndexes() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	var repos []string
	for _, entry := range entries {
		if entry.IsDir() {
			repos = append(repos, entry.Name())
		}
	}

	return repos, nil
}

// sanitizeName sanitizes a repository name for use as a directory name
func sanitizeName(name string) string {
	// Replace path separators and other problematic characters
	name = filepath.Base(name)
	// Remove any remaining problematic characters
	sanitized := ""
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			sanitized += string(ch)
		} else {
			sanitized += "_"
		}
	}
	return sanitized
}

// SaveMetadata saves repository metadata to disk
func (s *Storage) SaveMetadata(cacheKey string, metadata *RepoMetadata) error {
	repoDir := filepath.Join(s.baseDir, cacheKey)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}

	metadataPath := filepath.Join(repoDir, MetadataFileName)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// LoadMetadata loads repository metadata from disk
func (s *Storage) LoadMetadata(cacheKey string) (*RepoMetadata, error) {
	metadataPath := filepath.Join(s.baseDir, cacheKey, MetadataFileName)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("metadata not found for cache key: %s", cacheKey)
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata RepoMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// GetAllCachedRepos returns all cached repository agents with metadata
func (s *Storage) GetAllCachedRepos() ([]map[string]interface{}, error) {
	// Get all cache keys
	cacheKeys, err := s.ListIndexes()
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}

	var cachedAgents []map[string]interface{}

	for _, cacheKey := range cacheKeys {
		// Load metadata
		metadata, err := s.LoadMetadata(cacheKey)
		if err != nil {
			// Skip if no metadata, but continue with other agents
			continue
		}

		// Load index to get size and other details
		index, err := s.LoadIndex(cacheKey)
		if err != nil {
			// Skip if no index, but continue with other agents
			continue
		}

		// Calculate cache size (approximate)
		cacheSize := int64(0)
		if index != nil {
			cacheSize = index.TotalSize
		}

		// Generate agent name from path
		agentName := filepath.Base(metadata.Path) + " Expert"

		agent := map[string]interface{}{
			"type":       "repo",
			"name":       agentName,
			"path":       metadata.Path,
			"last_used":  index.LastIndexed.Format("2006-01-02T15:04:05Z"),
			"cache_size": cacheSize,
			"metadata": map[string]interface{}{
				"agent_names":   metadata.AgentNames,
				"file_count":    index.FileCount,
				"last_indexed":  index.LastIndexed.Format("2006-01-02T15:04:05Z"),
				"code_patterns": index.CodePatterns,
			},
		}

		cachedAgents = append(cachedAgents, agent)
	}

	return cachedAgents, nil
}
