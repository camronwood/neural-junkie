package agent

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/pipeline"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
	"github.com/camronwood/neural-junkie/internal/trace"
	"github.com/google/uuid"
)

type turnOutcome int

const (
	turnContinue turnOutcome = iota
	turnDone
	turnFailed
	turnCancelled
)

// turnState carries per-turn pipeline state through explicit composable steps.
type turnState struct {
	agent          *Agent
	ctx            context.Context
	genCancel      context.CancelFunc
	msg            *protocol.Message
	clearResponded func()

	traceRecorder *trace.Recorder
	intent        TurnIntent
	knowledgePlan routing.KnowledgePlan

	outcome turnOutcome

	response            string
	streamMsgID         string
	reasoningText       string
	genErr              error
	eff                 ai.AIProvider
	implSessionProposed bool
	implSessionFiles    []string
	implSessionOutcome  map[string]interface{}
	proposedFileChange  bool
	proposedGitChange   bool
	responseMsg         *protocol.Message
	collabID            string
	collabPhase         string
	responseHasImage    bool
	toolSteps           []map[string]interface{}
}

func (a *Agent) runTurnPipeline(ctx context.Context, msg *protocol.Message, clearResponded func()) {
	st := &turnState{
		agent:          a,
		msg:            msg,
		clearResponded: clearResponded,
		outcome:        turnContinue,
		toolSteps:      make([]map[string]interface{}, 0),
	}
	st.traceRecorder = trace.NewRecorder(msg.ID, msg.Channel, a.Info.ID)
	st.ctx = trace.WithRecorder(ctx, st.traceRecorder)
	root := st.traceRecorder.StartSpan("turn", nil)
	defer func() {
		root.End(nil)
		if st.genCancel != nil {
			st.genCancel()
		}
	}()

	steps := a.defaultTurnPipeline(st)
	if err := pipeline.Run(st.ctx, steps); err != nil && !errors.Is(err, errTurnPipelineHalt) {
		log.Printf("[%s] turn pipeline error: %v", a.Info.Name, err)
	}
}

func (a *Agent) defaultTurnPipeline(st *turnState) []pipeline.Step {
	return []pipeline.Step{
		pipeline.FuncStep{StepName: "prepare_turn", Fn: st.stepPrepareTurn},
		pipeline.FuncStep{StepName: "intent_classify", Fn: st.stepIntentClassify},
		pipeline.FuncStep{StepName: "knowledge_plan", Fn: st.stepKnowledgePlan},
		pipeline.FuncStep{StepName: "knowledge_execute", Fn: st.stepKnowledgeExecute},
		pipeline.FuncStep{StepName: "governance_record", Fn: st.stepGovernanceRecord},
		pipeline.FuncStep{StepName: "provider_route", Fn: st.stepProviderRoute},
		pipeline.FuncStep{StepName: "generate", Fn: st.stepGenerate},
		pipeline.FuncStep{StepName: "post_process", Fn: st.stepPostProcess},
		pipeline.FuncStep{StepName: "stamp_metadata", Fn: st.stepStampMetadata},
		pipeline.FuncStep{StepName: "deliver_response", Fn: st.stepDeliverResponse},
	}
}

func (st *turnState) skipIfDone() error {
	if st.outcome != turnContinue {
		return errTurnPipelineHalt
	}
	return nil
}

var errTurnPipelineHalt = errors.New("turn pipeline halted")

func (st *turnState) stepPrepareTurn(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "prepare_turn", nil)
	defer span.End(nil)

	a := st.agent
	msg := st.msg
	a.resetRoutingSnapshot()
	a.resetCompressSnapshot()
	a.syncWorkspaceFromMessage(msg)

	log.Printf("[%s] ⬇️ RECEIVED msg ID %s from %s (mentions: %v)", a.Info.Name, msg.ID[:8], msg.From.Name, msg.Mentions)
	log.Printf("[%s] ✅ MARKED msg %s as responded", a.Info.Name, msg.ID[:8])

	a.sendThinkingStatus(msg, protocol.ThinkingStatusStarted)
	a.resetCADWrittenPaths()

	genCtx, genCancel := collabGenerationContext(ctx, msg)
	timeoutCancel := genCancel
	genCtx, genCancel = context.WithCancel(genCtx)
	genID := a.registerGenCancel(msg.Channel, genCancel)
	st.genCancel = func() {
		a.unregisterGenCancel(msg.Channel, genID)
		genCancel()
		timeoutCancel()
	}
	st.ctx = trace.WithRecorder(genCtx, st.traceRecorder)
	return nil
}

