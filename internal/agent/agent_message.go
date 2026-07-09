package agent

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

func (a *Agent) handleMessage(ctx context.Context, msg *protocol.Message) {
	// Handle agent status messages for provider/model updates
	if msg.Type == protocol.MessageTypeAgentStatus {
		if msg.Metadata != nil {
			if v, ok := msg.Metadata[protocol.MetadataChannelInterjectAbort].(bool); ok && v && msg.Channel != "" {
				a.AbortChannel(msg.Channel)
				return
			}
			if _, ok := msg.Metadata[protocol.MetadataChannelHold]; ok && msg.Channel != "" {
				return
			}
			if v, ok := msg.Metadata["history_resync"].(bool); ok && v && msg.Channel != "" {
				if old := a.channelHistory(msg.Channel); len(old) > 0 {
					for _, m := range old {
						if m != nil && m.ID != "" {
							delete(a.respondedMessages, m.ID)
						}
					}
				}
				if hist, err := a.bootstrapChannelHistory(msg.Channel); err == nil {
					a.replaceChannelHistory(msg.Channel, hist)
				}
				return
			}
		}
		// Check if this status message is for us (updating our provider)
		if msg.From.ID == a.Info.ID {
			// Extract provider info from metadata
			if aiProvider, ok := msg.Metadata["ai_provider"].(string); ok {
				aiModel, _ := msg.Metadata["ai_model"].(string)
				// Update our info to match
				a.Info.AIProvider = aiProvider
				a.Info.AIModel = aiModel
				if aiModel != "" {
					a.Info.Model = aiModel
				}
				// Note: The actual AI provider instance would need to be updated separately
				// This is a limitation - we'd need access to create a new provider instance
				log.Printf("[%s] 📝 Updated provider info to %s (%s)", a.Info.Name, aiProvider, aiModel)
			}
		}
		// Don't process agent status messages further
		return
	}

	// Ignore own messages - check BOTH ID and Name (since ID changes on restart)
	if msg.From.ID == a.Info.ID || msg.From.Name == a.Info.Name {
		return
	}

	// Specialized-agent interception path (e.g., Assistant deterministic actions).
	if a.messageInterceptor != nil && a.messageInterceptor(ctx, msg) {
		return
	}

	// Check if we've already responded to this message (atomic check-and-set)
	a.respondedMutex.Lock()
	if a.respondedMessages[msg.ID] {
		a.respondedMutex.Unlock()
		return
	}
	a.respondedMutex.Unlock()

	// Add to history first (so we have context for decision)
	a.addToHistory(msg)

	if a.unansweredTracker != nil {
		a.unansweredTracker.observe(msg)
	}

	if a.Hub != nil && a.Hub.IsChannelHeld(msg.Channel) {
		return
	}

	// Decide if we should respond BEFORE marking as responded
	// This allows other agents to process the message if we don't respond
	if !a.shouldRespond(msg) {
		return
	}

	// Reserve this message so duplicate listeners do not start a second generation.
	a.respondedMutex.Lock()
	if a.respondedMessages[msg.ID] {
		a.respondedMutex.Unlock()
		return
	}
	a.respondedMessages[msg.ID] = true
	a.respondedMutex.Unlock()

	// Hub broadcasts one *Message to every subscriber; concurrent agents may otherwise
	// read/write msg.Metadata during prompt build (fatal map race).
	if work, err := protocol.CloneMessage(msg); err == nil && work != nil {
		msg = work
	}

	clearResponded := func() {
		a.respondedMutex.Lock()
		delete(a.respondedMessages, msg.ID)
		a.respondedMutex.Unlock()
	}

	// Log that we're processing this message
	a.resetRoutingSnapshot()
	a.resetCompressSnapshot()
	a.syncWorkspaceFromMessage(msg)
	a.recordKnowledgeRoute(msg)
	a.applyKnowledgePlanEarly(msg)
	a.recordTurnGovernance(msg)

	log.Printf("[%s] ⬇️ RECEIVED msg ID %s from %s (mentions: %v)", a.Info.Name, msg.ID[:8], msg.From.Name, msg.Mentions)
	log.Printf("[%s] ✅ MARKED msg %s as responded", a.Info.Name, msg.ID[:8])

	log.Printf("[%s] 💬 WILL RESPOND to msg %s from %s: %s", a.Info.Name, msg.ID[:8], msg.From.Name, msg.Content[:min(50, len(msg.Content))])
	log.Printf("[%s] 🔍 Message details - ThreadID: '%s', IsThreadReply: %v, ReplyTo: '%s'", a.Info.Name, msg.ThreadID, msg.IsThreadReply, msg.ReplyTo)

	// Send thinking status
	a.sendThinkingStatus(msg, protocol.ThinkingStatusStarted)
	a.resetCADWrittenPaths()

	genCtx, genCancel := collabGenerationContext(ctx, msg)
	timeoutCancel := genCancel
	genCtx, genCancel = context.WithCancel(genCtx)
	genID := a.registerGenCancel(msg.Channel, genCancel)
	defer func() {
		a.unregisterGenCancel(msg.Channel, genID)
		genCancel()
		timeoutCancel()
	}()

	// Try streaming path first, fall back to batch
	var response string
	var streamMsgID string
	var reasoningText string
	var err error

	eff := a.EffectiveAIProvider(genCtx, msg)
	if eff == nil {
		eff = a.GetAIProvider()
	}
	reason := "default_agent_provider"
	source := "rules"
	if msg.Type != protocol.MessageTypeCollabTask && !shouldRunImplementationSession(a, msg) {
		if base := a.GetAIProvider(); base != nil && eff != nil {
			if bm, em := strings.TrimSpace(base.GetModel()), strings.TrimSpace(eff.GetModel()); em != "" && bm != em {
				reason = "capability_routing"
				source = "capabilities"
			}
		}
		a.RecordRoutingFromProvider(eff, reason, source)
		a.recordClassifierRouting(msg)
	} else if msg.Type == protocol.MessageTypeCollabTask {
		snap := a.LastRoutingSnapshot()
		if snap.Reason != "" {
			reason = snap.Reason
		}
		if snap.Source != "" {
			source = snap.Source
		}
		a.RecordRoutingFromProvider(eff, reason, source)
	} else if shouldRunImplementationSession(a, msg) {
		a.recordClassifierRouting(msg)
	}
	a.broadcastRoutingTelemetry(msg)

	var implSessionProposed bool
	var implSessionFiles []string
	var implSessionOutcome map[string]interface{}
	if resp, ok := a.tryImplementationStatusCheckShortcut(msg); ok {
		response = resp
	} else if resp, redirectOutcome, ok := a.tryBootFixImplementerRedirect(msg); ok {
		response = resp
		implSessionOutcome = redirectOutcome
	} else if resp, destructiveOutcome, ok := a.tryDenyDestructiveImplementationSession(msg); ok {
		response = resp
		implSessionOutcome = destructiveOutcome
	} else if shouldRunImplementationSession(a, msg) {
		log.Printf("[%s] 🔧 Implementation session...", a.Info.Name)
		if implementationBestOfK(msg) > 1 {
			if sp, ok := eff.(ai.StreamingProvider); ok && sp.SupportsStreaming() {
				streamMsgID = uuid.New().String()
				response, streamMsgID, implSessionProposed, implSessionFiles, implSessionOutcome, err = a.runImplementationSessionBestOfK(genCtx, msg, eff, streamMsgID)
				reasoningText = ""
			} else {
				response, streamMsgID, implSessionProposed, implSessionFiles, implSessionOutcome, err = a.runImplementationSessionBestOfK(genCtx, msg, eff, "")
			}
		} else if sp, ok := eff.(ai.StreamingProvider); ok && sp.SupportsStreaming() {
			streamMsgID = uuid.New().String()
			response, streamMsgID, implSessionProposed, implSessionFiles, implSessionOutcome, err = a.runImplementationSessionStreaming(genCtx, msg, eff, streamMsgID)
			reasoningText = ""
		} else {
			response, implSessionProposed, implSessionFiles, implSessionOutcome, err = a.runImplementationSession(genCtx, msg, eff)
		}
	} else if sp, ok := eff.(ai.StreamingProvider); ok && sp.SupportsStreaming() {
		log.Printf("[%s] 📡 Streaming response...", a.Info.Name)
		response, streamMsgID, reasoningText, err = a.generateResponseStreaming(genCtx, msg, eff)
	} else {
		log.Printf("[%s] 📝 Generating response (batch)...", a.Info.Name)
		response, err = a.generateResponse(genCtx, msg, eff)
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("[%s] Generation cancelled (interject)", a.Info.Name)
			clearResponded()
			a.sendThinkingStatus(msg, protocol.ThinkingStatusAborted)
			if a.Collab != nil && msg.Type == protocol.MessageTypeCollabDiscussion {
				cid := msg.GetCollaborationID()
				phase := msg.GetCollaborationPhase()
				if cid != "" && phase == "planning" && a.Collab.IsActive(cid) {
					turnCount := a.Collab.ParticipantTurnCount(cid, a.Info.ID)
					a.scheduleCollaborationTurnHandoffRetry(msg, cid, a.Info.ID, turnCount)
				}
			}
			return
		}
		log.Printf("[%s] Error generating response: %v", a.Info.Name, err)
		clearResponded()
		a.sendThinkingStatus(msg, protocol.ThinkingStatusError)

		a.sendGenerationFailureMessages(msg, err)
		return
	}
	response = sanitizeInternalToolNames(response)
	response = sanitizeAbsolutePathFileChangeFromResponse(response)
	collabCtx := a.getCollaborationContext(msg)
	if collabCtx.ID != "" {
		response = sanitizeCollabDiscussionResponse(response, collabCtx, a.Info.Type)
	}
	var proposedFileChange bool
	var proposedGitChange bool
	var proposalErr error
	if implSessionProposed {
		proposedFileChange = len(implSessionFiles) > 0
	} else if isAskModeReadOnly(msg) {
		response = sanitizeAskModeResponse(response)
	} else {
		response, proposedFileChange, proposalErr = a.maybeSubmitFileChangeFromResponse(context.Background(), response, msg.Channel, msg)
		if !proposedFileChange && proposalErr == nil {
			response, proposedGitChange, proposalErr = a.maybeSubmitGitChangeFromResponse(context.Background(), response, msg.Channel, msg)
		}
		if !proposedFileChange && !proposedGitChange && proposalErr == nil {
			response, proposedFileChange, proposalErr = a.maybeProposeCombinedDeliveryExport(context.Background(), msg, response)
		}
	}
	if proposalErr != nil {
		log.Printf("[%s] Failed to submit file change proposal from response: %v", a.Info.Name, proposalErr)
	}
	if proposedFileChange && shouldAppendFileChangeApprovalPrompt(msg) {
		if strings.TrimSpace(response) == "" {
			response = "I submitted a file change proposal for your approval."
		} else {
			response += "\n\nI submitted a file change proposal for your approval."
		}
	}
	if proposedGitChange {
		if strings.TrimSpace(response) == "" {
			response = "I submitted a git change proposal for your approval."
		} else {
			response += "\n\nI submitted a git change proposal for your approval."
		}
	}
	if isAskModeReadOnly(msg) {
		response = sanitizeAskModeResponse(response)
	}
	log.Printf("[%s] ✍️  Generated response: %s", a.Info.Name, response[:min(50, len(response))])

	// Send response -- reuse the stream message ID when available so the
	// frontend can correlate deltas with the final persisted message.
	responseType := protocol.MessageTypeChat
	if msg.GetCollaborationID() != "" &&
		a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeCollaboration {
		responseType = protocol.MessageTypeCollabDiscussion
	}
	responseMsg := protocol.NewMessage(
		responseType,
		msg.Channel,
		a.Info,
		response,
	)
	if streamMsgID != "" {
		responseMsg.ID = streamMsgID
	}
	if strings.TrimSpace(reasoningText) != "" {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata["reasoning_text"] = reasoningText
	}
	if consulted := a.TakeDelegationConsulted(); len(consulted) > 0 {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata["delegation_consulted"] = consulted
	}
	if implSessionOutcome != nil || (shouldRunImplementationSession(a, msg) && implSessionProposed) {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata[protocol.IdeMetaImplementationComplete] = implSessionProposed || implSessionOutcome != nil
		if len(implSessionFiles) > 0 {
			responseMsg.Metadata[protocol.IdeMetaImplementationFiles] = implSessionFiles
		}
		if implSessionOutcome != nil {
			responseMsg.Metadata[protocol.IdeMetaImplementationOutcome] = implSessionOutcome
		}
	}
	if paths := a.takeCADWrittenPaths(); len(paths) > 0 {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata[protocol.IdeMetaCADFilesWritten] = paths
	}
	responseMsg.ReplyTo = msg.ID

	// If responding to a thread message, keep it in the thread
	if msg.IsInThread() {
		responseMsg.ThreadID = msg.ThreadID
		responseMsg.IsThreadReply = true
		log.Printf("[%s] 🧵 Responding in thread %s (IsInThread: true)", a.Info.Name, msg.ThreadID[:8])
	} else {
		log.Printf("[%s] 📢 Responding in main channel (IsInThread: false)", a.Info.Name)
	}
	log.Printf("[%s] 📤 Response details - ThreadID: '%s', IsThreadReply: %v, ReplyTo: '%s'", a.Info.Name, responseMsg.ThreadID, responseMsg.IsThreadReply, responseMsg.ReplyTo)

	// Check if this is a review request and add metadata
	if msg.ReplyTo != "" {
		handledReviewMetadata := false
		// Look for the message being replied to
		for _, histMsg := range a.channelHistory(msg.Channel) {
			if histMsg.ID == msg.ReplyTo {
				// Check if it's from another agent (review scenario)
				isFromAgent := histMsg.From.Type == protocol.AgentTypeFrontend ||
					histMsg.From.Type == protocol.AgentTypeBackend ||
					histMsg.From.Type == protocol.AgentTypeDatabase ||
					histMsg.From.Type == protocol.AgentTypeSecurity ||
					histMsg.From.Type == protocol.AgentTypeArchitecture ||
					histMsg.From.Type == protocol.AgentTypeCodeReview ||
					histMsg.From.Type == protocol.AgentTypeDevOps ||
					histMsg.From.Type == protocol.AgentTypeRepo ||
					histMsg.From.Type == protocol.AgentTypeExpert ||
					histMsg.From.Type == protocol.AgentTypeCLI

				if isFromAgent {
					// This is a review - track metadata
					currentDepth := msg.GetReviewDepth()
					responseMsg.SetReviewDepth(currentDepth + 1)
					responseMsg.SetReviewedMessageID(msg.ReplyTo)

					// Track original question if available
					originalQuestionID := msg.GetOriginalQuestionID()
					if originalQuestionID == "" {
						// Find the original question by looking back in history
						if histMsg.ReplyTo != "" {
							originalQuestionID = histMsg.ReplyTo
						}
					}
					if originalQuestionID != "" {
						responseMsg.SetOriginalQuestionID(originalQuestionID)
					}

					log.Printf("[%s] 📋 Review metadata: depth=%d, reviewing=%s",
						a.Info.Name, currentDepth+1, msg.ReplyTo[:8])
					handledReviewMetadata = true
				}
				break
			}
		}

		// Fallback for review flows where the replied message is not in local
		// history (for example, reply target was ephemeral status metadata).
		if !handledReviewMetadata && (msg.IsReviewRequest() || msg.GetReviewDepth() > 0) {
			currentDepth := msg.GetReviewDepth()
			responseMsg.SetReviewDepth(currentDepth + 1)
			responseMsg.SetReviewedMessageID(msg.ReplyTo)
			if originalQuestionID := msg.GetOriginalQuestionID(); originalQuestionID != "" {
				responseMsg.SetOriginalQuestionID(originalQuestionID)
			}
			log.Printf("[%s] 📋 Review metadata fallback: depth=%d, reviewing=%s",
				a.Info.Name, currentDepth+1, msg.ReplyTo[:8])
		}
	}

	ApplyCollaborationTaskMetadataOnReply(responseMsg, msg, response)
	a.ApplyRoutingMetadataToResponse(responseMsg)
	a.ApplyCompressMetadataToResponse(responseMsg)

	// Detect commands in the response and add them to metadata
	commandDetector := protocol.NewCommandDetector(nil)
	suggestions := commandDetector.DetectCommands(response, a.Info.Name, responseMsg.ID)
	suggestions = filterCollabCommandSuggestions(msg, suggestions)
	if cwd := collaborationWorkingDirectoryForMessage(a, msg); cwd != "" && len(suggestions) > 0 {
		for i := range suggestions {
			suggestions[i].Cwd = cwd
		}
	}
	if len(suggestions) > 0 {
		responseMsg.Metadata["suggested_commands"] = suggestions
		log.Printf("[%s] 🔧 Detected %d command suggestions", a.Info.Name, len(suggestions))
	}

	responseHasImage := false
	if a.isCLIAgent() {
		workDir := a.resolveCLIWorkDir(msg)
		responseHasImage = AttachGeneratedImageFromResponse(responseMsg, workDir)
		if responseHasImage {
			log.Printf("[%s] 🖼️ Attached generated image from CLI response path", a.Info.Name)
		}
	}

	// Record planning/review discussion turns before broadcast so peers see updated turn order.
	var collabID string
	var collabPhase string
	if cid := responseMsg.GetCollaborationID(); cid != "" && a.Collab != nil && msg.Type != protocol.MessageTypeCollabRecap {
		collabPhase = a.Collab.GetCollaboration(cid, a.Info.ID).Phase
		if collabPhase != "executing" {
			collabID = cid
			if err := a.Collab.RecordMessage(collabID, responseMsg); err != nil {
				log.Printf("[%s] Warning: failed to record collaboration message: %v", a.Info.Name, err)
				if collabPhase == "planning" || collabPhase == "reviewing" {
					clearResponded()
					a.sendThinkingStatus(msg, protocol.ThinkingStatusError)
					return
				}
			} else {
				a.Collab.AnalyzeConsensus(collabID, responseMsg)
			}
		}
	}

	log.Printf("[%s] 📤 Sending response msg ID %s (replying to %s)...", a.Info.Name, responseMsg.ID[:8], msg.ID[:8])
	if streamMsgID != "" {
		a.broadcastStreamEnd(msg, streamMsgID)
	}
	if err := a.Hub.SendMessage(responseMsg); err != nil {
		log.Printf("[%s] Error sending message: %v", a.Info.Name, err)
		a.sendThinkingStatus(msg, protocol.ThinkingStatusError)
		return
	}
	learning.MaybeSuggestAfterAgentReply(msg.Channel, a.Info.ID, a.Info.Name, string(a.Info.Type), msg.Content, response)
	log.Printf("[%s] ✅ Response sent successfully!", a.Info.Name)
	// Keep local history in sync with hub (own replies are ignored on Subscribe).
	a.addToHistory(responseMsg)
	a.MaybePostHubGeneratedImageForCLI(msg, responseHasImage)
	a.sendThinkingStatus(msg, protocol.ThinkingStatusCompleted)

	if collabID != "" && collabPhase == "planning" {
		a.promptNextCollaborationTurn(responseMsg, collabID)
	}
}

