package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ProviderConfig struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"` // ollama, anthropic, openai-compatible, cursor-cli, gemini-cli
	Name     string            `json:"name"`
	Endpoint string            `json:"endpoint,omitempty"`
	APIKey   string            `json:"api_key,omitempty"`
	Model    string            `json:"model,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	WorkDir  string            `json:"work_dir,omitempty"`
}

type AgentConfig struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	ProviderID string `json:"provider_id,omitempty"`
}

type AIConfig struct {
	DefaultProviderID string           `json:"default_provider_id"`
	Providers         []ProviderConfig `json:"providers"`
}

type OllamaConfig struct {
	AutoStart      bool     `json:"auto_start"`
	ModelsToEnsure []string `json:"models_to_ensure"`
}

type UpdateConfig struct {
	AutoCheck bool `json:"auto_check"`
}

type Config struct {
	Server  ServerConfig  `json:"server"`
	AI      AIConfig      `json:"ai"`
	Agents  []AgentConfig `json:"agents"`
	Ollama  OllamaConfig  `json:"ollama"`
	Updates UpdateConfig  `json:"updates"`

	mu       sync.RWMutex `json:"-"`
	filePath string       `json:"-"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		AI: AIConfig{
			DefaultProviderID: "ollama-local",
			Providers: []ProviderConfig{
				{
					ID:       "ollama-local",
					Type:     "ollama",
					Name:     "Local Ollama",
					Endpoint: "http://localhost:11434",
					Model:    "qwen2.5-coder:14b",
				},
			},
		},
		Agents: []AgentConfig{
			{Type: "backend", Name: "GoExpert", Enabled: true},
			{Type: "frontend", Name: "ReactExpert", Enabled: true},
			{Type: "devops", Name: "DevOpsPro", Enabled: true},
			{Type: "database", Name: "SQLMaster", Enabled: true},
			{Type: "security", Name: "SecurityExpert", Enabled: true},
			{Type: "rust", Name: "RustExpert", Enabled: true},
		},
		Ollama: OllamaConfig{
			AutoStart:      true,
			ModelsToEnsure: []string{"qwen2.5-coder:14b", "qwen2.5:7b"},
		},
		Updates: UpdateConfig{
			AutoCheck: true,
		},
	}
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".neural-junkie"), nil
}

func configFilePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	fp, err := configFilePath()
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	cfg.filePath = fp

	data, err := os.ReadFile(fp)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.mergeEnvVars()
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.migrateIfNeeded(data)
	cfg.mergeEnvVars()
	return cfg, nil
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fp := c.filePath
	if fp == "" {
		var err error
		fp, err = configFilePath()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(fp)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmpFile := fp + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := os.Rename(tmpFile, fp); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

func (c *Config) Exists() bool {
	fp := c.filePath
	if fp == "" {
		var err error
		fp, err = configFilePath()
		if err != nil {
			return false
		}
	}
	_, err := os.Stat(fp)
	return err == nil
}

func (c *Config) GetProvider(id string) *ProviderConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i := range c.AI.Providers {
		if c.AI.Providers[i].ID == id {
			return &c.AI.Providers[i]
		}
	}
	return nil
}

func (c *Config) GetDefaultProvider() *ProviderConfig {
	return c.GetProvider(c.AI.DefaultProviderID)
}

func (c *Config) AddProvider(p ProviderConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, existing := range c.AI.Providers {
		if existing.ID == p.ID {
			return fmt.Errorf("provider with ID %q already exists", p.ID)
		}
	}
	c.AI.Providers = append(c.AI.Providers, p)
	return nil
}

func (c *Config) UpdateProvider(p ProviderConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.AI.Providers {
		if c.AI.Providers[i].ID == p.ID {
			c.AI.Providers[i] = p
			return nil
		}
	}
	return fmt.Errorf("provider %q not found", p.ID)
}

