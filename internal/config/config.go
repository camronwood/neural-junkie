package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	// TimeoutSeconds is primarily used by CLI providers to control max
	// runtime for a single invocation. If unset or <= 0, provider default applies.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
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

// HFConfig holds Hugging Face Hub download and token defaults.
type HFConfig struct {
	// Token for Hub downloads and biology ESMFold (Settings → AI & providers).
	Token string `json:"token,omitempty"`
	// CacheDir overrides HF cache (default ~/.cache/huggingface/hub).
	CacheDir string `json:"cache_dir,omitempty"`
}

type UpdateConfig struct {
	AutoCheck bool `json:"auto_check"`
}

// CollaborationConfig controls multi-agent collaboration behavior.
type CollaborationConfig struct {
	// SmartRoutingEnabled selects a configured AI provider per collaboration
	// execution task (MessageTypeCollabTask with task_id) using a static heuristic.
	SmartRoutingEnabled bool `json:"smart_routing_enabled"`
	// AssetsRoot is the parent directory for per-collaboration execution sandboxes.
	// Each run uses <AssetsRoot>/<collaboration-id>/. Empty uses ~/.neural-junkie/collaborations.
	// Overridden by NEURAL_JUNKIE_COLLAB_ASSETS_DIR when set.
	AssetsRoot string `json:"assets_root,omitempty"`
}

type Config struct {
	Server         ServerConfig         `json:"server"`
	AI             AIConfig             `json:"ai"`
	Agents         []AgentConfig        `json:"agents"`
	Packs          PacksConfig          `json:"packs"`
	MCP            MCPConfig            `json:"mcp"`
	Ollama         OllamaConfig         `json:"ollama"`
	HF             HFConfig             `json:"hf"`
	Updates        UpdateConfig         `json:"updates"`
	Collaboration  CollaborationConfig  `json:"collaboration"`

	mu       sync.RWMutex `json:"-"`
	filePath string       `json:"-"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 18765,
		},
		AI: AIConfig{
			DefaultProviderID: "ollama-local",
			Providers: []ProviderConfig{
				{
					ID:       "ollama-local",
					Type:     "ollama",
					Name:     "Local Ollama",
					Endpoint: "http://localhost:11434",
					Model:    UtilityOllamaModel,
				},
			},
		},
		Agents: []AgentConfig{},
		Ollama: OllamaConfig{
			AutoStart:      true,
			ModelsToEnsure: []string{UtilityOllamaModel},
		},
		Updates: UpdateConfig{
			AutoCheck: true,
		},
		Collaboration: CollaborationConfig{
			SmartRoutingEnabled: false,
		},
		Packs: DefaultPacksConfig(),
		MCP:   DefaultMCPConfig(),
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
	cfg.migrateSoftwareDevelopmentPackIfNeeded()
	cfg.EnsureMCPDefaults()
	cfg.mergeEnvVars()
	cfg.SyncAgentsFromPacks()
	return cfg, nil
}

// EnsureMCPDefaults fills MCP agent map when missing (full defaults come from migrateIfNeeded when "mcp" is absent).
func (c *Config) EnsureMCPDefaults() {
	if c == nil {
		return
	}
	if c.MCP.Agents == nil {
		c.MCP.Agents = DefaultMCPConfig().Agents
	}
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

// FirstOllamaEndpoint returns the endpoint of the first Ollama-type provider, or "" if none.
func (c *Config) FirstOllamaEndpoint() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, p := range c.AI.Providers {
		if p.Type == "ollama" {
			return strings.TrimSpace(p.Endpoint)
		}
	}
	return ""
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
		if !a.Enabled {
			continue
		}
		t := strings.ToLower(strings.TrimSpace(a.Type))
		if packID := packForAgentType(t); packID != "" && !c.packEnabledLocked(packID) {
			continue
		}
		result = append(result, a)
	}
	return result
}

func (c *Config) ProviderForAgent(a AgentConfig) *ProviderConfig {
	pid := a.ProviderID
	if pid == "" {
		pid = c.AI.DefaultProviderID
	}
	p := c.GetProvider(pid)
	if p == nil {
		return nil
	}
	if a.Type == "biology" && c.IsPackEnabled(PackLifeSciences) {
		copy := *p
		m := strings.TrimSpace(copy.Model)
		if m == "" || m == BioOllamaTag {
			copy.Model = BioOllamaChatModel
		}
		return &copy
	}
	return p
}

// ListProvidersSnapshot returns a copy of configured providers (thread-safe).
func (c *Config) ListProvidersSnapshot() []ProviderConfig {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ProviderConfig, len(c.AI.Providers))
	copy(out, c.AI.Providers)
	return out
}

// Redacted returns a copy with API keys masked for safe API exposure.
func (c *Config) Redacted() *Config {
	c.mu.RLock()
	server := c.Server
	defaultPID := c.AI.DefaultProviderID
	srcProviders := c.AI.Providers
	agents := append([]AgentConfig(nil), c.Agents...)
	ollama := c.Ollama
	hf := c.HF
	updates := c.Updates
	collab := c.Collaboration
	packs := c.Packs
	if packs.Enabled == nil {
		packs.Enabled = make(map[string]bool)
	}
	mcpCfg := c.MCP
	filePath := c.filePath
	c.mu.RUnlock()

	redactedProviders := make([]ProviderConfig, len(srcProviders))
	for i, p := range srcProviders {
		redactedProviders[i] = p
		if p.APIKey != "" {
			if len(p.APIKey) > 8 {
				redactedProviders[i].APIKey = p.APIKey[:4] + "..." + p.APIKey[len(p.APIKey)-4:]
			} else {
				redactedProviders[i].APIKey = "***"
			}
		}
	}
	redactedHF := hf
	if redactedHF.Token != "" {
		if len(redactedHF.Token) > 8 {
			redactedHF.Token = redactedHF.Token[:4] + "..." + redactedHF.Token[len(redactedHF.Token)-4:]
		} else {
			redactedHF.Token = "***"
		}
	}
	return &Config{
		Server:        server,
		AI:            AIConfig{DefaultProviderID: defaultPID, Providers: redactedProviders},
		Agents:        agents,
		Packs:         packs,
		MCP:           mcpCfg,
		Ollama:        ollama,
		HF:            redactedHF,
		Updates:       updates,
		Collaboration: collab,
		filePath:      filePath,
	}
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
	var probe struct {
		MCP *json.RawMessage `json:"mcp"`
	}
	if err := json.Unmarshal(raw, &probe); err == nil && probe.MCP == nil {
		c.MCP = DefaultMCPConfig()
	}

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
