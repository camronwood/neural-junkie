package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/contextcompress"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

var assistantExportContinuationRE = regexp.MustCompile(`(?i)\b(save|store|export|write)\s+(it|that|this)\b`)

const (
	implSessionMaxToolIterations = 20
	implSessionMaxEditRounds     = 3
	implSessionTimeout           = 480 * time.Second
	implSessionFrontendTimeout   = 600 * time.Second
	editorTrustAutoApply         = "auto_apply_edits"
)

func collaborationAllowsImplementationSession(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	phase := strings.TrimSpace(msg.GetCollaborationPhase())
	if phase != string(collaboration.PhaseExecuting) {
		return false
	}
	caps := protocol.ResolveTurnCapabilities(msg)
	return caps.CanRunImplSession || msg.IdeEditorModeIsExport() || msg.ImplementationSession()
}

type implSessionStateKey struct{}
type implSessionRoundKey struct{}

// ImplementationSessionState tracks progress during a multi-step implementation run.
type ImplementationSessionState struct {
	EditRound       int
	FilesChanged    []string
	ProposedCount   int
	VerifyOutput    string
	VerifyFailed    bool
	VerifySkipped   bool
	RepairUsed      bool
	RepairAttempts  int
	Phase           string
	StackManifest   *StackManifest
	SeedsLoaded     int
	DiscoverTools   []string
	PreflightErrors []string
	TrustMode                  string
	BootFixIntent              bool
	FixLikeIntent              bool
	ReproCommand               string
	ReproExitCode              int
	ReproOutput                string
	ClarifyQuestionsAsked      int
	ReproBootstrapActive       bool
	DiagnosticBootstrapDone    bool
	CommandHistory             []CommandRunRecord
	LastReadPaths              []string
	BootReadPaths              []string
	SinceLastCommandReadOrEdit bool
	CommandOnlyRounds          int
	CommandFailuresSinceEdit   map[string]int
	LastCommandOutputText      string
	LastFailedCommand          string
	CircuitBreakerFired        bool
	PlaybookUsedName           string
	RegisteredFiles            []string
	RegistrationErrors         []string
	DiagnosePhaseRequired      bool
	DiagnosePhaseComplete      bool
	LastRepairFailureKind      RepairFailureKind
	LastVerifyFailureKind      RepairFailureKind
	PrematureStopAttempts      int
	ToolStepCount              int
	DialogueSummary            string
	DeterministicFallbackUsed  bool
}

func withImplementationSessionState(ctx context.Context, s *ImplementationSessionState) context.Context {
	return context.WithValue(ctx, implSessionStateKey{}, s)
}

func implementationSessionStateFromContext(ctx context.Context) *ImplementationSessionState {
	if ctx == nil {
		return nil
	}
	s, _ := ctx.Value(implSessionStateKey{}).(*ImplementationSessionState)
	return s
}

func withImplementationSessionRound(ctx context.Context, round int) context.Context {
	return context.WithValue(ctx, implSessionRoundKey{}, round)
}

func implementationSessionRoundFromContext(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	r, _ := ctx.Value(implSessionRoundKey{}).(int)
	return r
}

// assistantAllowsImplementationSession gates the personal Assistant out of the file-edit
// implementation loop unless the user clearly wants code changes or export-to-file work.
// Composer export/implementation_session metadata alone is not enough (e.g. travel planning in a DM).
func assistantAllowsImplementationSession(a *Agent, msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if ConversationModeFromMessage(msg) == ConversationModeChat {
		if isAdvisoryImplementationQuestion(msg.Content) {
			return false
		}
	}
	if userRequestsImplementation(msg.Content) || userRequestsFileExport(msg.Content) {
		return true
	}
	if userReferencesPriorAssistantContent(msg.Content) {
		return true
	}
	if msg.IdeEditorModeIsExport() && assistantExportContinuationRE.MatchString(msg.Content) {
		return true
	}
	history := a.channelHistorySafe(msg.Channel)
	if channelHasRecentImplementationActivity(history, msg.ID, a.Info.ID) {
		return userAffirmsPendingImplementation(msg.Content) ||
			userRequestsImplementation(msg.Content) ||
			userRequestsImplementationStatusCheck(msg.Content) ||
			userRequestsFileExport(msg.Content)
	}
	return false
}

// shouldRunImplementationSession reports whether to use the bounded implementation loop.
func shouldRunImplementationSession(a *Agent, msg *protocol.Message) bool {
	if a == nil || msg == nil {
		return false
	}
	if !channelAllowsImplementationSession(msg.Channel, msg) {
		return false
	}
	if msg.GetCollaborationID() != "" && !collaborationAllowsImplementationSession(msg) {
		return false
	}
	if msg.IdeEditorModeIsAsk() || msg.IdeEditorModeIsPlan() || msg.IdeEditorMode() == "ask" || msg.IdeEditorMode() == "plan" {
		return false
	}
	if isAskModeReadOnly(msg) {
		return false
	}
	if a.Info.Type == protocol.AgentTypeAssistant && !assistantAllowsImplementationSession(a, msg) {
		return false
	}
	// Short status follow-ups ("is it fixed?") get a conversational reply, not a new session.
	if userRequestsImplementationStatusCheck(msg.Content) &&
		!userRequestsImplementation(msg.Content) &&
		!messageHasBootOrBuildError(msg.Content) &&
		!msg.ImplementationSession() {
		return false
	}
	if shouldSkipAgentResponseOnFileExportApproval(a, msg) {
		return false
	}
	if shouldDeferImplSessionForCombinedDeliveryExport(a, msg) {
		return false
	}
	if !agentTypeCanShipFileChanges(a.Info.Type) {
		return false
	}
	// CodeReviewer is read-only — never run the file-edit implementation loop.
	if a.Info.Type == protocol.AgentTypeCodeReview {
		return false
	}
	if userRequestsCodeReview(msg.Content) {
		return false
	}
	// Explicit chat-mode turns are advisory only — unless continuing an active implementation
	// thread (status check or boot-fix follow-up).
	if ConversationModeFromMessage(msg) == ConversationModeChat &&
		!msg.ImplementationSession() && !msg.IdeEditorModeIsExport() {
		history := a.channelHistorySafe(msg.Channel)
		active := channelHasRecentImplementationActivity(history, msg.ID, a.Info.ID)
		if !active || (!userRequestsImplementationStatusCheck(msg.Content) &&
			!userRequestsImplementation(msg.Content) && !messageHasBootOrBuildError(msg.Content)) {
			return false
		}
	}

	history := a.channelHistorySafe(msg.Channel)
	if userReferencesPriorAssistantContent(msg.Content) &&
		findPriorAssistantContent(history, msg.ID, a.Info.ID, priorReferenceMinChars) == "" {
		return false
	}
	if vagueContinuationWithoutPriorThread(history, msg.ID, a.Info.ID, msg.Content) {
		return false
	}
	activeThread := channelHasRecentImplementationActivity(history, msg.ID, a.Info.ID)
	wantImpl := userRequestsImplementationForMessage(a, msg) || msg.ImplementationSession() || msg.IdeEditorModeIsExport()

	if userAffirmsPendingImplementation(msg.Content) && isWeakImplementationAffirmation(msg.Content) &&
		!affirmationContinuesImplementation(history, msg.ID, a.Info.ID, msg.Content) &&
		!userRequestsImplementation(msg.Content) {
		return false
	}

	if activeThread && (userAffirmsPendingImplementation(msg.Content) || userRequestsImplementation(msg.Content) ||
		userRequestsImplementationStatusCheck(msg.Content) ||
		userRequestsFileExport(msg.Content) || msg.ImplementationSession()) {
		if userAffirmsPendingImplementation(msg.Content) &&
			!affirmationContinuesImplementation(history, msg.ID, a.Info.ID, msg.Content) &&
			!userRequestsImplementation(msg.Content) {
			return false
		}
		return true
	}
	if !wantImpl {
		return false
	}
	if msg.IdeEditorMode() == "agent" || msg.IdeEditorModeIsExport() {
		if msg.IdeRouteAgentType() != "" || msg.ImplementationSession() || userRequestsImplementation(msg.Content) ||
			userRequestsImplementationStatusCheck(msg.Content) || msg.IdeEditorModeIsExport() {
			return true
		}
	}
	return msg.ImplementationSession() || userRequestsFileExportForMessage(msg) || msg.IdeEditorModeIsExport()
}