func (st *turnState) stepIntentClassify(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "intent_classify", nil)
	st.intent = st.agent.classifyTurnIntentForMessage(st.msg)
	span.End(map[string]any{"intent": st.intent.String()})

	if st.intent == IntentClosure {
		if resp, ok := tryConversationalClosure(st.agent, st.msg); ok {
			log.Printf("[%s] Conversational closure (fast path): %q", st.agent.Info.Name, truncateForLog(st.msg.Content, 60))
			if err := st.agent.sendQuickChatReply(st.msg, resp); err != nil {
				log.Printf("[%s] Error sending closure: %v", st.agent.Info.Name, err)
				st.clearResponded()
				st.outcome = turnFailed
			} else {
				st.outcome = turnDone
			}
			return errTurnPipelineHalt
		}
	}
	return nil
}

func (st *turnState) stepKnowledgePlan(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "knowledge_plan", nil)
	defer span.End(nil)

	st.agent.recordKnowledgeRoute(st.msg, st.intent)
	st.knowledgePlan = st.agent.effectiveKnowledgePlan(st.msg, st.intent)
	span.End(map[string]any{
		"reason":  st.knowledgePlan.Reason,
		"targets": routeTargetsToStrings(st.knowledgePlan.Targets),
	})
	return nil
}

func (st *turnState) stepKnowledgeExecute(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "knowledge_execute", nil)
	defer span.End(nil)

	st.agent.applyKnowledgePlanEarly(st.msg, st.intent)
	st.agent.ExecuteKnowledgePlan(ctx, st.msg, st.knowledgePlan, st.intent, KnowledgePhasePrompt)
	return nil
}

func (st *turnState) stepGovernanceRecord(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "governance_record", nil)
	defer span.End(nil)

	st.agent.recordTurnGovernance(st.msg)
	return nil
}

func (st *turnState) stepProviderRoute(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "provider_route", nil)
	defer span.End(nil)

	a := st.agent
	msg := st.msg
	log.Printf("[%s] 💬 WILL RESPOND to msg %s from %s: %s", a.Info.Name, msg.ID[:8], msg.From.Name, msg.Content[:min(50, len(msg.Content))])

	eff := a.EffectiveAIProvider(st.ctx, msg)
	if eff == nil {
		eff = a.GetAIProvider()
	}
	st.eff = eff
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
	return nil
}