// promptNextCollaborationTurn emits a deterministic handoff prompt so the next
// participant receives an explicit trigger after each accepted collaboration turn.
func (a *Agent) sendThinkingStatus(originalMsg *protocol.Message, status protocol.ThinkingStatus) {
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		originalMsg.Channel,
		a.Info,
		"", // Empty content for status messages
	)
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = make(map[string]interface{})
	}
	statusMsg.Metadata["thinking_status"] = string(status)
	statusMsg.Metadata["question_id"] = originalMsg.ID

	// Fire and forget - don't block on sending status
	go func() {
		if err := a.Hub.SendMessage(statusMsg); err != nil {
			log.Printf("[%s] Warning: failed to send thinking status: %v", a.Info.Name, err)
		}
	}()
}

// effectiveChannelType resolves channel classification for routing. DM rooms use
// the "dm-" name prefix; if hub metadata is missing or wrong, still treat as DM
// so 1:1 agents answer the user.
func (a *Agent) effectiveChannelType(channel string) protocol.ChannelType {
	if channel == "" {
		return protocol.ChannelTypePublic
	}
	t := a.Hub.GetChannelType(channel)
	if t == protocol.ChannelTypeDM {
		return protocol.ChannelTypeDM
	}
	if strings.HasPrefix(strings.ToLower(channel), "dm-") {
		return protocol.ChannelTypeDM
	}
	if t == protocol.ChannelTypeCollaboration {
		return protocol.ChannelTypeCollaboration
	}
	if strings.HasPrefix(strings.ToLower(channel), "collab-") {
		return protocol.ChannelTypeCollaboration
	}
	return t
}

