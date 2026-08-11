package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ActionEvidence is an authoritative outcome observed during this turn.
type ActionEvidence struct {
	Kind     EvidenceKind `json:"kind"`
	Tool     string       `json:"tool,omitempty"`
	Status   string       `json:"status"`
	ExitCode *int         `json:"exit_code,omitempty"`
	Detail   string       `json:"detail,omitempty"`
}

// ActionEvidenceLedger is intentionally per-turn and persistence-neutral.
type ActionEvidenceLedger struct {
	mu      sync.Mutex
	entries []ActionEvidence
}

func (l *ActionEvidenceLedger) Record(e ActionEvidence) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, e)
}

func (l *ActionEvidenceLedger) Entries() []ActionEvidence {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]ActionEvidence(nil), l.entries...)
}

func (l *ActionEvidenceLedger) Has(kind EvidenceKind) bool {
	for _, e := range l.Entries() {
		if e.Kind == kind && e.Status == "succeeded" {
			return true
		}
	}
	return false
}

func (l *ActionEvidenceLedger) recordToolEvent(ev ai.ToolStepEvent) {
	if ev.Kind != "result" && ev.Kind != "done" && ev.Kind != "error" {
		return
	}
	status := "succeeded"
	if ev.Kind == "error" {
		status = "failed"
	}
	switch ev.Name {
	case proposeFileEditToolName, searchReplaceToolName, applyPatchToolName:
		l.Record(ActionEvidence{Kind: EvidenceEditProposed, Tool: ev.Name, Status: status, Detail: ev.Preview})
	case createArtifactToolName, updateArtifactToolName, mapsCreateToolName, mapsUpdateToolName:
		l.Record(ActionEvidence{Kind: EvidenceArtifactCreated, Tool: ev.Name, Status: status, Detail: ev.Preview})
	}
}

type actionEvidenceContextKey struct{}

func contextWithActionEvidence(ctx context.Context, ledger *ActionEvidenceLedger) context.Context {
	return context.WithValue(ctx, actionEvidenceContextKey{}, ledger)
}

func actionEvidenceFromContext(ctx context.Context) *ActionEvidenceLedger {
	if ctx == nil {
		return nil
	}
	ledger, _ := ctx.Value(actionEvidenceContextKey{}).(*ActionEvidenceLedger)
	return ledger
}

func (st *turnState) buildActionEvidence() {
	ledger := st.evidence
	if ledger == nil {
		return
	}
	if st.proposedFileChange || st.proposedGitChange || st.implSessionProposed {
		ledger.Record(ActionEvidence{Kind: EvidenceEditProposed, Status: "succeeded"})
	}
	outcome := st.implSessionOutcome
	if outcome == nil {
		return
	}
	if value, _ := outcome["outcome"].(string); value != "" {
		switch value {
		case "proposals_submitted":
			ledger.Record(ActionEvidence{Kind: EvidenceEditProposed, Status: "succeeded", Detail: value})
		case "applied_and_verified":
			ledger.Record(ActionEvidence{Kind: EvidenceEditApplied, Status: "succeeded", Detail: value})
		case "applied_verify_failed":
			ledger.Record(ActionEvidence{Kind: EvidenceEditApplied, Status: "failed", Detail: value})
		}
	}
	if raw, ok := outcome["repro_exit_code"]; ok {
		if code, ok := integerValue(raw); ok {
			ledger.Record(ActionEvidence{Kind: EvidenceCommandRun, Tool: "run_command", Status: "succeeded", ExitCode: &code})
			if code == 0 {
				ledger.Record(ActionEvidence{Kind: EvidenceCommandPass, Tool: "run_command", Status: "succeeded", ExitCode: &code})
			}
		}
	}
}