func (a *Agent) runImplementationSession(ctx context.Context, msg *protocol.Message, eff ai.AIProvider) (string, bool, []string, map[string]interface{}, error) {
	text, _, proposed, files, outcome, err := a.runImplementationSessionStreaming(ctx, msg, eff, "")
	return text, proposed, files, outcome, err
}

func (a *Agent) runImplementationSessionStreaming(ctx context.Context, msg *protocol.Message, eff ai.AIProvider, streamMsgID string) (string, string, bool, []string, map[string]interface{}, error) {
	if streamMsgID == "" {
		streamMsgID = uuid.New().String()
	}
	frontend := false
	wsPathEarly := a.resolveWorkspacePath(msg)
	if wsPathEarly != "" {
		if m := DetectStackManifest(wsPathEarly); m != nil && (m.HasReact || m.HasTailwind) {
			frontend = true
		}
	}
	sessionTimeout := implSessionTimeoutForMessage(msg, frontend)

	sessionCtx, cancel := context.WithTimeout(ctx, sessionTimeout)
	defer cancel()

	if eff == nil {
		eff = a.GetAIProvider()
	}
	if eff == nil {
		eff = a.EffectiveImplementationProvider(sessionCtx, msg)
	}

	history := a.channelHistory(msg.Channel)
	state := &ImplementationSessionState{
		Phase:     "discover",
		TrustMode: resolveImplementationTrustMode(msg),
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath != "" {
		state.StackManifest = DetectStackManifest(wsPath)
		state.BootFixIntent = messageImpliesBootFix(msg.Content, history)
		state.FixLikeIntent = messageImpliesFixLikeIntent(msg.Content, history)
		if state.FixLikeIntent && !state.BootFixIntent {
			state.BootFixIntent = state.FixLikeIntent
		}
		state.ReproCommand = inferReproCommand(wsPath, state.StackManifest, msg.Content)
		state.DiagnosePhaseRequired = requiresDiagnoseGate(msg, state, wsPath)
	}
	maxToolIter, maxEditRounds, maxFiles := implSessionLimitsForState(msg, state)
	sessionCtx = ai.WithToolLoopMaxIterations(sessionCtx, maxToolIter)
	perf := performanceFromHub()
	sessionCtx = contextcompress.WithAgentRetrieveBudget(sessionCtx, perf.AgentRuntimeMaxRetrievePerTurnOrDefault())

	a.sendThinkingActivity(msg, protocol.ThinkingActivityImplementation, "exploring workspace and applying fixes")

	sessionCtx = withImplementationSessionState(sessionCtx, state)
	sessionCtx = shared.ContextWithImplementationSession(sessionCtx, true)
	sessionCtx = attachImplSessionCommandPolicy(sessionCtx, state)
	sessionCtx = ContextWithImplementationRoutingHints(sessionCtx, ImplementationRoutingHints{
		RepairAttempts: state.RepairAttempts,
		VerifyFailed:   state.VerifyFailed,
		BootFixIntent:  state.BootFixIntent,
	})
	restoreImplSessionFromCheckpoint(msg, state)

	if question, ask := maybeAskFixClarification(msg, state, wsPath); ask {
		outcome := a.buildImplementationSessionOutcome(msg, state, false)
		return question, streamMsgID, false, nil, outcome, nil
	}

	if userRequestsDestructiveCommand(msg.Content) {
		state.FilesChanged = nil
		state.ProposedCount = 0
		summary := "I can't run destructive cleanup commands against your workspace. No file changes were made."
		outcome := a.buildImplementationSessionOutcome(msg, state, false)
		return summary, streamMsgID, false, nil, outcome, nil
	}

	if a.tryEarlyCorruptAppJSBootFix(sessionCtx, msg, wsPath, state) {
		state.Phase = "verify"
		verifyOut, verifyFailed, verifySkipped := a.runVerifyForState(sessionCtx, msg, state)
		state.VerifyOutput = verifyOut
		state.VerifyFailed = verifyFailed
		state.VerifySkipped = verifySkipped
		summary := a.formatImplementationSessionSummary("", state, true, msg)
		outcome := a.buildImplementationSessionOutcome(msg, state, true)
		return summary, streamMsgID, true, state.FilesChanged, outcome, nil
	}

	if a.tryEarlyCommandEvidencePlaybook(sessionCtx, msg, wsPath, state) {
		state.Phase = "verify"
		var verifyOut string
		var verifyFailed, verifySkipped bool
		// Re-running make start-all after adding the target launches dev servers and can
		// block until session timeout; the playbook already matched pasted failure evidence.
		if state.PlaybookUsedName == "missing_start_all_target" {
			verifySkipped = true
			state.VerifySkipped = true
		} else {
			verifyOut, verifyFailed, verifySkipped = a.runVerifyForState(sessionCtx, msg, state)
			state.VerifyOutput = verifyOut
			state.VerifyFailed = verifyFailed
			state.VerifySkipped = verifySkipped
		}
		proposed := state.hasRegisteredProposals() || state.ProposedCount > 0 || len(state.FilesChanged) > 0
		summary := a.formatImplementationSessionSummary("", state, proposed, msg)
		outcome := a.buildImplementationSessionOutcome(msg, state, proposed)
		return summary, streamMsgID, proposed, state.FilesChanged, outcome, nil
	}

	var repairNote string
	if state.FixLikeIntent && wsPath != "" && !state.DiagnosticBootstrapDone {
		bootstrapApplied, bootstrapNote := a.runReproBootstrap(sessionCtx, msg, state, wsPath)
		if bootstrapApplied {
			state.Phase = "verify"
			verifyOut, verifyFailed, verifySkipped := a.runReproVerify(sessionCtx, msg, state)
			state.VerifyOutput = verifyOut
			state.VerifyFailed = verifyFailed
			state.VerifySkipped = verifySkipped
			proposed := state.hasRegisteredProposals()
			summary := a.formatImplementationSessionSummary("", state, proposed, msg)
			outcome := a.buildImplementationSessionOutcome(msg, state, proposed)
			return summary, streamMsgID, proposed, state.RegisteredFiles, outcome, nil
		}
		repairNote = bootstrapNote
		a.sendInterimFixUpdate(msg, formatFixInterimProgress(state))
	}

	completeFixDiagnoseFromRepro(state)
	if state.FixLikeIntent && strings.TrimSpace(state.LastCommandOutput()) != "" {
		a.broadcastImplementationProgress(msg, streamMsgID, formatFixInterimProgress(state))
	}

	if a.tryEarlyMissingNpmModuleFix(sessionCtx, msg, wsPath, state) {
		a.runBootFixNpmInstallAfterDepProposal(sessionCtx, msg, state, wsPath)
		state.Phase = "verify"
		verifyOut, verifyFailed, verifySkipped := a.runReproVerify(sessionCtx, msg, state)
		state.VerifyOutput = verifyOut
		state.VerifyFailed = verifyFailed
		state.VerifySkipped = verifySkipped
		proposed := state.hasRegisteredProposals()
		summary := a.formatImplementationSessionSummary("", state, proposed, msg)
		outcome := a.buildImplementationSessionOutcome(msg, state, proposed)
		return summary, streamMsgID, proposed, state.RegisteredFiles, outcome, nil
	}

	var lastResponse string
	proposedAny := false

	if a.tryEarlyGoMathFixtureFix(sessionCtx, msg, wsPath, state) {
		state.Phase = "verify"
		verifyOut, verifyFailed, verifySkipped := a.runVerifyForState(sessionCtx, msg, state)
		state.VerifyOutput = verifyOut
		state.VerifyFailed = verifyFailed
		state.VerifySkipped = verifySkipped
		summary := a.formatImplementationSessionSummary("", state, true, msg)
		outcome := a.buildImplementationSessionOutcome(msg, state, true)
		return summary, streamMsgID, true, state.FilesChanged, outcome, nil
	}

	if a.tryEarlyTypeScriptCompileFix(sessionCtx, msg, wsPath, state) {
		state.Phase = "verify"
		verifyOut, verifyFailed, verifySkipped := a.runVerifyForState(sessionCtx, msg, state)
		state.VerifyOutput = verifyOut
		state.VerifyFailed = verifyFailed
		state.VerifySkipped = verifySkipped
		summary := a.formatImplementationSessionSummary("", state, true, msg)
		outcome := a.buildImplementationSessionOutcome(msg, state, true)
		return summary, streamMsgID, true, state.FilesChanged, outcome, nil
	}

	if a.tryEarlyGoMainFixtureFix(sessionCtx, msg, wsPath, state) {
		state.Phase = "verify"
		verifyOut, verifyFailed, verifySkipped := a.runVerifyForState(sessionCtx, msg, state)
		state.VerifyOutput = verifyOut
		state.VerifyFailed = verifyFailed
		state.VerifySkipped = verifySkipped
		proposed := state.hasRegisteredProposals() || len(state.FilesChanged) > 0
		summary := a.formatImplementationSessionSummary("", state, proposed, msg)
		outcome := a.buildImplementationSessionOutcome(msg, state, proposed)
		return summary, streamMsgID, proposed, state.FilesChanged, outcome, nil
	}

	if a.tryEarlyThemeCSSFix(sessionCtx, msg, wsPath, state) {
		state.Phase = "verify"
		verifyOut, verifyFailed, verifySkipped := a.runVerifyForState(sessionCtx, msg, state)
		state.VerifyOutput = verifyOut
		state.VerifyFailed = verifyFailed
		state.VerifySkipped = verifySkipped
		proposed := state.hasRegisteredProposals() || len(state.FilesChanged) > 0
		summary := a.formatImplementationSessionSummary("", state, proposed, msg)
		outcome := a.buildImplementationSessionOutcome(msg, state, proposed)
		return summary, streamMsgID, proposed, state.FilesChanged, outcome, nil
	}

	if a.tryEarlyThemeToggleFix(sessionCtx, msg, wsPath, state) {
		state.Phase = "verify"
		verifyOut, verifyFailed, verifySkipped := a.runVerifyForState(sessionCtx, msg, state)
		state.VerifyOutput = verifyOut
		state.VerifyFailed = verifyFailed
		state.VerifySkipped = verifySkipped
		proposed := state.hasRegisteredProposals() || len(state.FilesChanged) > 0
		summary := a.formatImplementationSessionSummary("", state, proposed, msg)
		outcome := a.buildImplementationSessionOutcome(msg, state, proposed)
		return summary, streamMsgID, proposed, state.FilesChanged, outcome, nil
	}

	toolModel := a.resolveImplementationToolModel("")
	if globalImplementationRouting != nil {
		plan, routedEff := globalImplementationRouting.Plan(sessionCtx, eff, a.Info, msg)
		toolModel = a.resolveImplementationToolModel(plan.ToolModel)
		if routedEff != nil {
			eff = routedEff
		}
	}
	sessionCtx = ai.WithImplementationToolModel(sessionCtx, toolModel)

	for fileCycle := 0; fileCycle < maxFiles; fileCycle++ {
		cycleProposed := false

	for round := 0; round < maxEditRounds; round++ {
		state.EditRound = round
		state.Phase = "discover"
		roundCtx := withImplementationSessionRound(sessionCtx, round)
		if repairNote != "" {
			roundCtx = withRepairNote(roundCtx, repairNote)
		}

		toolCtx := ai.WithToolStepObserver(roundCtx, func(ev ai.ToolStepEvent) {
			a.observeImplementationSessionToolStep(roundCtx, msg, state, streamMsgID, ev)
		})
		toolCtx = attachImplSessionCommandPolicy(toolCtx, state)

		if round == 0 && repairNote != "" {
			a.sendThinkingActivity(msg, protocol.ThinkingActivityReasoning, "planning fix from build output")
		}

		response, err := a.generateImplementationRound(toolCtx, msg, eff)
		if err != nil {
			if sessionCtx.Err() != nil {
				summary := a.formatImplementationSessionSummary(lastResponse, state, proposedAny, msg)
				outcome := a.buildImplementationSessionOutcome(msg, state, proposedAny)
				return summary, streamMsgID, proposedAny, state.FilesChanged, outcome, nil
			}
			outcome := a.buildImplementationSessionOutcome(msg, state, proposedAny)
			return "", streamMsgID, proposedAny, state.FilesChanged, outcome, err
		}
		lastResponse = response

		if state.diagnoseGateBlocksProposals() && responseContainsDiagnosis(response) {
			state.DiagnosePhaseComplete = true
			repairNote = formatDiagnoseCompleteRepairNote()
			if round < maxEditRounds-1 {
				continue
			}
		}
		if state.diagnoseGateBlocksProposals() && (fileChangeBlockRegex.MatchString(response) || strings.Contains(response, `"path"`)) {
			state.recordRepairFailureKind(RepairFailureGrounding)
			repairNote = formatDiagnoseRequiredRepairNote(state)
			if round < maxEditRounds-1 {
				continue
			}
		}

		proposalsBefore := state.ProposedCount
		cleaned, fileChangeProposed, propErr := a.maybeSubmitFileChangeFromResponse(toolCtx, response, msg.Channel, msg)
		toolProposed := state.ProposedCount > proposalsBefore

		if propErr != nil {
			log.Printf("[%s] impl session file proposal error: %v", a.Info.Name, propErr)
			state.recordRepairFailureKind(RepairFailurePreflight)
			repairNote = formatPreflightTypedRepairNote(state.PreflightErrors, state.StackManifest)
			if round < maxEditRounds-1 {
				continue
			}
		}

		if state.diagnoseGateBlocksProposals() && (toolProposed || fileChangeProposed) {
			state.ProposedCount = proposalsBefore
			state.recordRepairFailureKind(RepairFailureGrounding)
			repairNote = formatDiagnoseRequiredRepairNote(state)
			if round < maxEditRounds-1 {
				continue
			}
		}

		if toolProposed || fileChangeProposed {
			proposedAny = true
			cycleProposed = true
			state.Phase = "edit"
			paths := extractChangedPathsFromResponse(response)
			state.FilesChanged = appendUnique(state.FilesChanged, paths)
			if toolProposed && len(paths) == 0 {
				// paths recorded in executeProposeFileEditTool via FilesChanged
			}
			lastResponse = cleaned
			break
		}

		if round == 0 && state.groundingSatisfied() {
			if ok, paths := a.attemptDeterministicImplementationFallback(roundCtx, msg); ok {
				proposedAny = true
				cycleProposed = true
				state.FilesChanged = appendUnique(state.FilesChanged, paths)
				lastResponse = ""
				break
			}
		}

		if round < maxEditRounds-1 {
			if reject, note := shouldRejectPrematureStop(a, msg, state, cycleProposed, round, maxEditRounds); reject {
				notePrematureStopAttempt(state)
				state.recordRepairFailureKind(RepairFailureAdvisory)
				repairNote = note
			} else if responseClaimsPrematureDone(response) || responseIsAdvisoryOnly(response) {
				notePrematureStopAttempt(state)
				state.recordRepairFailureKind(RepairFailureAdvisory)
				if state.VerifyFailed {
					repairNote = formatAntiStopRepairNote("Verification still fails — continue with concrete file edits.")
				} else if state.diagnoseGateBlocksProposals() {
					repairNote = formatDiagnoseRequiredRepairNote(state)
				} else {
					repairNote = formatAdvisoryOnlyRepairNote()
				}
			} else if len(state.PreflightErrors) > 0 {
				state.recordRepairFailureKind(RepairFailurePreflight)
				repairNote = formatPreflightTypedRepairNote(state.PreflightErrors, state.StackManifest)
			} else if !state.groundingSatisfied() {
				state.recordRepairFailureKind(RepairFailureGrounding)
				repairNote = formatGroundingRepairNote("")
			} else if state.FixLikeIntent {
				repairNote = fixLikePostDiscoverRepairNote(state)
			} else {
				state.recordRepairFailureKind(RepairFailureAdvisory)
				repairNote = formatAdvisoryOnlyRepairNote()
			}
		}
	}

	if !cycleProposed && strings.TrimSpace(lastResponse) != "" {
		fallbackCtx := withImplementationSessionState(sessionCtx, state)
		cleaned, ok, ferr := a.maybeSubmitFileChangeFromResponse(fallbackCtx, lastResponse, msg.Channel, msg)
		if ferr != nil {
			log.Printf("[%s] impl session end fallback error: %v", a.Info.Name, ferr)
		}
		if ok {
			proposedAny = true
			cycleProposed = true
			lastResponse = cleaned
			paths := extractChangedPathsFromResponse(lastResponse)
			state.FilesChanged = appendUnique(state.FilesChanged, paths)
		}
	}
	if !cycleProposed {
		if ok, paths := a.attemptDeterministicImplementationFallback(sessionCtx, msg); ok {
			proposedAny = true
			cycleProposed = true
			state.FilesChanged = appendUnique(state.FilesChanged, paths)
		}
	}
	if !cycleProposed {
		if ok, paths := a.attemptPlaybookForSessionState(sessionCtx, msg, state); ok {
			proposedAny = true
			cycleProposed = true
			state.FilesChanged = appendUnique(state.FilesChanged, paths)
		}
	}
	state.noteCommandOnlyRound(cycleProposed)
	if !cycleProposed && state.commandThrashingDetected() {
		if ok, paths := a.attemptPlaybookForSessionState(sessionCtx, msg, state); ok {
			proposedAny = true
			cycleProposed = true
			state.FilesChanged = appendUnique(state.FilesChanged, paths)
		} else {
			break
		}
	}
	a.repairTailwindDarkModeIfNeeded(sessionCtx, msg, state)
	a.repairAppThemeIfNeeded(sessionCtx, msg, state)
	a.repairCorruptAppJSEntryIfNeeded(sessionCtx, msg, state)
	if !cycleProposed {
		for _, p := range state.FilesChanged {
			if strings.Contains(strings.ToLower(p), "tailwind.config") || strings.HasSuffix(strings.ToLower(p), "app.tsx") {
				cycleProposed = true
				proposedAny = true
				break
			}
		}
	}

	if cycleProposed {
		a.maybeSendImplementationEarlyReply(msg, state)
		state.Phase = "verify"
		verifyOut, verifyFailed, verifySkipped := a.runVerifyForState(sessionCtx, msg, state)
		state.VerifyOutput = verifyOut
		state.VerifyFailed = verifyFailed
		state.VerifySkipped = verifySkipped

		if verifyFailed {
			if a.shouldSkipVerifyRepairAfterAutoApply(msg, state) {
				log.Printf("[%s] skipping verify repair after auto-applied edit (deterministic or Go-only)", a.Info.Name)
			} else if state.FixLikeIntent {
				a.sendInterimFixUpdate(msg, "Repro still failing — continuing with fixes…")
			}
			maxRepairs := 1
			if agentRuntimeV2ForMessage(msg) {
				maxRepairs = agentRuntimeMaxRepairRounds
			}
			if !a.shouldSkipVerifyRepairAfterAutoApply(msg, state) && state.RepairAttempts < maxRepairs {
			state.RepairAttempts++
			state.RepairUsed = state.RepairAttempts > 0
			state.Phase = "repair"
			verifyInfo := classifyVerifyFailure(verifyOut, detectVerifyCommandsForSession(wsPath, state, msg))
			state.LastVerifyFailureKind = verifyInfo.Kind
			state.recordRepairFailureKind(verifyInfo.Kind)
			repairNote = formatVerifyRepairNote(verifyInfo, verifyOut)
			sessionCtx = ContextWithImplementationRoutingHints(sessionCtx, ImplementationRoutingHints{
				RepairAttempts: state.RepairAttempts,
				VerifyFailed:   true,
				BootFixIntent:  state.BootFixIntent,
			})
			if globalImplementationRouting != nil {
				plan, repairEff := globalImplementationRouting.Plan(sessionCtx, eff, a.Info, msg)
				if repairEff != nil {
					eff = repairEff
				}
				toolModel := a.resolveImplementationToolModel(plan.ToolModel)
				sessionCtx = ai.WithImplementationToolModel(sessionCtx, toolModel)
			}
			roundCtx := withImplementationSessionRound(sessionCtx, state.EditRound+1)
			roundCtx = withRepairNote(roundCtx, repairNote)
			roundCtx = ai.WithToolLoopMaxIterations(roundCtx, maxToolIter)
			toolCtx := ai.WithToolStepObserver(roundCtx, func(ev ai.ToolStepEvent) {
				a.observeImplementationSessionToolStep(roundCtx, msg, state, streamMsgID, ev)
			})
			toolCtx = attachImplSessionCommandPolicy(toolCtx, state)
			response, err := a.generateImplementationRound(toolCtx, msg, eff)
			if err == nil {
				proposalsBefore := state.ProposedCount
				cleaned, proposed, propErr := a.maybeSubmitFileChangeFromResponse(toolCtx, response, msg.Channel, msg)
				if propErr != nil {
					log.Printf("[%s] impl session repair proposal error: %v", a.Info.Name, propErr)
				}
				if proposed || state.ProposedCount > proposalsBefore {
					proposedAny = true
					lastResponse = cleaned
					verifyOut2, verifyFailed2, _ := a.runVerifyForState(sessionCtx, msg, state)
					state.VerifyOutput = verifyOut2
					state.VerifyFailed = verifyFailed2
				} else {
					lastResponse = response
				}
			}
			}
		}
	}

	if !cycleProposed && !proposedAny {
		if reject, note := shouldRejectPrematureStop(a, msg, state, cycleProposed, state.EditRound, maxEditRounds); reject {
			notePrematureStopAttempt(state)
			state.recordRepairFailureKind(RepairFailureAdvisory)
			repairNote = note
			state.Phase = "discover"
			continue
		}
		break
	}
	if msg != nil && (msg.IdeEditorModeIsExport() || userRequestsFileExportForMessage(msg)) && proposedAny {
		break
	}
	if cont, note := shouldContinueImplementationSession(a, msg, state); cont {
		repairNote = note
		state.Phase = "discover"
		persistImplSessionCheckpoint(msg, state, fileCycle)
		continue
	}
	break
	} // fileCycle

	persistImplSessionCheckpoint(msg, state, -1)

	proposedAny = state.hasRegisteredProposals()
	summary := a.formatImplementationSessionSummary(lastResponse, state, proposedAny, msg)
	outcome := a.buildImplementationSessionOutcome(msg, state, proposedAny)
	return summary, streamMsgID, proposedAny, state.RegisteredFiles, outcome, nil
}

func (a *Agent) buildImplementationSessionOutcome(msg *protocol.Message, state *ImplementationSessionState, proposed bool) map[string]interface{} {
	outcome := map[string]interface{}{
		"repair_used":    false,
		"verify_failed":  false,
		"verify_skipped": false,
		"files_changed":  []string{},
		"outcome":        "no_changes",
	}
	if state == nil {
		return outcome
	}
	outcome["repair_used"] = state.RepairUsed
	if state.RepairAttempts > 0 {
		outcome["repair_attempts"] = state.RepairAttempts
	}
	outcome["verify_failed"] = state.VerifyFailed
	outcome["verify_skipped"] = state.VerifySkipped
	if len(state.RegisteredFiles) > 0 {
		outcome["files_changed"] = append([]string(nil), state.RegisteredFiles...)
	} else if len(state.FilesChanged) > 0 {
		outcome["files_changed"] = append([]string(nil), state.FilesChanged...)
	}
	trust := ""
	if msg != nil {
		trust = msg.EditorAgentTrust()
	}
	if state.TrustMode != "" {
		trust = state.TrustMode
	}
	applied := len(state.RegisteredFiles) > 0 && trust == editorTrustAutoApply && msg != nil &&
		sessionFilesOnDisk(a.resolveWorkspacePath(msg), state.RegisteredFiles)
	reproVerified := fixLikeSessionSucceeded(state)
	attempted := state.ProposedCount > 0 || len(state.FilesChanged) > 0
	registered := len(state.RegisteredFiles) > 0
	switch {
	case attempted && !registered && len(state.RegistrationErrors) > 0:
		outcome["outcome"] = "proposal_registration_failed"
		outcome["registration_errors"] = append([]string(nil), state.RegistrationErrors...)
	case !proposed:
		outcome["outcome"] = "no_changes"
	case trust == editorTrustAutoApply && applied && state.FixLikeIntent && strings.TrimSpace(state.ReproCommand) != "" && !reproVerified:
		outcome["outcome"] = "applied_verify_failed"
	case trust == editorTrustAutoApply && applied && state.VerifyFailed:
		outcome["outcome"] = "applied_verify_failed"
	case trust == editorTrustAutoApply && applied && state.FixLikeIntent && strings.TrimSpace(state.ReproCommand) != "" && reproVerified:
		outcome["outcome"] = "applied_and_verified"
	case trust == editorTrustAutoApply && applied && !state.VerifyFailed && !state.VerifySkipped:
		outcome["outcome"] = "applied_and_verified"
	case proposed:
		outcome["outcome"] = "proposals_submitted"
	}
	if state.FixLikeIntent && strings.TrimSpace(state.ReproCommand) != "" {
		outcome["repro_command"] = state.ReproCommand
		outcome["repro_exit_code"] = state.ReproExitCode
	}
	if len(state.CommandHistory) > 0 {
		failures := state.CommandFailureSummary()
		if len(failures) > 0 {
			rows := make([]map[string]interface{}, 0, len(failures))
			for _, f := range failures {
				rows = append(rows, map[string]interface{}{
					"cmd":   f.Command,
					"count": f.Count,
				})
			}
			outcome["command_failures"] = rows
		}
	}
	if state.PlaybookUsedName != "" {
		outcome["playbook_used"] = state.PlaybookUsedName
	}
	outcome["circuit_breaker_triggered"] = state.CircuitBreakerFired
	if ft := state.failureTypeForOutcome(); ft != "" {
		outcome["failure_type"] = ft
	}
	if state.PrematureStopAttempts > 0 {
		outcome["premature_stop_pushes"] = state.PrematureStopAttempts
	}
	if state.DiagnosePhaseRequired {
		outcome["diagnose_gate_required"] = true
		outcome["diagnose_gate_complete"] = state.DiagnosePhaseComplete
	}
	if state.DialogueSummary != "" {
		outcome["dialogue_summary_used"] = true
	}
	if snap := a.LastRoutingSnapshot(); snap.Reason != "" {
		outcome["routing_reason"] = snap.Reason
		if snap.ToolModel != "" {
			outcome["routing_tool_model"] = snap.ToolModel
		}
	}
	return outcome
}

type implRepairNoteKey struct{}

func withRepairNote(ctx context.Context, note string) context.Context {
	return context.WithValue(ctx, implRepairNoteKey{}, note)
}

func repairNoteFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	n, _ := ctx.Value(implRepairNoteKey{}).(string)
	return n
}

