package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/delegation"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/memory"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/workflow"
	"github.com/google/uuid"
)

func (h *Hub) SendMessage(msg *protocol.Message) error {
	h.inheritCollaborationFromChannel(msg)
	agent.AttachUserRulesMetadataIfMissing(msg)
	if msg != nil && msg.IdeRouteAgentType() != "" && msg.Channel != "" {
		h.MarkChannelDurable(msg.Channel)
	}
	// Human message clears channel hold (user interject / resume).
	if msg != nil && protocol.IsUserLikeSender(msg.From) && msg.Channel != "" && h.IsChannelHeld(msg.Channel) {
		h.SetChannelHold(msg.Channel, false, "")
		h.broadcastChannelHold(msg.Channel, false)
	}
	if err := h.enforceExecutionMessageBudget(msg); err != nil {
		return err
	}
	if err := h.rejectClosedCollaborationChannel(msg); err != nil {
		return err
	}
	h.normalizeCollabTurnHandoffMentions(msg)
	h.processCollaborationLifecycle(msg)
	h.attachCollaborationData(msg)

	// Only parse actionable @mentions for user-like senders. Agent responses
	// naturally contain @ symbols (file paths, code references) that should
	// not be treated as mention attempts.
	mentionStrings := []string{}
	allowMentionValidationErrors := protocol.ShouldParseMentions(msg.Type, msg.From)
	if allowMentionValidationErrors || h.shouldParseCollaborationMentions(msg) {
		mentionStrings = protocol.ParseMentions(msg.Content)
		if h.shouldParseCollaborationMentions(msg) {
			mentionStrings = protocol.FilterCollabTemplateMentions(mentionStrings)
		}
	}
	hasInvalidMentions := false
	isDMInbound := protocol.IsUserLikeSender(msg.From) && h.isChannelDM(msg.Channel)

	if len(mentionStrings) > 0 {
		// Resolve mentions and check for unresolved ones
		resolvedMentions := make(map[string]bool) // track which mentions were resolved
		agentIDs := h.ResolveMentionsWithValidation(mentionStrings, resolvedMentions, msg.Channel)
		wakeIDs, consultIDs := h.partitionCollabMentionTargets(msg, agentIDs)
		msg.Mentions = wakeIDs
		h.EnsureRoomMentionedAgents(msg.Channel, wakeIDs)

		h.maybeRequestCollaborationParticipants(msg, agentIDs)
		h.maybeCollabConsult(msg, consultIDs)

		// Send system messages for unresolved mentions (user-authored mentions only).
		if allowMentionValidationErrors && !isDMInbound {
			for _, mention := range mentionStrings {
				if !resolvedMentions[mention] {
					hasInvalidMentions = true

					// Send error message for not found agent
					errorMsg := protocol.NewMessage(
						protocol.MessageTypeSystemInfo,
						msg.Channel,
						protocol.AgentInfo{
							ID:   "system",
							Name: "System",
							Type: protocol.AgentTypeGeneral,
						},
						fmt.Sprintf("❌ Agent @%s not found. Available agents: %s",
							mention, h.getAgentListString()),
					)

					// Lock and send error message immediately
					h.mu.Lock()
					h.appendChannelMessageLocked(msg.Channel, errorMsg)
					h.broadcast(msg.Channel, errorMsg)
					h.mu.Unlock()
				}
			}
		}

		// If all mentions were invalid, don't process the message further
		// This prevents agents from responding to invalid @mentions
		if allowMentionValidationErrors && hasInvalidMentions && len(agentIDs) == 0 {
			if isDMInbound {
				h.normalizeDMMentionRouting(msg, mentionStrings, agentIDs)
			} else {
				// Set mentions to a dummy value so agents will see HasMentions() = true
				// but IsMentioned(agentID) = false, preventing all agents from responding
				msg.Mentions = []string{"__INVALID__"}

				// Store the message for history so user can see what they typed
				h.mu.Lock()
				h.appendChannelMessageLocked(msg.Channel, msg)
				h.mu.Unlock()
				return nil
			}
		}
	}

	if isDMInbound {
		h.normalizeDMMentionRouting(msg, mentionStrings, msg.Mentions)
	}

	// Echo eligible user turns to the UI before semantic classification. The
	// classifier can take seconds (Ollama); without this, /api/send and the
	// WebSocket echo both stall until Resolve returns.
	uiEchoed := h.echoUserTurnToUI(msg)

	h.resolveSemanticTurn(context.Background(), msg)

	// Slash commands: persist the human command line, then the system response (if any).
	if h.commandHandler != nil && len(msg.Content) > 0 && msg.Content[0] == '/' {
		commandMsg, cloneErr := protocol.CloneMessage(msg)
		if cloneErr != nil {
			return fmt.Errorf("slash command clone: %w", cloneErr)
		}
		if commandMsg.Metadata == nil {
			commandMsg.Metadata = make(map[string]interface{})
		}
		commandMsg.Metadata[protocol.MetadataSlashCommand] = true

		ctx := context.Background()
		response, err := h.commandHandler.ProcessCommand(ctx, msg)
		if err != nil {
			return fmt.Errorf("command processing error: %w", err)
		}
		if response != nil {
			h.attachCollaborationData(response)
		}
		defer h.noteChannelActivity(commandMsg)
		return h.persistSlashCommandExchange(commandMsg, response)
	}

	// Check for automatic repository agent creation
	if h.commandHandler != nil && !msg.IsFromSystem() {
		// Detect local file paths in the message
		pathResult := protocol.DetectLocalPaths(msg.Content)
		if pathResult.Found {
			// Get the best path for repository analysis
			bestPath := protocol.GetBestPathForRepository(pathResult.Paths)
			if bestPath != nil {
				// Check if we should auto-create a repo agent
				shouldAutoCreate := h.shouldAutoCreateRepoAgent(msg, bestPath.Path)
				if shouldAutoCreate {
					// Auto-create repository agent
					h.autoCreateRepoAgent(msg, bestPath.Path)
				}
			}
		}
	}

	// Intercept file change proposals and register with FileChangeManager
	var fileChangeRegErr error
	if msg.Type == protocol.MessageTypeFileChange && msg.Metadata != nil {
		if proposalRaw, ok := msg.Metadata["file_change_proposal"]; ok {
			fileChangeRegErr = h.registerFileChangeProposal(msg, proposalRaw)
			if fileChangeRegErr != nil {
				if msg.Metadata == nil {
					msg.Metadata = make(map[string]interface{})
				}
				msg.Metadata[protocol.MetaFileChangeRegistrationError] = fileChangeRegErr.Error()
			}
		}
	}
	if msg.Metadata != nil {
		if proposalRaw, ok := msg.Metadata["git_change_proposal"]; ok {
			h.registerGitChangeProposal(msg, proposalRaw)
		}
	}

	activityMsg := msg
	defer h.noteChannelActivity(activityMsg)

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[msg.Channel]; !ok {
		return fmt.Errorf("channel %s not found", msg.Channel)
	}

	// Ephemeral message types -- broadcast to subscribers but don't persist in history
	if msg.Type == protocol.MessageTypeStreamDelta || msg.Type == protocol.MessageTypeStreamEnd || msg.Type == protocol.MessageTypeAgentStatus {
		h.broadcast(msg.Channel, msg)
		return nil
	}

	// Closed-collab file proposals must not appear in transcript history. Registration
	// already rejected them; skipping append keeps deny_file_change_after_cancel green.
	if fileChangeRegErr != nil && isClosedCollabFileChangeRejection(fileChangeRegErr) {
		h.postFileChangeRegistrationFailureLocked(msg, fileChangeRegErr)
		return fileChangeRegErr
	}

	// Handle thread messages separately
	if msg.IsInThread() {
		threadID := msg.GetThreadID()

		// Track the parent message author (first time we see this thread)
		if _, exists := h.threadParentAuthors[threadID]; !exists {
			// Find the parent message (threadID == parent message ID)
			for _, channelMsg := range h.messages[msg.Channel] {
				if channelMsg.ID == threadID {
					h.threadParentAuthors[threadID] = channelMsg.From.ID
					break
				}
			}
		}

		// Add to thread storage
		h.appendThreadMessageLocked(threadID, msg)

		// Update thread metadata
		h.updateThreadMetadata(threadID, msg)

		// Broadcast to thread subscribers (for thread panel UI updates)
		h.broadcastToThread(threadID, msg)

		// ALSO add to channel message history so agents can see it when polling
		h.appendChannelMessageLocked(msg.Channel, msg)

		// ALSO broadcast to channel subscribers (so agents can see mentions)
		h.broadcast(msg.Channel, msg)
	} else if uiEchoed {
		// History already has the pre-classify echo; replace with stamped metadata
		// and deliver to agents only (UI already saw the message).
		h.appendChannelMessageLocked(msg.Channel, msg)
		if deliverToAgentTier(msg) {
			h.deliverToSubscribers(h.subscribers[msg.Channel], msg)
		}
		// Refresh UI with stamped turn decision / governance.
		h.deliverToSubscribers(h.uiSubscribers[msg.Channel], msg)
	} else {
		// Regular channel message
		if h.shouldSkipHumanJoinAnnouncementLocked(msg.Channel, msg) {
			return nil
		}
		h.appendChannelMessageLocked(msg.Channel, msg)
		h.broadcast(msg.Channel, msg)
	}

	if fileChangeRegErr != nil {
		h.postFileChangeRegistrationFailureLocked(msg, fileChangeRegErr)
		return fileChangeRegErr
	}

	return nil
}