// taskAssigneeFromMetadata reads task_assigned_to from collaboration_task metadata.
// JSON decoding can surface non-string types; normalize so assignee routing matches.
func isSlashCommandMessage(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if msg.Metadata != nil {
		if v, ok := msg.Metadata[protocol.MetadataSlashCommand].(bool); ok && v {
			return true
		}
	}
	content := strings.TrimSpace(msg.Content)
	return len(content) > 0 && content[0] == '/'
}

func (a *Agent) shouldRespond(msg *protocol.Message) bool {
	if a.Hub != nil && a.Hub.IsChannelHeld(msg.Channel) {
		return false
	}
	// Repopulate Mentions before collab turn-prompt routing (history replay may drop them).
	a.backfillMentionsFromContent(msg)
	// Slack bridge policy routing (e.g. always → Assistant on every line).
	if routed := msg.SlackRoutedAgentID(); routed != "" {
		return routed == a.Info.ID
	}
	// Forward-mode inbox lines are for manual owner reply, not agent auto-response.
	if msg.SlackInboxAwaitingManualReply() {
		return false
	}

	// Never respond to slash commands — let the command handler process them (must run before IDE routing).
	if isSlashCommandMessage(msg) {
		return false
	}

	// IDE file-tab routing applies only when the user did not @mention a specific agent.
	// Skip in DMs: the channel partner is always the intended recipient.
	if routedType := msg.IdeRouteAgentType(); routedType != "" && !msg.HasMentions() && !a.isDMChannel(msg.Channel) {
		if userRequestsCodeReview(msg.Content) {
			return false
		}
		return strings.EqualFold(string(a.Info.Type), routedType)
	}

	// Repo-path project reviews defer to repo expert agents (pending or indexed).
	if a.Hub != nil && messageDefersToRepoExpert(a, msg) {
		return false
	}

	if protocol.IsGeneratedImageDelivery(msg) {
		return false
	}

	if shouldSkipAgentResponseOnFileExportApproval(a, msg) {
		return false
	}

	if len(protocol.ExtractUserImages(msg)) > 0 && !a.Info.SupportsVision && !a.isCLIAgent() {
		return false
	}

	// Special handling for design analysis requests
	if designAnalysis, ok := msg.Metadata["design_analysis"].(bool); ok && designAnalysis {
		if !a.Info.SupportsVision {
			return false
		}
		if msg.HasMentions() && msg.IsMentioned(a.Info.ID) {
			log.Printf("[%s] 🎨 DESIGN ANALYSIS request detected - will respond", a.Info.Name)
			return true
		}
		return false
	}

	// COLLABORATION: orchestration messages (turn prompts, tasks) are sent from
	// System — evaluate before the generic "ignore System" rule below.
	if collabID := msg.GetCollaborationID(); collabID != "" && a.Collab != nil {
		if msg.Type == protocol.MessageTypeCollabRecap && msg.Metadata != nil {
			if assignee, ok := recapAssigneeFromMetadata(msg.Metadata); ok && assignee == a.Info.ID {
				log.Printf("[%s] ✅ COLLABORATION RECAP - will respond (collab %s)", a.Info.Name, collabID[:8])
				return true
			}
			return false
		}
		if msg.Metadata != nil {
			if internal, ok := msg.Metadata["collab_internal_event"].(bool); ok && internal {
				// Seed banners and status noise stay ignored; turn prompts wake only @mentioned agent.
				if !isCollabTurnPromptForAgent(msg, collabID, a.Info.ID, a.Collab) {
					return false
				}
				if isCollabTurnHandoffContent(msg.Content) {
					log.Printf("[%s] ✅ COLLABORATION TURN HANDOFF - will respond (collab %s)", a.Info.Name, collabID[:8])
					return true
				}
			}
		}
		if a.Collab.IsParticipant(collabID, a.Info.ID) && a.Collab.IsActive(collabID) {
			collabPhase := a.Collab.GetCollaboration(collabID, a.Info.ID).Phase
			if collabPhase == "planning" && a.Collab.PlanningSpeakerCooldownBlocked(collabID, a.Info.ID) {
				if !(msg.IsMentioned(a.Info.ID) && collabOutOfTurnMentionOK(msg, collabPhase)) {
					log.Printf("[%s] ⏸ planning cooldown — waiting for other participants (collab %s)", a.Info.Name, collabID[:8])
					return false
				}
			}
			if collabPhase == "executing" {
				if msg.Type == protocol.MessageTypeCollabTask && msg.Metadata != nil {
					if assignee, ok := taskAssigneeFromMetadata(msg.Metadata); ok && assignee == a.Info.ID {
						if !a.collabTaskRateLimitOK(collabID, msg.GetTaskID()) {
							log.Printf("[%s] ⏳ COLLABORATION TASK rate-limited (collab %s)", a.Info.Name, collabID[:8])
							return false
						}
						log.Printf("[%s] ✅ COLLABORATION TASK (assignee metadata) - will respond (collab %s)", a.Info.Name, collabID[:8])
						return true
					}
				}
				if msg.Type == protocol.MessageTypeCollabRecap && msg.Metadata != nil {
					if assignee, ok := recapAssigneeFromMetadata(msg.Metadata); ok && assignee == a.Info.ID {
						log.Printf("[%s] ✅ COLLABORATION RECAP - will respond (collab %s)", a.Info.Name, collabID[:8])
						return true
					}
				}
				if !msg.IsFromSystem() && msg.IsMentioned(a.Info.ID) && isHumanCollabSpeaker(msg) {
					log.Printf("[%s] ✅ HUMAN @mention during execution - will respond (collab %s)", a.Info.Name, collabID[:8])
					return true
				}
				log.Printf("[%s] ⏸ execution phase — task prompts only (collab %s)", a.Info.Name, collabID[:8])
				return false
			}
			if msg.Type == protocol.MessageTypeCollabTask && msg.Metadata != nil {
				if assignee, ok := taskAssigneeFromMetadata(msg.Metadata); ok && assignee == a.Info.ID {
					if !a.collabTaskRateLimitOK(collabID, msg.GetTaskID()) {
						log.Printf("[%s] ⏳ COLLABORATION TASK rate-limited (collab %s)", a.Info.Name, collabID[:8])
						return false
					}
					log.Printf("[%s] ✅ COLLABORATION TASK (assignee metadata) - will respond (collab %s)", a.Info.Name, collabID[:8])
					return true
				}
			}
			if a.Collab.IsAgentTurn(collabID, a.Info.ID) {
				if collabPhase == "planning" && a.Collab.PlanningSpeakerCooldownBlocked(collabID, a.Info.ID) &&
					!isHumanCollabSpeaker(msg) {
					log.Printf("[%s] ⏸ planning cooldown (turn held for other participants) (collab %s)", a.Info.Name, collabID[:8])
					return false
				}
				log.Printf("[%s] ✅ COLLABORATION TURN - will respond (collab %s)", a.Info.Name, collabID[:8])
				return true
			}
			if !msg.IsFromSystem() && msg.IsMentioned(a.Info.ID) && isHumanCollabSpeaker(msg) &&
				collabOutOfTurnMentionOK(msg, collabPhase) {
				log.Printf("[%s] ✅ HUMAN @mention during %s - will respond (collab %s)", a.Info.Name, collabPhase, collabID[:8])
				return true
			}
			if msg.IsMentioned(a.Info.ID) && a.Collab.AgentOutOfTurnMentionAllowed(collabID) &&
				collabOutOfTurnMentionOK(msg, collabPhase) {
				log.Printf("[%s] ✅ MENTIONED in collaboration - will respond (collab %s)", a.Info.Name, collabID[:8])
				return true
			}
			return false
		}
	}

	// Collaboration channel without metadata: still block agent chatter after discussion limits.
	if a.Collab != nil && a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeCollaboration {
		if info := a.Collab.GetCollaborationForAgent(a.Info.ID); info.ID != "" && info.Channel == msg.Channel {
			isFromAgent := msg.From.Type == protocol.AgentTypeFrontend ||
				msg.From.Type == protocol.AgentTypeBackend ||
				msg.From.Type == protocol.AgentTypeDatabase ||
				msg.From.Type == protocol.AgentTypeSecurity ||
				msg.From.Type == protocol.AgentTypeRust ||
				msg.From.Type == protocol.AgentTypeArchitecture ||
				msg.From.Type == protocol.AgentTypeCodeReview ||
				protocol.IsLifeSciencesAgentType(msg.From.Type) ||
				msg.From.Type == protocol.AgentTypeDevOps ||
				msg.From.Type == protocol.AgentTypeRepo ||
				msg.From.Type == protocol.AgentTypeExpert ||
				msg.From.Type == protocol.AgentTypeAssistant ||
				msg.From.Type == protocol.AgentTypeModerator ||
				msg.From.Type == protocol.AgentTypeCLI ||
				msg.From.Type == protocol.AgentTypeConfluence
			if isFromAgent && msg.From.ID != a.Info.ID {
				if info.Phase == "executing" {
					// Peer task replies are collaboration_discussion; only task prompts wake assignees (handled above).
					return false
				}
				if !a.Collab.IsAgentTurn(info.ID, a.Info.ID) && !a.Collab.AgentOutOfTurnMentionAllowed(info.ID) {
					log.Printf("[%s] ⏸ collaboration discussion closed — ignoring (collab %s)", a.Info.Name, info.ID[:8])
					return false
				}
			}
		}
	}

	// Never respond to system messages (errors, notifications, join/leave, etc.)
	if msg.From.Name == "System" || msg.From.ID == "system" {
		return false
	}
	if msg.Type == protocol.MessageTypeSystemInfo || msg.Type == protocol.MessageTypeAgentJoin || msg.Type == protocol.MessageTypeAgentLeave {
		return false
	}

	// Never respond to our own messages
	if msg.From.ID == a.Info.ID {
		return false
	}

	// DM channels: answer the human before any collaboration turn logic. The user
	// is always talking to this agent in a 1:1 room.
	if a.isDMChannel(msg.Channel) {
		isFromAgent := msg.From.Type == protocol.AgentTypeFrontend ||
			msg.From.Type == protocol.AgentTypeBackend ||
			msg.From.Type == protocol.AgentTypeDatabase ||
			msg.From.Type == protocol.AgentTypeSecurity ||
			msg.From.Type == protocol.AgentTypeRust ||
			msg.From.Type == protocol.AgentTypeArchitecture ||
			msg.From.Type == protocol.AgentTypeCodeReview ||
			protocol.IsLifeSciencesAgentType(msg.From.Type) ||
			msg.From.Type == protocol.AgentTypeDevOps ||
			msg.From.Type == protocol.AgentTypeRepo ||
			msg.From.Type == protocol.AgentTypeExpert ||
			msg.From.Type == protocol.AgentTypeAssistant ||
			msg.From.Type == protocol.AgentTypeModerator ||
			msg.From.Type == protocol.AgentTypeCLI ||
			msg.From.Type == protocol.AgentTypeConfluence
		if !isFromAgent {
			if DMHumanMessageShouldRespond(msg, a.Info.ID) {
				log.Printf("[%s] ✅ DM CHANNEL - will respond", a.Info.Name)
				return true
			}
			return false
		}
		return false
	}

	channelType := a.effectiveChannelType(msg.Channel)

	// THREAD HANDLING: In threads, respond if mentioned OR if we posted the parent message
	if msg.IsInThread() {
		threadID := msg.GetThreadID()

		// Check if we posted the parent message (thread was created from our message)
		parentAuthorID := a.Hub.GetThreadParentAuthor(threadID)
		if parentAuthorID == a.Info.ID {
			log.Printf("[%s] ✅ THREAD PARENT AUTHOR - will respond (thread created from our message)", a.Info.Name)
			return true
		}

		// Check if explicitly mentioned
		if msg.HasMentions() && msg.IsMentioned(a.Info.ID) {
			log.Printf("[%s] ✅ MENTIONED in thread - will respond", a.Info.Name)
			return true
		}

		return false
	}

	// Assistant stays passive on public channels unless mentioned or asking NJ platform help.
	if a.Info.Type == protocol.AgentTypeAssistant && channelType == protocol.ChannelTypePublic {
		return assistantPublicShouldRespond(a, msg)
	}

	// Check if message is from another agent (not a human)
	// We need to check this BEFORE mention checking, but handle mentions specially
	isFromAgent := msg.From.Type == protocol.AgentTypeFrontend ||
		msg.From.Type == protocol.AgentTypeBackend ||
		msg.From.Type == protocol.AgentTypeDatabase ||
		msg.From.Type == protocol.AgentTypeSecurity ||
		msg.From.Type == protocol.AgentTypeRust ||
		msg.From.Type == protocol.AgentTypeArchitecture ||
		msg.From.Type == protocol.AgentTypeCodeReview ||
		protocol.IsLifeSciencesAgentType(msg.From.Type) ||
		msg.From.Type == protocol.AgentTypeDevOps ||
		msg.From.Type == protocol.AgentTypeRepo ||
		msg.From.Type == protocol.AgentTypeExpert ||
		msg.From.Type == protocol.AgentTypeAssistant ||
		msg.From.Type == protocol.AgentTypeModerator ||
		msg.From.Type == protocol.AgentTypeCLI

	// If message has @mentions, ONLY respond if explicitly mentioned.
	// History replay may drop Mentions; re-parse @tokens from content first.
	a.backfillMentionsFromContent(msg)
	contentMentions := exclusiveMentionTokens(msg)
	if msg.HasMentions() || len(contentMentions) > 0 {
		if msg.IsMentioned(a.Info.ID) {
			// Check if this is a review request (replying to another agent's message)
			if msg.ReplyTo != "" {
				// Enforce max review depth from explicit metadata even when the
				// replied message is missing from local history.
				if msg.GetReviewDepth() >= 1 {
					return false
				}

				// Find the message being replied to
				var repliedToMsg *protocol.Message
				for _, histMsg := range a.channelHistory(msg.Channel) {
					if histMsg.ID == msg.ReplyTo {
						repliedToMsg = histMsg
						break
					}
				}

				// Check if the replied-to message is from an agent
				if repliedToMsg != nil {
					isRepliedToAgent := repliedToMsg.From.Type == protocol.AgentTypeFrontend ||
						repliedToMsg.From.Type == protocol.AgentTypeBackend ||
						repliedToMsg.From.Type == protocol.AgentTypeDatabase ||
						repliedToMsg.From.Type == protocol.AgentTypeSecurity ||
						repliedToMsg.From.Type == protocol.AgentTypeRust ||
						repliedToMsg.From.Type == protocol.AgentTypeArchitecture ||
						repliedToMsg.From.Type == protocol.AgentTypeCodeReview ||
						protocol.IsLifeSciencesAgentType(repliedToMsg.From.Type) ||
						repliedToMsg.From.Type == protocol.AgentTypeDevOps ||
						repliedToMsg.From.Type == protocol.AgentTypeRepo ||
						repliedToMsg.From.Type == protocol.AgentTypeExpert ||
						repliedToMsg.From.Type == protocol.AgentTypeAssistant ||
						repliedToMsg.From.Type == protocol.AgentTypeModerator ||
						repliedToMsg.From.Type == protocol.AgentTypeCLI

					if isRepliedToAgent {
						// This is a review request - check depth limits
						repliedToDepth := repliedToMsg.GetReviewDepth()
						if repliedToDepth >= 1 {
							return false
						}

						// Valid review request (depth 0 -> will become depth 1)
						log.Printf("[%s] ✅ REVIEW REQUEST detected (replied message depth %d, replying to %s)",
							a.Info.Name, repliedToDepth, msg.ReplyTo[:8])
						return true
					}
				}
			}

			// Not a review, or regular mention - always respond
			log.Printf("[%s] ✅ EXPLICITLY MENTIONED - will respond", a.Info.Name)
			return true
		}
		if a.isMentionedByContentTokens(contentMentions) {
			log.Printf("[%s] ✅ MENTIONED (content token) - will respond", a.Info.Name)
			return true
		}
		// Not mentioned but message has mentions - don't respond
		return false
	}

	// If no mentions specified, don't respond to other agents to prevent loops
	// Only respond to human messages when not explicitly mentioned
	if isFromAgent {
		return false
	}

	if userRequestsImplementationStatusCheck(msg.Content) &&
		channelHasRecentImplementationActivity(a.channelHistory(msg.Channel), msg.ID, a.Info.ID) {
		log.Printf("[%s] ✅ IMPLEMENTATION STATUS CHECK — will respond", a.Info.Name)
		return true
	}

	// Always respond if mentioned by name in the content
	if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(a.Info.Name)) {
		return true
	}

	// Respond to questions related to expertise
	content := strings.ToLower(msg.Content)

	// In custom channels we allow intent-style requests (without "?") so
	// relevant specialists can auto-respond without explicit @mentions.
	isQuestion := msg.Type == protocol.MessageTypeQuestion ||
		strings.Contains(content, "?")
	if !isQuestion && (channelType == protocol.ChannelTypeCustom || channelType == protocol.ChannelTypeCollaboration) {
		isQuestion = looksLikeUserRequest(content)
	}

	if !isQuestion {
		return false
	}

	// Check if STRONGLY related to our expertise
	// Use word boundaries to prevent false positives (e.g., "task" matching "task management")
	words := strings.Fields(content)
	wordSet := make(map[string]bool)
	for _, word := range words {
		// Remove punctuation for matching
		word = strings.Trim(word, ".,!?;:")
		if len(word) >= 2 {
			wordSet[word] = true
		}
	}

	// Check expertise keywords - require whole word matches
	relevanceScore := 0
	for _, skill := range a.Info.Expertise {
		skillLower := strings.ToLower(skill)
		skillWords := strings.Fields(skillLower)

		// Check if any significant word from expertise appears in message
		for _, skillWord := range skillWords {
			skillWord = strings.Trim(skillWord, ".,!?;:")
			if len(skillWord) >= 2 && wordSet[skillWord] {
				relevanceScore += 2
			}
		}

		// Also check for full skill phrase match (for multi-word skills like "task management")
		if len(skillWords) > 1 {
			skillPhrase := strings.Join(skillWords, " ")
			if strings.Contains(content, skillPhrase) {
				relevanceScore += 3
			}
		}
	}

	// Check agent type keywords - require whole word matches only
	typeKeywords := a.getTypeKeywords()
	for _, keyword := range typeKeywords {
		// Must be a whole word match to prevent false positives
		if wordSet[keyword] ||
			strings.Contains(content, " "+keyword+" ") ||
			strings.HasPrefix(content, keyword+" ") ||
			strings.HasSuffix(content, " "+keyword) {
			relevanceScore++
		}
	}

	// Custom- and collaboration-channel behavior: prefer expertise-relevant replies, and only fall
	// back to broad prompts with a responder cap to reduce noise.
	if (channelType == protocol.ChannelTypeCustom || channelType == protocol.ChannelTypeCollaboration) && msg.From.Type == "human" && !msg.HasMentions() {
		if relevanceScore >= customChannelRelevanceMinScore {
			return true
		}
		if isCustomChannelPrompt(content) && a.allowCustomChannelBroadPromptReply(msg) {
			return true
		}
		return false
	}

	if relevanceScore > 0 {
		return true
	}

	return false
}

