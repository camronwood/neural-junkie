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
	// DurableChannels skips 24h age-based prune for listed channel names.
	DurableChannels []string `json:"durable_channels,omitempty"`
	// ListenAll binds 0.0.0.0 when true (requires hub token unless debug/relaxed local).
	ListenAll bool `json:"listen_all,omitempty"`
	CorsAny   bool `json:"cors_any,omitempty"`
	// CorsOrigins and WSOrigins are extra allowed origins (comma-separated in env).
	CorsOrigins []string `json:"cors_origins,omitempty"`
	WSOrigins   []string `json:"ws_origins,omitempty"`
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

type AgentModelProfile struct {
	InferenceModel     string `json:"inference_model,omitempty"`
	LoRAComposeBase    string `json:"lora_compose_base,omitempty"`
	ComposedTag        string `json:"composed_tag,omitempty"`
	UseComposedForChat bool   `json:"use_composed_for_chat,omitempty"`
}

type AgentConfig struct {
	Type            string             `json:"type"`
	Name            string             `json:"name"`
	Enabled         bool               `json:"enabled"`
	ProviderID      string             `json:"provider_id,omitempty"`
	Model           string             `json:"model,omitempty"` // overrides provider row model for this agent
	ModelProfile    *AgentModelProfile `json:"model_profile,omitempty"`
	Implementation  string             `json:"implementation,omitempty"` // pack AgentSpec implementation (e.g. builtin/music)
	CapabilityAllow []string           `json:"capability_allow,omitempty"`
	CapabilityDeny  []string           `json:"capability_deny,omitempty"`
}

type AgentCapabilityOverride struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

type CapabilityPolicyConfig struct {
	AllowSensitiveByDefault bool                               `json:"allow_sensitive_by_default"`
	HandoffsEnabled         *bool                              `json:"handoffs_enabled,omitempty"`
	AgentOverrides          map[string]AgentCapabilityOverride `json:"agent_overrides,omitempty"`
}

type AIConfig struct {
	DefaultProviderID string           `json:"default_provider_id"`
	Providers         []ProviderConfig `json:"providers"`
}