func (a *Agent) generateImplementationRound(ctx context.Context, msg *protocol.Message, eff ai.AIProvider) (string, error) {
	intent := a.classifyTurnIntentForMessage(msg)
	prompt := a.buildPromptForIntent(msg, intent)
	prompt = a.appendDelegationContext(ctx, msg, prompt)
	prompt = a.appendRepoConsultContext(ctx, msg, prompt, intent)

	if note := repairNoteFromContext(ctx); note != "" {
		prompt += "\n=== REPAIR REQUIRED ===\n" + note + "\n"
	}
	if st := implementationSessionStateFromContext(ctx); st != nil {
		prompt = appendDialogueSummaryPrompt(prompt, st)
		if st.diagnoseGateBlocksProposals() {
			prompt += "\n=== DIAGNOSE BEFORE EDIT ===\nProvide Analysis and Planned edits before any file proposals.\n"
		}
	}
	var sessionGuidance strings.Builder
	appendImplementationSessionToolGuidance(&sessionGuidance, a, msg)
	prompt += sessionGuidance.String()

		if round := implementationSessionRoundFromContext(ctx); round == 0 {
		wsPath := a.resolveWorkspacePath(msg)
		if wsPath != "" && a.shouldAugmentPromptWithWorkspace(intent, msg) {
			if st := implementationSessionStateFromContext(ctx); st != nil && st.StackManifest != nil {
				prompt += st.StackManifest.FormatPromptBlock()
				seedContent := msg.Content
				if userAffirmsPendingImplementation(msg.Content) {
					for i := len(a.channelHistory(msg.Channel)) - 1; i >= 0; i-- {
						m := a.channelHistory(msg.Channel)[i]
						if m == nil || m.ID == msg.ID {
							continue
						}
						if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
							seedContent = m.Content
							break
						}
					}
				}
				if rem := remainingImplementationTargets(wsPath, st.StackManifest, seedContent); len(rem) > 0 {
					prompt += "\n=== REQUIRED FILES (ship all in this session) ===\n"
					for _, rel := range rem {
						prompt += "- " + rel + "\n"
					}
					if len(rem) > 1 {
						prompt += "Emit one [FILE_CHANGE] block (or propose_file_edit) per file above — partial delivery is not acceptable.\n"
					}
					prompt += st.StackManifest.FormatRepairHints()
				}
			}
			var referencedFiles strings.Builder
			seedMsg := msg
			if userAffirmsPendingImplementation(msg.Content) {
				for i := len(a.channelHistory(msg.Channel)) - 1; i >= 0; i-- {
					m := a.channelHistory(msg.Channel)[i]
					if m == nil || m.ID == msg.ID {
						continue
					}
					if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
						seedMsg = m
						AppendReferencedFiles(&referencedFiles, m.Content, wsPath)
						break
					}
				}
			} else {
				AppendReferencedFiles(&referencedFiles, msg.Content, wsPath)
			}
			seeds := AppendImplementationSeedFiles(&referencedFiles, a, seedMsg, wsPath, a.Info.Type, collectIncludedFilePaths(msg))
			if st := implementationSessionStateFromContext(ctx); st != nil {
				st.SeedsLoaded += seeds
			}
			prompt += referencedFiles.String()
		}
	} else {
		prompt += "\n=== IMPLEMENTATION SESSION ===\nUse grep, glob_file_search, semantic_search, and read_file to find code. Do not guess file contents.\n\n"
	}

	history := a.conversationHistoryForIntent(msg, intent)
	prompt = appendPriorReferenceGuidance(prompt, msg, history, a.Info.ID)
	prompt, _ = applyContextBudgetForMessage(msg, prompt)

	approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
	eff = a.toolCapableProvider(approvalCtx, eff)
	if len(a.agentToolDefinitions(msg)) > 0 {
		return a.generateWithAgentTools(approvalCtx, msg, prompt, history, eff)
	}
	return eff.GenerateResponse(approvalCtx, prompt, historyToMessages(history))
}

