package packs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest is the on-disk pack.yaml schema.
type Manifest struct {
	ID              string            `yaml:"id"`
	Version         string            `yaml:"version"`
	Title           string            `yaml:"title"`
	Description     string            `yaml:"description"`
	Publisher       string            `yaml:"publisher,omitempty"`
	PackKind        string            `yaml:"pack_kind,omitempty"` // customer | domain
	LayoutProfile   string            `yaml:"layout_profile"` // team | ide
	Capabilities    []string          `yaml:"capabilities"`
	RequiresPacks   []string          `yaml:"requires_packs,omitempty"`
	SettingsOverlay map[string]string `yaml:"settings_overlay,omitempty"`
	Assets          PackAssetsSpec    `yaml:"assets,omitempty"`
	Agents          []AgentSpec       `yaml:"agents"`
	ModelsToEnsure  []string          `yaml:"models_to_ensure"`
	OllamaModel     string            `yaml:"ollama_model"`
	ExpertSlug      string            `yaml:"expert_slug"`
	ExpertLabel     string            `yaml:"expert_label"`
	ExpertPresets   []ExpertPreset    `yaml:"expert_presets"`
	MCPAgents       []string          `yaml:"mcp_agents"`
	LoRAAdapters    []LoRAAdapterSpec `yaml:"lora_adapters,omitempty"`
}

// AgentSpec declares one in-process specialist from a pack.
type AgentSpec struct {
	Type           string `yaml:"type"`
	Name           string `yaml:"name"`
	Implementation string `yaml:"implementation"`
	OllamaModel    string `yaml:"ollama_model,omitempty"`
}

// LoRAAdapterSpec declares a pack LoRA to download and compose in Ollama.
type LoRAAdapterSpec struct {
	AgentType     string `yaml:"agent_type"`
	RepoID        string `yaml:"repo_id"`
	Filename      string `yaml:"filename,omitempty"`
	OllamaTag     string `yaml:"ollama_tag"`
	BaseOllamaTag string `yaml:"base_ollama_tag,omitempty"`
}

// ExpertPreset is a /create-expert slug from a pack manifest.
type ExpertPreset struct {
	Slug  string `yaml:"slug"`
	Label string `yaml:"label"`
}

// CatalogEntry is a row in packs/catalog.json for the Pack Store.
type CatalogEntry struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IconKey     string `json:"icon_key,omitempty"`
	Publisher   string `json:"publisher,omitempty"`
	Builtin     bool   `json:"builtin,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
}

// Catalog is the remote/store listing file.
type Catalog struct {
	Version int            `json:"version"`
	Packs   []CatalogEntry `json:"packs"`
}

// LoadManifest reads and validates pack.yaml from dir.
func LoadManifest(dir string) (*Manifest, error) {
	path := filepath.Join(dir, "pack.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pack manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse pack manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks required manifest fields.
func (m *Manifest) Validate() error {
	if m == nil {
		return fmt.Errorf("nil manifest")
	}
	m.ID = strings.TrimSpace(m.ID)
	if m.ID == "" {
		return fmt.Errorf("pack manifest missing id")
	}
	if strings.TrimSpace(m.Title) == "" {
		return fmt.Errorf("pack %q missing title", m.ID)
	}
	if m.LayoutProfile != "" && m.LayoutProfile != "team" && m.LayoutProfile != "ide" {
		return fmt.Errorf("pack %q invalid layout_profile %q", m.ID, m.LayoutProfile)
	}
	for _, a := range m.Agents {
		if strings.TrimSpace(a.Type) == "" {
			return fmt.Errorf("pack %q agent missing type", m.ID)
		}
	}
	return nil
}

// DefaultLayoutProfile returns team when unset.
func (m *Manifest) DefaultLayoutProfile() string {
	if m.LayoutProfile == "ide" {
		return "ide"
	}
	return "team"
}

// HasCapability reports whether the manifest declares a capability token.
func (m *Manifest) HasCapability(cap string) bool {
	cap = strings.TrimSpace(cap)
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}
