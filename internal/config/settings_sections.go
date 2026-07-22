package config

import "strings"

// SecurityConfig holds hub auth and rate-limit settings (Settings → Security).
type SecurityConfig struct {
	AuthRequired           bool     `json:"auth_required"`
	RelaxedLocal           bool     `json:"relaxed_local"`
	ListenAll              bool     `json:"listen_all"`
	SessionTTLHours        int      `json:"session_ttl_hours,omitempty"`
	RateLimitEnabled       *bool    `json:"rate_limit_enabled,omitempty"` // nil = enabled
	RateReadPerMinute      int      `json:"rate_read_per_minute,omitempty"`
	RateMutatePerMinute    int      `json:"rate_mutate_per_minute,omitempty"`
	HubToken               string   `json:"hub_token,omitempty"`
	FullMetadataSecret     string   `json:"full_metadata_secret,omitempty"`
	RunCommandAllowExtra   []string `json:"run_command_allow_extra,omitempty"` // user-approved run_command prefixes
}

func DefaultSecurityConfig() SecurityConfig {
	enabled := true
	return SecurityConfig{
		RateLimitEnabled:    &enabled,
		RateReadPerMinute:   300,
		RateMutatePerMinute: 120,
		SessionTTLHours:     168,
	}
}

// SessionConfig controls last-session.json restore behavior at hub boot.
type SessionConfig struct {
	RestoreOnStartup  bool `json:"restore_on_startup"`
	SkipRestoreOnce   bool `json:"skip_restore_once"`
	ForceRestoreLarge bool `json:"force_restore_large"`
}

func DefaultSessionConfig() SessionConfig {
	return SessionConfig{RestoreOnStartup: true}
}

// SessionSummaryConfig configures async hub session summaries.
type SessionSummaryConfig struct {
	Model          string `json:"model,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

func DefaultSessionSummaryConfig() SessionSummaryConfig {
	return SessionSummaryConfig{
		TimeoutSeconds: 90,
	}
}

// ImageGenSettings configures hub image generation (Settings → Image generation).
type ImageGenSettings struct {
	Provider      string `json:"provider,omitempty"` // ollama, openai, none
	Model         string `json:"model,omitempty"`
	OllamaModel   string `json:"ollama_model,omitempty"`
	OpenAIBaseURL string `json:"openai_base_url,omitempty"`
	OpenAIAPIKey  string `json:"openai_api_key,omitempty"`
	KeepAlive     string `json:"keep_alive,omitempty"`
}

func DefaultImageGenSettings() ImageGenSettings {
	return ImageGenSettings{
		Provider: "ollama",
	}
}

// CLIAgentsConfig toggles CLI provider subprocess behavior.
type CLIAgentsConfig struct {
	DisableInteractive bool   `json:"disable_interactive"`
	CursorTrust        bool   `json:"cursor_trust"`
	DisablePTY         bool   `json:"disable_pty"`
	GeminiCLIPTY       bool   `json:"gemini_cli_pty"`
	GeminiCLIHome      string `json:"gemini_cli_home,omitempty"`
}

func DefaultCLIAgentsConfig() CLIAgentsConfig {
	return CLIAgentsConfig{CursorTrust: true}
}

// MCPResourcesConfig enables the legacy MCP resource HTTP server.
type MCPResourcesConfig struct {
	Enabled    bool   `json:"enabled"`
	Port       int    `json:"port,omitempty"`
	ExportsDir string `json:"exports_dir,omitempty"`
}

func DefaultMCPResourcesConfig() MCPResourcesConfig {
	return MCPResourcesConfig{
		Port: 8086,
	}
}

// DebugSettings enables debug routes and pprof.
type DebugSettings struct {
	Enabled   bool   `json:"enabled"`
	PprofAddr string `json:"pprof_addr,omitempty"`
}

func DefaultDebugSettings() DebugSettings {
	return DebugSettings{
		PprofAddr: "127.0.0.1:6060",
	}
}

// AutomationConfig holds scenario harness and automation defaults.
type AutomationConfig struct {
	ScenarioRepo                   string `json:"scenario_repo,omitempty"`
	ScenarioAllowFileFallback      bool   `json:"scenario_allow_file_fallback"`
	DeliverableJudgeProvider       string `json:"deliverable_judge_provider,omitempty"`
	DeliverableJudgeMode           string `json:"deliverable_judge_mode,omitempty"`
	DeliverableJudgeModel          string `json:"deliverable_judge_model,omitempty"`
	DeliverableJudgeGeminiModel    string `json:"deliverable_judge_gemini_model,omitempty"`
	DeliverableJudgeAgent          string `json:"deliverable_judge_agent,omitempty"`
	DeliverableJudgeTimeout        int    `json:"deliverable_judge_timeout,omitempty"`
	DeliverableJudgeSkip           bool   `json:"deliverable_judge_skip"`
	DeliverableJudgeFallbackOllama bool   `json:"deliverable_judge_fallback_ollama"`
	DeliverableJudgeMinIntervalS   int    `json:"deliverable_judge_min_interval_s,omitempty"`
	AgentPoll                      bool   `json:"agent_poll"`
	HumanName                      string `json:"human_name,omitempty"`
}

func DefaultAutomationConfig() AutomationConfig {
	return AutomationConfig{
		DeliverableJudgeProvider:       "claude",
		DeliverableJudgeMode:           "hub",
		DeliverableJudgeModel:          "qwen2.5-coder:14b",
		DeliverableJudgeTimeout:        180,
		DeliverableJudgeFallbackOllama: true,
		DeliverableJudgeMinIntervalS:   13,
	}
}

// StorageConfig overrides hub data paths.
type StorageConfig struct {
	RepoDir string `json:"repo_dir,omitempty"`
}

func (s SecurityConfig) RateLimitEnabledOrDefault() bool {
	if s.RateLimitEnabled == nil {
		return true
	}
	return *s.RateLimitEnabled
}

func (s SecurityConfig) SessionTTLOrDefault() int {
	if s.SessionTTLHours > 0 {
		return s.SessionTTLHours
	}
	return 168
}

func (s SessionSummaryConfig) ModelOrDefault() string {
	if m := strings.TrimSpace(s.Model); m != "" {
		return m
	}
	return SessionSummaryOllamaModel
}

func (s SessionSummaryConfig) TimeoutOrDefault() int {
	if s.TimeoutSeconds > 0 {
		return s.TimeoutSeconds
	}
	return 90
}