func (st *turnState) stepGenerate(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "generate", nil)
	defer func() {
		if st.genErr != nil {
			span.EndError(st.genErr, nil)
		} else {
			span.End(nil)
		}
	}()

	a := st.agent
	msg := st.msg
	eff := st.eff
	toolObserver := a.makeToolStepObserver(st.ctx, msg, &st.toolSteps)

	var response string
	var streamMsgID string
	var reasoningText string
	var err error

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
	} else if collabLightMarkdownEligible(msg) {
		log.Printf("[%s] 📝 Collab light markdown execution...", a.Info.Name)
		genCtx := a.withToolObserver(st.ctx, toolObserver)
		response, implSessionProposed, implSessionFiles, err = a.runCollabLightMarkdownExecution(genCtx, msg, eff)
	} else if shouldRunImplementationSession(a, msg) {
		log.Printf("[%s] 🔧 Implementation session...", a.Info.Name)
		genCtx := a.withToolObserver(st.ctx, toolObserver)
		if implementationBestOfK(msg) > 1 {
			if sp, ok := eff.(ai.StreamingProvider); ok && sp.SupportsStreaming() {
				streamMsgID = uuid.New().String()
				response, streamMsgID, implSessionProposed, implSessionFiles, implSessionOutcome, err = a.runImplementationSessionBestOfK(genCtx, msg, eff, streamMsgID)
			} else {
				response, streamMsgID, implSessionProposed, implSessionFiles, implSessionOutcome, err = a.runImplementationSessionBestOfK(genCtx, msg, eff, "")
			}
		} else if sp, ok := eff.(ai.StreamingProvider); ok && sp.SupportsStreaming() {
			streamMsgID = uuid.New().String()
			response, streamMsgID, implSessionProposed, implSessionFiles, implSessionOutcome, err = a.runImplementationSessionStreaming(genCtx, msg, eff, streamMsgID)
		} else {
			response, implSessionProposed, implSessionFiles, implSessionOutcome, err = a.runImplementationSession(genCtx, msg, eff)
		}
	} else if sp, ok := eff.(ai.StreamingProvider); ok && sp.SupportsStreaming() {
		log.Printf("[%s] 📡 Streaming response...", a.Info.Name)
		genCtx := a.withToolObserver(st.ctx, toolObserver)
		llmSpan := trace.StartSpan(genCtx, "llm_call", map[string]any{"mode": "stream"})
		response, streamMsgID, reasoningText, err = a.generateResponseStreaming(genCtx, msg, eff)
		if err != nil {
			llmSpan.EndError(err, nil)
		} else {
			llmSpan.End(nil)
		}
	} else {
		log.Printf("[%s] 📝 Generating response (batch)...", a.Info.Name)
		genCtx := a.withToolObserver(st.ctx, toolObserver)
		llmSpan := trace.StartSpan(genCtx, "llm_call", map[string]any{"mode": "batch"})
		response, err = a.generateResponse(genCtx, msg, eff)
		if err != nil {
			llmSpan.EndError(err, nil)
		} else {
			llmSpan.End(nil)
		}
	}

	st.response = response
	st.streamMsgID = streamMsgID
	st.reasoningText = reasoningText
	st.genErr = err
	st.implSessionProposed = implSessionProposed
	st.implSessionFiles = implSessionFiles
	st.implSessionOutcome = implSessionOutcome

	if err != nil {
		if errors.Is(err, context.Canceled) {
			st.outcome = turnCancelled
			return st.handleGenerationCancelled()
		}
		st.outcome = turnFailed
		return st.handleGenerationError(err)
	}
	return nil
}

func (st *turnState) handleGenerationCancelled() error {
	a := st.agent
	msg := st.msg
	log.Printf("[%s] Generation cancelled (interject)", a.Info.Name)
	st.clearResponded()
	a.sendThinkingStatus(msg, protocol.ThinkingStatusAborted)
	if a.Collab != nil && msg.Type == protocol.MessageTypeCollabDiscussion {
		cid := msg.GetCollaborationID()
		phase := msg.GetCollaborationPhase()
		if cid != "" && phase == "planning" && a.Collab.IsActive(cid) {
			a.kickPlanningTurnWatchdog(cid)
		}
	}
	return errTurnPipelineHalt
}

func (st *turnState) handleGenerationError(err error) error {
	a := st.agent
	msg := st.msg
	log.Printf("[%s] Error generating response: %v", a.Info.Name, err)
	st.clearResponded()
	a.sendThinkingStatus(msg, protocol.ThinkingStatusError)
	a.sendGenerationFailureMessages(msg, err)
	return errTurnPipelineHalt
}

