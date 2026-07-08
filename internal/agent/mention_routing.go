package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// mentionTokensForRouting returns @mention name tokens used for agent wake/routing.
func mentionTokensForRouting(msg *protocol.Message) []string {
	if msg == nil {
		return nil
	}
	if protocol.ShouldParseMentions(msg.Type, msg.From) {
		return protocol.ParseMentions(msg.Content)
	}
	if msg.IsFromSystem() {
		return nil
	}
	switch msg.Type {
	case protocol.MessageTypeCollabDiscussion, protocol.MessageTypeCollabPlan, protocol.MessageTypeCollabTask:
		if msg.GetCollaborationID() != "" {
			return protocol.FilterCollabTemplateMentions(protocol.ParseMentions(msg.Content))
		}
	}
	return nil
}

// exclusiveMentionTokens returns @mention name tokens from user-authored actionable messages.
func exclusiveMentionTokens(msg *protocol.Message) []string {
	if msg == nil || !protocol.ShouldParseMentions(msg.Type, msg.From) {
		return nil
	}
	return protocol.ParseMentions(msg.Content)
}

func resolveMentionTokens(tokens []string, agents []protocol.AgentInfo) []string {
	if len(tokens) == 0 || len(agents) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var ids []string
	for _, token := range tokens {
		for _, ag := range agents {
			if !strings.EqualFold(ag.Name, token) && !strings.EqualFold(string(ag.Type), token) {
				continue
			}
			if seen[ag.ID] {
				continue
			}
			seen[ag.ID] = true
			ids = append(ids, ag.ID)
		}
	}
	return ids
}

// backfillMentionsFromContent repopulates msg.Mentions when history replay dropped them.
func (a *Agent) backfillMentionsFromContent(msg *protocol.Message) {
	if msg == nil || msg.HasMentions() || a.Hub == nil {
		return
	}
	tokens := mentionTokensForRouting(msg)
	if len(tokens) == 0 {
		return
	}
	agents, err := a.Hub.GetChannelAgents(msg.Channel)
	if err != nil || len(agents) == 0 {
		return
	}
	if ids := resolveMentionTokens(tokens, agents); len(ids) > 0 {
		msg.Mentions = ids
	}
}

func (a *Agent) isMentionedByContentTokens(tokens []string) bool {
	for _, token := range tokens {
		if strings.EqualFold(a.Info.Name, token) || strings.EqualFold(string(a.Info.Type), token) {
			return true
		}
	}
	return false
}

func isAgentChatReply(m *protocol.Message) bool {
	if m == nil || protocol.IsUserLikeSender(m.From) {
		return false
	}
	switch m.Type {
	case protocol.MessageTypeChat, protocol.MessageTypeAnswer, protocol.MessageTypeCollabDiscussion:
		return true
	default:
		return false
	}
}

// mentionTargetAlreadyAnswered reports whether a targeted @mention was already answered
// by another agent (used to skip stale unanswered replay for late-joining agents).
func mentionTargetAlreadyAnswered(history []*protocol.Message, userIdx int, msg *protocol.Message) bool {
	if msg == nil || userIdx < 0 || userIdx >= len(history) {
		return false
	}
	targetIDs := make(map[string]struct{})
	for _, id := range msg.Mentions {
		if id != "" && id != "__INVALID__" {
			targetIDs[id] = struct{}{}
		}
	}
	targetTokens := exclusiveMentionTokens(msg)
	if len(targetIDs) == 0 && len(targetTokens) == 0 {
		return false
	}

	for j := userIdx + 1; j < len(history); j++ {
		m := history[j]
		if !isAgentChatReply(m) {
			continue
		}
		if len(targetIDs) > 0 {
			if _, ok := targetIDs[m.From.ID]; ok {
				return true
			}
		}
		for _, token := range targetTokens {
			if strings.EqualFold(m.From.Name, token) || strings.EqualFold(string(m.From.Type), token) {
				return true
			}
		}
	}
	return false
}