// echoUserTurnToUI persists and UI-broadcasts an eligible user message before
// semantic classification so the chat timeline updates immediately.
//
// The echo is a clone: resolveSemanticTurn mutates msg.Metadata after this returns,
// and agent history replay / UI subscribers must not share that map.
func (h *Hub) echoUserTurnToUI(msg *protocol.Message) bool {
	if h == nil || msg == nil || !semanticTurnEligible(msg) {
		return false
	}
	if strings.HasPrefix(strings.TrimSpace(msg.Content), "/") {
		return false
	}
	if msg.IsInThread() {
		return false
	}
	echo, err := protocol.CloneMessage(msg)
	if err != nil || echo == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.channels[msg.Channel]; !ok {
		return false
	}
	if h.shouldSkipHumanJoinAnnouncementLocked(msg.Channel, msg) {
		return false
	}
	h.appendChannelMessageLocked(msg.Channel, echo)
	h.deliverToSubscribers(h.uiSubscribers[msg.Channel], echo)
	return true
}

func (h *Hub) inheritCollaborationFromChannel(msg *protocol.Message) {
	if msg == nil || h.collabManager == nil || msg.GetCollaborationID() != "" {
		return
	}
	snapshot := h.collabManager.GetByChannel(msg.Channel)
	if snapshot == nil {
		return
	}
	msg.SetCollaborationID(snapshot.ID)
	if snapshot.Phase != "" {
		msg.SetCollaborationPhase(string(snapshot.Phase))
	}
}

func (h *Hub) shouldParseCollaborationMentions(msg *protocol.Message) bool {
	if msg == nil || h.collabManager == nil || msg.IsFromSystem() {
		return false
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return false
	}
	if msg.Type != protocol.MessageTypeCollabDiscussion &&
		msg.Type != protocol.MessageTypeChat &&
		msg.Type != protocol.MessageTypeAnswer &&
		msg.Type != protocol.MessageTypeCollabPlan {
		return false
	}
	if !h.collabManager.IsParticipant(collabID, msg.From.ID) {
		return false
	}
	snapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snapshot == nil {
		return false
	}
	return snapshot.Phase == collaboration.PhasePlanning || snapshot.Phase == collaboration.PhaseReviewing
}

func (h *Hub) maybeRequestCollaborationParticipants(msg *protocol.Message, mentionedAgentIDs []string) {
	if msg == nil || h.collabManager == nil || msg.IsFromSystem() {
		return
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" || len(mentionedAgentIDs) == 0 {
		return
	}
	snapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snapshot == nil {
		return
	}
	if snapshot.Phase != collaboration.PhasePlanning && snapshot.Phase != collaboration.PhaseReviewing {
		return
	}
	if !h.collabManager.IsParticipant(collabID, msg.From.ID) {
		return
	}
	if !snapshot.AllowAgentParticipantRequests {
		return
	}

	candidates := make([]string, 0, len(mentionedAgentIDs))
	for _, agentID := range mentionedAgentIDs {
		if agentID == "" || h.collabManager.IsParticipant(collabID, agentID) {
			continue
		}
		candidates = append(candidates, agentID)
	}
	if len(candidates) == 0 {
		return
	}

	requests, err := h.collabManager.RequestParticipantAdds(collabID, msg.From.ID, msg.From.Name, candidates)
	if err != nil || len(requests) == 0 {
		return
	}

	for _, req := range requests {
		h.sendParticipantAddRequestNotice(msg.Channel, collabID, string(snapshot.Phase), req)
	}
}

const maxCollabConsultsPerMessage = 2

// partitionCollabMentionTargets splits mentions into wake (participants / L2 path) vs L1 consult targets.
// When allow-agent-adds is off, non-participants are consult-only and must not be woken via channel mention.
func (h *Hub) partitionCollabMentionTargets(msg *protocol.Message, mentionedAgentIDs []string) (wakeIDs, consultIDs []string) {
	if msg == nil || h.collabManager == nil || len(mentionedAgentIDs) == 0 {
		return mentionedAgentIDs, nil
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return mentionedAgentIDs, nil
	}
	snapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snapshot == nil {
		return mentionedAgentIDs, nil
	}
	switch snapshot.Phase {
	case collaboration.PhasePlanning, collaboration.PhaseReviewing, collaboration.PhaseExecuting, collaboration.PhaseApproved:
	default:
		return mentionedAgentIDs, nil
	}
	if !h.collabManager.IsParticipant(collabID, msg.From.ID) {
		return mentionedAgentIDs, nil
	}
	if snapshot.AllowAgentParticipantRequests {
		// L2: keep mentions for join-request path; participants still wake normally.
		return mentionedAgentIDs, nil
	}
	wakeIDs = make([]string, 0, len(mentionedAgentIDs))
	consultIDs = make([]string, 0)
	for _, agentID := range mentionedAgentIDs {
		if agentID == "" {
			continue
		}
		if h.collabManager.IsParticipant(collabID, agentID) {
			wakeIDs = append(wakeIDs, agentID)
			continue
		}
		consultIDs = append(consultIDs, agentID)
	}
	return wakeIDs, consultIDs
}

// maybeCollabConsult posts visible L1 consult answers for non-participants when expansion is disabled.
func (h *Hub) maybeCollabConsult(msg *protocol.Message, consultAgentIDs []string) {
	if msg == nil || h.collabManager == nil || h.commandHandler == nil || msg.IsFromSystem() {
		return
	}
	if len(consultAgentIDs) == 0 {
		return
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return
	}
	snapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snapshot == nil {
		return
	}
	switch snapshot.Phase {
	case collaboration.PhasePlanning, collaboration.PhaseReviewing, collaboration.PhaseExecuting, collaboration.PhaseApproved:
	default:
		return
	}
	if snapshot.AllowAgentParticipantRequests {
		return
	}
	if !h.collabManager.IsParticipant(collabID, msg.From.ID) {
		return
	}

	ctx := context.Background()
	seen := make(map[string]bool)
	n := 0
	for _, agentID := range consultAgentIDs {
		if agentID == "" || seen[agentID] || h.collabManager.IsParticipant(collabID, agentID) {
			continue
		}
		seen[agentID] = true
		if n >= maxCollabConsultsPerMessage {
			break
		}
		info, err := h.GetAgent(agentID)
		if err != nil || info == nil {
			log.Printf("[Collaboration] L1 consult target %s not found: %v", agentID, err)
			continue
		}
		banner := protocol.NewMessage(
			protocol.MessageTypeCollabStatus,
			msg.Channel,
			protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
			fmt.Sprintf("🔎 Consulting @%s (ask only — not joining the collaboration)…", info.Name),
		)
		banner.SetCollaborationID(collabID)
		banner.SetCollaborationPhase(string(snapshot.Phase))
		if banner.Metadata == nil {
			banner.Metadata = map[string]interface{}{}
		}
		banner.Metadata["collab_internal_event"] = true
		banner.Metadata["event"] = "collab-consult-start"
		banner.Metadata["consulted_agent_id"] = info.ID
		banner.Metadata["consulted_agent_name"] = info.Name
		banner.Metadata["consulted_by_id"] = msg.From.ID
		banner.Metadata["consulted_by_name"] = msg.From.Name
		_ = h.SendMessage(banner)

		res, err := h.commandHandler.CollabVisibleConsult(ctx, delegation.ConsultRequest{
			FromID:      msg.From.ID,
			FromName:    msg.From.Name,
			ToID:        info.ID,
			SubQuestion: msg.Content,
			Channel:     msg.Channel,
			Depth:       0,
		})
		answer := strings.TrimSpace(res.Text)
		if err != nil || answer == "" {
			fail := protocol.NewMessage(
				protocol.MessageTypeCollabStatus,
				msg.Channel,
				protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
				fmt.Sprintf("⚠️ Consult to @%s failed: %v", info.Name, err),
			)
			fail.SetCollaborationID(collabID)
			fail.SetCollaborationPhase(string(snapshot.Phase))
			if fail.Metadata == nil {
				fail.Metadata = map[string]interface{}{}
			}
			fail.Metadata["collab_internal_event"] = true
			fail.Metadata["event"] = "collab-consult-error"
			_ = h.SendMessage(fail)
			continue
		}
		reply := protocol.NewMessage(
			protocol.MessageTypeAnswer,
			msg.Channel,
			*info,
			answer,
		)
		reply.SetCollaborationID(collabID)
		reply.SetCollaborationPhase(string(snapshot.Phase))
		if reply.Metadata == nil {
			reply.Metadata = map[string]interface{}{}
		}
		reply.Metadata["collab_internal_event"] = true
		reply.Metadata["event"] = "collab-consult"
		reply.Metadata["collab_consult"] = true
		reply.Metadata["consulted_by_id"] = msg.From.ID
		reply.Metadata["consulted_by_name"] = msg.From.Name
		if res.Intent != "" {
			reply.Metadata["consult_intent"] = string(res.Intent)
		}
		if err := h.SendMessage(reply); err != nil {
			log.Printf("[Collaboration] Failed to post L1 consult from %s: %v", info.Name, err)
			continue
		}
		n++
	}
}