// getTypeKeywords returns keywords related to the agent's type
func (a *Agent) getTypeKeywords() []string {
	switch a.Info.Type {
	case protocol.AgentTypeFrontend:
		return []string{"ui", "frontend", "react", "vue", "angular", "css", "html", "component", "user interface"}
	case protocol.AgentTypeBackend:
		return []string{"api", "backend", "server", "endpoint", "service", "database", "business logic"}
	case protocol.AgentTypeDevOps:
		return []string{"deploy", "deployment", "ci/cd", "docker", "kubernetes", "infrastructure", "monitoring",
			"aws", "azure", "gcp", "cloud", "terraform", "ansible", "pipeline", "ecs", "eks", "lambda"}
	case protocol.AgentTypeDatabase:
		return []string{"database", "sql", "query", "schema", "migration", "postgres", "mysql", "mongodb",
			"db", "documentdb", "dynamodb", "aurora", "rds", "nosql", "redis", "index"}
	case protocol.AgentTypeSecurity:
		return []string{"security", "auth", "authentication", "authorization", "encryption", "vulnerability", "xss", "sql injection",
			"iam", "ssl", "tls", "cors", "csrf", "rbac", "jwt", "oauth2", "secrets"}
	case protocol.AgentTypeRust:
		return []string{"rust", "cargo", "tokio", "ownership", "borrowing", "lifetime", "trait", "async", "unsafe", "wasm", "serde", "crate"}
	case protocol.AgentTypeArchitecture:
		return []string{"architecture", "architect", "system design", "design", "scalability", "reliability", "tradeoff", "migration", "service boundary", "integration"}
	case protocol.AgentTypeCodeReview:
		return []string{"review", "code review", "correctness", "maintainability", "testing", "refactor", "regression", "readability", "quality"}
	case protocol.AgentTypeBiology, protocol.AgentTypeGenomics, protocol.AgentTypeStructuralBiology, protocol.AgentTypeCheminformatics:
		return []string{"biology", "protein", "gene", "genome", "dna", "rna", "sequence", "assay", "crispr", "enzyme", "mutation", "pathway", "cell", "lab", "protocol"}
	default:
		return []string{}
	}
}