func (a *Agent) runImplementationVerify(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState) (output string, failed bool, skipped bool) {
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath == "" {
		return "", false, true
	}
	cmds := detectVerifyCommandsForSession(wsPath, state, msg)
	if len(cmds) == 0 {
		return "", false, true
	}
	a.sendThinkingActivity(msg, protocol.ThinkingActivityVerifying, strings.Join(cmds, "; "))
	mcpServer := mcpServerFromInterface(a.MCPServer)
	if mcpServer == nil {
		return "", false, true
	}

	var combined strings.Builder
	anyFailed := false
	for _, cmd := range cmds {
		toolCtx := a.contextWithWorkspaceBackend(ctx, msg)
		toolCtx = shared.ContextWithImplementationSession(toolCtx, true)
		var cancel context.CancelFunc
		if d := verifyCommandTimeoutForMessage(msg); d > 0 {
			toolCtx, cancel = context.WithTimeout(toolCtx, d)
		}
		input, _ := json.Marshal(map[string]string{"command": cmd})
		result, err := executeMCPTool(toolCtx, mcpServer, "run_command", input)
		if cancel != nil {
			cancel()
		}
		if combined.Len() > 0 {
			combined.WriteString("\n---\n")
		}
		combined.WriteString("$ ")
		combined.WriteString(cmd)
		combined.WriteString("\n")
		if err != nil {
			combined.WriteString(err.Error())
			anyFailed = true
			break
		}
		combined.WriteString(result)
		if verifyCommandResultFailed(result) {
			anyFailed = true
			break
		}
	}
	return combined.String(), anyFailed, false
}

