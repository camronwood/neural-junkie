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

// useOllamaContextGuardrails enables compact prompts and echo retries.
func (a *Agent) useOllamaContextGuardrails(msg *protocol.Message) bool {
	return a.useCompactOllamaPrompt(msg) || a.useCompactAssistantOllamaPrompt(msg)
}

// buildCompactAssistantOllamaPrompt is a minimal Assistant prompt for qwen2.5:7b and similar.
func (a *Agent) buildCompactAssistantOllamaPrompt(msg *protocol.Message) string {
	var system strings.Builder
	system.WriteString("You are the Assistant in Neural Junkie — reminders, tasks, notes, and meeting help.\n")
	system.WriteString(fmt.Sprintf("Model: %q via %q.\n", a.Info.AIModel, a.Info.AIProvider))
	system.WriteString("Continue this conversation using recent exchanges as ground truth. ")
	system.WriteString("Prefer the open thread over treating the latest line as an isolated new question. ")
	system.WriteString("Do not paste prior assistant replies verbatim as your entire answer.\n")
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

// buildUltraCompactAssistantOllamaPrompt is the last-resort retry prompt (shortened window still preferred by caller).
func (a *Agent) buildUltraCompactAssistantOllamaPrompt(msg *protocol.Message) string {
	var system strings.Builder
	fmt.Fprintf(&system,
		"You are the Assistant (%q / %q). Continue the open conversation in 1-3 sentences. "+
			"Prefer the thread context over starting a new topic. Do not paste prior replies verbatim.\n",
		a.Info.AIModel, a.Info.AIProvider,
	)
	AppendUserAndAgentRules(&system, msg, &a.Info, ResolveUserRulesHubFallback(msg), compactUserRulesMarkdownBytes)
	AppendLearningsForMessage(&system, msg, &a.Info)
	user := strings.TrimSpace(msg.Content)
	return system.String() + ai.SystemPromptSeparator + user
}

// shouldUseDialogueAssistantPrompt prefers a short chat persona over the full
// Assistant command encyclopedia for ordinary conversation turns.
func (a *Agent) shouldUseDialogueAssistantPrompt(msg *protocol.Message) bool {
	if a == nil || a.Info.Type != protocol.AgentTypeAssistant || msg == nil {
		return false
	}
	if assistantPersonalContextQuery(msg) {
		return false
	}
	if messageAsksAboutNJApp(msg.Content) {
		return false
	}
	if ConversationModeFromMessage(msg) == ConversationModeCode {
		return false
	}
	return true
}

// buildDialogueAssistantPrompt is the dialogue-first Assistant system prompt:
// short persona + continuity cues; command catalogs stay lazy-loaded.
func (a *Agent) buildDialogueAssistantPrompt(msg *protocol.Message) string {
	var system strings.Builder
	system.WriteString("You are the Assistant in Neural Junkie — a conversational helper for reminders, tasks, notes, planning, and general questions.\n")
	fmt.Fprintf(&system, "Model: %q via %q.\n", a.Info.AIModel, a.Info.AIProvider)
	system.WriteString("Continue this conversation. Recent exchanges are ground truth for the open thread. ")
	system.WriteString("Prefer continuing the thread over treating the latest line as a new isolated question. ")
	system.WriteString("Do not paste prior assistant replies verbatim as your entire answer.\n")
	system.WriteString("Use web_search only when the user needs current external facts; set query to their topic (never the tool name). ")
	system.WriteString("Full command catalogs and Neural Junkie app docs load when they ask about the app or features.\n")
	AppendUserAndAgentRules(&system, msg, &a.Info, ResolveUserRulesHubFallback(msg), compactUserRulesMarkdownBytes)
	a.appendMemoryForMessage(&system, msg, a.channelHistory(msg.Channel))
	AppendLearningsForMessage(&system, msg, &a.Info)

	var user strings.Builder
	user.WriteString(strings.TrimSpace(msg.Content))
	user.WriteString("\n")
	AppendPromptAttachments(&user, msg)
	return system.String() + ai.SystemPromptSeparator + user.String()
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