type OllamaConfig struct {
	AutoStart      bool     `json:"auto_start"`
	ModelsToEnsure []string `json:"models_to_ensure"`
	// ReactToolsEnabled runs MCP tool loops via ReAct text parsing for allowlisted models.
	ReactToolsEnabled bool `json:"react_tools_enabled"`
	// ReactToolModels lists Ollama tags that use ReAct instead of Qwen tool-loop swap.
	ReactToolModels []string `json:"react_tool_models,omitempty"`
	// NumCtx sets Ollama options.num_ctx (0 = model/server default).
	NumCtx int `json:"num_ctx,omitempty"`
	// NumPredict caps output tokens (0 = provider heuristics / model default).
	NumPredict int `json:"num_predict,omitempty"`
	// KeepAlive controls model unload (e.g. "5m", "0", "-1" for immediate unload). Empty = Ollama default.
	KeepAlive string `json:"keep_alive,omitempty"`
	// BundledBinary is the path to a bundled Ollama binary (NJ_BUNDLED_OLLAMA env override).
	BundledBinary string `json:"bundled_binary,omitempty"`
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

type ImplementationConfig struct {
	RoutingEnabled      bool     `json:"routing_enabled"`
	LocalProviderID     string   `json:"local_provider_id,omitempty"`
	LocalToolModel      string   `json:"local_tool_model,omitempty"`
	ReliableToolModel   string   `json:"reliable_tool_model,omitempty"`
	ReliableProviderID  string   `json:"reliable_provider_id,omitempty"`
	FallbackProviderIDs []string `json:"fallback_provider_ids,omitempty"`
}

// LocalToolModel returns the configured implementation tool-loop model or default.
func (c ImplementationConfig) LocalToolModelOrDefault() string {
	if m := strings.TrimSpace(c.LocalToolModel); m != "" {
		return m
	}
	return "qwen3.5:9b"
}

// ReliableToolModelOrDefault returns the heavier local tool model for repair/boot-fix tiers.
func (c ImplementationConfig) ReliableToolModelOrDefault() string {
	if m := strings.TrimSpace(c.ReliableToolModel); m != "" {
		return m
	}
	return "qwen2.5-coder:14b"
}

type CollaborationConfig struct {
	// SmartRoutingEnabled selects a configured AI provider per collaboration
	// execution task (MessageTypeCollabTask with task_id) using a static heuristic.
	SmartRoutingEnabled bool `json:"smart_routing_enabled"`
	// PlanningProviderID optionally routes collaboration planning discussion turns
	// (MessageTypeCollabDiscussion while phase=planning) through this provider.
	PlanningProviderID string `json:"planning_provider_id,omitempty"`
	// AutoApproveDeliverables auto-approves [FILE_CHANGE] proposals under collabs/<id>/
	// during executing collaborations. Nil/absent defaults to true.
	AutoApproveDeliverables *bool `json:"auto_approve_deliverables,omitempty"`
	// AssetsRoot is the parent directory for per-collaboration execution sandboxes.
	// Each run uses <AssetsRoot>/<collaboration-id>/. Empty uses ~/.neural-junkie/collaborations.
	// Overridden by NEURAL_JUNKIE_COLLAB_ASSETS_DIR when set.
	AssetsRoot string `json:"assets_root,omitempty"`
	// ExecutionTimeoutSeconds overrides the generation deadline for file-deliverable
	// collaboration tasks (default 180). Non-file collab tasks use 120s.
	ExecutionTimeoutSeconds int `json:"execution_timeout_seconds,omitempty"`
}

// AutoApproveDeliverablesEnabled reports whether collab deliverable files are auto-approved.
func (c CollaborationConfig) AutoApproveDeliverablesEnabled() bool {
	if c.AutoApproveDeliverables == nil {
		return true
	}
	return *c.AutoApproveDeliverables
}

// FeaturesConfig toggles optional product features (pack-gated elsewhere).
type FeaturesConfig struct {
	PersonalLearningEnabled        bool   `json:"personal_learning_enabled"`
	PersonalLearningSuggestEnabled bool   `json:"personal_learning_suggest_enabled"`
	ConversationMemoryEnabled      *bool  `json:"conversation_memory_enabled,omitempty"`
	LearningEmbedModel             string `json:"learning_embed_model,omitempty"`
	CodebaseEmbedModel             string `json:"codebase_embed_model,omitempty"`
	// AgentRuntimeV2 enables open-ended native agent loop (Cursor parity).
	AgentRuntimeV2 *bool `json:"agent_runtime_v2,omitempty"`
	// LegacyFileChangeParse enables the legacy [FILE_CHANGE] parser.
	LegacyFileChangeParse bool `json:"legacy_file_change_parse,omitempty"`
}

type Config struct {
	Server            ServerConfig                      `json:"server"`
	AI                AIConfig                          `json:"ai"`
	Agents            []AgentConfig                     `json:"agents"`
	Packs             PacksConfig                       `json:"packs"`
	MCP               MCPConfig                         `json:"mcp"`
	Ollama            OllamaConfig                      `json:"ollama"`
	HF                HFConfig                          `json:"hf"`
	Updates           UpdateConfig                      `json:"updates"`
	Collaboration     CollaborationConfig               `json:"collaboration"`
	Implementation    ImplementationConfig              `json:"implementation"`
	Delegation        DelegationConfig                  `json:"delegation"`
	CapabilityPolicy  CapabilityPolicyConfig            `json:"capability_policy"`
	WorkspaceIndex    WorkspaceIndexConfig              `json:"workspace_index"`
	Routing           RoutingConfig                     `json:"routing"`
	SpecialistCompose map[string]SpecialistComposeEntry `json:"specialist_compose,omitempty"`
	Features          FeaturesConfig                    `json:"features"`
	Slack             SlackConfig                       `json:"slack"`
	WebSearch         WebSearchConfig                   `json:"web_search"`
	Performance       PerformanceConfig                 `json:"performance"`
	Phoenix           PhoenixConfig                     `json:"phoenix"`
	AWS               AWSConfig                         `json:"aws"`
	Jira              JiraConfig                        `json:"jira"`
	Incident          IncidentConfig                    `json:"incident"`
	GitHubIssues      GitHubIssuesConfig                `json:"github_issues"`
	Linear            LinearConfig                      `json:"linear"`
	PagerDuty         PagerDutyConfig                   `json:"pagerduty"`
	Sentry            SentryConfig                      `json:"sentry"`
	Security          SecurityConfig                    `json:"security"`
	Session           SessionConfig                     `json:"session"`
	SessionSummary    SessionSummaryConfig              `json:"session_summary"`
	ImageGen          ImageGenSettings                  `json:"image_gen"`
	CLIAgents         CLIAgentsConfig                   `json:"cli_agents"`
	MCPResources      MCPResourcesConfig                `json:"mcp_resources"`
	Debug             DebugSettings                     `json:"debug"`
	Automation        AutomationConfig                  `json:"automation"`
	Storage           StorageConfig                     `json:"storage"`

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
			AutoStart:         true,
			ModelsToEnsure:    []string{UtilityOllamaModel},
			ReactToolsEnabled: true,
			ReactToolModels:   []string{"gemma3:12b"},
		},
		Updates: UpdateConfig{
			AutoCheck: true,
		},
		Collaboration: CollaborationConfig{
			SmartRoutingEnabled:     false,
			AutoApproveDeliverables: boolPtr(true),
		},
		Implementation: ImplementationConfig{
			RoutingEnabled: true,
			LocalToolModel: "qwen2.5-coder:14b",
		},
		Delegation: DefaultDelegationConfig(),
		CapabilityPolicy: CapabilityPolicyConfig{
			HandoffsEnabled: boolPtr(true),
			AgentOverrides:  map[string]AgentCapabilityOverride{},
		},
		WorkspaceIndex: DefaultWorkspaceIndexConfig(),
		Routing:        DefaultRoutingConfig(),
		Features:       FeaturesConfig{PersonalLearningEnabled: false, AgentRuntimeV2: boolPtr(true)},
		Packs:          DefaultPacksConfig(),
		MCP:            DefaultMCPConfig(),
		AWS: AWSConfig{
			DefaultRegion: DefaultAWSRegion,
			ReadOnly:      boolPtr(true),
		},
		Incident: IncidentConfig{
			DefaultProvider: "jira",
			WriteMode:       boolPtr(false),
			RequireApproval: boolPtr(true),
		},
		Security:       DefaultSecurityConfig(),
		Session:        DefaultSessionConfig(),
		SessionSummary: DefaultSessionSummaryConfig(),
		ImageGen:       DefaultImageGenSettings(),
		CLIAgents:      DefaultCLIAgentsConfig(),
		MCPResources:   DefaultMCPResourcesConfig(),
		Debug:          DefaultDebugSettings(),
		Automation:     DefaultAutomationConfig(),
	}
}