func (h *Hub) sendParticipantAddRequestNotice(channel, collabID, phase string, req collaboration.ParticipantAddRequest) {
	notice := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("➕ @%s wants to add @%s to collaboration `%s`. Approve this in the app to invite them.", req.RequestedByName, req.AgentName, collabID[:8]),
	)
	notice.SetCollaborationID(collabID)
	notice.SetCollaborationPhase(phase)
	if notice.Metadata == nil {
		notice.Metadata = map[string]interface{}{}
	}
	notice.Metadata["collab_internal_event"] = true
	notice.Metadata["event"] = "collab-participant-add-request"
	notice.Metadata["requested_agent_id"] = req.AgentID
	notice.Metadata["requested_agent_name"] = req.AgentName
	notice.Metadata["requested_by_id"] = req.RequestedByID
	notice.Metadata["requested_by_name"] = req.RequestedByName
	if err := h.SendMessage(notice); err != nil {
		log.Printf("[Collaboration] Failed to broadcast participant add request: %v", err)
	}
}

// ApproveCollaborationParticipantRequest adds a user-approved pending participant.
func (h *Hub) ApproveCollaborationParticipantRequest(collabID, agentID string) (*collaboration.Collaboration, error) {
	if h.collabManager == nil {
		return nil, fmt.Errorf("collaboration manager unavailable")
	}
	snap, participant, err := h.collabManager.ApproveParticipantAddRequest(collabID, agentID)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if participant != nil {
		if err := h.AddAgentToChannel(participant.AgentID, snap.Channel); err != nil {
			return nil, err
		}
		if h.commandHandler != nil {
			if err := h.commandHandler.EnsureAgentSubscribedToChannel(context.Background(), participant.AgentID, snap.Channel); err != nil {
				log.Printf("[Collaboration] Failed to subscribe %s to %s: %v", participant.AgentName, snap.Channel, err)
			}
			h.commandHandler.setCollabClientOnAgent(participant.AgentID, participant.AgentName, h.NewCollaborationClientAdapter())
		}
	}
	if participant != nil {
		notice := protocol.NewMessage(
			protocol.MessageTypeCollabStatus,
			snap.Channel,
			protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
			fmt.Sprintf("➕ Added @%s to collaboration `%s` after user approval.", participant.AgentName, collabID[:8]),
		)
		notice.SetCollaborationID(collabID)
		notice.SetCollaborationPhase(string(snap.Phase))
		if notice.Metadata == nil {
			notice.Metadata = map[string]interface{}{}
		}
		notice.Metadata["collab_internal_event"] = true
		h.attachCollaborationData(notice)
		if err := h.SendMessage(notice); err != nil {
			log.Printf("[Collaboration] Failed to broadcast participant approval notice: %v", err)
		}
	}
	return h.collabManager.GetCollaborationSnapshot(collabID)
}

// DenyCollaborationParticipantRequest clears a pending participant add request.
func (h *Hub) DenyCollaborationParticipantRequest(collabID, agentID string) (*collaboration.Collaboration, error) {
	if h.collabManager == nil {
		return nil, fmt.Errorf("collaboration manager unavailable")
	}
	snap, err := h.collabManager.DenyParticipantAddRequest(collabID, agentID)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

func (h *Hub) processCollaborationLifecycle(msg *protocol.Message) {
	if msg == nil {
		return
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" || h.collabManager == nil {
		return
	}
	if h.maybeProcessRecapReply(msg) {
		return
	}
	if msg.Metadata != nil {
		if internal, ok := msg.Metadata["collab_internal_event"].(bool); ok && internal {
			return
		}
	}

	if protocol.IsUserLikeSender(msg.From) && !msg.IsFromSystem() {
		h.maybeKickPlanningDiscussionOnHumanMessage(collabID)
	}

	h.maybeIngestPlanArtifact(msg, collabID)
	h.maybeSyncTaskStatusFromPlanHandoff(msg, collabID)
	h.maybeUpdateTaskStatus(msg, collabID)
	NewCollabScheduler(h).OnGenerationError(collabID, msg)
}

func (h *Hub) maybeIngestPlanArtifact(msg *protocol.Message, collabID string) {
	if msg.IsFromSystem() {
		return
	}
	if phase := msg.GetCollaborationPhase(); phase != "" && phase != string(collaboration.PhasePlanning) {
		return
	}
	if msg.Type != protocol.MessageTypeChat &&
		msg.Type != protocol.MessageTypeAnswer &&
		msg.Type != protocol.MessageTypeCollabDiscussion {
		return
	}

	planContent := collaboration.ExtractPlanFromResponse(msg.Content)
	if strings.TrimSpace(planContent) == "" {
		return
	}

	collabSnapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || collabSnapshot == nil {
		return
	}
	if collabSnapshot.Phase != collaboration.PhasePlanning {
		return
	}
	// Exact-line goals: keep pinned task list; freestyle discussion plans often explode or under-parse.
	if collaboration.CollaborationPinsExactGoalTasks(collabSnapshot) {
		if err := h.collabManager.ApplyPinnedGoalTasks(collabID); err != nil {
			log.Printf("[Collaboration] Failed to keep pinned goal tasks for %s: %v", collabID[:8], err)
		}
		return
	}
	if collabSnapshot.Plan != nil && strings.TrimSpace(collabSnapshot.Plan.Content) == strings.TrimSpace(planContent) {
		return
	}
	if ok, reason := collaboration.ValidatePlanForCollaboration(collabSnapshot, planContent); !ok {
		log.Printf("[Collaboration] Skipping corrupt plan ingest for %s: %s", collabID[:8], reason)
		hint := protocol.NewMessage(
			protocol.MessageTypeSystemInfo,
			msg.Channel,
			protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
			fmt.Sprintf("⚠️ Plan update from @%s was not applied (%s). Revise task rows with @Agent and concrete file paths.", msg.From.Name, reason),
		)
		hint.SetCollaborationID(collabID)
		hint.SetCollaborationPhase(string(collaboration.PhasePlanning))
		if hint.Metadata == nil {
			hint.Metadata = map[string]interface{}{}
		}
		hint.Metadata["collab_internal_event"] = true
		_ = h.SendMessage(hint)
		return
	}

	if err := h.collabManager.UpdateArtifact(collabID, msg.From.ID, msg.From.Name, planContent); err != nil {
		log.Printf("[Collaboration] Failed to auto-update plan artifact for %s: %v", collabID[:8], err)
		return
	}

	updated, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || updated == nil {
		return
	}
	extractedTasks := collaboration.ExtractTasksFromPlan(planContent, updated.Agents)
	if len(extractedTasks) > 0 {
		if err := h.collabManager.SetTasks(collabID, extractedTasks); err != nil {
			log.Printf("[Collaboration] Failed to auto-set tasks for %s: %v", collabID[:8], err)
		}
		updated, _ = h.collabManager.GetCollaborationSnapshot(collabID)
	}

	planMsg := protocol.NewMessage(
		protocol.MessageTypeCollabPlan,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("🧩 Updated collaboration plan (v%d) based on @%s's proposal.",
			func() int {
				if updated != nil && updated.Plan != nil {
					return updated.Plan.Version
				}
				return 0
			}(),
			msg.From.Name,
		),
	)
	planMsg.SetCollaborationID(collabID)
	planMsg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	planMsg.SetArtifactAction("edit")
	if planMsg.Metadata == nil {
		planMsg.Metadata = map[string]interface{}{}
	}
	planMsg.Metadata["collab_internal_event"] = true
	if err := h.SendMessage(planMsg); err != nil {
		log.Printf("[Collaboration] Failed to broadcast plan update message: %v", err)
	}
	h.persistCollaborationReviewAssets(collabID)

	// Solo collab: short-circuit planning → reviewing after the first valid *agent* plan.
	// Do not short-circuit on user /collaborate text (goals often embed Task N examples),
	// which would skip discussion and break multi-collab isolation probes.
	if updated != nil && updated.IsSolo() &&
		!protocol.IsUserLikeSender(msg.From) && !msg.IsFromSystem() {
		if _, err := h.collabManager.TransitionToReviewing(collabID); err != nil {
			log.Printf("[Collaboration] Solo short-circuit to reviewing for %s: %v", collabID[:8], err)
		}
	}
}

func (h *Hub) maybeUpdateTaskStatus(msg *protocol.Message, collabID string) {
	// Task-assignment prompts carry task_id/task_status for routing; they are not
	// assignee status reports. Treating them here spuriously "updates" tasks to
	// pending and broadcasts duplicate collab_status noise.
	if msg.Type == protocol.MessageTypeCollabTask {
		return
	}
	if msg.Metadata != nil {
		if ge, ok := msg.Metadata["generation_error"].(bool); ok && ge {
			// Failed turns are healed by maybeRedispatchAfterCollabGenerationError.
			return
		}
	}

	taskID := msg.GetTaskID()
	if taskID == "" {
		return
	}

	collabSnapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || collabSnapshot == nil {
		return
	}

	var task *collaboration.CollaborationTask
	for i := range collabSnapshot.Tasks {
		if collabSnapshot.Tasks[i].ID == taskID {
			task = &collabSnapshot.Tasks[i]
			break
		}
	}
	if task == nil {
		return
	}

	strict := collabSnapshot.EffectiveExecutionPolicy().StrictTaskStatus
	inferred := collaboration.InferTaskStatusFromAgentReply(msg.Content, strict)
	var status collaboration.TaskStatus
	var ok bool
	if inferred != "" {
		status = inferred
		ok = true
	} else if metaStatus, metaOK := normalizeTaskStatus(msg.GetTaskStatus()); metaOK {
		if metaStatus != collaboration.TaskPending || msg.From.ID != task.AssignedTo {
			status = metaStatus
			ok = true
		}
	}
	if !ok && task.Status == collaboration.TaskPending && msg.From.ID == task.AssignedTo && !msg.IsFromSystem() {
		status = collaboration.TaskInProgress
		ok = true
	}
	if !ok {
		return
	}

	if status == collaboration.TaskCompleted {
		policy := collabSnapshot.EffectiveExecutionPolicy().BlockedUpstreamPolicy
		if !collaboration.UpstreamTasksComplete(*task, collabSnapshot.Tasks, policy) {
			h.broadcastCollabSystem(msg.Channel, collabID, fmt.Sprintf(
				"⚠️ **Task not marked complete** (`%s`) — @%s reported done but upstream task(s) are not finished yet.",
				collabID[:8], msg.From.Name,
			))
			if task.Status != collaboration.TaskInProgress {
				_, _ = h.collabManager.UpdateTaskStatusWithEffects(collabID, task.ID, collaboration.TaskInProgress, "Awaiting upstream tasks")
			}
			return
		}
		if collaboration.AgentReplyContainsStalePlanning(msg.Content) {
			h.broadcastCollabSystem(msg.Channel, collabID, fmt.Sprintf(
				"⚠️ **Task not marked complete** (`%s`) — @%s echoed planning/approval language during execution. Finish the assigned deliverable or use `/collab-task-done` when real output exists.",
				collabID[:8], msg.From.Name,
			))
			if task.Status != collaboration.TaskInProgress {
				_, _ = h.collabManager.UpdateTaskStatusWithEffects(collabID, task.ID, collaboration.TaskInProgress, "Awaiting execution deliverable")
			}
			return
		}
		if h.maybeWarnPrematureTaskCompletion(msg, collabID, task, collabSnapshot) {
			return
		}
	}

	output := strings.TrimSpace(msg.GetTaskOutput())
	if output == "" && (status == collaboration.TaskCompleted || status == collaboration.TaskBlocked) {
		output = strings.TrimSpace(msg.Content)
	}
	effects, err := h.collabManager.UpdateTaskStatusWithEffects(collabID, taskID, status, output)
	if err != nil {
		log.Printf("[Collaboration] Failed to update task %s in %s: %v", taskID, collabID[:8], err)
		return
	}
	if effects.ShouldFailRun {
		if snap, err := h.collabManager.CancelCollaboration(collabID); err != nil {
			log.Printf("[Collaboration] fail_run cancel %s: %v", collabID[:8], err)
		} else {
			h.syncRunbookRunIndex(snap)
			h.cancelCollaborationRecaps(collabID)
			// Stop any in-flight work now that the collaboration is cancelled.
			if h.commandHandler != nil {
				cancelChannel := strings.TrimSpace(snap.Channel)
				if cancelChannel == "" {
					cancelChannel = msg.Channel
				}
				h.commandHandler.AbortRuntimeAgentsOnChannel(cancelChannel)
				h.broadcastChannelInterjectAbort(cancelChannel)
			}
		}
		h.broadcastCollabSystem(msg.Channel, collabID, "🚫 **Run stopped:** "+effects.FailRunReason)
		return
	}

	updated, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err == nil && updated != nil {
		for i := range updated.Tasks {
			if updated.Tasks[i].ID == taskID {
				task = &updated.Tasks[i]
				break
			}
		}
	}

	statusMsg := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("📌 Task update: **%s** is now **%s** (assigned to @%s).",
			func() string {
				if task != nil {
					return task.Title
				}
				return taskID
			}(),
			status,
			func() string {
				if task != nil && task.AssignedName != "" {
					return task.AssignedName
				}
				return "unknown"
			}(),
		),
	)
	statusMsg.SetCollaborationID(collabID)
	statusMsg.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	statusMsg.SetTaskID(taskID)
	statusMsg.SetTaskStatus(string(status))
	if output != "" {
		statusMsg.SetTaskOutput(output)
	}
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = map[string]interface{}{}
	}
	statusMsg.Metadata["collab_internal_event"] = true
	if err := h.SendMessage(statusMsg); err != nil {
		log.Printf("[Collaboration] Failed to broadcast task status update message: %v", err)
	}

	if h.collabManager.AllTasksComplete(collabID) {
		h.requestFinalRecapAndFinalize(collabID, msg.Channel, "All tasks are done.", collaboration.FinalizeOptions{})
		return
	}

	if effects.ShouldDispatchWave {
		if fresh, err := h.collabManager.GetCollaborationSnapshot(collabID); err == nil && fresh != nil && fresh.Phase == collaboration.PhaseExecuting {
			h.dispatchReadyCollabTasks(fresh, msg, false)
		}
	}
}

