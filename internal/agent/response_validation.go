package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/ai"
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
	case createArtifactToolName, updateArtifactToolName:
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
	artifactClaimRE   = regexp.MustCompile(`(?i)\b(created|generated|posted|made|updated)\b.{0,48}\b(neural canvas|canvas|artifact|report|chart|timeline|diagram)\b|\b(neural canvas|canvas|artifact)\b.{0,48}\b(created|generated|posted|ready|updated)\b`)
	imageClaimRE      = regexp.MustCompile(`(?i)\b(generated|created|posted|made)\b.{0,48}\b(image|picture|illustration|logo|visual)\b|\b(image|picture|illustration|logo|visual)\b.{0,48}\b(generated|created|posted|ready)\b`)
	runClaimRE        = regexp.MustCompile(`(?i)\b(i|we)\s+(ran|executed|tested|built|linted)\b|\b(command|tests?|build|lint)\s+(ran|completed|finished)\b`)
	passClaimRE       = regexp.MustCompile(`(?i)\b(tests?|build|lint|checks?)\s+(all\s+)?(pass|passed|succeeded|green)\b|\bpasses\b`)
	actionPassClaimRE = regexp.MustCompile(`(?i)\b(pass|passed|succeeded|green)\b`)
	actionEditClaimRE = regexp.MustCompile(`(?i)\b(applied|implemented|completed|complete|done|saved|written|updated)\b`)
	editClaimRE       = regexp.MustCompile(`(?i)\b(applied|saved|written|updated|modified|implemented|completed)\b.{0,64}\b(file|code|change|implementation|fix)\b|\b(file|code|change|implementation|fix)\b.{0,64}\b(applied|saved|written|updated|complete|completed)\b`)
	deflectRE         = regexp.MustCompile(`(?i)\b(i can help (you )?(with|do)|here(?:'s| is) how (you can|to)|you can (run|create|generate|edit)|would you like me to|i can guide you)\b`)
)

type responseValidationIssue string

const (
	issueUnsupportedArtifact responseValidationIssue = "unsupported_artifact_claim"
	issueUnsupportedImage    responseValidationIssue = "unsupported_image_claim"
	issueUnsupportedRun      responseValidationIssue = "unsupported_run_claim"
	issueUnsupportedPass     responseValidationIssue = "unsupported_pass_claim"
	issueUnsupportedEdit     responseValidationIssue = "unsupported_edit_claim"
	issueActionDeflection    responseValidationIssue = "action_deflection"
	issueDirectness          responseValidationIssue = "direct_answer_failure"
)

func validateResponseAgainstEvidence(goal TurnGoal, ledger *ActionEvidenceLedger, msg *protocol.Message, response string, history []*protocol.Message) []responseValidationIssue {
	var issues []responseValidationIssue
	if artifactClaimRE.MatchString(response) && !ledger.Has(EvidenceArtifactCreated) {
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
		if deflectRE.MatchString(response) || len(issues) == 0 {
			issues = append(issues, issueActionDeflection)
		}
	}
	directFailure := looksLikeEchoOfPriorUserTurn(msg, response, history) ||
		looksLikeReAskAfterAffirmation(msg, response, history) ||
		looksLikeAsksUserToPasteWorkspaceFiles(msg, response) ||
		looksLikeIgnoresCodebaseAttachments(msg, response) ||
		looksLikeIgnoresWorkspaceVisibility(msg, response)
	if directFailure {
		issues = append(issues, issueDirectness)
	}
	return uniqueValidationIssues(issues)
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
	default:
		return "I couldn't complete the requested action in this turn."
	}
}

func buildResponseValidationRetryPrompt(goal TurnGoal, ledger *ActionEvidenceLedger, msg *protocol.Message) string {
	return fmt.Sprintf(
		"Answer the user's latest request directly. Do not repeat questions, ignore supplied context, or claim actions not shown by the evidence.\n"+
			"TURN GOAL: %s\nACTION: %s\nEVIDENCE: %+v\n",
		goal.NormalizedRequest, goal.Action, ledger.Entries(),
	) + ai.SystemPromptSeparator + strings.TrimSpace(msg.Content)
}
