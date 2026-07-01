package routing

import (
	"strings"

	unified "github.com/camronwood/neural-junkie/internal/routing"
)

// LoRAInput is the context for selecting a composed Ollama LoRA tag.
type LoRAInput struct {
	TaskText      string
	AgentType     string
	AgentModel    string
	InstalledTags map[string]struct{}
}

func tagInstalled(tags map[string]struct{}, tag string) bool {
	tag = strings.TrimSpace(tag)
	if tag == "" || tags == nil {
		return false
	}
	_, ok := tags[tag]
	return ok
}

// SelectComposedTag delegates to unified LoRA selection (v2).
func SelectComposedTag(in LoRAInput) (tag string, reason string) {
	return unified.SelectLoRATag(unified.Input{
		Text:          in.TaskText,
		AgentType:     in.AgentType,
		AgentModel:    in.AgentModel,
		InstalledTags: in.InstalledTags,
	})
}
