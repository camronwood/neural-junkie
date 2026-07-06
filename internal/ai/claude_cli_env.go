package ai

import (
	"os"
	"strings"
)

// Env vars that override Claude Code OAuth (LiteLLM proxy, API keys, custom models).
var claudeProxyEnvKeys = []string{
	"ANTHROPIC_BASE_URL",
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_MODEL",
	"ANTHROPIC_SMALL_FAST_MODEL",
}

func claudeCustomRoutingEnabled() bool {
	return strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_CLAUDE_CUSTOM_ROUTING")) == "1"
}

// appendClaudeCLIEnv prepares subprocess env for Claude Code.
// Strips env auth overrides so `claude login` OAuth and claude.ai connectors work.
func appendClaudeCLIEnv(cmdEnv []string) []string {
	if claudeCustomRoutingEnabled() {
		return cmdEnv
	}
	out := make([]string, 0, len(cmdEnv))
	for _, entry := range cmdEnv {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			out = append(out, entry)
			continue
		}
		if isClaudeProxyEnvKey(key) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func isClaudeProxyEnvKey(key string) bool {
	key = strings.TrimSpace(key)
	for _, k := range claudeProxyEnvKeys {
		if key == k {
			return true
		}
	}
	return false
}
