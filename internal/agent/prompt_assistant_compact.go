package agent

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// useCompactAssistantOllamaPrompt uses a short prompt for Assistant on small local models.
func (a *Agent) useCompactAssistantOllamaPrompt(msg *protocol.Message) bool {
	if a.Info.Type != protocol.AgentTypeAssistant {
		return false
	}
	if msg != nil && a.getCollaborationContext(msg).ID != "" {
		return false
	}
	if !strings.Contains(strings.ToLower(a.Info.AIProvider), "ollama") {
		return false
	}
	return ai.OllamaSmallChatModel(a.Info.AIModel)
}

// useOllamaContextGuardrails enables compact prompts, user-only history, and echo retries.
func (a *Agent) useOllamaContextGuardrails(msg *protocol.Message) bool {
	return a.useCompactOllamaPrompt(msg) || a.useCompactAssistantOllamaPrompt(msg)
}

// buildCompactAssistantOllamaPrompt is a minimal Assistant prompt for qwen2.5:7b and similar.
func (a *Agent) buildCompactAssistantOllamaPrompt(msg *protocol.Message) string {
	var system strings.Builder
	system.WriteString("You are the Assistant in Neural Junkie — reminders, tasks, notes, and meeting help.\n")
	system.WriteString(fmt.Sprintf("Model: %q via %q.\n", a.Info.AIModel, a.Info.AIProvider))
	system.WriteString("Answer ONLY the user's latest message. ")
	system.WriteString("Do not repeat or re-answer earlier questions from the conversation.\n")
	system.WriteString("Meeting notes and emails load when the user asks about meetings or email. ")
	system.WriteString("Full Neural Junkie app knowledge (shortcuts, settings, packs) loads when they ask about the NJ app or features.\n")
	AppendUserAndAgentRules(&system, msg, &a.Info, ResolveUserRulesHubFallback(msg), compactUserRulesMarkdownBytes)
	a.appendMemoryForMessage(&system, msg, a.channelHistory(msg.Channel))
	AppendLearningsForMessage(&system, msg, &a.Info)

	var user strings.Builder
	user.WriteString(strings.TrimSpace(msg.Content))
	user.WriteString("\n")
	AppendPromptAttachments(&user, msg)

	return system.String() + ai.SystemPromptSeparator + user.String()
}

// buildUltraCompactAssistantOllamaPrompt is the last-resort retry prompt (no history).
func (a *Agent) buildUltraCompactAssistantOllamaPrompt(msg *protocol.Message) string {
	var system strings.Builder
	fmt.Fprintf(&system,
		"You are the Assistant (%q / %q). Reply in 1-3 sentences to the user only. Do not mention prior topics.\n",
		a.Info.AIModel, a.Info.AIProvider,
	)
	AppendUserAndAgentRules(&system, msg, &a.Info, ResolveUserRulesHubFallback(msg), compactUserRulesMarkdownBytes)
	AppendLearningsForMessage(&system, msg, &a.Info)
	user := strings.TrimSpace(msg.Content)
	return system.String() + ai.SystemPromptSeparator + user
}

// userAsksAboutModelIdentity detects questions about which LLM/model the agent runs.
func userAsksAboutModelIdentity(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	markers := []string{
		"what model", "which model", "what llm", "which llm",
		"what ai model", "which ai model", "running on", "powered by",
		"are you running", "model are you", "llm are you",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

// looksLikeContextStackEcho reports replies that re-answer earlier topics (common on small Ollama models).
func looksLikeContextStackEcho(msg *protocol.Message, text string) bool {
	if msg == nil {
		return false
	}
	if msg.From.ID != "" && (messageAsksAboutMeetings(msg.Content) || messageAsksAboutEmail(msg.Content)) {
		return false
	}
	t := strings.TrimSpace(text)
	q := strings.ToLower(strings.TrimSpace(msg.Content))

	if userAsksAboutModelIdentity(msg.Content) || userAsksAboutPromptContext(msg.Content) {
		if strings.Contains(t, "```") {
			return true
		}
		if strings.Contains(strings.ToLower(t), "meeting notes") || strings.Contains(strings.ToLower(t), "def add_") {
			return true
		}
	}

	if len(t) < 120 {
		return false
	}

	narrow := len(q) < 70 || userAsksAboutModelIdentity(msg.Content)
	if !narrow {
		return false
	}
	if strings.Contains(t, "```") && !strings.Contains(q, "python") && !strings.Contains(q, "function") &&
		!strings.Contains(q, "code") && !strings.Contains(q, "def ") {
		return true
	}
	if strings.Count(t, "\n\n") >= 2 && len(t) > 280 {
		return true
	}
	return false
}
