package confluence

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// StorageDir is the base directory for storing Confluence indexes
	StorageDir = ".neural-junkie/confluence"
)

// Storage handles persistent storage of Confluence indexes
type Storage struct {
	baseDir string
}

// NewStorage creates a new storage handler
func NewStorage() (*Storage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(home, StorageDir)

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &Storage{
		baseDir: baseDir,
	}, nil
}

// GetStorageDir returns the storage directory for a space
func (s *Storage) GetStorageDir(spaceKey string) string {
	// Sanitize space key for filesystem
	sanitized := strings.ToLower(spaceKey)
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	return filepath.Join(s.baseDir, sanitized)
}

// SaveIndex saves an index to persistent storage with compression
func (s *Storage) SaveIndex(index *ConfluenceIndex) error {
	spaceDir := s.GetStorageDir(index.SpaceKey)

	// Create space directory if it doesn't exist
	if err := os.MkdirAll(spaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create space directory: %w", err)
	}

	indexPath := filepath.Join(spaceDir, "index.json.gz")

	// Create temporary file
	tempPath := indexPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	// Encode index as JSON
	encoder := json.NewEncoder(gzWriter)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(index); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to encode index: %w", err)
	}

	// Close writers to flush
	if err := gzWriter.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, indexPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// LoadIndex loads an index from persistent storage
func (s *Storage) LoadIndex(spaceKey string) (*ConfluenceIndex, error) {
	indexPath := filepath.Join(s.GetStorageDir(spaceKey), "index.json.gz")

	// Check if index exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("index not found for space: %s", spaceKey)
	}

	// Open index file
	file, err := os.Open(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index file: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Decode index
	var index ConfluenceIndex
	decoder := json.NewDecoder(gzReader)

	if err := decoder.Decode(&index); err != nil {
		return nil, fmt.Errorf("failed to decode index: %w", err)
	}

	return &index, nil
}

// IndexExists checks if an index exists for a space
func (s *Storage) IndexExists(spaceKey string) bool {
	indexPath := filepath.Join(s.GetStorageDir(spaceKey), "index.json.gz")
	_, err := os.Stat(indexPath)
	return err == nil
}

// DeleteIndex removes an index from storage
func (s *Storage) DeleteIndex(spaceKey string) error {
	spaceDir := s.GetStorageDir(spaceKey)

	if err := os.RemoveAll(spaceDir); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	return nil
}

// ListIndexes returns a list of all stored space keys
func (s *Storage) ListIndexes() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	var spaceKeys []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if index file exists
			indexPath := filepath.Join(s.baseDir, entry.Name(), "index.json.gz")
			if _, err := os.Stat(indexPath); err == nil {
				spaceKeys = append(spaceKeys, entry.Name())
			}
		}
	}

	return spaceKeys, nil
}

// GetIndexSize returns the size of a stored index in bytes
func (s *Storage) GetIndexSize(spaceKey string) (int64, error) {
	indexPath := filepath.Join(s.GetStorageDir(spaceKey), "index.json.gz")

	info, err := os.Stat(indexPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat index: %w", err)
	}

	return info.Size(), nil
}

// SaveMetadata saves space metadata (for quick lookups without loading full index)
func (s *Storage) SaveMetadata(spaceKey string, metadata map[string]interface{}) error {
	spaceDir := s.GetStorageDir(spaceKey)

	if err := os.MkdirAll(spaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create space directory: %w", err)
	}

	metadataPath := filepath.Join(spaceDir, "metadata.json")

	file, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	return nil
}

// LoadMetadata loads space metadata
func (s *Storage) LoadMetadata(spaceKey string) (map[string]interface{}, error) {
	metadataPath := filepath.Join(s.GetStorageDir(spaceKey), "metadata.json")

	file, err := os.Open(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer file.Close()

	var metadata map[string]interface{}
	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return metadata, nil
}
