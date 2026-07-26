package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sort"
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

const maxChatValidationEscalations = 3

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
	contextSpan   *trace.SpanHandle
	intent        TurnIntent
	goal          TurnGoal
	context       protocol.TurnContextEnvelope
	evidence      *ActionEvidenceLedger
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
	validationRetried   bool
	validationAttempts  int
	contextRecovered    bool
	actionValidated     bool
}

func (a *Agent) runTurnPipeline(ctx context.Context, msg *protocol.Message, clearResponded func()) {
	st := &turnState{
		agent:          a,
		msg:            msg,
		clearResponded: clearResponded,
		outcome:        turnContinue,
		toolSteps:      make([]map[string]interface{}, 0),
		evidence:       &ActionEvidenceLedger{},
	}
	st.traceRecorder = trace.NewRecorder(msg.ID, msg.Channel, a.Info.ID)
	st.ctx = trace.WithRecorder(ctx, st.traceRecorder)
	root := st.traceRecorder.StartSpan("turn", nil)
	defer func() {
		root.End(nil)
		finalTrace := st.traceRecorder.Close()
		if err := trace.Persist(finalTrace); err != nil {
			log.Printf("[%s] failed to persist final turn trace: %v", a.Info.Name, err)
		}
		if st.genCancel != nil {
			st.genCancel()
		}
		a.endTurnRouting(msg.ID)
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
		pipeline.FuncStep{StepName: "context_select", Fn: st.stepContextSelect},
		pipeline.FuncStep{StepName: "knowledge_plan", Fn: st.stepKnowledgePlan},
		pipeline.FuncStep{StepName: "knowledge_execute", Fn: st.stepKnowledgeExecute},
		pipeline.FuncStep{StepName: "governance_record", Fn: st.stepGovernanceRecord},
		pipeline.FuncStep{StepName: "provider_route", Fn: st.stepProviderRoute},
		pipeline.FuncStep{StepName: "generate", Fn: st.stepGenerate},
		pipeline.FuncStep{StepName: "post_process", Fn: st.stepPostProcess},
		pipeline.FuncStep{StepName: "validate_response", Fn: st.stepValidateResponse},
		pipeline.FuncStep{StepName: "stamp_metadata", Fn: st.stepStampMetadata},
		pipeline.FuncStep{StepName: "deliver_response", Fn: st.stepDeliverResponse},
	}
}

func (st *turnState) stepContextSelect(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "context_select", nil)
	st.contextSpan = span
	st.context = st.agent.selectTurnContext(st.msg)
	st.ctx = contextWithTurnEnvelope(st.ctx, st.context)
	span.End(contextSelectionTraceAttrs(st.context))
	return nil
}

func contextSelectionTraceAttrs(envelope protocol.TurnContextEnvelope) map[string]any {
	selectedIDs := make([]string, 0, len(envelope.Provenance))
	sectionSet := map[string]bool{}
	for _, item := range envelope.Provenance {
		if item.ID != "" {
			selectedIDs = append(selectedIDs, item.ID)
		}
		if item.Section != "" {
			sectionSet[item.Section] = true
		}
	}
	if envelope.Goal != nil {
		sectionSet["goal"] = true
	}
	for section, count := range map[string]int{
		"decisions": len(envelope.Decisions), "unresolved_actions": len(envelope.UnresolvedActions),
		"corrections": len(envelope.Corrections), "recent_exchanges": len(envelope.RecentExchanges),
		"retrieved_memories": len(envelope.RetrievedMemories), "workspace_evidence": len(envelope.WorkspaceEvidence),
	} {
		if count > 0 {
			sectionSet[section] = true
		}
	}
	digestVersion := 0
	if envelope.Summary != nil {
		digestVersion = envelope.Summary.Version
		sectionSet["summary"] = true
	}
	selectedSections := make([]string, 0, len(sectionSet))
	for section := range sectionSet {
		selectedSections = append(selectedSections, section)
	}
	sort.Strings(selectedSections)
	omissions := make(map[string]string, len(envelope.SupersededMessageIDs))
	for _, id := range envelope.SupersededMessageIDs {
		if id != "" {
			omissions[id] = "superseded"
		}
	}
	return map[string]any{
		"selected_context_ids": selectedIDs,
		"selected_sections":    selectedSections,
		"dropped_context_ids":  append([]string(nil), envelope.SupersededMessageIDs...),
		"omission_reasons":     omissions,
		"provenance":           append([]protocol.TurnContextProvenance(nil), envelope.Provenance...),
		"digest_version":       digestVersion,
		"section_sizes":        contextEnvelopeSectionSizes(envelope),
		"section_budgets":      envelope.SectionBudgets,
		"compression": map[string]any{
			"summary_checkpoint": envelope.Summary != nil,
		},
		"recovery": map[string]any{
			"active":             len(envelope.Corrections) > 0 || len(envelope.SupersededMessageIDs) > 0,
			"correction_count":   len(envelope.Corrections),
			"superseded_count":   len(envelope.SupersededMessageIDs),
			"unresolved_actions": len(envelope.UnresolvedActions),
		},
	}
}