func verifyCommandResultFailed(result string) bool {
	result = strings.TrimSpace(result)
	if result == "" {
		return false
	}
	if strings.HasPrefix(result, "ERROR:") {
		return true
	}
	lower := strings.ToLower(result)
	if strings.Contains(lower, "not allowlisted") {
		return true
	}
	return strings.Contains(result, "exit_code=") && !strings.Contains(result, "exit_code=0")
}

const implVerifyCommandTimeout = 120 * time.Second

func verifyCommandTimeoutForMessage(msg *protocol.Message) time.Duration {
	if msg != nil && strings.TrimSpace(msg.Channel) == "implement-scenarios" {
		return implVerifyCommandTimeout
	}
	return 0
}

func detectVerifyCommands(wsPath string) []string {
	return detectVerifyCommandsForSession(wsPath, nil, nil)
}

func detectVerifyCommandsForSession(wsPath string, state *ImplementationSessionState, msg *protocol.Message) []string {
	var hintPaths []string
	userContent := ""
	if state != nil {
		hintPaths = append(hintPaths, state.RegisteredFiles...)
		hintPaths = append(hintPaths, state.FilesChanged...)
	}
	if msg != nil {
		userContent = msg.Content
	}
	if goCmds := detectGoBuildVerifyCommands(wsPath, hintPaths, userContent); len(goCmds) > 0 {
		return goCmds
	}
	if _, err := os.Stat(filepath.Join(wsPath, "go.mod")); err == nil {
		return []string{"go test ./..."}
	}
	if _, err := os.Stat(filepath.Join(wsPath, "Cargo.toml")); err == nil {
		return []string{"cargo test"}
	}
	if _, err := os.Stat(filepath.Join(wsPath, "package.json")); err == nil {
		return detectNodeVerifyCommands(wsPath)
	}
	if _, err := os.Stat(filepath.Join(wsPath, "pyproject.toml")); err == nil {
		return []string{"python -m pytest -q"}
	}
	if _, err := os.Stat(filepath.Join(wsPath, "requirements.txt")); err == nil {
		return []string{"python -m pytest -q"}
	}
	if matches, _ := filepath.Glob(filepath.Join(wsPath, "*.tf")); len(matches) > 0 {
		return []string{"terraform validate"}
	}
	return nil
}

