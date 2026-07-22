package hub

import (
	"context"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type SemanticTurnRouter interface {
	Resolve(context.Context, intent.TurnFeatures) intent.TurnDecision
}

func (h *Hub) SetSemanticTurnRouter(router SemanticTurnRouter) {
	h.mu.Lock()
	h.semanticTurnRouter = router
	h.mu.Unlock()
}

func (h *Hub) resolveSemanticTurn(ctx context.Context, msg *protocol.Message) {
	if h == nil || msg == nil || !protocol.IsUserLikeSender(msg.From) {
		return
	}
	if msg.Metadata != nil {
		// Canonical decisions are server-owned. Never accept a client-authored
		// decision that could request mutation or select a privileged recipient.
		delete(msg.Metadata, protocol.MetadataTurnDecision)
		delete(msg.Metadata, protocol.TurnMetaGovernance)
	}
	h.mu.RLock()
	router := h.semanticTurnRouter
	h.mu.RUnlock()
	if router == nil {
		return
	}

	stampCanonicalGovernance(msg)
	features := h.semanticTurnFeatures(msg)
	decision := router.Resolve(ctx, features)
	if err := protocol.StampTurnDecision(msg, decision); err != nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	if len(msg.Mentions) == 0 && !features.IsDirectMessage && decision.RecipientType != "" {
		msg.Metadata[protocol.IdeMetaRouteAgentType] = decision.RecipientType
	}
	switch decision.Action {
	case intent.ActionDebug, intent.ActionEdit, intent.ActionContinue:
		msg.Metadata[protocol.IdeMetaImplementationSession] = true
		msg.Metadata[protocol.TurnMetaCanProposeFiles] = decision.Mutation == intent.MutationWorkspace
		msg.Metadata[protocol.TurnMetaCanRunImplSession] = true
		msg.Metadata[protocol.TurnMetaRequiresWorkspace] = true
	}
	governance, _ := protocol.ExtractTurnGovernance(msg)
	governance.RequiresWorkspace = decision.Action == intent.ActionDebug ||
		decision.Action == intent.ActionEdit || decision.Action == intent.ActionContinue
	governance.CanProposeFiles = decision.Mutation == intent.MutationWorkspace
	governance.CanRunImplSession = governance.RequiresWorkspace
	if governance.ComposerMode == "ask" || governance.ComposerMode == "plan" {
		governance.CanProposeFiles = false
		governance.CanRunImplSession = false
		governance.RequiresWorkspace = false
	}
	protocol.StampTurnGovernance(msg, governance)
}

func stampCanonicalGovernance(msg *protocol.Message) {
	if msg == nil {
		return
	}
	mode := protocol.ComposerModeFromMessage(msg)
	if mode == "" {
		mode = "agent"
	}
	canAct := mode == "agent" || mode == "export"
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: mode, ContextTier: protocol.ContextScopeFromMessage(msg),
		CanProposeFiles: canAct, CanRunImplSession: canAct,
		Provenance: "server_canonical",
	})
}

func (h *Hub) semanticTurnFeatures(msg *protocol.Message) intent.TurnFeatures {
	mode := protocol.ComposerModeFromMessage(msg)
	if mode == "" {
		mode = "agent"
	}
	canMutate := mode == "agent" || mode == "export"
	features := intent.TurnFeatures{
		Text:                 strings.TrimSpace(msg.Content),
		ComposerMode:         mode,
		ExplicitRecipient:    strings.TrimSpace(msg.IdeRouteAgentType()),
		ReplyTarget:          strings.TrimSpace(msg.ReplyTo),
		CollaborationPhase:   strings.TrimSpace(msg.GetCollaborationPhase()),
		IsSlashCommand:       strings.HasPrefix(strings.TrimSpace(msg.Content), "/"),
		IsDirectMessage:      h.isChannelDM(msg.Channel),
		HasExplicitMention:   len(msg.Mentions) > 0,
		HasWorkspace:         semanticMessageHasWorkspace(msg),
		CanProposeFiles:      canMutate,
		CanRunImplementation: canMutate,
	}
	if msg.Metadata != nil {
		if raw, ok := msg.Metadata["requested_action"].(string); ok {
			features.ExplicitAction = intent.Action(strings.TrimSpace(raw))
			if err := (&intent.SemanticIntent{
				SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
				RequestedAction: features.ExplicitAction, MutationRequested: intent.MutationNone, Confidence: 1,
			}).Validate(); err != nil {
				features.ExplicitAction = ""
			}
		}
	}

	state := h.GetChannelConversationState(msg.Channel)
	if state != nil {
		for _, action := range state.Actions {
			if action.CompletedAt != nil {
				continue
			}
			if features.PendingActionID == "" || action.PromisedAt.After(pendingActionTime(state, features.PendingActionID)) {
				features.PendingActionID = action.ID
				features.PendingDescription = action.Description
				features.PendingAction = intent.Action(action.Action)
			}
		}
	}
	features.RecentExchanges = h.semanticRecentExchanges(msg.Channel, msg.ID, 6)
	return features
}

func pendingActionTime(state *ChannelConversationState, id string) (zeroTime time.Time) {
	if state == nil {
		return zeroTime
	}
	if action, ok := state.Actions[id]; ok {
		return action.PromisedAt
	}
	return zeroTime
}

func (h *Hub) semanticRecentExchanges(channel, skipID string, limit int) []intent.Exchange {
	h.mu.RLock()
	messages := append([]*protocol.Message(nil), h.messages[channel]...)
	h.mu.RUnlock()
	if limit <= 0 {
		limit = 6
	}
	out := make([]intent.Exchange, 0, limit)
	for i := len(messages) - 1; i >= 0 && len(out) < limit; i-- {
		message := messages[i]
		if message == nil || message.ID == skipID {
			continue
		}
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		if len(content) > 500 {
			content = content[:500]
		}
		role := "assistant"
		if protocol.IsUserLikeSender(message.From) {
			role = "user"
		}
		out = append(out, intent.Exchange{Role: role, Content: content})
	}
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}

func semanticMessageHasWorkspace(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	for _, key := range []string{"workspace_context", "workspace_path", "workspace_root", "repo_path"} {
		value, ok := msg.Metadata[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return true
			}
		case map[string]interface{}:
			if len(typed) > 0 {
				return true
			}
		default:
			return true
		}
	}
	return false
}