func (st *turnState) stepPostProcess(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "post_process", nil)
	defer span.End(nil)

	a := st.agent
	msg := st.msg
	response := sanitizeInternalToolNames(st.response)
	response = sanitizeAbsolutePathFileChangeFromResponse(response)
	collabCtx := a.getCollaborationContext(msg)
	if collabCtx.ID != "" {
		response = sanitizeCollabDiscussionResponse(response, collabCtx, a.Info.Type)
	}

	var proposedFileChange bool
	var proposedGitChange bool
	var proposalErr error
	if st.implSessionProposed {
		proposedFileChange = len(st.implSessionFiles) > 0
	} else if isAskModeReadOnly(msg) {
		response = sanitizeAskModeResponse(response)
	} else if ctx.Err() != nil {
		log.Printf("[%s] Skipping post-process file proposals: %v", a.Info.Name, ctx.Err())
	} else if err := a.refuseInactiveCollabProposal(msg); err != nil {
		log.Printf("[%s] Skipping post-process file proposals: %v", a.Info.Name, err)
	} else {
		// Use the turn ctx so /cancel-plan abort stops late proposals (never Background).
		response, proposedFileChange, proposalErr = a.maybeSubmitFileChangeFromResponse(ctx, response, msg.Channel, msg)
		if !proposedFileChange && proposalErr == nil {
			response, proposedGitChange, proposalErr = a.maybeSubmitGitChangeFromResponse(ctx, response, msg.Channel, msg)
		}
		if !proposedFileChange && !proposedGitChange && proposalErr == nil {
			response, proposedFileChange, proposalErr = a.maybeProposeCombinedDeliveryExport(ctx, msg, response)
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

	st.response = response
	st.proposedFileChange = proposedFileChange
	st.proposedGitChange = proposedGitChange

	responseType := protocol.MessageTypeChat
	if msg.GetCollaborationID() != "" &&
		a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeCollaboration {
		responseType = protocol.MessageTypeCollabDiscussion
	}
	responseMsg := protocol.NewMessage(responseType, msg.Channel, a.Info, response)
	if st.streamMsgID != "" {
		responseMsg.ID = st.streamMsgID
	}
	if strings.TrimSpace(st.reasoningText) != "" {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata["reasoning_text"] = st.reasoningText
	}
	if consulted := a.TakeDelegationConsulted(); len(consulted) > 0 {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata["delegation_consulted"] = consulted
	}
	if st.implSessionOutcome != nil || (shouldRunImplementationSession(a, msg) && st.implSessionProposed) {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata[protocol.IdeMetaImplementationComplete] = st.implSessionProposed || st.implSessionOutcome != nil
		if len(st.implSessionFiles) > 0 {
			responseMsg.Metadata[protocol.IdeMetaImplementationFiles] = st.implSessionFiles
		}
		if st.implSessionOutcome != nil {
			responseMsg.Metadata[protocol.IdeMetaImplementationOutcome] = st.implSessionOutcome
		}
	}
	if paths := a.takeCADWrittenPaths(); len(paths) > 0 {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata[protocol.IdeMetaCADFilesWritten] = paths
	}
	responseMsg.ReplyTo = msg.ID
	if msg.IsInThread() {
		responseMsg.ThreadID = msg.ThreadID
		responseMsg.IsThreadReply = true
	}
	st.applyReviewMetadata(responseMsg, msg)
	ApplyCollaborationTaskMetadataOnReply(responseMsg, msg, response)

	commandDetector := protocol.NewCommandDetector(nil)
	suggestions := commandDetector.DetectCommands(response, a.Info.Name, responseMsg.ID)
	suggestions = filterCollabCommandSuggestions(msg, suggestions)
	if cwd := collaborationWorkingDirectoryForMessage(a, msg); cwd != "" && len(suggestions) > 0 {
		for i := range suggestions {
			suggestions[i].Cwd = cwd
		}
	}
	if len(suggestions) > 0 {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata["suggested_commands"] = suggestions
	}

	st.responseHasImage = false
	if a.isCLIAgent() {
		workDir := a.resolveCLIWorkDir(msg)
		st.responseHasImage = AttachGeneratedImageFromResponse(responseMsg, workDir)
	}
	st.responseMsg = responseMsg
	return nil
}

func (st *turnState) applyReviewMetadata(responseMsg, msg *protocol.Message) {
	a := st.agent
	if msg.ReplyTo == "" {
		return
	}
	handledReviewMetadata := false
	for _, histMsg := range a.channelHistory(msg.Channel) {
		if histMsg.ID == msg.ReplyTo {
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
				currentDepth := msg.GetReviewDepth()
				responseMsg.SetReviewDepth(currentDepth + 1)
				responseMsg.SetReviewedMessageID(msg.ReplyTo)
				originalQuestionID := msg.GetOriginalQuestionID()
				if originalQuestionID == "" && histMsg.ReplyTo != "" {
					originalQuestionID = histMsg.ReplyTo
				}
				if originalQuestionID != "" {
					responseMsg.SetOriginalQuestionID(originalQuestionID)
				}
				handledReviewMetadata = true
			}
			break
		}
	}
	if !handledReviewMetadata && (msg.IsReviewRequest() || msg.GetReviewDepth() > 0) {
		currentDepth := msg.GetReviewDepth()
		responseMsg.SetReviewDepth(currentDepth + 1)
		responseMsg.SetReviewedMessageID(msg.ReplyTo)
		if originalQuestionID := msg.GetOriginalQuestionID(); originalQuestionID != "" {
			responseMsg.SetOriginalQuestionID(originalQuestionID)
		}
	}
}

