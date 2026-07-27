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
	// Join announcements, system_info, approvals, and other non-chat traffic must
	// not invoke the classifier or stamp implementation/continuation authority.
	if !semanticTurnEligible(msg) {
		return
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
		// Keep client/scenario routing (e.g. implement harness ide_route_agent_type=frontend).
		// Overwriting with a classifier recipient (often "assistant") silences the target agent
		// on public scenario channels where Assistant is not listening.
		if strings.TrimSpace(msg.IdeRouteAgentType()) == "" {
			msg.Metadata[protocol.IdeMetaRouteAgentType] = decision.RecipientType
		}
	}
	wantsImpl := decision.Mutation == intent.MutationWorkspace &&
		(decision.Action == intent.ActionDebug || decision.Action == intent.ActionEdit ||
			decision.Action == intent.ActionContinue || decision.Action == intent.ActionRun)
	mode := protocol.ComposerModeFromMessage(msg)
	planOrAsk := mode == "ask" || mode == "plan"
	// Preserve client/scenario implementation_session when semantic does not request mutation
	// (e.g. boot-fix paste classified as Answer).
	explicitSession := msg.ImplementationSession() && !planOrAsk
	// conversation_mode=chat is advisory: do not promote semantic Edit into an
	// implementation session unless the client/scenario already opted in.
	chatAdvisory := false
	if raw, ok := msg.Metadata["conversation_mode"].(string); ok {
		chatAdvisory = strings.EqualFold(strings.TrimSpace(raw), "chat")
	}
	advisoryQuestion := intent.LooksLikeAdvisoryImplementationQuestion(msg.Content)
	if (chatAdvisory || advisoryQuestion) && !explicitSession {
		wantsImpl = false
		// Keep advisory turns as answer/none so design/topic-switch prompts
		// are not stamped inspect/edit/image (scenarios and tooling rely on this).
		// Explicit image/artifact phrases are re-promoted later in turn-goal derivation.
		decision.Action = intent.ActionAnswer
		decision.RequestedAction = intent.ActionAnswer
		decision.Mutation = intent.MutationNone
		if err := protocol.StampTurnDecision(msg, decision); err != nil {
			return
		}
	}
	if explicitSession {
		wantsImpl = true
	}
	if wantsImpl && !planOrAsk {
		msg.Metadata[protocol.IdeMetaImplementationSession] = true
		msg.Metadata[protocol.TurnMetaCanProposeFiles] = true
		msg.Metadata[protocol.TurnMetaCanRunImplSession] = true
		msg.Metadata[protocol.TurnMetaRequiresWorkspace] = true
	}
	if (chatAdvisory || advisoryQuestion) && !explicitSession && !planOrAsk {
		delete(msg.Metadata, protocol.IdeMetaImplementationSession)
		delete(msg.Metadata, protocol.TurnMetaCanProposeFiles)
		delete(msg.Metadata, protocol.TurnMetaCanRunImplSession)
		delete(msg.Metadata, protocol.TurnMetaRequiresWorkspace)
	}
	governance, _ := protocol.ExtractTurnGovernance(msg)
	if governance.ComposerMode == "" {
		governance.ComposerMode = mode
	}
	governance.RequiresWorkspace = wantsImpl && !planOrAsk
	governance.CanProposeFiles = (decision.Mutation == intent.MutationWorkspace || explicitSession) && !planOrAsk && !((chatAdvisory || advisoryQuestion) && !explicitSession)
	governance.CanRunImplSession = wantsImpl && !planOrAsk
	if (chatAdvisory || advisoryQuestion) && !explicitSession {
		governance.CanProposeFiles = false
		governance.CanRunImplSession = false
		governance.RequiresWorkspace = false
	}
	if governance.ComposerMode == "ask" || governance.ComposerMode == "plan" {
		governance.CanProposeFiles = false
		governance.CanRunImplSession = false
		governance.RequiresWorkspace = false
		delete(msg.Metadata, protocol.TurnMetaCanProposeFiles)
		delete(msg.Metadata, protocol.TurnMetaCanRunImplSession)
		delete(msg.Metadata, protocol.TurnMetaRequiresWorkspace)
	}
	protocol.StampTurnGovernance(msg, governance)
}

// semanticTurnEligible reports whether a message is a real user turn that should
// receive a canonical semantic decision.
func semanticTurnEligible(msg *protocol.Message) bool {
	if msg == nil || !protocol.IsUserLikeSender(msg.From) || msg.IsFromSystem() {
		return false
	}
	switch msg.Type {
	case protocol.MessageTypeChat, protocol.MessageTypeQuestion:
		return strings.TrimSpace(msg.Content) != ""
	default:
		return false
	}
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
	if id, renderer, title := h.semanticOpenCanvasArtifact(msg.Channel, msg.ID); id != "" {
		features.OpenArtifactID = id
		features.OpenArtifactRenderer = renderer
		features.OpenArtifactTitle = title
	}
	return features
}

func (h *Hub) semanticOpenCanvasArtifact(channel, skipID string) (id, renderer, title string) {
	h.mu.RLock()
	messages := append([]*protocol.Message(nil), h.messages[channel]...)
	h.mu.RUnlock()
	seen := 0
	for i := len(messages) - 1; i >= 0 && seen < 40; i-- {
		message := messages[i]
		if message == nil || message.ID == skipID {
			continue
		}
		seen++
		ref, ok := messageArtifactReference(message)
		if !ok || strings.TrimSpace(ref.ID) == "" {
			continue
		}
		rid := strings.TrimSpace(ref.RendererID)
		if rid == "" {
			continue
		}
		return ref.ID, rid, strings.TrimSpace(ref.Title)
	}
	return "", "", ""
}

func messageArtifactReference(msg *protocol.Message) (protocol.ArtifactReference, bool) {
	if msg == nil || msg.Metadata == nil {
		return protocol.ArtifactReference{}, false
	}
	raw, ok := msg.Metadata["artifact_ref"]
	if !ok || raw == nil {
		return protocol.ArtifactReference{}, false
	}
	switch v := raw.(type) {
	case protocol.ArtifactReference:
		return v, strings.TrimSpace(v.ID) != ""
	case *protocol.ArtifactReference:
		if v == nil {
			return protocol.ArtifactReference{}, false
		}
		return *v, strings.TrimSpace(v.ID) != ""
	case map[string]interface{}:
		ref := protocol.ArtifactReference{}
		if id, _ := v["id"].(string); id != "" {
			ref.ID = id
		}
		if t, _ := v["title"].(string); t != "" {
			ref.Title = t
		}
		if rid, _ := v["renderer_id"].(string); rid != "" {
			ref.RendererID = rid
		}
		if media, _ := v["media_type"].(string); media != "" {
			ref.MediaType = media
		}
		return ref, strings.TrimSpace(ref.ID) != ""
	default:
		return protocol.ArtifactReference{}, false
	}
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