func integerValue(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

var (
	artifactClaimRE = regexp.MustCompile(`(?i)\b(created|generated|posted|made|updated|create|creating|will create|i will create)\b.{0,64}\b(neural\s+canvas|canvas|artifact|report|chart|timeline|diagram)\b|\b(neural\s+canvas|canvas|artifact)\b.{0,64}\b(created|generated|posted|ready|updated)\b`)
	// Fake in-chat "Neural Canvas: Title" dumps without a real artifact tool call.
	fakeNeuralCanvasHeaderRE = regexp.MustCompile(`(?i)(\*\*)?neural\s+canvas\s*:\s*\S`)
	imageClaimRE             = regexp.MustCompile(`(?i)\b(generated|created|posted|made)\b.{0,48}\b(image|picture|illustration|logo|visual)\b|\b(image|picture|illustration|logo|visual)\b.{0,48}\b(generated|created|posted|ready)\b`)
	runClaimRE               = regexp.MustCompile(`(?i)\b(i|we)\s+(ran|executed|tested|built|linted)\b|\b(command|tests?|build|lint)\s+(ran|completed|finished)\b`)
	passClaimRE              = regexp.MustCompile(`(?i)\b(tests?|build|lint|checks?)\s+(all\s+)?(pass|passed|succeeded|green)\b|\bpasses\b`)
	actionPassClaimRE        = regexp.MustCompile(`(?i)\b(pass|passed|succeeded|green)\b`)
	actionEditClaimRE        = regexp.MustCompile(`(?i)\b(applied|implemented|completed|complete|done|saved|written|updated)\b`)
	editClaimRE              = regexp.MustCompile(`(?i)\b(applied|saved|written|updated|modified|implemented|completed)\b.{0,64}\b(file|code|change|implementation|fix)\b|\b(file|code|change|implementation|fix)\b.{0,64}\b(applied|saved|written|updated|complete|completed)\b`)
	deflectRE                = regexp.MustCompile(`(?i)\b(i can help (you )?(with|do)|here(?:'s| is) how (you can|to)|you can (run|create|generate|edit)|would you like me to|i can guide you)\b`)
)

type responseValidationIssue string

const (
	issueUnsupportedArtifact     responseValidationIssue = "unsupported_artifact_claim"
	issueUnsupportedImage        responseValidationIssue = "unsupported_image_claim"
	issueUnsupportedRun          responseValidationIssue = "unsupported_run_claim"
	issueUnsupportedPass         responseValidationIssue = "unsupported_pass_claim"
	issueUnsupportedEdit         responseValidationIssue = "unsupported_edit_claim"
	issueActionDeflection        responseValidationIssue = "action_deflection"
	issueMissingRequiredEvidence responseValidationIssue = "missing_required_evidence"
	issueDirectness              responseValidationIssue = "direct_answer_failure"
	issueCorrectionIgnored       responseValidationIssue = "correction_ignored"
)

func validateResponseAgainstEvidence(goal TurnGoal, ledger *ActionEvidenceLedger, msg *protocol.Message, response string, history []*protocol.Message) []responseValidationIssue {
	var issues []responseValidationIssue
	claimedArtifact := artifactClaimRE.MatchString(response) ||
		(fakeNeuralCanvasHeaderRE.MatchString(response) && msg != nil && intent.LooksLikeCanvasCreateOrFillAsk(msg.Content))
	if claimedArtifact && !ledger.Has(EvidenceArtifactCreated) {
		issues = append(issues, issueUnsupportedArtifact)
	}
	if imageClaimRE.MatchString(response) && !ledger.Has(EvidenceImagePosted) {
		issues = append(issues, issueUnsupportedImage)
	}
	if runClaimRE.MatchString(response) && !ledger.Has(EvidenceCommandRun) {
		issues = append(issues, issueUnsupportedRun)
	}
	actionPassClaim := goal.Action == ActionRun && actionPassClaimRE.MatchString(response)
	if (passClaimRE.MatchString(response) || actionPassClaim) && !ledger.Has(EvidenceCommandPass) {
		issues = append(issues, issueUnsupportedPass)
	}
	actionEditClaim := (goal.Action == ActionEdit || goal.Action == ActionContinue) && actionEditClaimRE.MatchString(response)
	if (editClaimRE.MatchString(response) || actionEditClaim) && !ledger.Has(EvidenceEditApplied) {
		issues = append(issues, issueUnsupportedEdit)
	}
	if goal.RequiresActionEvidence() && !goalHasExpectedEvidence(goal, ledger) {
		switch goal.Action {
		case ActionArtifact, ActionImage:
			// Explicit canvas/image goals require tool evidence — FILE_CHANGE prose is not a substitute.
			issues = append(issues, issueMissingRequiredEvidence)
		default:
			// Missing tool evidence alone is not a failure when the model answered
			// without claiming the action (common under misclassified turn goals).
			// Soft-fail only for empty replies or explicit "I can help you / you can run" deflection.
			if deflectRE.MatchString(response) || strings.TrimSpace(response) == "" {
				issues = append(issues, issueActionDeflection)
			}
		}
	}
	directFailure := looksLikeEchoOfPriorUserTurn(msg, response, history) ||
		looksLikeReAskAfterAffirmation(msg, response, history) ||
		looksLikeAsksUserToPasteWorkspaceFiles(msg, response) ||
		looksLikeIgnoresCodebaseAttachments(msg, response) ||
		looksLikeIgnoresWorkspaceVisibility(msg, response) ||
		looksLikeShallowImplementationReply(msg, response) ||
		looksLikeWrongModalAccessibilityAnswer(msg, response, history) ||
		looksLikeSuperficialCodeFixReply(msg, response, history) ||
		looksLikeUnverifiedRepoEndpointClaim(msg, response) ||
		looksLikeFabricatedEndpointEndorsement(msg, response) ||
		looksLikeRepoFactChallengeDoubleDown(msg, response)
	if directFailure {
		issues = append(issues, issueDirectness)
	}
	return uniqueValidationIssues(issues)
}

// validateActiveCorrectionsHonored flags continue/summary replies that drop or revive renamed labels.
func validateActiveCorrectionsHonored(envelope protocol.TurnContextEnvelope, msg *protocol.Message, response string) []responseValidationIssue {
	if msg == nil || strings.TrimSpace(response) == "" {
		return nil
	}
	if len(envelope.Corrections) == 0 && pinnedGoalToken(envelope) == "" {
		return nil
	}
	lowerAsk := strings.ToLower(msg.Content)
	if !(strings.Contains(lowerAsk, "summar") || strings.Contains(lowerAsk, "continue") ||
		strings.Contains(lowerAsk, "final") || strings.Contains(lowerAsk, "after the correction")) {
		return nil
	}
	for _, correction := range envelope.Corrections {
		target := correctionRenameTarget(correction.Instruction)
		if target == "" {
			continue
		}
		if !strings.Contains(response, target) {
			return []responseValidationIssue{issueCorrectionIgnored}
		}
		if superseded := supersededComponentName(envelope, target); superseded != "" {
			staleRE := regexp.MustCompile(`(?i)(final|component)\s+[^\n.]{0,40}` + regexp.QuoteMeta(superseded))
			if staleRE.MatchString(response) {
				return []responseValidationIssue{issueCorrectionIgnored}
			}
		}
	}
	if pinned := pinnedGoalToken(envelope); pinned != "" {
		correctionTarget := ""
		for _, correction := range envelope.Corrections {
			if t := correctionRenameTarget(correction.Instruction); t != "" {
				correctionTarget = t
				break
			}
		}
		if !strings.Contains(response, pinned) && (correctionTarget == "" || !strings.Contains(response, correctionTarget)) {
			return []responseValidationIssue{issueCorrectionIgnored}
		}
	}
	return nil
}

var pinnedCamelRE = regexp.MustCompile(`\b([A-Z][a-z0-9]+(?:[A-Z][a-z0-9]+)+)\b`)

func pinnedGoalToken(envelope protocol.TurnContextEnvelope) string {
	if envelope.Goal == nil {
		return ""
	}
	for _, text := range []string{envelope.Goal.PinnedText, envelope.Goal.Text} {
		if tok := componentNameFromText(text); tok != "" {
			return tok
		}
		if m := pinnedCamelRE.FindStringSubmatch(text); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

// shouldRewriteAsSafeFailure reports whether validation issues should replace the
// model response with safeActionFailure. Unsupported action claims still rewrite;
// a substantive answer under a misclassified goal is kept.
func shouldRewriteAsSafeFailure(issues []responseValidationIssue, response string) bool {
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		switch issue {
		case issueUnsupportedArtifact, issueUnsupportedImage, issueUnsupportedRun,
			issueUnsupportedPass, issueUnsupportedEdit, issueMissingRequiredEvidence:
			return true
		case issueActionDeflection:
			if deflectRE.MatchString(response) || strings.TrimSpace(response) == "" {
				return true
			}
		}
	}
	return false
}

// shouldRewriteAsSafeFailureForGoal keeps successful map/artifact replies when the
// ledger proves the canvas deliverable landed, even if the turn goal was misclassified
// as run/edit (local classifiers often map "map from A to B" onto ActionRun).
func shouldRewriteAsSafeFailureForGoal(goal TurnGoal, ledger *ActionEvidenceLedger, issues []responseValidationIssue, response string) bool {
	if !shouldRewriteAsSafeFailure(issues, response) {
		return false
	}
	if ledger != nil && ledger.Has(EvidenceArtifactCreated) {
		for _, issue := range issues {
			switch issue {
			case issueUnsupportedRun, issueUnsupportedPass, issueUnsupportedEdit, issueActionDeflection:
				continue
			case issueUnsupportedArtifact, issueUnsupportedImage, issueDirectness:
				return true
			}
		}
		return false
	}
	_ = goal
	return true
}

// shouldKeepChatSurfaceResponse keeps a non-empty chat answer when the user
// asked for text in-thread (or a meeting-note summary) but the turn was
// mis-stamped ActionArtifact because a canvas tab was focused.
func shouldKeepChatSurfaceResponse(msg *protocol.Message, goal TurnGoal, issues []responseValidationIssue, response string) bool {
	if msg == nil || strings.TrimSpace(response) == "" {
		return false
	}
	if goal.Action != ActionArtifact {
		return false
	}
	if !intent.PrefersChatOverOpenCanvas(msg.Content) {
		return false
	}
	for _, issue := range issues {
		switch issue {
		case issueMissingRequiredEvidence:
			continue
		default:
			return false
		}
	}
	return len(issues) > 0
}

// shouldKeepOpenCanvasReviseResponse suppresses Edit/AskUser soft-fails when the
// user clearly asked to revise an open Neural Canvas. Promote should normally
// stamp ActionArtifact first; this is a safety net so "wasn't able to make
// changes" / "question prompt" does not replace a usable model reply.
func shouldKeepOpenCanvasReviseResponse(msg *protocol.Message, goal TurnGoal, issues []responseValidationIssue) bool {
	if msg == nil || len(issues) == 0 {
		return false
	}
	switch goal.Action {
	case ActionEdit, ActionContinue, ActionAskUser:
	default:
		return false
	}
	if !messageHasOpenCanvasArtifact(msg) {
		return false
	}
	if !intent.LooksLikeOpenCanvasReviseAsk(msg.Content) && !intent.LooksLikeOpenCanvasFillAsk(msg.Content) {
		return false
	}
	for _, issue := range issues {
		switch issue {
		case issueUnsupportedEdit, issueActionDeflection, issueMissingRequiredEvidence:
			continue
		default:
			return false
		}
	}
	return true
}

func messageHasOpenCanvasArtifact(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	raw, ok := msg.Metadata["open_artifact"]
	if !ok || raw == nil {
		return false
	}
	switch v := raw.(type) {
	case map[string]interface{}:
		id, _ := v["id"].(string)
		return strings.TrimSpace(id) != "" && strings.TrimSpace(id) != "__library__"
	case map[string]string:
		id := strings.TrimSpace(v["id"])
		return id != "" && id != "__library__"
	default:
		return false
	}
}

func goalHasExpectedEvidence(goal TurnGoal, ledger *ActionEvidenceLedger) bool {
	for _, kind := range goal.ExpectedEvidence {
		if ledger.Has(kind) {
			return true
		}
	}
	return !goal.RequiresActionEvidence()
}

func uniqueValidationIssues(in []responseValidationIssue) []responseValidationIssue {
	seen := make(map[responseValidationIssue]bool)
	out := make([]responseValidationIssue, 0, len(in))
	for _, issue := range in {
		if !seen[issue] {
			seen[issue] = true
			out = append(out, issue)
		}
	}
	return out
}

func safeActionFailure(goal TurnGoal, ledger *ActionEvidenceLedger) string {
	switch goal.Action {
	case ActionArtifact:
		return "I couldn't create the requested Neural Canvas artifact in this turn."
	case ActionImage:
		return "I couldn't generate or post the requested image in this turn."
	case ActionRun:
		return "I wasn't able to run the requested command in this turn."
	case ActionDebug:
		if ledger.Has(EvidenceCommandPass) {
			return "I ran the workspace diagnostic, but it passed and did not reproduce the reported failure. Which command or screen fails?"
		}
		if ledger.Has(EvidenceCommandRun) {
			return "I reproduced the workspace failure, but I couldn't complete a grounded repair in this turn."
		}
		return "I wasn't able to run the requested workspace diagnosis in this turn."
	case ActionEdit, ActionContinue:
		if ledger.Has(EvidenceEditProposed) {
			return "I submitted the file changes as proposals; they have not been applied yet."
		}
		if ledger.Has(EvidenceCommandPass) {
			return "I ran the workspace diagnostic, but it passed and did not reproduce the reported failure. Which command or screen fails when you try to start the app?"
		}
		if ledger.Has(EvidenceCommandRun) {
			return "I reproduced the workspace failure, but I couldn't produce a grounded file change in this turn."
		}
		return "I wasn't able to make or propose the requested changes in this turn."
	case ActionAskUser:
		return "I wasn't able to open the requested question prompt in this turn."
	case ActionInspect:
		return "I wasn't able to run the requested git/workspace inspection in this turn."
	default:
		return "I couldn't complete the requested action in this turn."
	}
}

func buildResponseValidationRetryPrompt(goal TurnGoal, ledger *ActionEvidenceLedger, msg *protocol.Message) string {
	prefix := "Answer the user's latest request directly. Do not repeat questions, ignore supplied context, or claim actions not shown by the evidence.\n" +
		"Honor ACTIVE CORRECTION names from durable conversation state — never revive superseded labels.\n"
	if msg != nil && looksLikeCodeCritiqueFollowUp(msg.Content) {
		prefix += "The user is correcting bugs in code you previously showed. Fix EVERY issue they named in a complete code block — assign refs before restore, remove duplicate handlers, avoid broken event-target guards, call onClose on Escape (never console.log placeholders), keep Tab focus-trap cycling, capture document.activeElement for restore (no getElementById hacks), call hooks before early returns, and never assign to hook boolean params like isOpen = false.\n"
	}
	if msg != nil && modalAccessibilityGapFollowUp(msg.Content) {
		prefix += "For modal accessibility gap-fill: querySelectorAll does NOT exclude display:none — filter with Array.from(...).filter(el => el.offsetParent !== null); put tabIndex={-1} on the dialog in JSX (not setAttribute at runtime); guard the ENTIRE keydown handler (Escape AND Tab) with dialogRef.current.contains(document.activeElement) so nested modals/popovers on top are not affected.\n"
	}
	if msg != nil && looksLikeConcreteCodeRequest(msg.Content) {
		prefix += "The user asked for working code, not a strategy memo. Lead with a complete copy-paste-ready code block. For modal focus traps: intercept Tab/Shift+Tab ONLY at the first/last focusable boundary (document.activeElement === first|last) so normal tabbing works inside the dialog; call onClose on Escape; capture document.activeElement to a ref and restore it in effect cleanup; guard with if (!isOpen) return before querying modalRef when the component returns null while closed — do not assign isOpen = false inside handlers.\n"
	}
	if msg != nil && (looksLikeRepoFactAsk(msg.Content) || looksLikeRepoFactChallengeFollowUp(msg.Content)) {
		prefix += "Do not invent HTTP paths or endorse arbitrary user-suggested endpoints as correct. Admit uncertainty unless verified in workspace/tool output.\n"
	}
	return fmt.Sprintf(
		prefix+"TURN GOAL: %s\nACTION: %s\nEVIDENCE: %+v\n",
		goal.NormalizedRequest, goal.Action, ledger.Entries(),
	) + ai.SystemPromptSeparator + strings.TrimSpace(msg.Content)
}
