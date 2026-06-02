package routing

import "strings"

// Default light Ollama models tried in order for exploration-style collab tasks.
var DefaultLightOllamaModels = []string{
	"qwen2.5:3b",
	"qwen2.5:1.5b",
	"phi3:mini",
	"gemma2:2b",
	"llama3.2:3b",
}

// LooksLightweightCollabTask reports tasks that can use a smaller/faster local model.
func LooksLightweightCollabTask(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if looksSecurity(text) {
		return false
	}
	if synthesisKeywords(text) {
		return false
	}
	keywords := []string{
		"identify", "list ", "find relevant", "locate", "scan ", "explore",
		"inventory", "catalog", "enumerate", "discover", "grep", "search for",
		"analyze schema", "analyze the schema",
	}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return len(text) < 900 && looksCheap(text)
}

func synthesisKeywords(text string) bool {
	for _, k := range []string{
		"compile findings", "from the above tasks", "synthesize", "consolidate findings",
		"aggregate findings", "produce a markdown document",
	} {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

// SelectLightOllamaTag picks the first installed candidate from prefs (or defaults).
func SelectLightOllamaTag(installed map[string]struct{}, prefs ...string) (tag string, reason string) {
	candidates := prefs
	if len(candidates) == 0 {
		candidates = DefaultLightOllamaModels
	}
	for _, tag := range candidates {
		tag = strings.TrimSpace(tag)
		if tag != "" && tagInstalled(installed, tag) {
			return tag, "light_local_model"
		}
	}
	return "", "no_light_model"
}
