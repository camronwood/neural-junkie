package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is a minimal devcontainer.json subset.
type Config struct {
	Name             string `json:"name"`
	Image            string `json:"image"`
	WorkspaceFolder  string `json:"workspaceFolder"`
	WorkspaceMount   string `json:"workspaceMount"`
	RemoteUser       string `json:"remoteUser"`
}

// Load reads .devcontainer/devcontainer.json from repoRoot.
func Load(repoRoot string) (*Config, error) {
	path := filepath.Join(repoRoot, ".devcontainer", "devcontainer.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read devcontainer.json: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse devcontainer.json: %w", err)
	}
	if cfg.WorkspaceFolder == "" {
		cfg.WorkspaceFolder = "/workspaces/" + filepath.Base(repoRoot)
	}
	return &cfg, nil
}

// AttachPlan describes how NJ should connect to a dev container.
type AttachPlan struct {
	ContainerName   string `json:"container_name"`
	WorkspaceFolder string `json:"workspace_folder"`
	Image           string `json:"image"`
	SidecarPort     int    `json:"sidecar_port"`
}

// PlanFromConfig builds an attach plan from devcontainer config.
func PlanFromConfig(repoRoot string, cfg *Config) AttachPlan {
	name := cfg.Name
	if name == "" {
		name = filepath.Base(repoRoot) + "-dev"
	}
	return AttachPlan{
		ContainerName:   name,
		WorkspaceFolder: cfg.WorkspaceFolder,
		Image:           cfg.Image,
		SidecarPort:     19876,
	}
}