func detectGoBuildVerifyCommands(wsPath string, hintPaths []string, userContent string) []string {
	if _, err := os.Stat(filepath.Join(wsPath, "go.mod")); err == nil {
		return nil
	}
	goPaths := collectGoVerifyHintPaths(hintPaths, userContent)
	if len(goPaths) == 0 {
		return nil
	}
	pkgDirs := uniqueGoPackageDirs(goPaths)
	cmds := make([]string, 0, len(pkgDirs))
	for _, pkg := range pkgDirs {
		if pkg == "." {
			cmds = append(cmds, "go build .")
		} else {
			cmds = append(cmds, "go build ./"+pkg)
		}
	}
	return cmds
}

func collectGoVerifyHintPaths(hintPaths []string, userContent string) []string {
	var out []string
	for _, p := range hintPaths {
		p = normalizeFileChangeRelPath(p)
		if strings.HasSuffix(strings.ToLower(p), ".go") {
			out = appendUnique(out, []string{p})
		}
	}
	if len(out) > 0 {
		return out
	}
	for _, m := range goPathInContentRE.FindAllString(userContent, -1) {
		out = appendUnique(out, []string{normalizeFileChangeRelPath(m)})
	}
	return out
}

var goPathInContentRE = regexp.MustCompile(`(?i)(?:[\w.-]+/)*[\w.-]+\.go`)