// maybeRedispatchAfterCollabGenerationError returns a failed assignee turn to the
// ready-pending pool and re-sends the task prompt when the agent is free.
func (h *Hub) maybeRedispatchAfterCollabGenerationError(msg *protocol.Message, collabID string) {
	if h == nil || h.collabManager == nil || msg == nil || msg.Metadata == nil {
		return
	}
	ge, _ := msg.Metadata["generation_error"].(bool)
	if !ge {
		return
	}
	taskID := strings.TrimSpace(msg.GetTaskID())
	if taskID == "" {
		return
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil || snap.Phase != collaboration.PhaseExecuting {
		return
	}
	var task *collaboration.CollaborationTask
	for i := range snap.Tasks {
		if snap.Tasks[i].ID == taskID {
			task = &snap.Tasks[i]
			break
		}
	}
	if task == nil {
		return
	}
	switch task.Status {
	case collaboration.TaskPending, collaboration.TaskInProgress, collaboration.TaskBlocked:
	default:
		return
	}
	if err := h.collabManager.ClearTaskPromptDispatched(collabID, taskID); err != nil {
		log.Printf("[Collaboration] ClearTaskPromptDispatched after generation_error: %v", err)
		return
	}
	if h.isAssigneeBusy(task.AssignedTo) {
		return
	}
	fresh, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || fresh == nil || !h.CollaborationCanDispatchTasks(fresh) {
		return
	}
	filter := func(t collaboration.CollaborationTask) bool { return t.ID == taskID }
	if n := h.dispatchCollabTaskMessagesFilter(fresh, nil, filter, true); n > 0 {
		short := taskID
		if len(short) > 8 {
			short = short[:8]
		}
		log.Printf("[Collaboration] Redispatched task %s after generation_error for %s", short, collabID[:8])
	}
}

func (h *Hub) finalizeAndBroadcastCollaboration(collabID, channel, reason string, opts collaboration.FinalizeOptions) {
	if h.collabManager == nil {
		return
	}
	c, err := h.collabManager.FinalizeCollaboration(collabID, opts)
	if err != nil {
		log.Printf("[Collaboration] Failed to finalize collaboration %s: %v", collabID[:8], err)
		return
	}
	h.syncRunbookRunIndex(c)
	h.persistCollaborationReviewAssets(collabID)
	if channel == "" {
		channel = c.Channel
	}
	if channel == "" {
		channel = "general"
	}

	completed := 0
	for _, t := range c.Tasks {
		if t.Status == collaboration.TaskCompleted {
			completed++
		}
	}
	total := len(c.Tasks)
	summary := fmt.Sprintf("✅ Collaboration `%s` completed. %s", collabID[:8], reason)
	if total > 0 {
		summary += fmt.Sprintf(" (%d/%d tasks done", completed, total)
		if ch := strings.TrimSpace(c.Channel); ch != "" {
			summary += fmt.Sprintf(", channel #%s", ch)
		}
		summary += ")."
	} else if ch := strings.TrimSpace(c.Channel); ch != "" {
		summary += fmt.Sprintf(" (channel #%s).", ch)
	}
	if strings.TrimSpace(c.SessionRecap) != "" {
		summary += "\n\n---\n\n" + strings.TrimSpace(c.SessionRecap)
	}

	completedMsg := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		summary,
	)
	completedMsg.SetCollaborationID(collabID)
	completedMsg.SetCollaborationPhase(string(collaboration.PhaseCompleted))
	if completedMsg.Metadata == nil {
		completedMsg.Metadata = map[string]interface{}{}
	}
	completedMsg.Metadata["collab_internal_event"] = true
	h.attachCollaborationData(completedMsg)
	if err := h.SendMessage(completedMsg); err != nil {
		log.Printf("[Collaboration] Failed to broadcast collaboration completion message: %v", err)
	}
}

