package hub

import (
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// parseRepoAgentCreateArgs interprets tokens after /create-repo-agent <path>.
// Supports: [agent-name] [provider] [model] with --model stripped by parseCreateRepoAgentFlags.
func parseRepoAgentCreateArgs(parts []string, flagModel string) (agentName, provider, model string) {
	provider = "ollama"
	model = flagModel
	if len(parts) < 3 {
		return "", provider, model
	}
	rest := parts[2:]
	provIdx := -1
	for i, tok := range rest {
		if isRepoAgentProviderToken(tok) {
			provIdx = i
			break
		}
	}
	if provIdx >= 0 {
		if provIdx > 0 {
			agentName = protocol.NormalizeAgentName(strings.Join(rest[:provIdx], " "))
		}
		provider = rest[provIdx]
		if provIdx+1 < len(rest) && model == "" {
			next := rest[provIdx+1]
			if !strings.HasPrefix(next, "--") {
				model = next
			}
		}
	} else if len(rest) > 0 {
		agentName = protocol.NormalizeAgentName(strings.Join(rest, " "))
	}
	return agentName, provider, model
}

func isRepoAgentProviderToken(tok string) bool {
	switch strings.ToLower(tok) {
	case "claude", "ollama", "lmstudio", "huggingface", "hf":
		return true
	default:
		return false
	}
}

func defaultRepoAgentName(repoPath string) string {
	return protocol.NormalizeAgentName(filepath.Base(repoPath) + "-expert")
}
