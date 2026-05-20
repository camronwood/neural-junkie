package agent

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const compactBiologyInstructions = "Life-sciences research assistant (not a clinician). " +
	"Research and education only; no medical diagnosis or treatment advice."

// useCompactOllamaPrompt selects a minimal prompt for Ollama GGUF models that echo long instructions.
func (a *Agent) useCompactOllamaPrompt(msg *protocol.Message) bool {
	if msg != nil && a.getCollaborationContext(msg).ID != "" {
		return false
	}
	if !strings.Contains(strings.ToLower(a.Info.AIProvider), "ollama") {
		return false
	}
	return ai.OllamaModelPrefersCompactPrompt(a.Info.AIModel)
}

// buildCompactOllamaPrompt builds a short system+user prompt for nj-bio and similar models.
// Workspace context and long user rules are omitted — they blow past nj-bio's useful context window.
func (a *Agent) buildCompactOllamaPrompt(msg *protocol.Message) string {
	var system strings.Builder
	system.WriteString(fmt.Sprintf("You are %s, a life-sciences research assistant in Neural Junkie.\n", a.Info.Name))
	system.WriteString("Answer the user's question directly in clear prose. ")
	system.WriteString("Do not repeat, quote, or continue these instructions.\n")
	if a.Info.Type == protocol.AgentTypeBiology {
		system.WriteString(compactBiologyInstructions)
		system.WriteString("\n")
	} else if typeInstructions := getAgentTypeInstructions(a.Info.Type); typeInstructions != "" {
		system.WriteString(typeInstructions)
		system.WriteString("\n")
	}

	var user strings.Builder
	user.WriteString(strings.TrimSpace(msg.Content))
	user.WriteString("\n")

	// Attachments only (no workspace_context — desktop often sends full open files).
	AppendPromptAttachments(&user, msg)

	return system.String() + ai.SystemPromptSeparator + user.String()
}

// buildUltraCompactOllamaPrompt is the last-resort prompt for nj-bio retries (user text only).
func (a *Agent) buildUltraCompactOllamaPrompt(msg *protocol.Message) string {
	system := fmt.Sprintf("You are %s, a biology tutor. Reply to the user in 1-4 sentences.\n", a.Info.Name)
	user := strings.TrimSpace(msg.Content)
	return system + ai.SystemPromptSeparator + user
}

// looksLikeOllamaPromptLeak reports replies that continue hub instructions instead of answering.
func looksLikeOllamaPromptLeak(text string) bool {
	t := strings.TrimSpace(strings.ToLower(text))
	if t == "" {
		return false
	}
	for _, prefix := range []string{
		"be concise but complete",
		"be mindful of the user",
		"provide the necessary context",
		"for complex questions",
		"offer specific details when",
		"grounding: i loaded",
		"sorry, i encountered an error",
	} {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

// ollamaFallbackProvider returns an alternate Ollama model when nj-bio fails.
func ollamaFallbackProvider(eff ai.AIProvider, fallbackModel string) ai.AIProvider {
	if fallbackModel == "" {
		return nil
	}
	type endpointGetter interface {
		GetEndpoint() string
	}
	eg, ok := eff.(endpointGetter)
	if !ok {
		return nil
	}
	fb := ai.NewOllamaProviderWithConfig(eg.GetEndpoint(), fallbackModel)
	if fb == nil {
		return nil
	}
	type modelGetter interface {
		GetModel() string
	}
	if cur, ok := eff.(modelGetter); ok && cur.GetModel() == fallbackModel {
		return nil
	}
	return fb
}