func contextEnvelopeSectionSizes(envelope protocol.TurnContextEnvelope) map[string]map[string]int {
	sections := map[string]any{
		"goal":               envelope.Goal,
		"decisions":          envelope.Decisions,
		"unresolved_actions": envelope.UnresolvedActions,
		"corrections":        envelope.Corrections,
		"recent_exchanges":   envelope.RecentExchanges,
		"retrieved_memories": envelope.RetrievedMemories,
		"workspace_evidence": envelope.WorkspaceEvidence,
		"summary":            envelope.Summary,
	}
	counts := map[string]int{
		"goal": boolInt(envelope.Goal != nil), "decisions": len(envelope.Decisions),
		"unresolved_actions": len(envelope.UnresolvedActions), "corrections": len(envelope.Corrections),
		"recent_exchanges": len(envelope.RecentExchanges), "retrieved_memories": len(envelope.RetrievedMemories),
		"workspace_evidence": len(envelope.WorkspaceEvidence), "summary": boolInt(envelope.Summary != nil),
	}
	out := make(map[string]map[string]int, len(sections))
	for name, value := range sections {
		encoded, _ := json.Marshal(value)
		out[name] = map[string]int{"items": counts[name], "bytes": len(encoded)}
	}
	return out
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
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
	a.beginTurnRouting(msg.ID)
	st.ctx = contextWithTurnRouting(st.ctx, msg.ID)
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
	st.goal = deriveTurnGoal(st.agent, st.msg, st.intent)
	st.intent = st.goal.Intent
	st.goal = persistTurnConversationState(st.agent, st.msg, st.goal)
	st.ctx = contextWithTurnGoal(st.ctx, st.goal)
	st.ctx = contextWithActionEvidence(st.ctx, st.evidence)
	attrs := map[string]any{"intent": st.intent.String(), "action": string(st.goal.Action), "goal_id": st.goal.ID}
	if decision, ok := protocol.ExtractTurnDecision(st.msg); ok {
		attrs["semantic_schema_version"] = decision.SchemaVersion
		attrs["semantic_source"] = decision.Source
		attrs["semantic_confidence"] = decision.Confidence
		attrs["semantic_classifier_model"] = decision.ClassifierModel
		attrs["semantic_classifier_latency_ms"] = decision.ClassifierLatencyMS
		attrs["semantic_policy_overrides"] = decision.PolicyOverrides
		attrs["semantic_abstention"] = decision.AbstentionReason
	}
	span.End(attrs)

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
	if ua, ok := eff.(ai.UsageAware); ok {
		ua.ResetSessionUsage()
	}
	reason := "default_agent_provider"
	source := "rules"
	if msg.Type != protocol.MessageTypeCollabTask && !turnGoalRunsImplementationSession(st.goal) {
		if base := a.GetAIProvider(); base != nil && eff != nil {
			if bm, em := strings.TrimSpace(base.GetModel()), strings.TrimSpace(eff.GetModel()); em != "" && bm != em {
				reason = "capability_routing"
				source = "capabilities"
			}
		}
		a.RecordRoutingFromProviderFor(msg.ID, eff, reason, source)
		a.recordClassifierRouting(msg)
	} else if msg.Type == protocol.MessageTypeCollabTask {
		snap := a.LastRoutingSnapshotFor(msg.ID)
		if snap.Reason != "" {
			reason = snap.Reason
		}
		if snap.Source != "" {
			source = snap.Source
		}
		a.RecordRoutingFromProviderFor(msg.ID, eff, reason, source)
	} else if turnGoalRunsImplementationSession(st.goal) {
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
		if err == nil && !implSessionProposed {
			// No grounded sources — fall back to normal generation instead of shipping an empty stub.
			log.Printf("[%s] light markdown deferred; falling back to generation...", a.Info.Name)
			if sp, ok := eff.(ai.StreamingProvider); ok && sp.SupportsStreaming() {
				log.Printf("[%s] 📡 Streaming response...", a.Info.Name)
				llmSpan := trace.StartSpan(genCtx, "llm_call", map[string]any{"mode": "stream", "light_fallback": true})
				response, streamMsgID, reasoningText, err = a.generateResponseStreaming(genCtx, msg, eff)
				if err != nil {
					llmSpan.EndError(err, nil)
				} else {
					llmSpan.End(nil)
				}
			} else {
				log.Printf("[%s] 📝 Generating response (batch)...", a.Info.Name)
				llmSpan := trace.StartSpan(genCtx, "llm_call", map[string]any{"mode": "batch", "light_fallback": true})
				response, err = a.generateResponse(genCtx, msg, eff)
				if err != nil {
					llmSpan.EndError(err, nil)
				} else {
					llmSpan.End(nil)
				}
			}
		}
	} else if turnGoalRunsImplementationSession(st.goal) {
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
	if st.implSessionOutcome != nil || (turnGoalRunsImplementationSession(st.goal) && st.implSessionProposed) {
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
	} else if msg != nil && msg.ImplementationSession() && !msg.IdeEditorModeIsAsk() && !msg.IdeEditorModeIsPlan() {
		// Inbound requested an implementation session but the turn took a non-session
		// path (e.g. conversational answer). Still emit a minimal outcome so harness
		// assert_message_metadata(require_keys) can observe the decision.
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		if responseMsg.Metadata[protocol.IdeMetaImplementationOutcome] == nil {
			responseMsg.Metadata[protocol.IdeMetaImplementationOutcome] = map[string]interface{}{
				"outcome":             "no_changes",
				"files_changed":       []string{},
				"routing_reason":      "session_not_run",
				"verify_failed":       false,
				"implementation_skip": true,
			}
			responseMsg.Metadata[protocol.IdeMetaImplementationComplete] = false
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

func (st *turnState) stepValidateResponse(ctx context.Context) error {
	if err := st.skipIfDone(); err != nil {
		return err
	}
	span := trace.StartSpan(ctx, "validate_response", nil)
	defer span.End(nil)

	st.buildActionEvidence()
	history := st.agent.conversationHistoryForIntent(st.msg, st.intent)
	issues := validateResponseAgainstEvidence(st.goal, st.evidence, st.msg, st.response, history)
	st.actionValidated = len(issues) == 0 && goalHasExpectedEvidence(st.goal, st.evidence)
	if len(issues) > 0 {
		switch {
		case st.goal.RequiresActionEvidence() && shouldRewriteAsSafeFailure(issues, st.response):
			st.response = safeActionFailure(st.goal, st.evidence)
		case st.goal.RequiresActionEvidence():
			// Misclassified action goals often still get a good conversational answer.
			// Keep it instead of replacing with a canned soft-fail.
		default:
			st.validationRetried = true
			for st.validationAttempts < maxChatValidationEscalations {
				retryProvider, ok := st.agent.EscalateConversationProvider(st.ctx, st.msg)
				if !ok || retryProvider == nil {
					break
				}
				st.validationAttempts++
				st.eff = retryProvider

				retryPrompt := buildResponseValidationRetryPrompt(st.goal, st.evidence, st.msg)
				recovered := false
				if !st.contextRecovered && turnContextRecoveryApplicable(st.context) {
					recoverySpan := trace.StartSpan(ctx, "context_recovery", map[string]any{
						"attempt": st.validationAttempts,
						"reason":  ConversationReasonQualityGateFailure,
					})
					recoveredContext := st.agent.selectTurnContext(st.msg)
					if turnContextRecoveryApplicable(recoveredContext) {
						st.context = recoveredContext
					}
					st.ctx = contextWithTurnEnvelope(st.ctx, st.context)
					retryPrompt = appendDurableConversationContext(retryPrompt, st.context)
					st.contextRecovered = true
					recovered = true
					recoverySpan.End(contextSelectionTraceAttrs(st.context))
				}

				attemptSpan := trace.StartSpan(ctx, "model_attempt", validationAttemptTraceAttrs(
					st.validationAttempts, retryProvider, recovered,
				))
				retry, err := retryProvider.GenerateResponse(
					ai.WithToolApprovalChannel(st.ctx, st.msg.Channel),
					retryPrompt,
					nil,
				)
				retry = sanitizeInternalToolNames(retry)
				retry = sanitizeAbsolutePathFileChangeFromResponse(retry)
				retryIssues := validateResponseAgainstEvidence(st.goal, st.evidence, st.msg, retry, history)
				if err == nil && strings.TrimSpace(retry) != "" && len(retryIssues) == 0 {
					attemptSpan.End(map[string]any{"validation": "passed"})
					st.response = retry
					issues = nil
					break
				}
				attemptSpan.EndError(err, map[string]any{
					"validation": "failed",
					"issues":     validationIssueNames(retryIssues),
				})
				markLastRoutingAttemptFailed(st.msg, ConversationReasonQualityGateFailure)
				st.agent.recordRoutingEvidenceFromMessage(st.msg)
				if len(retryIssues) > 0 {
					issues = retryIssues
				}
			}
			if len(issues) > 0 {
				if literal, ok := tryCodebaseReturnLiteralAnswer(st.msg); ok {
					st.response = literal
				} else {
					st.response = "I couldn't produce a sufficiently grounded answer from the available context."
				}
			}
		}
	}
	if st.responseMsg != nil {
		st.responseMsg.Content = st.response
		if st.responseMsg.Metadata == nil {
			st.responseMsg.Metadata = make(map[string]interface{})
		}
		st.responseMsg.Metadata["turn_goal"] = st.goal
		st.responseMsg.Metadata["action_intent"] = string(st.goal.Action)
		st.responseMsg.Metadata["action_evidence"] = st.evidence.Entries()
		st.responseMsg.Metadata["response_validation_retry"] = st.validationRetried
		st.responseMsg.Metadata["response_validation_attempts"] = st.validationAttempts
		st.responseMsg.Metadata["response_context_recovered"] = st.contextRecovered
		if len(issues) > 0 {
			names := make([]string, 0, len(issues))
			for _, issue := range issues {
				names = append(names, string(issue))
			}
			st.responseMsg.Metadata["response_validation_issues"] = names
			delete(st.responseMsg.Metadata, "suggested_commands")
		}
	}
	return nil
}

func turnContextRecoveryApplicable(envelope protocol.TurnContextEnvelope) bool {
	return envelope.Goal != nil || envelope.Summary != nil ||
		len(envelope.Decisions) > 0 || len(envelope.UnresolvedActions) > 0 ||
		len(envelope.Corrections) > 0 || len(envelope.SupersededMessageIDs) > 0
}

func validationAttemptTraceAttrs(attempt int, provider ai.AIProvider, recovered bool) map[string]any {
	attrs := map[string]any{
		"attempt":          attempt,
		"context_recovery": recovered,
	}
	if provider != nil {
		attrs["provider_id"] = providerIDFromAI(provider)
		attrs["model"] = provider.GetModel()
	}
	return attrs
}

func validationIssueNames(issues []responseValidationIssue) []string {
	names := make([]string, 0, len(issues))
	for _, issue := range issues {
		names = append(names, string(issue))
	}
	return names
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
	if decision, ok := protocol.ExtractTurnDecision(st.msg); ok {
		_ = protocol.StampTurnDecision(responseMsg, decision)
	}
	a.ApplyRoutingMetadataToResponseFor(st.msg.ID, responseMsg)
	a.ApplyCompressMetadataToResponse(responseMsg)
	st.stampContextObservability(responseMsg)
	a.ApplyTraceMetadataToResponse(responseMsg, st.traceRecorder, st.toolSteps)
	a.applyUsageTelemetry(st)
	return nil
}

func (st *turnState) stampContextObservability(responseMsg *protocol.Message) {
	if st == nil || responseMsg == nil || st.msg == nil {
		return
	}
	if responseMsg.Metadata == nil {
		responseMsg.Metadata = make(map[string]interface{})
	}
	for _, key := range []string{
		"injected_memory_count", "injected_memory_ids",
		"injected_codebase_count",
	} {
		if value, ok := st.msg.Metadata[key]; ok {
			responseMsg.Metadata[key] = value
		}
	}
	if st.contextSpan == nil {
		return
	}
	attrs := map[string]any{}
	if raw, ok := st.msg.Metadata[contextBudgetStatsMetadata]; ok {
		switch stats := raw.(type) {
		case ContextBudgetStats:
			attrs["compression"] = map[string]any{
				"summary_checkpoint":  st.context.Summary != nil,
				"applied":             stats.Truncated,
				"original_bytes":      stats.OriginalBytes,
				"final_bytes":         stats.FinalBytes,
				"compressed_sections": stats.CompressedSections,
				"recoverable":         st.msg.Metadata[contextRetrieveCapabilityMetadata] == true,
			}
			if len(stats.CompressedSections) > 0 {
				omissions := map[string]string{}
				for _, section := range stats.CompressedSections {
					omissions[section] = "compressed_to_budget"
				}
				attrs["budget_omission_reasons"] = omissions
			}
		}
	}
	attrs["retrieval_counts"] = map[string]int{
		"memory":   metadataInt(st.msg.Metadata, "injected_memory_count"),
		"codebase": metadataInt(st.msg.Metadata, "injected_codebase_count"),
	}
	st.contextSpan.Annotate(attrs)
}

func metadataInt(metadata map[string]interface{}, key string) int {
	switch value := metadata[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
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
	if st.actionValidated {
		st.completePersistedAction()
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
		if ledger := actionEvidenceFromContext(ctx); ledger != nil {
			ledger.recordToolEvent(ev)
		}
		if ev.Kind == "start" || ev.Kind == "result" || ev.Kind == "error" {
			*steps = append(*steps, map[string]interface{}{
				"name":           ev.Name,
				"kind":           ev.Kind,
				"iteration":      ev.Iteration,
				"preview":        ev.Preview,
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