func normalizeTaskStatus(raw string) (collaboration.TaskStatus, bool) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case string(collaboration.TaskPending):
		return collaboration.TaskPending, true
	case string(collaboration.TaskInProgress):
		return collaboration.TaskInProgress, true
	case string(collaboration.TaskCompleted):
		return collaboration.TaskCompleted, true
	case string(collaboration.TaskBlocked):
		return collaboration.TaskBlocked, true
	default:
		return "", false
	}
}

func (h *Hub) attachCollaborationData(msg *protocol.Message) {
	if msg == nil || h.collabManager == nil {
		return
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return
	}
	_, _ = h.collabManager.EnsureExecutionTasks(collabID)
	snapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snapshot == nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["collaboration_data"] = snapshot.ToUIPayload()
}

// GetMessages returns messages from a channel
func (h *Hub) shouldAutoCreateRepoAgent(msg *protocol.Message, repoPath string) bool {
	// Don't auto-create if it's a system message
	if msg.IsFromSystem() {
		return false
	}

	if messageHasSharedWorkspaceForRepo(msg, repoPath) {
		return false
	}

	// Don't auto-create if there's already a pending review for this path
	if h.commandHandler.HasPendingReview(repoPath) {
		return false
	}

	// Don't auto-create if there's already a repo agent for this path
	agents := h.ListAgents()
	for _, agent := range agents {
		if agent.Type == protocol.AgentTypeRepo && agent.RepositoryPath == repoPath {
			return false
		}
	}

	// Only auto-create if the message mentions a regular agent (not repo agent)
	// and contains repository-related keywords
	hasRegularAgentMention := false
	hasRepoKeywords := false

	// Check for regular agent mentions
	for _, mention := range msg.Mentions {
		agent, err := h.GetAgent(mention)
		if err == nil && agent.Type != protocol.AgentTypeRepo {
			hasRegularAgentMention = true
			break
		}
	}

	// Check for repository-related keywords
	content := strings.ToLower(msg.Content)
	repoKeywords := []string{
		"review", "analyze", "check", "examine", "look at", "code review",
		"architecture", "structure", "codebase", "repository", "project",
		"help with", "assist with", "understand", "explain",
	}

	for _, keyword := range repoKeywords {
		if strings.Contains(content, keyword) {
			hasRepoKeywords = true
			break
		}
	}

	return hasRegularAgentMention && hasRepoKeywords
}

func messageHasSharedWorkspaceForRepo(msg *protocol.Message, repoPath string) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	raw, ok := msg.Metadata["workspace_context"].(map[string]interface{})
	if !ok {
		return false
	}
	wsPath, _ := raw["workspace_path"].(string)
	wsPath = strings.TrimSpace(wsPath)
	if wsPath == "" {
		return false
	}
	return sameRepoPath(wsPath, repoPath)
}