func uniqueGoPackageDirs(goFiles []string) []string {
	seen := map[string]bool{}
	var dirs []string
	for _, f := range goFiles {
		dir := filepath.ToSlash(filepath.Dir(f))
		if dir == "." || dir == "" {
			dir = "."
		}
		if seen[dir] {
			continue
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	return dirs
}

func (a *Agent) shouldSkipVerifyRepairAfterAutoApply(msg *protocol.Message, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || !state.VerifyFailed {
		return false
	}
	trust := resolveImplementationTrustMode(msg)
	if state.TrustMode != "" {
		trust = state.TrustMode
	}
	if trust != editorTrustAutoApply {
		return false
	}
	paths := state.RegisteredFiles
	if len(paths) == 0 {
		paths = state.FilesChanged
	}
	if !sessionFilesOnDisk(a.resolveWorkspacePath(msg), paths) {
		return false
	}
	if state.DeterministicFallbackUsed {
		return true
	}
	return !state.FixLikeIntent && allPathsGoSource(paths)
}

func allPathsGoSource(paths []string) bool {
	if len(paths) == 0 {
		return false
	}
	for _, p := range paths {
		if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(p)), ".go") {
			return false
		}
	}
	return true
}

func (a *Agent) maybeSendImplementationEarlyReply(msg *protocol.Message, state *ImplementationSessionState) {
	if a == nil || msg == nil || state == nil || a.Hub == nil {
		return
	}
	trust := resolveImplementationTrustMode(msg)
	if state.TrustMode != "" {
		trust = state.TrustMode
	}
	if trust != editorTrustAutoApply {
		return
	}
	paths := state.RegisteredFiles
	if len(paths) == 0 {
		paths = state.FilesChanged
	}
	if len(paths) == 0 || !sessionFilesOnDisk(a.resolveWorkspacePath(msg), paths) {
		return
	}
	text := fmt.Sprintf("Implementation session — applied changes (changes to: %s); verifying workspace…", strings.Join(paths, ", "))
	a.sendInterimFixUpdate(msg, text)
}

func detectNodeVerifyCommands(wsPath string) []string {
	nodeModules := filepath.Join(wsPath, "node_modules")
	if _, err := os.Stat(nodeModules); err != nil {
		if cmd := shared.TypeScriptCheckShellCommand(wsPath); cmd != "" {
			return []string{cmd}
		}
		return nil
	}
	var cmds []string
	if hasPackageScript(wsPath, "build") {
		cmds = append(cmds, "npm run build")
	} else if cmd := shared.TypeScriptCheckShellCommand(wsPath); cmd != "" {
		cmds = append(cmds, cmd)
	}
	if hasPackageScript(wsPath, "typecheck") {
		cmds = append(cmds, "npm run typecheck")
	}
	cmds = append(cmds, "CI=true npm test --if-present -- --watchAll=false --passWithNoTests")
	return cmds
}