func looksLikeUserRequest(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	requestPrefixes := []string{
		"how ", "what ", "why ", "where ", "when ", "can ", "could ", "would ",
		"please ", "help ", "show ", "build ", "create ", "fix ", "debug ",
		"review ", "explain ", "plan ", "implement ",
	}
	for _, prefix := range requestPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return strings.Contains(trimmed, " please ") || strings.HasSuffix(trimmed, " please")
}

// shouldInjectWorkspaceCode decides whether to proactively inject workspace code
// context for a message. We only do this for code-analysis intents, not for
// capability/permission/tasking questions (e.g. "can you create files?").
func shouldInjectWorkspaceCode(content string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(content))
	if trimmed == "" {
		return false
	}

	// Some "can you ..." prompts are actually concrete code-access requests
	// and should load workspace code context.
	if shouldTreatCapabilityAsCodeRequest(trimmed) {
		return true
	}

	// Capability/permission style prompts should stay direct and not be drowned
	// by large workspace code context.
	capabilityPrefixes := []string{
		"can you ", "could you ", "are you able", "are you allowed",
		"do you support", "can i ", "could i ",
	}
	for _, p := range capabilityPrefixes {
		if strings.HasPrefix(trimmed, p) {
			return false
		}
	}
	capabilityPhrases := []string{
		"create files", "create a file", "add a readme", "write files",
		"edit files", "modify files", "make changes",
	}
	for _, p := range capabilityPhrases {
		if strings.Contains(trimmed, p) {
			return false
		}
	}

	// Positive signals for code-level analysis where source context is helpful.
	codeIntentPhrases := []string{
		"review", "analyze", "audit", "debug", "trace", "walk through",
		"explain this code", "why is this failing", "where is", "find in code",
		"refactor", "fix bug", "line ", "function ", "struct ", "trait ",
	}
	for _, p := range codeIntentPhrases {
		if strings.Contains(trimmed, p) {
			return true
		}
	}

	// Explicit file paths strongly indicate code-context intent.
	return len(DetectFilePaths(content)) > 0
}