func sameRepoPath(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil {
		return filepath.Clean(absA) == filepath.Clean(absB)
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// autoCreateRepoAgent automatically creates a repository agent for the given path
func (h *Hub) autoCreateRepoAgent(originalMsg *protocol.Message, repoPath string) {
	// Generate agent name from path
	agentName := protocol.NormalizeAgentName(filepath.Base(repoPath) + "Expert")

	// Send initial feedback message
	feedbackMsg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		originalMsg.Channel,
		protocol.AgentInfo{
			ID:   "system",
			Name: "System",
			Type: protocol.AgentTypeGeneral,
		},
		fmt.Sprintf("🤖 Detected repository path: %s\n"+
			"Creating repository expert agent automatically...\n"+
			"This will take 30-60 seconds for first-time indexing.\n"+
			"The agent will respond to your question once ready.",
			repoPath),
	)

	// Send feedback message
	h.mu.Lock()
	h.appendChannelMessageLocked(originalMsg.Channel, feedbackMsg)
	h.broadcast(originalMsg.Channel, feedbackMsg)
	h.mu.Unlock()

	// Add to pending reviews
	h.commandHandler.AddPendingReview(repoPath, originalMsg, agentName)

	// Create the repo agent in a goroutine to avoid blocking
	go func() {
		ctx := context.Background()

		// Create a synthetic message for the command handler
		createMsg := &protocol.Message{
			ID:      originalMsg.ID + "_auto_create",
			Type:    protocol.MessageTypeQuestion,
			Channel: originalMsg.Channel,
			From:    originalMsg.From,
			Content: fmt.Sprintf("/create-repo-agent %s %s", repoPath, agentName),
			Metadata: map[string]interface{}{
				"auto_created":    true,
				"original_msg_id": originalMsg.ID,
			},
		}

		// Process the create command
		response, err := h.commandHandler.ProcessCommand(ctx, createMsg)
		if err != nil {
			// Send error message
			errorMsg := protocol.NewMessage(
				protocol.MessageTypeSystemInfo,
				originalMsg.Channel,
				protocol.AgentInfo{
					ID:   "system",
					Name: "System",
					Type: protocol.AgentTypeGeneral,
				},
				fmt.Sprintf("❌ Failed to auto-create repository agent: %v", err),
			)

			h.mu.Lock()
			h.appendChannelMessageLocked(originalMsg.Channel, errorMsg)
			h.broadcast(originalMsg.Channel, errorMsg)
			h.mu.Unlock()

			// Remove from pending reviews
			h.commandHandler.RemovePendingReview(repoPath)
		} else if response != nil {
			// Send the response message
			h.mu.Lock()
			h.appendChannelMessageLocked(originalMsg.Channel, response)
			h.broadcast(originalMsg.Channel, response)
			h.mu.Unlock()
		}
	}()
}

// GetCommandHandler returns the command handler for external access
func (h *Hub) persistCollaborationReviewAssets(collabID string) {
	if h == nil || h.collabManager == nil || strings.TrimSpace(collabID) == "" {
		return
	}
	baseDir, err := h.collabManager.CollabAssetsBaseDir()
	if err != nil {
		log.Printf("[Collaboration] review assets root for %s: %v", shortCollabID(collabID), err)
		return
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		log.Printf("[Collaboration] review assets snapshot for %s: %v", shortCollabID(collabID), err)
		return
	}
	paths, err := collaboration.WriteReviewAssets(baseDir, snap)
	if err != nil {
		log.Printf("[Collaboration] write review assets for %s: %v", shortCollabID(collabID), err)
		return
	}
	channel := ""
	if snap != nil {
		channel = strings.TrimSpace(snap.Channel)
	}
	memory.IndexReviewAssetPaths(paths, collabID, channel)
}

func shortCollabID(collabID string) string {
	if len(collabID) <= 8 {
		return collabID
	}
	return collabID[:8]
}

// WaitCollabAsync blocks until approve-plan background work finishes.
func (h *Hub) WaitCollabAsync() {
	if h == nil {
		return
	}
	h.collabAsyncWG.Wait()
}

// ListCollaborationSnapshots returns collaboration snapshots suitable for UI
// consumption. Data is deep-copied by the collaboration manager.
func (h *Hub) RedispatchOpenCollaborationTasksAfterSessionRestore() {
	if h.collabManager == nil {
		return
	}
	h.collabManager.ReconcileRestoredAgentIDs()
	open := func(t collaboration.CollaborationTask) bool {
		return t.Status == collaboration.TaskPending ||
			t.Status == collaboration.TaskInProgress ||
			t.Status == collaboration.TaskBlocked
	}
	active := h.collabManager.ListActive()
	for _, c := range active {
		if c == nil || c.Phase != collaboration.PhaseReviewing {
			continue
		}
		if c.PlanningRecapStatus == collaboration.RecapStatusPending && c.PlanningRecapAgentID == "" {
			log.Printf("[Collaboration] Session restore: dispatching missing planning recap for collaboration %s", c.ID[:8])
			h.onCollaborationEnterReviewing(c.ID)
		}
	}
	for _, c := range active {
		if c == nil || c.Phase != collaboration.PhaseExecuting {
			continue
		}
		if _, err := h.collabManager.EnsureExecutionTasks(c.ID); err != nil {
			log.Printf("[Collaboration] session-restore redispatch EnsureExecutionTasks for %s: %v", c.ID[:8], err)
			continue
		}
		snap, err := h.collabManager.GetCollaborationSnapshot(c.ID)
		if err != nil || snap == nil {
			continue
		}
		n := 0
		for _, t := range snap.Tasks {
			if open(t) {
				n++
			}
		}
		if n == 0 {
			continue
		}
		sent := h.dispatchCollabTaskMessagesFilter(snap, nil, open, true)
		log.Printf("[Collaboration] Session restore: re-sent %d open task prompt(s) for executing collaboration %s", sent, c.ID[:8])
	}
}

// dispatchReadyCollabTasks sends collaboration_task prompts for DAG-ready pending tasks.
func (h *Hub) dispatchReadyCollabTasks(snap *collaboration.Collaboration, inheritFrom *protocol.Message, forceRedispatch bool) int {
	return h.dispatchCollabTaskMessagesFilter(snap, inheritFrom, nil, forceRedispatch)
}

// DispatchReadyCollabTasksForSnapshot dispatches ready tasks (exported for runbook start API).
func (h *Hub) DispatchReadyCollabTasksForSnapshot(snap *collaboration.Collaboration, forceRedispatch bool) int {
	return h.dispatchReadyCollabTasks(snap, nil, forceRedispatch)
}

// dispatchCollabTaskMessages sends collaboration_task messages so assignees
// receive task_assigned_to metadata (mirrors /approve-plan). Used after the
// manager heals missing assignees on executing collaborations.
func (h *Hub) dispatchCollabTaskMessages(snap *collaboration.Collaboration, inheritFrom *protocol.Message, forceRedispatch bool) {
	h.dispatchReadyCollabTasks(snap, inheritFrom, forceRedispatch)
}

// dispatchCollabTaskMessagesFilter sends collaboration_task messages for tasks
// selected by include when set, otherwise DAG-ready pending tasks only.
// Returns the number of prompts sent.
func (h *Hub) dispatchCollabTaskMessagesFilter(snap *collaboration.Collaboration, inheritFrom *protocol.Message, include func(collaboration.CollaborationTask) bool, forceRedispatch bool) int {
	if snap == nil || snap.Phase != collaboration.PhaseExecuting || len(snap.Tasks) == 0 {
		return 0
	}
	if snap.DispatchPaused {
		return 0
	}
	if !h.CollaborationCanDispatchTasks(snap) {
		return 0
	}
	maxParallel := snap.EffectiveExecutionPolicy().MaxConcurrentTasks
	collabID := snap.ID
	if h.collabManager != nil {
		fresh, err := h.collabManager.GetCollaborationSnapshot(collabID)
		if err == nil && fresh != nil {
			snap = fresh
		}
	}
	ch := snap.Channel
	sent := 0
	cacheCompleted := 0
	h.annotateCollaborationTaskRouting(snap)
	for _, task := range snap.Tasks {
		if include != nil && !include(task) {
			continue
		}
		if include == nil {
			if task.Status != collaboration.TaskPending {
				continue
			}
			if !forceRedispatch && task.PromptDispatched {
				continue
			}
			if !collaboration.IsTaskReadyForCollab(task, snap) {
				continue
			}
		} else if forceRedispatch {
			switch task.Status {
			case collaboration.TaskPending:
				if !collaboration.IsTaskReadyForCollab(task, snap) {
					continue
				}
			case collaboration.TaskInProgress, collaboration.TaskBlocked:
				// allow resume redispatch
			default:
				continue
			}
		} else {
			if task.Status != collaboration.TaskPending || task.PromptDispatched {
				continue
			}
			if !collaboration.IsTaskReadyForCollab(task, snap) {
				continue
			}
		}

		if maxParallel > 0 && sent >= maxParallel {
			break
		}

		if cached, ok := h.cachedOrchestrationTaskResult(context.Background(), snap, task); ok {
			if _, err := h.collabManager.UpdateTaskStatusWithEffects(
				collabID, task.ID, collaboration.TaskCompleted, string(cached),
			); err != nil {
				log.Printf("[Orchestration] apply cached task result %s: %v", shortOrchestrationID(task.ID), err)
			} else {
				cacheCompleted++
			}
			continue
		}

		attempt, claimErr := h.claimOrchestrationTask(context.Background(), snap, task)
		if claimErr != nil && forceRedispatch {
			attempt, claimErr = h.activeOrchestrationAttempt(context.Background(), collabID, task.ID)
		}
		if claimErr != nil {
			if h.durableOrchestrationEnforced() {
				log.Printf("[Orchestration] task claim blocked %s: %v", shortOrchestrationID(task.ID), claimErr)
				continue
			}
			log.Printf("[Orchestration] shadow claim mismatch %s: %v", shortOrchestrationID(task.ID), claimErr)
			attempt = nil
		}

		// Action tasks: execute on hub then mark complete (wave continues via status handler).
		if task.EffectiveKind() == collaboration.TaskKindAction && task.Action != nil {
			if h.executeCollabActionTask(snap, task) {
				sent++
			}
			h.syncOrchestrationState(context.Background())
			continue
		}

		mentionName := task.AssignedName
		if mentionName == "" {
			mentionName = "team"
		}
		handoffLimit := snap.EffectiveExecutionPolicy().HandoffMaxChars
		workspaceNote := ""
		if snap.ExecutionMode != collaboration.ExecutionModeWorktree && strings.TrimSpace(snap.SourceRepoPath) != "" {
			outDir := collaboration.PlannedOutputDirectory(snap, "")
			if outDir == "" {
				outDir = snap.WorkingDirectory
			}
			rel := collaboration.ProjectCollabRelPath(snap.ID)
			workspaceNote = fmt.Sprintf(
				"\n\n**Project workspace:** `%s` (read/inspect the codebase here).\n**Write deliverables under:** `%s` (absolute: `%s`).\nEmit [FILE_CHANGE] blocks with paths relative to the project root (e.g. `%s/draft.md`).",
				snap.SourceRepoPath, rel, outDir, rel,
			)
		}
		body := formatCollabTaskDispatchBody(snap, task, mentionName, handoffLimit, workspaceNote)
		body += collaboration.TaskDispatchFileDeliverableNote(task, snap.Description)
		for _, t := range snap.Tasks {
			if t.ID == task.ID {
				task = t
				break
			}
		}
		body += formatCollabTaskRoutingNote(task)
		taskMsg := protocol.NewMessage(
			protocol.MessageTypeCollabTask,
			ch,
			protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
			body,
		)
		taskMsg.SetCollaborationID(collabID)
		taskMsg.SetCollaborationPhase(string(collaboration.PhaseExecuting))
		taskMsg.SetTaskID(task.ID)
		taskMsg.SetTaskStatus(string(task.Status))
		if ws := h.collaborationWorkspaceContextSnapshot(snap); ws != nil {
			if taskMsg.Metadata == nil {
				taskMsg.Metadata = map[string]interface{}{}
			}
			taskMsg.Metadata["workspace_context"] = ws
		}
		if task.AssignedTo != "" {
			taskMsg.Mentions = []string{task.AssignedTo}
			if taskMsg.Metadata == nil {
				taskMsg.Metadata = map[string]interface{}{}
			}
			taskMsg.Metadata["task_assigned_to"] = task.AssignedTo
		}
		if task.Options != nil && task.Options.ProviderID != "" {
			if taskMsg.Metadata == nil {
				taskMsg.Metadata = map[string]interface{}{}
			}
			taskMsg.Metadata["task_provider_id"] = task.Options.ProviderID
		}
		var optionPaths []string
		if task.Options != nil {
			optionPaths = task.Options.ContextPaths
		}
		contextPaths := collaboration.MergeContextPaths(
			collaboration.InferTaskContextPaths(task, snap.SourceRepoPath),
			optionPaths,
		)
		if taskMsg.Metadata == nil {
			taskMsg.Metadata = map[string]interface{}{}
		}
		policy := collaboration.NewDeliverablePolicy(task, snap.Description, contextPaths)
		taskMsg.Metadata["task_title"] = strings.TrimSpace(task.Title)
		taskMsg.Metadata["task_description"] = strings.TrimSpace(task.Description)
		taskMsg.Metadata["deliverable_kind"] = policy.Kind()
		if len(contextPaths) > 0 {
			taskMsg.Metadata["task_context_paths"] = contextPaths
			taskMsg.Metadata["context_scope"] = "focus"
		} else if strings.TrimSpace(snap.SourceRepoPath) != "" {
			taskMsg.Metadata["context_scope"] = "hint"
		}
		if len(contextPaths) > 0 && strings.TrimSpace(snap.SourceRepoPath) != "" && policy.RequiresFile() {
			mergeTaskContextFilesIntoMessage(taskMsg, snap.SourceRepoPath, contextPaths)
		}
		applyCollabTaskRoutingMetadata(task, taskMsg)
		timeoutSec := collaboration.ExecutionTimeoutSeconds(task, h.collabExecutionTimeoutOverride())
		if taskMsg.Metadata == nil {
			taskMsg.Metadata = map[string]interface{}{}
		}
		taskMsg.Metadata["execution_timeout_seconds"] = timeoutSec
		dispatchToken := uuid.New().String()
		taskMsg.Metadata[protocol.MetadataDispatchToken] = dispatchToken
		if attempt != nil {
			taskMsg.Metadata["orchestration_attempt_id"] = attempt.ID
			taskMsg.Metadata["orchestration_lease_token"] = attempt.LeaseToken
			taskMsg.Metadata["orchestration_attempt_number"] = attempt.Number
		}
		if policy.RequiresImplementationSession() {
			taskMsg.Metadata[protocol.IdeMetaImplementationSession] = true
		}
		if rules := h.collabUserRulesMarkdown(snap, inheritFrom); rules != "" {
			if _, ok := taskMsg.Metadata[agent.MetadataUserRulesMarkdown]; !ok {
				taskMsg.Metadata[agent.MetadataUserRulesMarkdown] = rules
			}
		}
		if err := h.SendMessage(taskMsg); err != nil {
			log.Printf("[Collaboration] Failed to send task message (redispatch): %v", err)
			h.orchestrationDispatchFailed(context.Background(), attempt, err, task)
			continue
		}
		workflow.LogTaskDispatched(collabID, task.ID, dispatchToken)
		if h.collabManager != nil {
			if err := h.collabManager.MarkTaskPromptDispatched(collabID, task.ID); err != nil {
				log.Printf("[Collaboration] MarkTaskPromptDispatched %s: %v", task.ID[:8], err)
			}
		}
		sent++
	}
	if sent > 0 && h.collabManager != nil {
		_ = h.collabManager.MarkTasksDispatched(collabID)
	}
	if cacheCompleted > 0 && h.collabManager != nil {
		if fresh, err := h.collabManager.GetCollaborationSnapshot(collabID); err == nil {
			sent += h.dispatchReadyCollabTasks(fresh, inheritFrom, false)
		}
	}
	return sent
}

func formatCollabTaskDispatchBody(snap *collaboration.Collaboration, task collaboration.CollaborationTask, mentionName string, handoffLimit int, workspaceNote string) string {
	var b strings.Builder
	b.WriteString("@")
	b.WriteString(mentionName)
	b.WriteString(" -- ")
	if snap != nil {
		if goal := strings.TrimSpace(snap.Description); goal != "" {
			b.WriteString("**Collaboration goal (original ask):** ")
			b.WriteString(goal)
			b.WriteString("\n\n")
		}
	}
	b.WriteString("**Your assigned task:**\n\n**")
	b.WriteString(task.Title)
	b.WriteString("**\n\n")
	b.WriteString(task.Description)
	if snap != nil {
		b.WriteString(collaboration.FormatDependencyHandoffWithLimit(task, snap.Tasks, handoffLimit))
	}
	b.WriteString(workspaceNote)
	b.WriteString("\n\nComplete this task now. Ship concrete output ([FILE_CHANGE] and/or findings in the deliverables folder). End your reply with `TASK_STATUS: completed` or `TASK_STATUS: blocked`. @mention others only if blocked.")
	return b.String()
}

// NewCollaborationClientAdapter creates an adapter that implements
// agent.CollaborationClient by delegating to the real CollaborationManager.
func (h *Hub) NewCollaborationClientAdapter() agent.CollaborationClient {
	return &collabClientAdapter{cm: h.collabManager}
}

// collabClientAdapter bridges the agent.CollaborationClient interface
// to the concrete collaboration.CollaborationManager.
type collabClientAdapter struct {
	cm *collaboration.CollaborationManager
}

func (a *collabClientAdapter) IsParticipant(collabID, agentID string) bool {
	return a.cm.IsParticipant(collabID, agentID)
}

func (a *collabClientAdapter) IsAgentTurn(collabID, agentID string) bool {
	return a.cm.IsAgentTurn(collabID, agentID)
}

func (a *collabClientAdapter) IsActive(collabID string) bool {
	return a.cm.IsActive(collabID)
}

func (a *collabClientAdapter) AgentOutOfTurnMentionAllowed(collabID string) bool {
	return a.cm.AgentOutOfTurnMentionAllowed(collabID)
}

func (a *collabClientAdapter) PlanningSpeakerCooldownBlocked(collabID, agentID string) bool {
	return a.cm.PlanningSpeakerCooldownBlocked(collabID, agentID)
}

func (a *collabClientAdapter) ParticipantTurnCount(collabID, agentID string) int {
	return a.cm.ParticipantTurnCount(collabID, agentID)
}

func (a *collabClientAdapter) GetCurrentTurnAgent(collabID string) (string, error) {
	return a.cm.GetCurrentTurnAgent(collabID)
}

func (a *collabClientAdapter) GetCollaborationForAgent(agentID string) agent.CollaborationInfo {
	c := a.cm.GetCollaborationForAgent(agentID)
	if c == nil {
		return agent.CollaborationInfo{}
	}
	return collaborationInfoForAgent(c, agentID)
}

func (a *collabClientAdapter) GetCollaboration(collabID, agentID string) agent.CollaborationInfo {
	collabID = strings.TrimSpace(collabID)
	if collabID == "" || !a.cm.IsParticipant(collabID, agentID) {
		return agent.CollaborationInfo{}
	}
	c, err := a.cm.GetCollaboration(collabID)
	if err != nil || c == nil {
		return agent.CollaborationInfo{}
	}
	return collaborationInfoForAgent(c, agentID)
}

func collaborationInfoForAgent(c *collaboration.Collaboration, agentID string) agent.CollaborationInfo {
	if c == nil {
		return agent.CollaborationInfo{}
	}

	agentRole := ""
	for _, ag := range c.Agents {
		if ag.AgentID == agentID {
			agentRole = ag.Role
			break
		}
	}

	agents := make([]agent.CollaborationAgentSummary, 0, len(c.Agents))
	for _, ag := range c.Agents {
		agents = append(agents, agent.CollaborationAgentSummary{
			Name:      ag.AgentName,
			Type:      string(ag.AgentType),
			Role:      ag.Role,
			Expertise: ag.Expertise,
		})
	}

	planContent := ""
	planVersion := 0
	if c.Plan != nil {
		planContent = c.Plan.Content
		planVersion = c.Plan.Version
	}

	return agent.CollaborationInfo{
		ID:                     c.ID,
		Description:            c.Description,
		Phase:                  string(c.Phase),
		PlanContent:            planContent,
		PlanVersion:            planVersion,
		AgentRole:              agentRole,
		Agents:                 agents,
		Channel:                c.Channel,
		ExecutionMode:          string(c.ExecutionMode),
		SourceRepoPath:         c.SourceRepoPath,
		SourceWorkspaceContext: c.SourceWorkspaceContext,
		AttachWorkspaceContext: c.AttachWorkspaceContext,
		WorktreeBranch:         c.WorktreeBranch,
		WorkingDirectory:       c.WorkingDirectory,
	}
}

func (a *collabClientAdapter) GetCollaborationWorkingDirectory(collabID string) string {
	if collabID == "" {
		return ""
	}
	c, err := a.cm.GetCollaboration(collabID)
	if err != nil || c == nil {
		return ""
	}
	return c.WorkingDirectory
}

func (a *collabClientAdapter) RecordMessage(collabID string, msg *protocol.Message) error {
	return a.cm.RecordMessage(collabID, msg)
}

func (a *collabClientAdapter) AnalyzeConsensus(collabID string, msg *protocol.Message) string {
	return string(a.cm.AnalyzeConsensus(collabID, msg))
}

func (h *Hub) postFileChangeRegistrationFailureLocked(msg *protocol.Message, regErr error) {
	if h == nil || msg == nil || regErr == nil {
		return
	}
	path := fileChangeProposalPath(msg)
	agentName := msg.From.Name
	if agentName == "" {
		agentName = "Agent"
	}
	content := fmt.Sprintf("File change proposal for %s was not registered: %s", path, regErr.Error())
	sys := protocol.NewMessage(protocol.MessageTypeSystemInfo, msg.Channel, protocol.AgentInfo{
		ID:   "system",
		Name: "System",
		Type: protocol.AgentTypeGeneral,
	}, content)
	h.appendChannelMessageLocked(msg.Channel, sys)
	h.broadcast(msg.Channel, sys)
	log.Printf("[FileChange] Registration failed for %s from %s: %v", path, agentName, regErr)
}

func fileChangeProposalPath(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return "(unknown path)"
	}
	raw, ok := msg.Metadata["file_change_proposal"]
	if !ok {
		return "(unknown path)"
	}
	proposalBytes, err := json.Marshal(raw)
	if err != nil {
		return "(unknown path)"
	}
	var proposal protocol.FileChangeProposal
	if json.Unmarshal(proposalBytes, &proposal) != nil {
		return "(unknown path)"
	}
	path := strings.TrimSpace(proposal.FilePath)
	if path == "" {
		return "(unknown path)"
	}
	return path
}