func (c *Config) RemoveProvider(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, a := range c.Agents {
		if a.ProviderID == id {
			return fmt.Errorf("cannot remove provider %q: agent %q references it", id, a.Name)
		}
	}
	if c.AI.DefaultProviderID == id {
		return fmt.Errorf("cannot remove the default provider %q", id)
	}

	for i := range c.AI.Providers {
		if c.AI.Providers[i].ID == id {
			c.AI.Providers = append(c.AI.Providers[:i], c.AI.Providers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("provider %q not found", id)
}

func (c *Config) SetAgentEnabled(agentType string, enabled bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.Agents {
		if c.Agents[i].Type == agentType {
			c.Agents[i].Enabled = enabled
			return nil
		}
	}
	return fmt.Errorf("agent type %q not found", agentType)
}

func (c *Config) SetAgentProvider(agentType, providerID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	found := false
	for _, p := range c.AI.Providers {
		if p.ID == providerID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("provider %q not found", providerID)
	}

	for i := range c.Agents {
		if c.Agents[i].Type == agentType {
			c.Agents[i].ProviderID = providerID
			return nil
		}
	}
	return fmt.Errorf("agent type %q not found", agentType)
}

func (c *Config) EnabledAgents() []AgentConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var result []AgentConfig
	for _, a := range c.Agents {
		if a.Enabled {
			result = append(result, a)
		}
	}
	return result
}

func (c *Config) ProviderForAgent(a AgentConfig) *ProviderConfig {
	pid := a.ProviderID
	if pid == "" {
		pid = c.AI.DefaultProviderID
	}
	return c.GetProvider(pid)
}

// Redacted returns a copy with API keys masked for safe API exposure.
func (c *Config) Redacted() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cp := *c
	cp.AI.Providers = make([]ProviderConfig, len(c.AI.Providers))
	for i, p := range c.AI.Providers {
		cp.AI.Providers[i] = p
		if p.APIKey != "" {
			if len(p.APIKey) > 8 {
				cp.AI.Providers[i].APIKey = p.APIKey[:4] + "..." + p.APIKey[len(p.APIKey)-4:]
			} else {
				cp.AI.Providers[i].APIKey = "***"
			}
		}
	}
	return &cp
}

// mergeEnvVars overlays environment variables onto the config. Env vars take
// precedence when set, allowing the existing env.local workflow to coexist
// with the new config.json.
func (c *Config) mergeEnvVars() {
	if v := os.Getenv("SERVER_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("SERVER_HOST"); v != "" {
		c.Server.Host = v
	}

	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		found := false
		for i := range c.AI.Providers {
			if c.AI.Providers[i].Type == "anthropic" {
				c.AI.Providers[i].APIKey = v
				found = true
				break
			}
		}
		if !found {
			c.AI.Providers = append(c.AI.Providers, ProviderConfig{
				ID:     "anthropic",
				Type:   "anthropic",
				Name:   "Claude (Anthropic)",
				APIKey: v,
				Model:  "claude-3-5-sonnet-20241022",
			})
		}
	}

	if v := os.Getenv("OLLAMA_ENDPOINT"); v != "" {
		for i := range c.AI.Providers {
			if c.AI.Providers[i].Type == "ollama" {
				c.AI.Providers[i].Endpoint = v
				break
			}
		}
	}
	if v := os.Getenv("OLLAMA_MODEL"); v != "" {
		for i := range c.AI.Providers {
			if c.AI.Providers[i].Type == "ollama" {
				c.AI.Providers[i].Model = v
				break
			}
		}
	}
}

// migrateIfNeeded handles migration from older config formats.
// Currently checks for the legacy flat ai.* schema and converts to the
// providers array format.
func (c *Config) migrateIfNeeded(raw []byte) {
	var legacy struct {
		AI struct {
			DefaultProvider  string `json:"default_provider"`
			OllamaEndpoint   string `json:"ollama_endpoint"`
			OllamaModel      string `json:"ollama_model"`
			AnthropicAPIKey  string `json:"anthropic_api_key"`
			LMStudioEndpoint string `json:"lmstudio_endpoint"`
		} `json:"ai"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return
	}

	if legacy.AI.OllamaEndpoint != "" && len(c.AI.Providers) == 0 {
		c.AI.Providers = append(c.AI.Providers, ProviderConfig{
			ID:       "ollama",
			Type:     "ollama",
			Name:     "Local Ollama",
			Endpoint: legacy.AI.OllamaEndpoint,
			Model:    legacy.AI.OllamaModel,
		})
		if c.AI.DefaultProviderID == "" {
			c.AI.DefaultProviderID = "ollama"
		}
	}

	if legacy.AI.AnthropicAPIKey != "" {
		found := false
		for _, p := range c.AI.Providers {
			if p.Type == "anthropic" {
				found = true
				break
			}
		}
		if !found {
			c.AI.Providers = append(c.AI.Providers, ProviderConfig{
				ID:     "anthropic",
				Type:   "anthropic",
				Name:   "Claude (Anthropic)",
				APIKey: legacy.AI.AnthropicAPIKey,
				Model:  "claude-3-5-sonnet-20241022",
			})
		}
	}

	if legacy.AI.LMStudioEndpoint != "" {
		found := false
		for _, p := range c.AI.Providers {
			if p.Type == "openai-compatible" && p.ID == "lmstudio" {
				found = true
				break
			}
		}
		if !found {
			c.AI.Providers = append(c.AI.Providers, ProviderConfig{
				ID:       "lmstudio",
				Type:     "openai-compatible",
				Name:     "LM Studio",
				Endpoint: legacy.AI.LMStudioEndpoint,
			})
		}
	}
}
