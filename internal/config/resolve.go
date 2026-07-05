package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func envBool(key string) (bool, bool) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return false, false
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes":
		return true, true
	case "0", "false", "no":
		return false, true
	default:
		return false, false
	}
}

func envString(key string) (string, bool) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return "", false
	}
	return v, true
}

func envInt(key string) (int, bool) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

func splitCSVEnv(key string) []string {
	raw, ok := envString(key)
	if !ok {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ResolvedSecurity returns security settings with env overrides applied.
func (c *Config) ResolvedSecurity() SecurityConfig {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.Security
	if out.RateLimitEnabled == nil {
		enabled := true
		out.RateLimitEnabled = &enabled
	}
	if out.RateReadPerMinute <= 0 {
		out.RateReadPerMinute = 300
	}
	if out.RateMutatePerMinute <= 0 {
		out.RateMutatePerMinute = 120
	}
	if out.SessionTTLHours <= 0 {
		out.SessionTTLHours = 168
	}
	// Sync listen_all from server when security block empty
	if !out.ListenAll && c.Server.ListenAll {
		out.ListenAll = true
	}
	if v, ok := envBool("NEURAL_JUNKIE_AUTH_REQUIRED"); ok {
		out.AuthRequired = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_AUTH_REQUIRED")) == "1" {
		out.AuthRequired = true
	}
	if v, ok := envBool("NEURAL_JUNKIE_RELAXED_LOCAL"); ok {
		out.RelaxedLocal = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_RELAXED_LOCAL")) == "1" {
		out.RelaxedLocal = true
	}
	if v, ok := envBool("NEURAL_JUNKIE_LISTEN_ALL"); ok {
		out.ListenAll = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_LISTEN_ALL")) == "1" {
		out.ListenAll = true
	}
	if v, ok := envInt("NEURAL_JUNKIE_SESSION_TTL_HOURS"); ok && v > 0 {
		out.SessionTTLHours = v
	}
	if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_RATE_LIMIT")) == "0" {
		disabled := false
		out.RateLimitEnabled = &disabled
	}
	if v, ok := envInt("NEURAL_JUNKIE_RATE_READ"); ok && v > 0 {
		out.RateReadPerMinute = v
	}
	if v, ok := envInt("NEURAL_JUNKIE_RATE_MUTATE"); ok && v > 0 {
		out.RateMutatePerMinute = v
	}
	if v, ok := envString("NEURAL_JUNKIE_HUB_TOKEN"); ok {
		out.HubToken = v
	}
	if v, ok := envString("NEURAL_JUNKIE_FULL_METADATA_SECRET"); ok {
		out.FullMetadataSecret = v
	}
	return out
}

// ResolvedServer returns server bind settings with env overrides.
func (c *Config) ResolvedServer() ServerConfig {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.Server
	if out.Port <= 0 {
		out.Port = 18765
	}
	if out.Host == "" {
		out.Host = "localhost"
	}
	if v, ok := envString("SERVER_HOST"); ok {
		out.Host = v
	}
	if v, ok := envInt("SERVER_PORT"); ok && v > 0 {
		out.Port = v
	}
	sec := c.ResolvedSecurity()
	out.ListenAll = sec.ListenAll
	if v, ok := envBool("NEURAL_JUNKIE_CORS_ANY"); ok {
		out.CorsAny = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_CORS_ANY")) == "1" {
		out.CorsAny = true
	}
	if v := splitCSVEnv("NEURAL_JUNKIE_CORS_ORIGINS"); len(v) > 0 {
		out.CorsOrigins = v
	}
	if v := splitCSVEnv("NEURAL_JUNKIE_WS_ORIGINS"); len(v) > 0 {
		out.WSOrigins = v
	}
	return out
}

// ResolvedSession returns session restore settings with env overrides.
func (c *Config) ResolvedSession() SessionConfig {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.Session
	if v, ok := envBool("NEURAL_JUNKIE_RESTORE_LAST_SESSION"); ok {
		out.RestoreOnStartup = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_RESTORE_LAST_SESSION")) == "1" {
		out.RestoreOnStartup = true
	}
	if v, ok := envBool("NEURAL_JUNKIE_SKIP_SESSION_RESTORE"); ok {
		out.SkipRestoreOnce = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SKIP_SESSION_RESTORE")) == "1" {
		out.SkipRestoreOnce = true
	}
	if v, ok := envBool("NEURAL_JUNKIE_FORCE_SESSION_RESTORE"); ok {
		out.ForceRestoreLarge = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_FORCE_SESSION_RESTORE")) == "1" {
		out.ForceRestoreLarge = true
	}
	return out
}

// ResolvedSessionSummary returns session summary model/timeout with env overrides.
func (c *Config) ResolvedSessionSummary() SessionSummaryConfig {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.SessionSummary
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = 90
	}
	if v, ok := envString("NJ_SESSION_SUMMARY_MODEL"); ok {
		out.Model = v
	}
	if v, ok := envString("NJ_SESSION_SUMMARY_TIMEOUT"); ok {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			out.TimeoutSeconds = int(d.Seconds())
		}
	}
	return out
}