func (st *turnState) stepStampMetadata(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "stamp_metadata", nil)
	defer span.End(nil)

	a := st.agent
	responseMsg := st.responseMsg
	if responseMsg == nil {
		return nil
	}
	a.ApplyRoutingMetadataToResponse(responseMsg)
	a.ApplyCompressMetadataToResponse(responseMsg)
	a.ApplyTraceMetadataToResponse(responseMsg, st.traceRecorder, st.toolSteps)
	return nil
}

func (st *turnState) stepDeliverResponse(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "deliver_response", nil)
	defer span.End(nil)

	a := st.agent
	msg := st.msg
	responseMsg := st.responseMsg
	if responseMsg == nil {
		return nil
	}

	var collabID string
	var collabPhase string
	if cid := responseMsg.GetCollaborationID(); cid != "" && a.Collab != nil && msg.Type != protocol.MessageTypeCollabRecap {
		collabPhase = a.Collab.GetCollaboration(cid, a.Info.ID).Phase
		if collabPhase != "executing" {
			collabID = cid
		}
	}

	if st.streamMsgID != "" {
		a.broadcastStreamEnd(msg, st.streamMsgID)
	}
	// Channel-visible delivery first. Recording discussion before SendMessage caused
	// dual-collab isolation flakes where hub msgs advanced with an empty channel transcript.
	if err := a.Hub.SendMessage(responseMsg); err != nil {
		log.Printf("[%s] Error sending message: %v", a.Info.Name, err)
		a.sendThinkingStatus(msg, protocol.ThinkingStatusError)
		if collabID != "" && (collabPhase == "planning" || collabPhase == "reviewing") {
			st.clearResponded()
			if collabPhase == "planning" && a.Collab.IsActive(collabID) {
				a.kickPlanningTurnWatchdog(collabID)
			}
		}
		st.outcome = turnFailed
		return errTurnPipelineHalt
	}

	if collabID != "" && a.Collab != nil {
		if err := a.Collab.RecordMessage(collabID, responseMsg); err != nil {
			log.Printf("[%s] Warning: failed to record collaboration message: %v", a.Info.Name, err)
			if collabPhase == "planning" || collabPhase == "reviewing" {
				st.clearResponded()
				a.sendThinkingStatus(msg, protocol.ThinkingStatusError)
				if collabPhase == "planning" && a.Collab.IsActive(collabID) {
					a.kickPlanningTurnWatchdog(collabID)
				}
				st.outcome = turnFailed
				return errTurnPipelineHalt
			}
		} else {
			a.Collab.AnalyzeConsensus(collabID, responseMsg)
		}
	}

	learning.MaybeSuggestAfterAgentReply(msg.Channel, a.Info.ID, a.Info.Name, string(a.Info.Type), msg.Content, st.response)
	a.addToHistory(responseMsg)
	a.MaybePostHubGeneratedImageForCLI(msg, st.responseHasImage)
	a.sendThinkingStatus(msg, protocol.ThinkingStatusCompleted)

	if collabID != "" && collabPhase == "planning" {
		a.promptNextCollaborationTurn(responseMsg, collabID)
	}
	st.outcome = turnDone
	return nil
}

func (a *Agent) makeToolStepObserver(ctx context.Context, msg *protocol.Message, steps *[]map[string]interface{}) func(ai.ToolStepEvent) {
	return func(ev ai.ToolStepEvent) {
		toolSpan := trace.StartSpan(ctx, "tool_call", map[string]any{
			"name":      ev.Name,
			"kind":      ev.Kind,
			"iteration": ev.Iteration,
		})
		if ev.Kind == "error" {
			toolSpan.End(map[string]any{"preview": ev.Preview})
		} else {
			toolSpan.End(nil)
		}
		streamMsgID := StreamMessageIDFromContext(ctx)
		a.broadcastToolStep(ctx, msg, streamMsgID, ev)
		if ev.Kind == "start" || ev.Kind == "result" || ev.Kind == "error" {
			*steps = append(*steps, map[string]interface{}{
				"name":       ev.Name,
				"kind":       ev.Kind,
				"iteration":  ev.Iteration,
				"preview":    ev.Preview,
				"max_iterations": ev.MaxIterations,
			})
		}
	}
}

func (a *Agent) withToolObserver(ctx context.Context, obs func(ai.ToolStepEvent)) context.Context {
	if obs == nil {
		return ctx
	}
	return ai.WithToolStepObserver(ctx, obs)
}
