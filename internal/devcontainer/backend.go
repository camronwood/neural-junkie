package devcontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

const devcontainerRel = ".devcontainer/devcontainer.json"

// LoadViaBackend reads devcontainer.json through a workspace backend (local or remote).
func LoadViaBackend(ctx context.Context, b workspacebackend.Backend) (*Config, error) {
	if b == nil {
		return nil, fmt.Errorf("backend required")
	}
	data, err := b.ReadFile(ctx, devcontainerRel)
	if err != nil {
		return nil, fmt.Errorf("read devcontainer.json: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse devcontainer.json: %w", err)
	}
	if cfg.WorkspaceFolder == "" {
		cfg.WorkspaceFolder = "/workspaces/" + filepath.Base(b.Root())
	}
	return &cfg, nil
}