func shouldTreatCapabilityAsCodeRequest(trimmedLower string) bool {
	codeAccessSignals := []string{
		"open ", "share ", "show ", "read ", "inspect ",
		"source file", "source files", "source code",
		"implementation details", "implementation", "how it works",
		".rs", ".go", ".py", ".ts", ".tsx", ".js",
		"src/", "cargo.toml", "main.rs", "lib.rs",
	}
	for _, s := range codeAccessSignals {
		if strings.Contains(trimmedLower, s) {
			return true
		}
	}
	return false
}

func isCustomChannelPrompt(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	channelPrompts := []string{
		"who's here", "whos here", "who is here", "who all is here", "who all here",
		"in this channel", "anyone here", "everyone here", "roll call",
		"can you all", "could you all", "all of you", "team", "together",
	}
	for _, phrase := range channelPrompts {
		if strings.Contains(trimmed, phrase) {
			return true
		}
	}
	return false
}

func (a *Agent) allowCustomChannelBroadPromptReply(msg *protocol.Message) bool {
	// Best-effort cap: count existing agent replies to this message.
	recent, err := a.Hub.GetMessages(msg.Channel, 80)
	if err == nil {
		replies := 0
		for _, m := range recent {
			if m.Type != protocol.MessageTypeChat || m.ReplyTo != msg.ID {
				continue
			}
			if isAgentType(m.From.Type) {
				replies++
			}
		}
		if replies >= customChannelBroadPromptResponderCap {
			return false
		}
	}

	// Stable deterministic ordering so the same small subset responds.
	agentIDs := []string{}
	channelAgents, err := a.Hub.GetChannelAgents(msg.Channel)
	if err != nil {
		return true
	}
	for _, ag := range channelAgents {
		if isAgentType(ag.Type) {
			agentIDs = append(agentIDs, ag.ID)
		}
	}
	if len(agentIDs) <= customChannelBroadPromptResponderCap {
		return true
	}
	sort.Slice(agentIDs, func(i, j int) bool {
		hi := sha1.Sum([]byte(msg.ID + ":" + agentIDs[i]))
		hj := sha1.Sum([]byte(msg.ID + ":" + agentIDs[j]))
		return strings.Compare(fmt.Sprintf("%x", hi), fmt.Sprintf("%x", hj)) < 0
	})
	limit := customChannelBroadPromptResponderCap
	for i := 0; i < limit && i < len(agentIDs); i++ {
		if agentIDs[i] == a.Info.ID {
			return true
		}
	}
	return false
}

func isAgentType(t protocol.AgentType) bool {
	return t == protocol.AgentTypeFrontend ||
		t == protocol.AgentTypeBackend ||
		t == protocol.AgentTypeDatabase ||
		t == protocol.AgentTypeSecurity ||
		t == protocol.AgentTypeRust ||
		t == protocol.AgentTypeArchitecture ||
		t == protocol.AgentTypeCodeReview ||
		protocol.IsLifeSciencesAgentType(t) ||
		t == protocol.AgentTypeDevOps ||
		t == protocol.AgentTypeRepo ||
		t == protocol.AgentTypeExpert ||
		t == protocol.AgentTypeAssistant ||
		t == protocol.AgentTypeModerator ||
		t == protocol.AgentTypeCLI
}

// generateResponse generates an AI response based on the message and context
