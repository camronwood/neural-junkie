package agent

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ClosureKind classifies low-signal user follow-ups that should not re-invoke the LLM.
type ClosureKind int

const (
	ClosureNone ClosureKind = iota
	ClosureThanks
	ClosureAlreadyAnswered
	ClosureBriefAck
)

var (
	thanksRE = regexp.MustCompile(`(?i)^(ok\s+)?(thanks|thank you|thx|ty)[\s!.]*$`)
	briefAckRE = regexp.MustCompile(`(?i)^(ok|cool|nice|great|perfect|got it|sounds good)[\s!.]*$`)
)

// classifyConversationalClosure detects thanks, repetition complaints, or brief acks.
func classifyConversationalClosure(content string) ClosureKind {
	s := strings.TrimSpace(content)
	if s == "" || strings.HasPrefix(s, "/") {
		return ClosureNone
	}
	if thanksRE.MatchString(s) {
		return ClosureThanks
	}
	if isAlreadyAnsweredCorrection(s) {
		return ClosureAlreadyAnswered
	}
	if briefAckRE.MatchString(s) && len(s) < 40 {
		return ClosureBriefAck
	}
	return ClosureNone
}

func isAlreadyAnsweredCorrection(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	triggers := []string{
		"already said",
		"you said that",
		"told me that",
		"said that already",
		"you already told",
		"you already answered",
		"know you said",
	}
	found := false
	for _, t := range triggers {
		if strings.Contains(lower, t) {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	if strings.Contains(lower, "?") {
		return false
	}
	if len(lower) > 100 {
		return false
	}
	return true
}

func conversationalClosureResponse(kind ClosureKind) string {
	switch kind {
	case ClosureThanks:
		return "You're welcome! Let me know if you need anything else."
	case ClosureAlreadyAnswered:
		return "You're right — I won't repeat that. What would you like to do next?"
	case ClosureBriefAck:
		return "Glad that helped. Anything else I can help with?"
	default:
		return ""
	}
}

// recentAgentAnsweredInChannel reports whether this agent posted a chat/answer recently.
func recentAgentAnsweredInChannel(history []*protocol.Message, agentID string) bool {
	if agentID == "" {
		return false
	}
	checked := 0
	for i := len(history) - 1; i >= 0 && checked < 6; i-- {
		m := history[i]
		if m == nil {
			continue
		}
		checked++
		switch m.Type {
		case protocol.MessageTypeChat, protocol.MessageTypeAnswer:
			if m.From.ID == agentID {
				return true
			}
		case protocol.MessageTypeQuestion:
			return false
		}
	}
	return false
}

// tryConversationalClosure returns a canned reply when the user is closing the topic.
func tryConversationalClosure(a *Agent, msg *protocol.Message) (string, bool) {
	if msg == nil || msg.From.ID == a.Info.ID {
		return "", false
	}
	if msg.Type != protocol.MessageTypeQuestion && msg.Type != protocol.MessageTypeChat {
		return "", false
	}

	kind := classifyConversationalClosure(msg.Content)
	if kind == ClosureNone {
		return "", false
	}

	history := a.Context.History[msg.Channel]
	if kind == ClosureBriefAck && !recentAgentAnsweredInChannel(history, a.Info.ID) {
		return "", false
	}

	resp := conversationalClosureResponse(kind)
	if resp == "" {
		return "", false
	}
	return resp, true
}

// conversationHistoryForGeneration returns LLM history rows for the current message.
func (a *Agent) conversationHistoryForGeneration(msg *protocol.Message) []*protocol.Message {
	raw := a.Context.History[msg.Channel]
	if a.Info.Type == protocol.AgentTypeAssistant {
		return filterAssistantHistory(raw, msg)
	}
	return historyForGeneration(raw, msg.ID)
}

func truncateForLog(s string, max int) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