func boolPtr(v bool) *bool {
	return &v
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
			cfg.SeedFromEnv()
			cfg.mergeEnvVars()
			SetAppConfig(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.migrateCapabilityPolicy(data)
	cfg.migrateIfNeeded(data)
	cfg.MigrateBiologyMCPModels()
	cfg.migrateSoftwareDevelopmentPackIfNeeded()
	cfg.migrateIdePackIfNeeded()
	cfg.migrateRetiredAbilityPackExperts()
	cfg.MigrateInstalledPacks()
	cfg.EnsureMCPDefaults()
	if err := cfg.DecryptSecretsAfterLoad(); err != nil {
		return nil, fmt.Errorf("decrypt config secrets: %w", err)
	}
	cfg.SeedFromEnv()
	cfg.mergeEnvVars()
	cfg.SyncAgentsFromPacks()
	SetAppConfig(cfg)
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

// SetChannelDurable adds or removes a channel from the durable (no 24h prune) list.
func (c *Config) SetChannelDurable(channel string, durable bool) {
	channel = strings.TrimSpace(channel)
	if channel == "" || c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var kept []string
	for _, ch := range c.Server.DurableChannels {
		if ch != channel {
			kept = append(kept, ch)
		}
	}
	if durable {
		kept = append(kept, channel)
	}
	c.Server.DurableChannels = kept
}

// IsChannelDurable reports whether a channel skips age-based message prune.
func (c *Config) IsChannelDurable(channel string) bool {
	channel = strings.TrimSpace(channel)
	if channel == "" || c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, ch := range c.Server.DurableChannels {
		if ch == channel {
			return true
		}
	}
	return false
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

	if err := c.EncryptSecretsBeforeSave(); err != nil {
		return fmt.Errorf("encrypt config secrets: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		_ = c.DecryptSecretsAfterLoad()
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmpFile := fp + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		_ = c.DecryptSecretsAfterLoad()
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := os.Rename(tmpFile, fp); err != nil {
		os.Remove(tmpFile)
		_ = c.DecryptSecretsAfterLoad()
		return fmt.Errorf("failed to save config: %w", err)
	}
	_ = os.Chmod(fp, 0o600)
	_ = c.DecryptSecretsAfterLoad()

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

// SetAgentRuntimeProvider updates provider and model for an agent matched by name or type.
func (c *Config) SetAgentRuntimeProvider(name, agentType, providerID, model string) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	name = strings.TrimSpace(name)
	agentType = strings.TrimSpace(agentType)
	for i := range c.Agents {
		if name != "" && c.Agents[i].Name == name {
			if providerID != "" {
				c.Agents[i].ProviderID = providerID
			}
			c.Agents[i].Model = strings.TrimSpace(model)
			return true
		}
	}
	for i := range c.Agents {
		if agentType != "" && c.Agents[i].Type == agentType {
			if providerID != "" {
				c.Agents[i].ProviderID = providerID
			}
			c.Agents[i].Model = strings.TrimSpace(model)
			return true
		}
	}
	return false
}

// PersonalLearningEnabled reports opt-in personal learning (also requires pack capability at runtime).
func (c *Config) PersonalLearningEnabled() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Features.PersonalLearningEnabled
}

// PersonalLearningSuggestEnabled reports opt-in agent-suggested learning proposals.
func (c *Config) PersonalLearningSuggestEnabled() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Features.PersonalLearningEnabled && c.Features.PersonalLearningSuggestEnabled
}

// AgentRuntimeV2Enabled reports whether open-ended agent runtime is active.
func (c *Config) AgentRuntimeV2Enabled() bool {
	if c == nil {
		return true
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Features.AgentRuntimeV2 == nil {
		return true
	}
	return *c.Features.AgentRuntimeV2
}

// LearningEmbedModel returns configured Ollama embed model or default.
func (c *Config) LearningEmbedModel() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return strings.TrimSpace(c.Features.LearningEmbedModel)
}

// CodebaseEmbedModel returns configured Ollama embed model for @codebase search.
func (c *Config) CodebaseEmbedModel() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return strings.TrimSpace(c.Features.CodebaseEmbedModel)
}

// ConversationMemoryEnabled reports whether conversation memory retrieval is on (default true).
func (c *Config) ConversationMemoryEnabled() bool {
	if c == nil {
		return true
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Features.ConversationMemoryEnabled == nil {
		return true
	}
	return *c.Features.ConversationMemoryEnabled
}

// ClearAllAgentModels removes per-agent model overrides.
func (c *Config) ClearAllAgentModels() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.Agents {
		c.Agents[i].Model = ""
	}
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
		if packID := packForAgentTypeLocked(c, t); packID != "" && !c.packEnabledLocked(packID) {
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
	copy := *p
	if chatModel := c.ChatModelForAgent(a.Type, a.Model); chatModel != "" {
		copy.Model = chatModel
	}
	if isDevSpecialistAgentType(a.Type) && c.IsPackEnabled(PackSoftwareDevelopment) {
		agentModel := strings.TrimSpace(a.Model)
		providerModel := strings.TrimSpace(copy.Model)
		if (agentModel == "" || isBiologyChatModel(agentModel)) &&
			(providerModel == "" || isBiologyChatModel(providerModel)) {
			copy.Model = DevOllamaCodeModel
		}
	}
	if m := strings.TrimSpace(a.Model); m != "" {
		if isDevSpecialistAgentType(a.Type) && c.IsPackEnabled(PackSoftwareDevelopment) && isBiologyChatModel(m) {
			copy.Model = DevOllamaCodeModel
		} else {
			copy.Model = m
		}
	}
	return &copy
}

func isDevSpecialistAgentType(agentType string) bool {
	t := strings.ToLower(strings.TrimSpace(agentType))
	for _, d := range devSpecialistTypes {
		if t == d {
			return true
		}
	}
	return false
}

func isBiologyChatModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return model == BioOllamaChatModel ||
		model == BioOllamaTag ||
		strings.Contains(model, "openbiollm")
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
	impl := c.Implementation
	delegation := c.Delegation.Normalized()
	workspaceIndex := c.WorkspaceIndex
	routing := c.Routing.Normalized()
	specialistCompose := c.SpecialistCompose
	features := c.Features
	packs := c.Packs
	if packs.Enabled == nil {
		packs.Enabled = make(map[string]bool)
	}
	mcpCfg := c.MCP
	awsCfg := c.AWS
	jiraCfg := c.Jira
	webSearch := c.WebSearch
	slack := c.Slack
	performance := c.Performance
	phoenix := c.Phoenix
	security := c.Security
	session := c.Session
	sessionSummary := c.SessionSummary
	imageGen := c.ImageGen
	cliAgents := c.CLIAgents
	mcpResources := c.MCPResources
	debug := c.Debug
	automation := c.Automation
	storage := c.Storage
	filePath := c.filePath
	c.mu.RUnlock()

	redactedProviders := redactProviderKeys(srcProviders)
	redactedHF := hf
	if redactedHF.Token != "" {
		redactedHF.Token = maskSecret(redactedHF.Token)
	}
	redactedWebSearch := webSearch
	if redactedWebSearch.APIKey != "" {
		redactedWebSearch.APIKey = maskSecret(redactedWebSearch.APIKey)
	}
	redactedJira := jiraCfg
	if redactedJira.APIToken != "" {
		redactedJira.APIToken = maskSecret(redactedJira.APIToken)
	}
	redactedSlack := slack
	if redactedSlack.AppToken != "" {
		redactedSlack.AppToken = maskSecret(redactedSlack.AppToken)
	}
	if redactedSlack.BotToken != "" {
		redactedSlack.BotToken = maskSecret(redactedSlack.BotToken)
	}
	if redactedSlack.ClientSecret != "" {
		redactedSlack.ClientSecret = maskSecret(redactedSlack.ClientSecret)
	}
	redactedSecurity := security
	if redactedSecurity.HubToken != "" {
		redactedSecurity.HubToken = maskSecret(redactedSecurity.HubToken)
	}
	if redactedSecurity.FullMetadataSecret != "" {
		redactedSecurity.FullMetadataSecret = maskSecret(redactedSecurity.FullMetadataSecret)
	}
	redactedImageGen := imageGen
	if redactedImageGen.OpenAIAPIKey != "" {
		redactedImageGen.OpenAIAPIKey = maskSecret(redactedImageGen.OpenAIAPIKey)
	}

	return &Config{
		Server:            server,
		AI:                AIConfig{DefaultProviderID: defaultPID, Providers: redactedProviders},
		Agents:            agents,
		Packs:             packs,
		MCP:               mcpCfg,
		AWS:               awsCfg,
		Jira:              redactedJira,
		Ollama:            ollama,
		HF:                redactedHF,
		Updates:           updates,
		Collaboration:     collab,
		Implementation:    impl,
		Delegation:        delegation,
		WorkspaceIndex:    workspaceIndex,
		Routing:           routing,
		SpecialistCompose: specialistCompose,
		Features:          features,
		Slack:             redactedSlack,
		WebSearch:         redactedWebSearch,
		Performance:       performance,
		Phoenix:           phoenix,
		Security:          redactedSecurity,
		Session:           session,
		SessionSummary:    sessionSummary,
		ImageGen:          redactedImageGen,
		CLIAgents:         cliAgents,
		MCPResources:      mcpResources,
		Debug:             debug,
		Automation:        automation,
		Storage:           storage,
		filePath:          filePath,
	}
}

func redactProviderKeys(src []ProviderConfig) []ProviderConfig {
	out := make([]ProviderConfig, len(src))
	for i, p := range src {
		out[i] = p
		if p.APIKey != "" {
			out[i].APIKey = maskSecret(p.APIKey)
		}
	}
	return out
}

func maskSecret(s string) string {
	if len(s) > 8 {
		return s[:4] + "..." + s[len(s)-4:]
	}
	return "***"
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

	if v := GeminiAPIKeyFromEnvOrFile(); v != "" {
		found := false
		for i := range c.AI.Providers {
			if c.AI.Providers[i].Type == "gemini-cli" || c.AI.Providers[i].ID == "gemini-cli" {
				c.AI.Providers[i].APIKey = v
				found = true
				break
			}
		}
		if !found {
			c.AI.Providers = append(c.AI.Providers, ProviderConfig{
				ID:     "gemini-cli",
				Type:   "gemini-cli",
				Name:   "Gemini (API key)",
				APIKey: v,
				Model:  "gemini-2.5-flash",
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
	if v := strings.TrimSpace(os.Getenv("OLLAMA_CODE_MODEL")); v != "" {
		for i := range c.Agents {
			if isDevSpecialistAgentType(c.Agents[i].Type) {
				c.Agents[i].Model = v
			}
		}
	}

	if v := os.Getenv("NEURAL_JUNKIE_SLACK_ENABLED"); v == "1" || strings.EqualFold(v, "true") {
		c.Slack.Enabled = true
	}
	if v := os.Getenv("NEURAL_JUNKIE_SLACK_APP_TOKEN"); v != "" {
		c.Slack.AppToken = v
	}
	if v := os.Getenv("NEURAL_JUNKIE_SLACK_BOT_TOKEN"); v != "" {
		c.Slack.BotToken = v
	}
	if v := os.Getenv("NEURAL_JUNKIE_SLACK_DISPLAY_NAME"); v != "" {
		c.Slack.DisplayName = v
	}
	if SlackDisabledByEnv() {
		c.Slack.Enabled = false
	}

	if v := os.Getenv("NEURAL_JUNKIE_WEB_SEARCH_ENABLED"); v == "1" || strings.EqualFold(v, "true") {
		c.WebSearch.Enabled = true
	}
	if v := os.Getenv("NEURAL_JUNKIE_WEB_SEARCH_API_KEY"); v != "" {
		c.WebSearch.APIKey = v
	}
	if v := os.Getenv("NEURAL_JUNKIE_WEB_SEARCH_PROVIDER"); v != "" {
		c.WebSearch.Provider = v
	}
	if v := os.Getenv("NEURAL_JUNKIE_WEB_SEARCH_KEYLESS"); v == "1" || strings.EqualFold(v, "true") {
		c.WebSearch.Keyless = true
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