// registerFileChangeProposal extracts a FileChangeProposal from message metadata
// and registers it with the FileChangeManager so it appears in the pending changes UI.
func (h *Hub) registerFileChangeProposal(msg *protocol.Message, proposalRaw interface{}) error {
	if msg != nil && (msg.IdeEditorModeIsAsk() || msg.IdeEditorModeIsPlan() ||
		msg.IdeEditorMode() == "ask" || msg.IdeEditorMode() == "plan") {
		log.Printf("[FileChange] Rejected proposal in read-only editor mode (%s) from %s",
			msg.IdeEditorMode(), msg.From.Name)
		return fmt.Errorf("file changes not allowed in ask/plan mode")
	}
	if err := h.rejectFileChangeOnClosedCollab(msg); err != nil {
		log.Printf("[FileChange] Rejected proposal on closed collaboration: %v", err)
		return err
	}

	// Convert the raw proposal to typed struct via JSON round-trip
	proposalBytes, err := json.Marshal(proposalRaw)
	if err != nil {
		log.Printf("[FileChange] Failed to marshal proposal: %v", err)
		return fmt.Errorf("marshal proposal: %w", err)
	}

	var proposal protocol.FileChangeProposal
	if err := json.Unmarshal(proposalBytes, &proposal); err != nil {
		log.Printf("[FileChange] Failed to unmarshal proposal: %v", err)
		return fmt.Errorf("unmarshal proposal: %w", err)
	}

	// Resolve workspace root from message context only.
	wsRoot := h.resolveWorkspaceRoot(msg)
	if wsRoot == "" {
		if proposal.Metadata != nil {
			if p, ok := proposal.Metadata["target_workspace_path"].(string); ok && strings.TrimSpace(p) != "" {
				wsRoot = strings.TrimSpace(p)
			}
		}
	}
	if wsRoot == "" {
		log.Printf("[FileChange] Missing workspace context for proposal from %s on channel %s",
			msg.From.Name, msg.Channel)
		return fmt.Errorf("missing workspace context for file change proposal")
	}

	// Resolve file path against workspace
	filePath, err := h.resolveWorkspacePath(proposal.FilePath, wsRoot)
	if err != nil {
		log.Printf("[FileChange] Failed to resolve file path %q: %v", proposal.FilePath, err)
		return fmt.Errorf("resolve file path %q: %w", proposal.FilePath, err)
	}

	// Map proposal operation string to filechange.FileOperation
	var operation filechange.FileOperation
	switch proposal.Operation {
	case "create":
		operation = filechange.FileOperationCreate
	case "edit":
		operation = filechange.FileOperationEdit
	case "delete":
		operation = filechange.FileOperationDelete
	case "move":
		operation = filechange.FileOperationMove
	default:
		log.Printf("[FileChange] Unknown operation: %s", proposal.Operation)
		return fmt.Errorf("unknown file change operation: %s", proposal.Operation)
	}

	// Resolve paths for move operations
	oldPath := proposal.OldPath
	newPath := proposal.NewPath
	if operation == filechange.FileOperationMove {
		oldPath, err = h.resolveWorkspacePath(proposal.OldPath, wsRoot)
		if err != nil {
			log.Printf("[FileChange] Failed to resolve move old path %q: %v", proposal.OldPath, err)
			return fmt.Errorf("resolve move old path %q: %w", proposal.OldPath, err)
		}
		newPath, err = h.resolveWorkspacePath(proposal.NewPath, wsRoot)
		if err != nil {
			log.Printf("[FileChange] Failed to resolve move new path %q: %v", proposal.NewPath, err)
			return fmt.Errorf("resolve move new path %q: %w", proposal.NewPath, err)
		}
	}

	var workspaceIO filechange.WorkspaceIO
	if h.fileChangeBackendFn != nil {
		workspaceIO = h.fileChangeBackendFn(wsRoot)
	}
	proposalExecutor := filechange.NewFileChangeExecutor(wsRoot)
	proposalExecutor.SetWorkspaceIO(workspaceIO)

	manifest := agent.DetectStackManifest(wsRoot)
	proposal.FilePath = agent.RedirectProposalPathForOp(proposal.FilePath, manifest, proposal.Operation)
	filePath, err = h.resolveWorkspacePath(proposal.FilePath, wsRoot)
	if err != nil {
		return fmt.Errorf("resolve redirected file path %q: %w", proposal.FilePath, err)
	}
	if operation == filechange.FileOperationCreate {
		if _, readErr := proposalExecutor.GetFileContent(filePath); readErr == nil {
			operation = filechange.FileOperationEdit
			proposal.Operation = "edit"
		}
	}
	propOp := agent.ProposalOpCreate
	if operation == filechange.FileOperationEdit {
		propOp = agent.ProposalOpEdit
	}
	// Deletes/moves are not create/edit — create preflight rejects src/App.js when the
	// stack entry is App.tsx, which blocks the corrupt App.js boot-fix delete.
	if operation != filechange.FileOperationDelete && operation != filechange.FileOperationMove {
		if err := agent.ValidateProposal(wsRoot, proposal.FilePath, propOp, manifest); err != nil {
			log.Printf("[FileChange] Preflight rejected %q: %v", proposal.FilePath, err)
			return fmt.Errorf("preflight rejected %q: %w", proposal.FilePath, err)
		}
	}

	if looksLikePlaceholderDeliverableContent(proposal.NewContent) {
		log.Printf("[FileChange] Rejected placeholder deliverable content for %q from %s",
			proposal.FilePath, msg.From.Name)
		return fmt.Errorf("placeholder deliverable content for %q", proposal.FilePath)
	}
	if operation == filechange.FileOperationEdit {
		current, readErr := proposalExecutor.GetFileContent(filePath)
		if readErr != nil {
			return fmt.Errorf("read current edit target %q: %w", proposal.FilePath, readErr)
		}
		proposal.OldContent = current
		if filechange.SanitizeFileChangeContent(proposal.NewContent) ==
			filechange.SanitizeFileChangeContent(proposal.OldContent) {
			return fmt.Errorf("no-op edit rejected for %q: resulting content is unchanged", proposal.FilePath)
		}
	}

	// Register with FileChangeManager
	change, err := h.fileChangeManager.ProposeFileChange(
		operation,
		filePath,
		oldPath,
		newPath,
		proposal.OldContent,
		proposal.NewContent,
		msg.From,
		msg.Channel,
	)
	if err != nil {
		log.Printf("[FileChange] Failed to register proposal: %v", err)
		return fmt.Errorf("register proposal: %w", err)
	}
	if err := h.fileChangeManager.BindExecutionContext(change.ID, wsRoot, workspaceIO); err != nil {
		return fmt.Errorf("bind proposal execution context: %w", err)
	}
	if change.Metadata == nil {
		change.Metadata = make(map[string]interface{})
	}
	change.Metadata["workspace_root"] = wsRoot
	h.persistFileChange(change)

	// Update the message metadata with the registered change ID so the UI can link them
	msg.Metadata["registered_change_id"] = change.ID

	log.Printf("[FileChange] Registered %s proposal for %s (change ID: %s) from %s",
		proposal.Operation, filePath, change.ID, msg.From.Name)

	h.maybeAutoApproveCollabFileChange(msg, change, operation, wsRoot)
	h.maybeAutoApproveIDEFileChange(msg, change, operation, wsRoot)
	refreshed, _ := h.fileChangeManager.GetFileChange(change.ID)
	if refreshed == nil {
		refreshed = change
	}
	if refreshed.Status != filechange.FileChangeStatusPending {
		if msg.Metadata == nil {
			msg.Metadata = map[string]interface{}{}
		}
		msg.Metadata[protocol.MetaFileChangeAutoApproved] = true
	}
	msg.Metadata[protocol.MetaChangeProposal] = protocol.ChangeProposalCard{
		Version:     1,
		Kind:        protocol.ChangeProposalKindFile,
		ID:          refreshed.ID,
		Status:      protocol.ChangeProposalStatus(refreshed.Status),
		Operation:   string(refreshed.Operation),
		FilePath:    refreshed.FilePath,
		OldPath:     refreshed.OldPath,
		NewPath:     refreshed.NewPath,
		RequestedAt: refreshed.RequestedAt,
		ExpiresAt:   refreshed.ExpiresAt,
		Reason:      refreshed.Reason,
	}
	return nil
}