// ResolvedImageGen returns image generation settings with env overrides.
func (c *Config) ResolvedImageGen() ImageGenSettings {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.ImageGen
	if out.Provider == "" {
		out.Provider = "ollama"
	}
	if v, ok := envString("NEURAL_JUNKIE_IMAGE_PROVIDER"); ok {
		out.Provider = strings.ToLower(v)
	}
	if v, ok := envString("NEURAL_JUNKIE_IMAGE_MODEL"); ok {
		out.Model = v
	}
	if v, ok := envString("OLLAMA_IMAGE_MODEL"); ok {
		out.OllamaModel = v
	}
	if v, ok := envString("OPENAI_BASE_URL"); ok {
		out.OpenAIBaseURL = v
	}
	if v, ok := envString("OPENAI_API_KEY"); ok {
		out.OpenAIAPIKey = v
	}
	if v, ok := envString("NEURAL_JUNKIE_IMAGE_KEEP_ALIVE"); ok {
		out.KeepAlive = v
	} else if v, ok := envString("OLLAMA_IMAGE_KEEP_ALIVE"); ok {
		out.KeepAlive = v
	}
	return out
}

// ResolvedCLIAgents returns CLI agent subprocess settings with env overrides.
func (c *Config) ResolvedCLIAgents() CLIAgentsConfig {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.CLIAgents
	if v, ok := envBool("NEURAL_JUNKIE_DISABLE_CLI_INTERACTIVE"); ok {
		out.DisableInteractive = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_DISABLE_CLI_INTERACTIVE")) == "1" {
		out.DisableInteractive = true
	}
	if v, ok := envBool("NEURAL_JUNKIE_CURSOR_TRUST"); ok {
		out.CursorTrust = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_CURSOR_TRUST")) == "0" {
		out.CursorTrust = false
	}
	if v, ok := envBool("NEURAL_JUNKIE_DISABLE_CLI_PTY"); ok {
		out.DisablePTY = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_DISABLE_CLI_PTY")) == "1" {
		out.DisablePTY = true
	}
	if v, ok := envBool("NEURAL_JUNKIE_GEMINI_CLI_PTY"); ok {
		out.GeminiCLIPTY = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_GEMINI_CLI_PTY")) == "1" {
		out.GeminiCLIPTY = true
	}
	if v, ok := envString("NEURAL_JUNKIE_GEMINI_CLI_HOME"); ok {
		out.GeminiCLIHome = v
	}
	return out
}

// ResolvedDebug returns debug settings with env overrides.
func (c *Config) ResolvedDebug() DebugSettings {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.Debug
	if out.PprofAddr == "" {
		out.PprofAddr = "127.0.0.1:6060"
	}
	if v, ok := envBool("NEURAL_JUNKIE_DEBUG"); ok {
		out.Enabled = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_DEBUG")) == "1" {
		out.Enabled = true
	}
	if v, ok := envString("NEURAL_JUNKIE_PPROF_ADDR"); ok {
		out.PprofAddr = v
	}
	return out
}

// ResolvedMCPResources returns MCP resource server settings with env overrides.
func (c *Config) ResolvedMCPResources() MCPResourcesConfig {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.MCPResources
	if out.Port <= 0 {
		out.Port = 8086
	}
	if v, ok := envBool("ENABLE_MCP_RESOURCES"); ok {
		out.Enabled = v
	} else if strings.EqualFold(strings.TrimSpace(os.Getenv("ENABLE_MCP_RESOURCES")), "true") {
		out.Enabled = true
	}
	if v, ok := envInt("MCP_RESOURCES_PORT"); ok && v > 0 {
		out.Port = v
	}
	if v, ok := envString("MCP_EXPORTS_DIR"); ok {
		out.ExportsDir = v
	}
	return out
}

// ResolvedAutomation returns automation/scenario harness settings with env overrides.
func (c *Config) ResolvedAutomation() AutomationConfig {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.Automation
	if out.DeliverableJudgeTimeout <= 0 {
		out.DeliverableJudgeTimeout = 180
	}
	if v, ok := envString("NEURAL_JUNKIE_SCENARIO_REPO"); ok {
		out.ScenarioRepo = v
	}
	if v, ok := envBool("NJ_SCENARIO_ALLOW_FILE_FALLBACK"); ok {
		out.ScenarioAllowFileFallback = v
	} else if strings.TrimSpace(os.Getenv("NJ_SCENARIO_ALLOW_FILE_FALLBACK")) == "1" {
		out.ScenarioAllowFileFallback = true
	}
	if v, ok := envString("NJ_DELIVERABLE_JUDGE_PROVIDER"); ok {
		out.DeliverableJudgeProvider = v
	}
	if v, ok := envString("NJ_DELIVERABLE_JUDGE_MODE"); ok {
		out.DeliverableJudgeMode = v
	}
	if v, ok := envString("NJ_DELIVERABLE_JUDGE_MODEL"); ok {
		out.DeliverableJudgeModel = v
	}
	if v, ok := envString("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"); ok {
		out.DeliverableJudgeGeminiModel = v
	}
	if v, ok := envString("NJ_DELIVERABLE_JUDGE_AGENT"); ok {
		out.DeliverableJudgeAgent = v
	}
	if v, ok := envInt("NJ_DELIVERABLE_JUDGE_TIMEOUT"); ok && v > 0 {
		out.DeliverableJudgeTimeout = v
	}
	if v, ok := envBool("NJ_DELIVERABLE_JUDGE_SKIP"); ok {
		out.DeliverableJudgeSkip = v
	}
	if v, ok := envBool("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA"); ok {
		out.DeliverableJudgeFallbackOllama = v
	}
	if v, ok := envInt("NJ_DELIVERABLE_JUDGE_MIN_INTERVAL_S"); ok && v > 0 {
		out.DeliverableJudgeMinIntervalS = v
	}
	if v, ok := envBool("NEURAL_JUNKIE_AGENT_POLL"); ok {
		out.AgentPoll = v
	} else if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_AGENT_POLL")) == "1" {
		out.AgentPoll = true
	}
	if v, ok := envString("NEURAL_JUNKIE_HUMAN_NAME"); ok {
		out.HumanName = v
	}
	return out
}

