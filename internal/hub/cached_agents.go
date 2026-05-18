package hub

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/repo"
)

// DeleteCachedAgent removes a cached agent entry (repo index, CLI record, etc.) when it is not loaded.
func DeleteCachedAgent(agentType, name, path string) (bool, error) {
	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)
	if name == "" && path == "" {
		return false, fmt.Errorf("agent name or path is required")
	}

	switch strings.ToLower(strings.TrimSpace(agentType)) {
	case "repo", "":
		if deleted, err := deleteCachedRepoAgent(name, path); err != nil || deleted {
			return deleted, err
		}
	case "cli":
		return deleteCachedCLIAgent(name)
	default:
		return false, fmt.Errorf("unsupported cached agent type %q", agentType)
	}

	// Fallback: try repo by name when type omitted
	if agentType == "" {
		return deleteCachedRepoAgent(name, path)
	}
	return false, nil
}

func deleteCachedRepoAgent(name, path string) (bool, error) {
	storage, err := repo.NewStorage()
	if err != nil {
		return false, err
	}

	if path != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return false, fmt.Errorf("invalid path: %w", err)
		}
		key, err := storage.GetCacheKeyForPath(absPath)
		if err != nil {
			return false, err
		}
		if !storage.IndexExists(key) {
			if _, err := storage.LoadMetadata(key); err != nil {
				return false, nil
			}
		}
		return true, storage.DeleteIndex(key)
	}

	if name == "" {
		return false, nil
	}

	keys, err := storage.ListIndexes()
	if err != nil {
		return false, err
	}
	for _, cacheKey := range keys {
		meta, err := storage.LoadMetadata(cacheKey)
		if err != nil {
			continue
		}
		if repoCacheMatchesAgentName(meta, name) {
			return true, storage.DeleteIndex(cacheKey)
		}
	}
	return false, nil
}

func repoCacheMatchesAgentName(meta *repo.RepoMetadata, name string) bool {
	if meta == nil {
		return false
	}
	derived := filepath.Base(meta.Path) + "Expert"
	derivedSpaced := filepath.Base(meta.Path) + " Expert"
	if strings.EqualFold(derived, name) || strings.EqualFold(derivedSpaced, name) {
		return true
	}
	for _, n := range meta.AgentNames {
		if strings.EqualFold(strings.TrimSpace(n), name) {
			return true
		}
	}
	return false
}

func deleteCachedCLIAgent(name string) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("agent name is required for CLI cache delete")
	}
	storage, err := agent.NewCLIAgentStorage()
	if err != nil {
		return false, err
	}
	return storage.DeleteByName(name)
}
