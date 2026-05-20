package agent

import (
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TurnIntent classifies a user turn for prompt tiering and history caps.
type TurnIntent int

const (
	IntentClosure TurnIntent = iota
	IntentSubstantive
	IntentMeta
	IntentSlashCommand
	IntentLowSignal
)

func (i TurnIntent) String() string {
	switch i {
	case IntentClosure:
		return "closure"
	case IntentSubstantive:
		return "substantive"
	case IntentMeta:
		return "meta"
	case IntentSlashCommand:
		return "slash"
	case IntentLowSignal:
		return "low_signal"
	default:
		return "unknown"
	}
}

func classifyTurnIntent(msg *protocol.Message, channelType protocol.ChannelType, agentID string, history []*protocol.Message) TurnIntent {
	if msg == nil {
		return IntentSubstantive
	}
	content := strings.TrimSpace(msg.Content)
	if content != "" && content[0] == '/' {
		return IntentSlashCommand
	}

	kind := classifyConversationalClosure(content)
	if kind != ClosureNone {
		if kind == ClosureBriefAck && !recentAgentAnsweredInChannel(history, agentID) {
			// Not a true closure turn; treat as low-signal minimal prompt.
		} else {
			return IntentClosure
		}
	}

	if userAsksAboutPromptContext(content) {
		return IntentMeta
	}

	if channelType == protocol.ChannelTypeCollaboration {
		return IntentSubstantive
	}

	if len(content) < 25 && !strings.Contains(content, "?") {
		return IntentLowSignal
	}
	return IntentSubstantive
}

func (a *Agent) classifyTurnIntentForMessage(msg *protocol.Message) TurnIntent {
	history := a.Context.History[msg.Channel]
	return classifyTurnIntent(msg, a.effectiveChannelType(msg.Channel), a.Info.ID, history)
}

func (a *Agent) sessionSummaryBlock(channel string) string {
	if a.Hub == nil {
		return ""
	}
	chType := a.effectiveChannelType(channel)
	if chType != protocol.ChannelTypeDM && chType != protocol.ChannelTypeCustom {
		return ""
	}
	summary := strings.TrimSpace(a.Hub.GetChannelSessionSummary(channel))
	if summary == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("=== SESSION SUMMARY ===\n")
	b.WriteString(summary)
	b.WriteString("\nAnswer the user's latest message. Do not repeat facts from the summary unless they ask again.\n\n")
	return b.String()
}

func (a *Agent) injectSessionSummary(prompt string, msg *protocol.Message) string {
	block := a.sessionSummaryBlock(msg.Channel)
	if block == "" {
		return prompt
	}
	return block + prompt
}

func (a *Agent) buildMinimalPrompt(msg *protocol.Message) string {
	var b strings.Builder
	specialty := string(a.Info.Type)
	if a.Info.Type == protocol.AgentTypeHelper && len(a.Info.Expertise) > 0 {
		specialty = a.Info.Expertise[0]
	}
	fmt.Fprintf(&b, "You are %s, a %s specialist in a multi-agent chat.\n\n", a.Info.Name, specialty)
	if block := a.sessionSummaryBlock(msg.Channel); block != "" {
		b.WriteString(block)
	}
	b.WriteString("Respond briefly and naturally to the user's latest message only.\n")
	b.WriteString("Do not repeat long prior answers or re-derive facts already covered in the session summary.\n\n")
	b.WriteString("USER MESSAGE:\n")
	b.WriteString(strings.TrimSpace(msg.Content))
	return b.String()
}

func (a *Agent) buildPromptForIntent(msg *protocol.Message, intent TurnIntent) string {
	if a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeCollaboration {
		return a.injectSessionSummary(a.buildPrompt(msg), msg)
	}
	switch intent {
	case IntentLowSignal, IntentMeta:
		return a.buildMinimalPrompt(msg)
	default:
		if a.customPromptBuilder != nil {
			return a.injectSessionSummary(a.customPromptBuilder(msg), msg)
		}
		return a.injectSessionSummary(a.buildPrompt(msg), msg)
	}
}

func (a *Agent) conversationHistoryForIntent(msg *protocol.Message, intent TurnIntent) []*protocol.Message {
	hasSummary := a.sessionSummaryBlock(msg.Channel) != ""
	max := maxHistoryForIntent(intent, hasSummary)
	raw := a.Context.History[msg.Channel]
	var base []*protocol.Message
	if a.Info.Type == protocol.AgentTypeAssistant {
		base = filterAssistantHistory(raw, msg)
	} else {
		base = historyForGeneration(raw, msg.ID)
	}
	return trimHistoryTail(base, max)
}

func maxHistoryForIntent(intent TurnIntent, hasSummary bool) int {
	switch intent {
	case IntentLowSignal, IntentMeta:
		return 2
	case IntentSubstantive:
		if hasSummary {
			return 4
		}
		return 8
	default:
		return MaxLLMHistoryMessages
	}
}

func trimHistoryTail(history []*protocol.Message, max int) []*protocol.Message {
	if max <= 0 || len(history) <= max {
		return history
	}
	return history[len(history)-max:]
}

func (a *Agent) shouldAugmentPromptWithWorkspace(intent TurnIntent, msg *protocol.Message) bool {
	if intent == IntentLowSignal || intent == IntentMeta {
		return false
	}
	if a.useCompactOllamaPrompt(msg) {
		return false
	}
	return true
}

func (a *Agent) logTurnIntent(intent TurnIntent, msg *protocol.Message) {
	hasSummary := a.sessionSummaryBlock(msg.Channel) != ""
	chars := 0
	if msg != nil {
		chars = len(strings.TrimSpace(msg.Content))
	}
	log.Printf("[%s] intent=%s channel=%s chars=%d summary=%t",
		a.Info.Name, intent.String(), msg.Channel, chars, hasSummary)
}
