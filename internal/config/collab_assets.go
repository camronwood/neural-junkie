package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultCollabAssetsDir is ~/.neural-junkie/collaborations when no root is configured.
func DefaultCollabAssetsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ".neural-junkie", "collaborations"), nil
}

// ResolveCollabAssetsRoot expands ~ and returns an absolute directory for collaboration
// execution sandboxes. Each collaboration still gets a subdirectory named by its ID.
func ResolveCollabAssetsRoot(configured string) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return DefaultCollabAssetsDir()
	}

	expanded := configured
	if expanded == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand ~: %w", err)
		}
		expanded = home
	} else if strings.HasPrefix(expanded, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand ~: %w", err)
		}
		expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
	}

	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("absolute path: %w", err)
	}
	return filepath.Clean(abs), nil
}

// CollabAssetsRoot returns the directory where per-collaboration sandboxes are created.
// NEURAL_JUNKIE_COLLAB_ASSETS_DIR overrides config.json collaboration.assets_root.
func CollabAssetsRoot(cfg *Config) string {
	if v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_COLLAB_ASSETS_DIR")); v != "" {
		if resolved, err := ResolveCollabAssetsRoot(v); err == nil {
			return resolved
		}
	}
	configured := ""
	if cfg != nil {
		cfg.mu.RLock()
		configured = cfg.Collaboration.AssetsRoot
		cfg.mu.RUnlock()
	}
	resolved, err := ResolveCollabAssetsRoot(configured)
	if err != nil {
		fallback, fbErr := DefaultCollabAssetsDir()
		if fbErr != nil {
			return configured
		}
		return fallback
	}
	return resolved
}