func hasPackageScript(wsPath, name string) bool {
	b, err := os.ReadFile(filepath.Join(wsPath, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return false
	}
	_, ok := pkg.Scripts[name]
	return ok
}

func sessionFilesOnDisk(wsPath string, paths []string) bool {
	wsPath = strings.TrimSpace(wsPath)
	if wsPath == "" || len(paths) == 0 {
		return false
	}
	for _, rel := range paths {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			return false
		}
		info, err := os.Stat(filepath.Join(wsPath, rel))
		if err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

func (a *Agent) formatImplementationSessionSummary(lastResponse string, state *ImplementationSessionState, proposed bool, msg *protocol.Message) string {
	var b strings.Builder
	trust := ""
	if msg != nil {
		trust = msg.EditorAgentTrust()
	}
	if state != nil && state.TrustMode != "" {
		trust = state.TrustMode
	}
	applied := false
	if proposed && msg != nil && trust == editorTrustAutoApply {
		paths := state.RegisteredFiles
		if len(paths) == 0 {
			paths = state.FilesChanged
		}
		applied = sessionFilesOnDisk(a.resolveWorkspacePath(msg), paths)
	}
	reproVerified := fixLikeSessionSucceeded(state)
	fixLikeRepro := state != nil && state.FixLikeIntent && strings.TrimSpace(state.ReproCommand) != ""

	switch {
	case !proposed:
		b.WriteString("Implementation session finished without file changes.\n\n")
		if state != nil {
			if line := state.formatCommandFailureSummaryLine(); line != "" {
				b.WriteString(line)
				b.WriteString("\n\n")
			}
			if len(state.RegistrationErrors) > 0 {
				b.WriteString("File change proposals were not registered:\n")
				for _, e := range state.RegistrationErrors {
					b.WriteString("- ")
					b.WriteString(e)
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}
		}
	case trust == editorTrustAutoApply && state != nil && applied && fixLikeRepro && !reproVerified:
		if len(state.FilesChanged) > 0 {
			b.WriteString(fmt.Sprintf("Fix session complete — applied changes but repro `%s` still fails (changes to: %s).\n\n", state.ReproCommand, strings.Join(state.FilesChanged, ", ")))
		} else {
			b.WriteString(fmt.Sprintf("Fix session complete — applied changes but repro `%s` still fails.\n\n", state.ReproCommand))
		}
	case trust == editorTrustAutoApply && state != nil && applied && fixLikeRepro && reproVerified:
		if len(state.FilesChanged) > 0 {
			b.WriteString(fmt.Sprintf("Fix session complete — repro `%s` passes (changes to: %s).\n\n", state.ReproCommand, strings.Join(state.FilesChanged, ", ")))
		} else {
			b.WriteString(fmt.Sprintf("Fix session complete — repro `%s` passes.\n\n", state.ReproCommand))
		}
	case trust == editorTrustAutoApply && state != nil && applied && state.VerifyFailed:
		if len(state.FilesChanged) > 0 {
			b.WriteString(fmt.Sprintf("Implementation session complete — applied but verification failed (changes to: %s).\n\n", strings.Join(state.FilesChanged, ", ")))
		} else {
			b.WriteString("Implementation session complete — applied but verification failed.\n\n")
		}
	case trust == editorTrustAutoApply && state != nil && applied && !state.VerifyFailed && !state.VerifySkipped:
		if len(state.FilesChanged) > 0 {
			b.WriteString(fmt.Sprintf("Implementation session complete — applied and verified (changes to: %s).\n\n", strings.Join(state.FilesChanged, ", ")))
		} else {
			b.WriteString("Implementation session complete — applied and verified.\n\n")
		}
	case proposed && state != nil && len(state.FilesChanged) > 0 && state.VerifyFailed:
		b.WriteString(fmt.Sprintf("Implementation session complete — proposals submitted (changes to: %s); verification failed on current workspace.\n\n", strings.Join(state.FilesChanged, ", ")))
	case proposed && state != nil && len(state.FilesChanged) > 0 && state.VerifyOutput != "" && !state.VerifyFailed && !state.VerifySkipped:
		b.WriteString(fmt.Sprintf("Implementation session complete — proposals submitted and workspace verifies clean (changes to: %s).\n\n", strings.Join(state.FilesChanged, ", ")))
	case proposed && state != nil && len(state.FilesChanged) > 0:
		b.WriteString(fmt.Sprintf("Implementation session complete — proposals submitted for approval (changes to: %s).\n\n", strings.Join(state.FilesChanged, ", ")))
	default:
		b.WriteString("Implementation session complete — proposals submitted for approval.\n\n")
	}

	if state != nil && len(state.PreflightErrors) > 0 && !proposed {
		b.WriteString("Preflight issues encountered:\n")
		for _, e := range state.PreflightErrors {
			b.WriteString("- ")
			b.WriteString(e)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if state != nil && state.VerifyOutput != "" {
		b.WriteString("Verification:\n```\n")
		b.WriteString(truncateImplLog(state.VerifyOutput, 2000))
		b.WriteString("\n```\n\n")
	} else if state != nil && state.VerifySkipped && proposed {
		b.WriteString("Verification skipped (interactive trust — approve proposals to apply changes).\n\n")
	}
	body := sanitizeFailedImplementationResponse(lastResponse, state)
	b.WriteString(body)
	return strings.TrimSpace(b.String())
}

func looksLikeListDirToolEcho(content string) bool {
	trim := strings.TrimSpace(content)
	if trim == "" {
		return false
	}
	lines := strings.Split(trim, "\n")
	dirLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, " (file)") || strings.HasSuffix(line, " (dir)") {
			dirLines++
		}
	}
	return dirLines >= 3 && dirLines >= len(lines)/2
}

func sanitizeFailedImplementationResponse(lastResponse string, state *ImplementationSessionState) string {
	trim := strings.TrimSpace(lastResponse)
	if state == nil || state.SeedsLoaded < 1 {
		return trim
	}
	lower := strings.ToLower(trim)
	if trim == "No matches found." || (strings.Contains(lower, "no matches found") && len(trim) < 120) {
		return "Workspace files were already loaded for this session, but search found no matches. " +
			"Use read_file on paths from the error log (e.g. src/App.js, src/main.tsx) and emit [FILE_CHANGE] fixes."
	}
	if looksLikeListDirToolEcho(trim) {
		return "Workspace files were already loaded for this session. " +
			"Do not echo list_dir output — read the seeded files and emit [FILE_CHANGE] fixes for the boot error."
	}
	asksForPaste := strings.Contains(lower, "please provide") ||
		strings.Contains(lower, "could you please share") ||
		strings.Contains(lower, "paste the content") ||
		strings.Contains(lower, "share the content of") ||
		strings.Contains(lower, "please share the content")
	if !asksForPaste {
		return trim
	}
	return "Workspace files were already loaded for this session, but I could not produce [FILE_CHANGE] proposals. " +
		"Diagnose using src/main.tsx, src/App.tsx, index.html, and src-tauri/tauri.conf.json, then emit concrete fixes via [FILE_CHANGE] or propose_file_edit."
}

// resolveImplementationToolModel prefers the agent's dedicated coder tag for tool loops
// (e.g. qwen2.5-coder:14b). General qwen3.5:27b specialists use the configured tool model.
func (a *Agent) resolveImplementationToolModel(planToolModel string) string {
	if a != nil {
		if agentModel := strings.TrimSpace(a.Info.AIModel); agentModel != "" {
			lower := strings.ToLower(agentModel)
			if strings.Contains(lower, "qwen2.5-coder") ||
				strings.Contains(lower, "qwen3-coder") ||
				strings.Contains(lower, "qwen3.5-coder") ||
				strings.Contains(lower, "codestral") {
				return agentModel
			}
		}
	}
	if m := strings.TrimSpace(planToolModel); m != "" {
		return m
	}
	return "qwen3.5:9b"
}

func truncateImplLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...(truncated)"
}

func extractChangedPathsFromResponse(response string) []string {
	var paths []string
	matches := fileChangeBlockRegex.FindAllStringSubmatch(response, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		if d, err := parseFileChangeDirective(m[1]); err == nil && d.Path != "" {
			paths = append(paths, d.Path)
		}
	}
	return paths
}

func appendUnique(dst []string, add []string) []string {
	seen := make(map[string]bool, len(dst))
	for _, p := range dst {
		seen[p] = true
	}
	for _, p := range add {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		dst = append(dst, p)
	}
	return dst
}

func appendImplementationSessionToolGuidance(prompt *strings.Builder, a *Agent, msg *protocol.Message) {
	if a == nil || prompt == nil {
		return
	}
	prompt.WriteString("\n=== IMPLEMENTATION SESSION (required delivery) ===\n")
	prompt.WriteString("Ship working file changes in this turn — advice-only is not acceptable.\n")
	if a.hasWorkspaceTools() {
		prompt.WriteString("Prefer search_replace (exact unique old_string) or apply_patch for edits; propose_file_edit for new files or full small-file rewrites.\n")
	}
	prompt.WriteString("Alternatively emit one or more [FILE_CHANGE] blocks with real relative paths and full file content.\n")
	prompt.WriteString("Do NOT re-plan or ask design questions when the user already approved or said to proceed — ship file changes in this turn.\n")
	if msg != nil {
		applied := channelRecentlyAppliedFilePaths(a.channelHistory(msg.Channel), msg.ID, a.Info.ID)
		if len(applied) > 0 {
			prompt.WriteString(fmt.Sprintf(
				"Already applied (do NOT re-propose): %s — edit the next required file instead.\n",
				strings.Join(applied, ", "),
			))
		}
		if userAffirmsPendingImplementation(msg.Content) {
			prompt.WriteString(
				"User affirmed continuation — ship the NEXT file change for the prior implementation task. " +
					"Do not re-propose already-applied paths or reply with advice only.\n",
			)
		}
	}
	appendFileChangeMachineBlockDocs(prompt)
	if a != nil && msg != nil {
		history := a.channelHistorySafe(msg.Channel)
		if messageImpliesBootFix(msg.Content, history) || messageHasBootOrBuildError(msg.Content) {
			prompt.WriteString("Boot-fix: read Makefile, package.json, and scripts/start-all.sh with read_file before running make start-all or npm run dev.\n")
		}
	}
}

const proposeFileEditToolName = "propose_file_edit"

func proposeFileEditToolDefinition() ai.ClaudeToolDefinition {
	schema := json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Relative file path under workspace root"},"content":{"type":"string","description":"Full new file content"},"operation":{"type":"string","description":"create, edit, or delete"}},"required":["path","content"]}`)
	return ai.ClaudeToolDefinition{
		Name:        proposeFileEditToolName,
		Description: "Propose a file create or edit in the shared workspace (submitted for approval)",
		InputSchema: schema,
	}
}

func (a *Agent) executeProposeFileEditTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	if isAskModeReadOnly(msg) {
		return "", fmt.Errorf("ask mode is read-only")
	}
	var args struct {
		Path      string `json:"path"`
		Content   string `json:"content"`
		Operation string `json:"operation"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid propose_file_edit input: %w", err)
		}
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	op := strings.ToLower(strings.TrimSpace(args.Operation))
	if op == "delete" {
		return "", fmt.Errorf("delete not supported via propose_file_edit in v1")
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	content := a.substituteFileExportContent(msg, args.Content)
	if userPath := preferFileExportTargetPath(msg); userPath != "" {
		path = userPath
	}
	var err error
	err = a.proposeFileChangePreferEditOrCreate(ctx, channel, path, content, msg)
	if err != nil {
		return "", err
	}
	if st := implementationSessionStateFromContext(ctx); st != nil {
		st.ProposedCount++
		st.FilesChanged = appendUnique(st.FilesChanged, []string{path})
		st.RecordEdit(path)
	}
	return fmt.Sprintf(`{"status":"proposed","path":%q}`, path), nil
}