// ResolvedStorage returns storage path overrides with env applied.
func (c *Config) ResolvedStorage() StorageConfig {
	if c == nil {
		c = DefaultConfig()
	}
	out := c.Storage
	if v, ok := envString("NEURAL_JUNKIE_REPO_DIR"); ok {
		out.RepoDir = v
	}
	return out
}

// ResolvedSlackDisabled reports whether Slack bridge must stay off (config or env).
func (c *Config) ResolvedSlackDisabled() bool {
	if c != nil && c.Slack.ForceDisabled {
		return true
	}
	if v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_DISABLED")); v == "1" || strings.EqualFold(v, "true") {
		return true
	}
	if v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_ENABLED")); v == "0" || strings.EqualFold(v, "false") {
		return true
	}
	return false
}

// SettingsRestartReasons lists config keys that require a hub process restart when changed.
func SettingsRestartReasons(before, after *Config) []string {
	if before == nil || after == nil {
		return nil
	}
	var reasons []string
	add := func(key string, changed bool) {
		if changed {
			reasons = append(reasons, key)
		}
	}
	bs, as := before.ResolvedServer(), after.ResolvedServer()
	add("server.host", bs.Host != as.Host)
	add("server.port", bs.Port != as.Port)
	add("server.listen_all", bs.ListenAll != as.ListenAll)
	add("server.cors_any", bs.CorsAny != as.CorsAny)
	bd, ad := before.ResolvedDebug(), after.ResolvedDebug()
	add("debug.enabled", bd.Enabled != ad.Enabled)
	add("debug.pprof_addr", bd.PprofAddr != ad.PprofAddr)
	bm, am := before.ResolvedMCPResources(), after.ResolvedMCPResources()
	add("mcp_resources.enabled", bm.Enabled != am.Enabled)
	add("mcp_resources.port", bm.Port != am.Port)
	bsess, asess := before.ResolvedSession(), after.ResolvedSession()
	add("session.restore_on_startup", bsess.RestoreOnStartup != asess.RestoreOnStartup)
	add("session.skip_restore_once", bsess.SkipRestoreOnce != asess.SkipRestoreOnce)
	add("session.force_restore_large", bsess.ForceRestoreLarge != asess.ForceRestoreLarge)
	bsec, asec := before.ResolvedSecurity(), after.ResolvedSecurity()
	add("security.session_ttl_hours", bsec.SessionTTLHours != asec.SessionTTLHours)
	return reasons
}

// PreserveRedactedSecrets copies unchanged secrets when client sends masked placeholders.
func PreserveRedactedSecrets(incoming, existing *Config) {
	if incoming == nil || existing == nil {
		return
	}
	if isRedactedSecret(incoming.Security.HubToken) {
		incoming.Security.HubToken = existing.Security.HubToken
	}
	if isRedactedSecret(incoming.Security.FullMetadataSecret) {
		incoming.Security.FullMetadataSecret = existing.Security.FullMetadataSecret
	}
	if isRedactedSecret(incoming.ImageGen.OpenAIAPIKey) {
		incoming.ImageGen.OpenAIAPIKey = existing.ImageGen.OpenAIAPIKey
	}
	if isRedactedSecret(incoming.Slack.AppToken) {
		incoming.Slack.AppToken = existing.Slack.AppToken
	}
	if isRedactedSecret(incoming.Slack.BotToken) {
		incoming.Slack.BotToken = existing.Slack.BotToken
	}
	if isRedactedSecret(incoming.Slack.ClientSecret) {
		incoming.Slack.ClientSecret = existing.Slack.ClientSecret
	}
	if isRedactedSecret(incoming.WebSearch.APIKey) {
		incoming.WebSearch.APIKey = existing.WebSearch.APIKey
	}
}

func isRedactedSecret(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || s == "***" || strings.Contains(s, "...")
}
