package ai

import (
	"strings"
	"sync"
)

// cliProviderModelOverrides holds runtime model selections from Settings > AI Providers
// so running CLI agents pick up profile changes without restart.
var cliProviderModelOverrides sync.Map

// SetCLIProviderModelOverride sets the model used for subsequent CLI invocations.
// Pass an empty model to clear the override.
func SetCLIProviderModelOverride(providerName, model string) {
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		return
	}
	model = strings.TrimSpace(model)
	if model == "" {
		cliProviderModelOverrides.Delete(providerName)
		return
	}
	cliProviderModelOverrides.Store(providerName, model)
}

// CLIModelFlagProviders receive a --model flag before the prompt when a model is configured.
var CLIModelFlagProviders = map[string]bool{
	"cursor-cli":  true,
	"claude-cli":  true,
	"codex-cli":   true,
	"copilot-cli": true,
}

// prependCLIModelArgs inserts --model when the provider supports it and model is non-empty.
// gemini-cli uses GEMINI_MODEL env instead (see syncGeminiModelEnv).
func prependCLIModelArgs(providerName string, baseArgs []string, model string) []string {
	model = strings.TrimSpace(model)
	if model == "" || providerName == "gemini-cli" {
		return baseArgs
	}
	if !CLIModelFlagProviders[providerName] {
		return baseArgs
	}
	// Skip if caller already configured --model.
	for i, arg := range baseArgs {
		if arg == "--model" && i+1 < len(baseArgs) && strings.TrimSpace(baseArgs[i+1]) != "" {
			return baseArgs
		}
	}
	out := make([]string, 0, len(baseArgs)+2)
	out = append(out, "--model", model)
	out = append(out, baseArgs...)
	return out
}

// EffectiveCLIModel returns the model to use for a CLI invocation (display + flags/env).
func (c *CLIAgentProvider) EffectiveCLIModel() string {
	if c == nil {
		return ""
	}
	if v, ok := cliProviderModelOverrides.Load(c.ProviderName); ok {
		if m := strings.TrimSpace(v.(string)); m != "" {
			return m
		}
	}
	if c.ProviderName == "gemini-cli" {
		if m := strings.TrimSpace(c.Env["GEMINI_MODEL"]); m != "" {
			return m
		}
	}
	if m := strings.TrimSpace(c.Model); m != "" {
		// Skip generic placeholder names that are not real CLI model ids.
		switch m {
		case "cursor-agent", "gemini-agent", "claude-agent", "copilot-agent", "codex-agent":
			return ""
		default:
			return m
		}
	}
	return ""
}