// resolveWorkspacePath resolves a potentially relative file path against the provided workspace root.
func (h *Hub) resolveWorkspacePath(filePath, workspaceRoot string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("empty file path")
	}
	if workspaceRoot == "" {
		return "", fmt.Errorf("missing workspace root for path: %s", filePath)
	}
	var candidate string
	if filepath.IsAbs(filePath) {
		candidate = filePath
	} else {
		candidate = filepath.Join(workspaceRoot, filePath)
	}
	return pathutil.WithinRoot(workspaceRoot, candidate)
}

// resolveWorkspaceRoot returns the workspace root path from message context only.
func (h *Hub) resolveWorkspaceRoot(msg *protocol.Message) string {
	if msg != nil && msg.Metadata != nil {
		if p, ok := msg.Metadata["target_workspace_path"].(string); ok && strings.TrimSpace(p) != "" {
			return strings.TrimSpace(p)
		}
	}
	// Try to get workspace path from message metadata (workspace_context)
	if msg.Metadata != nil {
		if wsCtx, ok := msg.Metadata["workspace_context"]; ok {
			if ctxMap, ok := wsCtx.(map[string]interface{}); ok {
				if wsPath, ok := ctxMap["workspace_path"].(string); ok && wsPath != "" {
					return wsPath
				}
			}
		}
	}

	return ""
}

// MetadataKeyHistoryResync is set on ephemeral agent_status messages after channel
// history pruning so clients and agents refetch from the hub.
