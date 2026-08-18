package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	semantic "github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const maxRecentlyAppliedFilePaths = 8

var (
	implementationAffirmRE      = regexp.MustCompile(`(?i)\b(approved|approve(d| it)?|keep going|please continue|continue(?: with (it|this|that|the work))?|looks good|that sounds good|sounds good|go[- ]?ahead|goadhead|do it( now)?|yes please|please do|proceed|make (the |those )?(changes|them)|apply (that|it|your plan)|do that now|ok please|sure,?\s*please|let's do it|please implement|sounds good[,!]?\s*(go|do)|that works[,!]?\s*(go|do)?|you can (start|begin|proceed)|yeah go ahead|yes[,!]?\s*(keep going|that sounds good|use that|please))\b`)
	weakImplementationAffirmRE  = regexp.MustCompile(`(?i)^(?:@\w+\s+)?(?:ok|okay|looks good|that works|sounds good|nice|great|cool|perfect)[!.?\s]*$`)
	themeImplementationRE       = regexp.MustCompile(`(?i)(?:\b(theme|themes|dark[/ ]?light|dark mode|light mode|ui theme)\b.{0,64}\b(add(?:ing)?|implement(?:ing)?|build(?:ing)?|wire|toggle|finish)\b|\b(add(?:ing)?|implement(?:ing)?|build(?:ing)?|wire|finish)\b.{0,64}\b(theme|themes|ui theme|dark mode|light mode|font size)\b)`)
	implementTypoRE             = regexp.MustCompile(`(?i)\bimpl[e]?ment\b`)
	workspaceDirectiveRE        = regexp.MustCompile(`(?i)\b(use|read|from)\s+(the\s+)?(open\s+)?workspace\b`)
	bootErrorIntentRE           = regexp.MustCompile(`(?i)(not booting|won't boot|will not boot|cannot boot|can't boot|fails? to boot|does not boot|failed to scan|esbuild|✘\s*\[ERROR\]|\[ERROR\].*Expected|make start-all|vite dev|syntax error|white screen|blank screen|exit_code=)`)
	// errorLogMarkerRE detects actual command/build output markers (not user phrasing) in a raw
	// transcript — used only for housekeeping like stale-summary scrubbing, never turn routing.
	errorLogMarkerRE = regexp.MustCompile(`(?i)(✘\s*\[ERROR\]|\[ERROR\].*Expected|esbuild|exit_code=\d|syntax error|panic:|fatal error:|traceback \(most recent call last\))`)
	implementationStatusCheckRE = regexp.MustCompile(`(?i)^(?:@\w+\s+)?(?:is it fixed|did (?:that|it) fix|does it work(?: now)?|is it working(?: now)?|still broken|still not (?:booting|working)|working now)\??[!.?\s]*$`)
	destructiveCommandRE        = regexp.MustCompile(`(?i)\brm\s+-rf\b|\brm\s+-r\b|\brmdir\s+/\b|>\s*/dev/`)
	contentDeliveryRE           = regexp.MustCompile(`(?i)\b(linkedin|blog post|blog article|article about|write (?:me )?(?:a |an )?article|marketing copy|press release|social media post|whitepaper|writeup|newsletter)\b`)
	fileExportRE                = regexp.MustCompile(`(?i)\b(store (?:that|it|in|the)|save (?:it|as|in|the)|fill (?:the file|.* with)|create (?:that |the )?file|please create (?:that |the )?file|write (?:it |that ).*(?:file|\.md)|markdown file)\b`)
	bareWorkspaceWrapperRE = regexp.MustCompile(`(?i)\b(can you|could you|please|for this|for that|to do this|now)\b`)
)

var workspaceDirectiveDocSeeds = []string{"README.md", "DOCS.md", "docs/README.md"}

var contentDeliveryDocSeeds = []string{
	"README.md",
	"DOCS.md",
	"docs/README.md",
	"docs/ARCHITECTURE.md",
	"package.json",
	"go.mod",
	"Cargo.toml",
	"desktop/package.json",
}

const maxImplementationSeedFiles = 8

func userRequestsDestructiveCommand(content string) bool {
	return destructiveCommandRE.MatchString(content)
}

// isAdvisoryImplementationQuestion reports hypothetical or placement-only asks that should
// stay conversational (no [FILE_CHANGE] / implementation session).
func isAdvisoryImplementationQuestion(content string) bool {
	return semantic.LooksLikeAdvisoryImplementationQuestion(content)
}

// agentMessageIsConversationalClosure reports canned closure replies that end an impl thread.
func agentMessageIsConversationalClosure(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	markers := []string{
		"you're welcome",
		"you are welcome",
		"won't repeat",
		"will not repeat",
		"glad that helped",
		"anything else i can help",
		"let me know if you need anything else",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

// userRequestsImplementation is a deprecated phrase-matching heuristic kept only for its
// call sites' signatures. Routing now trusts the stamped TurnDecision (see
// userRequestsImplementationForMessage) instead of natural-language phrase matching.
//
// Deprecated: always returns false. Do not add new call sites.
func userRequestsImplementation(content string) bool {
	return false
}

// userRequestsContentDelivery reports writing/marketing tasks that should not trigger file-edit fallback.
func userRequestsContentDelivery(content string) bool {
	return contentDeliveryRE.MatchString(strings.TrimSpace(content))
}

// userRequestsFileExportForMessage includes explicit composer export mode.
func userRequestsFileExportForMessage(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	// Composer export mode is structural metadata (not a phrase list).
	if msg.IdeEditorModeIsExport() {
		return true
	}
	if _, ok := protocol.ExtractTurnDecision(msg); ok {
		// Canonical decisions must not re-enter export via natural-language phrases.
		return false
	}
	return userRequestsFileExport(msg.Content)
}

// userRequestsFileExport is deprecated. Prefer explicit composer export mode
// (IdeEditorModeIsExport). Routing must not re-enter export via natural-language phrases.
//
// Deprecated: always returns false. Do not add new call sites.
func userRequestsFileExport(content string) bool {
	_ = content
	return false
}

// isBareWorkspaceDirective is deprecated. Fence fallback uses ImplementationSession /
// stamped edit, not NL workspace phrases.
//
// Deprecated: always returns false. Do not add new call sites.
func isBareWorkspaceDirective(content string) bool {
	_ = content
	return false
}


// userAffirmsPendingImplementation is deprecated. Continuations are stamped
// ActionContinue (or ReplyTarget + pending action at the hub).
//
// Deprecated: always returns false. Do not add new call sites.
func userAffirmsPendingImplementation(content string) bool {
	_ = content
	return false
}

// isWeakImplementationAffirmation is deprecated alongside userAffirmsPendingImplementation.
//
// Deprecated: always returns false.
func isWeakImplementationAffirmation(content string) bool {
	_ = content
	return false
}

func agentMessageIsFileContentDump(content string) bool {
	trim := strings.TrimSpace(content)
	return strings.HasPrefix(trim, "###") && strings.Contains(content, "```")
}

func agentMessageImplSessionFailed(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "finished without file changes")
}

func messageIsFileChangeApproval(m *protocol.Message) bool {
	return m != nil && m.FileChangeApproved()
}

func fileChangeApprovalTargetsAgent(m *protocol.Message, agentID string) bool {
	if !messageIsFileChangeApproval(m) {
		return false
	}
	if agentID == "" {
		return true
	}
	target := m.FileChangeApprovalAgentID()
	return target == "" || target == agentID
}

// channelRecentlyAppliedFilePaths returns workspace paths the user already approved via the UI
// in recent channel history (relative and absolute forms normalized for seed exclusion).
func channelRecentlyAppliedFilePaths(history []*protocol.Message, skipMsgID, agentID string) []string {
	seen := make(map[string]bool)
	var paths []string
	scanned := 0
	for i := len(history) - 1; i >= 0 && scanned < 16; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID {
			continue
		}
		scanned++
		if !messageIsFileChangeApproval(m) || !fileChangeApprovalTargetsAgent(m, agentID) {
			continue
		}
		raw := ""
		if m.Metadata != nil {
			raw, _ = m.Metadata[protocol.MetaFileChangePath].(string)
		}
		if raw == "" {
			for _, p := range DetectFilePaths(m.Content) {
				raw = p
				break
			}
		}
		for _, key := range appliedPathExcludeKeys(raw) {
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			paths = append(paths, key)
			if len(paths) >= maxRecentlyAppliedFilePaths {
				return paths
			}
		}
	}
	return paths
}

func appliedPathExcludeKeys(path string) []string {
	path = strings.Trim(strings.TrimSpace(path), "`")
	if path == "" {
		return nil
	}
	keys := []string{path}
	if rel := normalizeFileChangeRelPath(path); rel != "" && rel != path {
		keys = append(keys, rel)
	}
	if base := filepath.Base(path); base != "" && base != path && base != "." {
		keys = append(keys, base)
	}
	return keys
}

func seedPathExcluded(p string, exclude map[string]bool) bool {
	if exclude == nil {
		return false
	}
	p = strings.TrimSpace(p)
	if exclude[p] {
		return true
	}
	if base := filepath.Base(p); base != "" && exclude[base] {
		return true
	}
	return false
}

func mergeAppliedPathsIntoExclude(exclude map[string]bool, applied []string) map[string]bool {
	if len(applied) == 0 {
		return exclude
	}
	if exclude == nil {
		exclude = make(map[string]bool, len(applied))
	}
	for _, raw := range applied {
		for _, key := range appliedPathExcludeKeys(raw) {
			exclude[key] = true
		}
	}
	return exclude
}

// isVagueImplementationContinuation reports bare "pick up where we left off" asks without concrete task detail.
func isVagueImplementationContinuation(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	hasContinuation := strings.Contains(lower, "pick up where") ||
		strings.Contains(lower, "where we left off") ||
		strings.Contains(lower, "continue where we")
	if !hasContinuation {
		return false
	}
	if themeImplementationRE.MatchString(lower) || implementTypoRE.MatchString(lower) {
		return false
	}
	concrete := []string{
		"settings", "modal", "theme", "implement", "add ", "fix ", "build ",
		"tailwind", "component", "button", "src/", ".tsx", ".jsx", ".js", ".go",
		"finish that", "finish the", "yesterday we",
	}
	for _, p := range concrete {
		if strings.Contains(lower, p) {
			return false
		}
	}
	return len(lower) < 120
}

// vagueContinuationWithoutPriorThread blocks implementation sessions when the user only asks
// to resume work but the channel has no prior implementation activity to continue.
func vagueContinuationWithoutPriorThread(history []*protocol.Message, skipMsgID, agentID, content string) bool {
	if !isVagueImplementationContinuation(content) {
		return false
	}
	if channelHasRecentImplementationActivity(history, skipMsgID, agentID) {
		return false
	}
	if channelHasRecentFileChangeApproval(history, skipMsgID, agentID) {
		return false
	}
	return true
}

// channelHasRecentFileChangeApproval reports a UI or command approval the agent should treat as done.
func channelHasRecentFileChangeApproval(history []*protocol.Message, skipMsgID, agentID string) bool {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 16; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID {
			continue
		}
		seen++
		if messageIsFileChangeApproval(m) && fileChangeApprovalTargetsAgent(m, agentID) {
			return true
		}
	}
	return false
}

// channelHasPendingImplementationPlan reports whether the agent awaits approval of a plan or
// proposals — not a failed session or a read-only file dump.
func channelHasPendingImplementationPlan(history []*protocol.Message, skipMsgID, agentID string) bool {
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID {
			continue
		}
		if messageIsFileChangeApproval(m) && fileChangeApprovalTargetsAgent(m, agentID) {
			return false
		}
		if protocol.IsUserLikeSender(m.From) {
			continue
		}
		if agentID != "" && m.From.ID != agentID {
			continue
		}
		if m.Type == protocol.MessageTypeFileChange {
			return true
		}
		if m.Type != protocol.MessageTypeChat && m.Type != protocol.MessageTypeAnswer {
			continue
		}
		body := strings.ToLower(m.Content)
		if strings.Contains(body, "proposals submitted for approval") {
			return true
		}
		if agentMessageImplSessionFailed(m.Content) || agentMessageIsFileContentDump(m.Content) {
			return false
		}
		if strings.Contains(body, "[file_change]") ||
			strings.Contains(body, "proposals submitted for approval") {
			return true
		}
		promisesAction := strings.Contains(body, "i will ") ||
			strings.Contains(body, "i'll ") ||
			strings.Contains(body, "plan:")
		if promisesAction && agentMessagePromisesWorkspaceEdit(body) {
			return true
		}
		return false
	}
	return false
}

func agentMessagePromisesWorkspaceEdit(content string) bool {
	for _, marker := range []string{
		" edit ", " modify ", " update ", " implement ", " fix ", " patch ",
		" refactor ", " apply ", " file", " code", " source", " component",
	} {
		if strings.Contains(" "+strings.ToLower(content)+" ", marker) {
			return true
		}
	}
	return false
}

// affirmationContinuesImplementation gates weak affirmations so "looks good" after a file dump
// or failed session does not re-open the implementation loop.
func affirmationContinuesImplementation(history []*protocol.Message, skipMsgID, agentID, content string) bool {
	if channelHasRecentFileChangeApproval(history, skipMsgID, agentID) {
		return channelHasRecentImplementationAsk(history, skipMsgID) ||
			channelHasRecentImplementationActivity(history, skipMsgID, agentID)
	}
	if !userAffirmsPendingImplementation(content) {
		return false
	}
	hasAsk := channelHasRecentImplementationAsk(history, skipMsgID)
	if isWeakImplementationAffirmation(content) {
		return hasAsk && channelHasPendingImplementationPlan(history, skipMsgID, agentID)
	}
	return hasAsk || channelHasRecentImplementationActivity(history, skipMsgID, agentID)
}

// channelHasRecentImplementationActivity reports an active implementation thread in channel history.
func channelHasRecentImplementationActivity(history []*protocol.Message, skipMsgID string, agentID string) bool {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 16; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID {
			continue
		}
		seen++
		if messageIsFileChangeApproval(m) && fileChangeApprovalTargetsAgent(m, agentID) {
			return true
		}
		if m.Type == protocol.MessageTypeFileChange {
			if agentID == "" || m.From.ID == agentID {
				return true
			}
		}
		if m.Type == protocol.MessageTypeChat || m.Type == protocol.MessageTypeAnswer {
			if agentID != "" && m.From.ID != agentID {
				continue
			}
			if agentMessageIsConversationalClosure(m.Content) {
				return false
			}
			body := strings.ToLower(m.Content)
			if strings.Contains(body, "implementation session complete") ||
				strings.Contains(body, "proposals submitted for approval") ||
				strings.Contains(body, "finished without file changes") {
				return true
			}
		}
		if protocol.IsUserLikeSender(m.From) {
			if classifyConversationalClosure(m.Content) != ClosureNone {
				return false
			}
			if messageStampedImplAction(m) {
				return true
			}
		}
	}
	return false
}

// channelHasRecentFileExportAsk scans recent user turns for save/store/export requests.
func channelHasRecentFileExportAsk(history []*protocol.Message, skipMsgID string) bool {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 16; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID {
			continue
		}
		if !protocol.IsUserLikeSender(m.From) {
			continue
		}
		seen++
		if userRequestsFileExport(m.Content) || m.IdeEditorModeIsExport() {
			return true
		}
		if userReferencesPriorAssistantContent(m.Content) {
			lower := strings.ToLower(m.Content)
			if strings.Contains(lower, "markdown") || strings.Contains(lower, ".md") ||
				strings.Contains(lower, "save") || strings.Contains(lower, "store") {
				return true
			}
		}
	}
	return false
}

// channelHasRecentCodeImplementationAsk scans for coding/build tasks (not content export).
func channelHasRecentCodeImplementationAsk(history []*protocol.Message, skipMsgID string) bool {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 16; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID {
			continue
		}
		if !protocol.IsUserLikeSender(m.From) {
			continue
		}
		seen++
		if userRequestsFileExport(m.Content) || m.IdeEditorModeIsExport() {
			return false
		}
		if messageStampedImplAction(m) {
			return true
		}
	}
	return false
}

// shouldSkipAgentResponseOnFileExportApproval reports hub approval echoes that should not
// re-open an export loop after the user already applied a file change.
func shouldSkipAgentResponseOnFileExportApproval(a *Agent, msg *protocol.Message) bool {
	if a == nil || msg == nil || !msg.FileChangeApproved() {
		return false
	}
	history := a.channelHistory(msg.Channel)
	if !channelHasRecentFileExportAsk(history, msg.ID) {
		return false
	}
	return !channelHasRecentCodeImplementationAsk(history, msg.ID)
}

// channelHasRecentImplementationAsk scans recent user turns for an implementation request.
func channelHasRecentImplementationAsk(history []*protocol.Message, skipMsgID string) bool {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 12; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID {
			continue
		}
		if !protocol.IsUserLikeSender(m.From) {
			continue
		}
		seen++
		if messageStampedImplAction(m) {
			return true
		}
	}
	return false
}

// userRequestsImplementationForMessage is stamp-first: it trusts the semantic TurnDecision
// stamped on the turn (Edit/Debug/Continue/Run → true, everything else → false). When no
// stamp is present it falls back to structural signals only (composer implementation-session
// state, export mode, or a UI file-change approval continuing a recent implementation thread)
// — never natural-language phrase matching.
func userRequestsImplementationForMessage(a *Agent, msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		switch decision.Action {
		case semantic.ActionDebug, semantic.ActionEdit, semantic.ActionContinue, semantic.ActionRun:
			return true
		default:
			return false
		}
	}
	if msg.ImplementationSession() || msg.IdeEditorModeIsExport() {
		return true
	}
	if a == nil {
		return false
	}
	if msg.FileChangeApproved() {
		return !shouldSkipAgentResponseOnFileExportApproval(a, msg)
	}
	history := a.channelHistory(msg.Channel)
	if channelHasRecentFileChangeApproval(history, msg.ID, a.Info.ID) &&
		(channelHasRecentImplementationAsk(history, msg.ID) ||
			channelHasRecentImplementationActivity(history, msg.ID, a.Info.ID)) {
		if channelHasRecentFileExportAsk(history, msg.ID) &&
			!channelHasRecentCodeImplementationAsk(history, msg.ID) {
			return false
		}
		return true
	}
	return false
}

// ShouldForceSessionSummaryRefresh reports user turns that should refresh a stale session
// summary. This is a low-stakes cache-invalidation heuristic used only as a last resort when
// no stamped TurnDecision is available on the message (see ShouldForceSessionSummaryRefreshForMessage)
// — erring toward refreshing more often is safe, so a short affirmation regex is kept live here
// even though the equivalent turn-routing helpers (userAffirmsPendingImplementation, etc.) are
// deprecated stubs.
func ShouldForceSessionSummaryRefresh(content string) bool {
	if implementationAffirmRE.MatchString(content) {
		return true
	}
	// Any mention of reviewing code invalidates a stale summary — this is intentionally
	// broader than userRequestsCodeReview (which gates whole-project vs. targeted-fix
	// routing), since over-refreshing here is harmless.
	lower := strings.ToLower(content)
	if strings.Contains(lower, "review") && strings.Contains(lower, "code") {
		return true
	}
	if userRequestsCodeReview(content) {
		return true
	}
	return false
}

// ShouldForceSessionSummaryRefreshForMessage includes UI file-change approvals.
func ShouldForceSessionSummaryRefreshForMessage(msg *protocol.Message) bool {
	if msg != nil && msg.FileChangeApproved() {
		return true
	}
	if msg != nil && msg.Type == protocol.MessageTypeSystemInfo &&
		strings.Contains(msg.Content, "Applied change") {
		return true
	}
	if msg == nil {
		return false
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		switch decision.Action {
		case semantic.ActionDebug, semantic.ActionEdit, semantic.ActionContinue, semantic.ActionRun:
			return true
		case semantic.ActionInspect:
			return decision.Domain == "code_review" || decision.RecipientType == "code-review"
		default:
			return decision.Interaction == semantic.InteractionContinuation ||
				decision.Interaction == semantic.InteractionCorrection
		}
	}
	return ShouldForceSessionSummaryRefresh(msg.Content)
}

// ScrubStaleSessionSummary removes summary bullets contradicted by a boot/error log in the transcript.
func ScrubStaleSessionSummary(summary, transcript string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" || !errorLogMarkerRE.MatchString(transcript) {
		return summary
	}
	var kept []string
	for _, line := range strings.Split(summary, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		lower := strings.ToLower(trim)
		if strings.Contains(lower, "still needed") &&
			(strings.Contains(lower, "error message") || strings.Contains(lower, "symptoms") || strings.Contains(lower, "specific error")) {
			continue
		}
		if strings.Contains(lower, "open questions") &&
			(strings.Contains(lower, "what happens when") || strings.Contains(lower, "try to start")) {
			continue
		}
		kept = append(kept, trim)
	}
	if len(kept) == 0 {
		return summary
	}
	return strings.Join(kept, "\n")
}

// ShouldForceSessionSummaryRefreshOnAgentResponse reports agent replies that invalidate a stale summary.
func ShouldForceSessionSummaryRefreshOnAgentResponse(content string) bool {
	trim := strings.TrimSpace(content)
	if trim == "" {
		return false
	}
	lower := strings.ToLower(trim)
	if strings.Contains(lower, "grounding: i loaded") {
		return true
	}
	if agentMessageIsFileContentDump(trim) {
		return true
	}
	if agentMessageImplSessionFailed(trim) ||
		strings.Contains(lower, "implementation session complete") ||
		strings.Contains(lower, "proposals submitted for approval") {
		return true
	}
	return false
}

// appendAntiRepeatFileDumpGuidance tells the model not to repeat a prior workspace file dump.
func appendAntiRepeatFileDumpGuidance(prompt *strings.Builder, history []*protocol.Message, agentID string) {
	if prompt == nil || len(history) == 0 {
		return
	}
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m == nil {
			continue
		}
		if agentID != "" && m.From.ID != agentID {
			continue
		}
		if m.Type != protocol.MessageTypeChat && m.Type != protocol.MessageTypeAnswer {
			continue
		}
		if !agentMessageIsFileContentDump(m.Content) {
			return
		}
		prompt.WriteString("\n=== RESPONSE GUIDANCE ===\n")
		prompt.WriteString("Your previous reply already showed file contents from the workspace. ")
		prompt.WriteString("Do NOT repeat the same file dump. Advance the diagnosis, propose fixes via [FILE_CHANGE], ")
		prompt.WriteString("or ask one specific follow-up question.\n\n")
		return
	}
}

func agentTypeCanShipFileChanges(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeAssistant, protocol.AgentTypeFrontend, protocol.AgentTypeBackend, protocol.AgentTypeDatabase,
		protocol.AgentTypeSecurity, protocol.AgentTypeArchitecture, protocol.AgentTypeCodeReview,
		protocol.AgentTypeDevOps, protocol.AgentTypeExpert, protocol.AgentTypeRust:
		return true
	default:
		return false
	}
}

// shouldProactiveScanWorkspace limits bulk workspace scans on implementation turns so
// models do not reply with multi-file architecture tours (e.g. after "package.json" appears
// in constraint text). This is a prompt-context hint, not a turn-routing decision, so the
// content-only themeImplementationRE signal (add/implement/finish + theme/dark-mode language)
// stays live here even though userRequestsImplementation is a deprecated routing stub.
func shouldProactiveScanWorkspace(content string) bool {
	if themeImplementationRE.MatchString(content) {
		return true
	}
	return shouldInjectWorkspaceCode(content)
}

// shouldProactiveScanWorkspaceForMessage includes affirmation follow-ups in the same thread.
func shouldProactiveScanWorkspaceForMessage(a *Agent, msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if explicitCodebaseLookupWithChunks(msg) {
		return false
	}
	if a != nil && userRequestsImplementationForMessage(a, msg) {
		return true
	}
	return shouldProactiveScanWorkspace(msg.Content)
}

// shouldForceWorkspaceGroundingOpener is false for chat / context_scope=none turns
// (e.g. dm-topic-switch casual opinions must not open with "Grounding: I loaded").
func shouldForceWorkspaceGroundingOpener(msg *protocol.Message) bool {
	if msg == nil {
		return true
	}
	if ConversationModeFromMessage(msg) == ConversationModeChat {
		return false
	}
	if ResolveContextScope(msg) == ContextScopeNone {
		return false
	}
	return true
}

func workspaceGroundingRequirement(totalLoaded int, content string, implementation ...bool) string {
	if totalLoaded <= 0 {
		return ""
	}
	impl := len(implementation) > 0 && implementation[0]
	if !impl {
		impl = userRequestsImplementation(content)
	}
	if impl {
		return fmt.Sprintf(
			"\nGrounding requirement: Start your answer with exactly this one line:\n"+
				"\"Grounding: I loaded %d file(s) from the workspace context for this answer.\"\n"+
				"Then ship concrete changes via [FILE_CHANGE] blocks (one per file you modify). "+
				"Do not write a codebase tour or architecture summary unless the user asked for one.\n\n",
			totalLoaded,
		)
	}
	if userRequestsEditorDocumentReview(content) {
		return fmt.Sprintf(
			"\nGrounding requirement: Start your answer with exactly this one line:\n"+
				"\"Grounding: I loaded %d file(s) from the workspace context for this answer.\"\n"+
				"Then immediately review the ACTIVE open file from WORKSPACE CONTEXT. "+
				"Name that file path in your next sentence. Do not ask which file or claim you cannot see their editor/tab.\n\n",
			totalLoaded,
		)
	}
	return fmt.Sprintf(
		"\nGrounding requirement: Start your answer with exactly this one line:\n"+
			"\"Grounding: I loaded %d file(s) from the workspace context for this answer.\"\nThen continue with your analysis.\n\n",
		totalLoaded,
	)
}

func appendContentDeliveryGuidance(prompt *strings.Builder, msg *protocol.Message) {
	if msg == nil || !userRequestsContentDelivery(msg.Content) {
		return
	}
	if ResolveContextScope(msg) == ContextScopeNone && !messageHasWorkspaceContext(msg) {
		return
	}
	prompt.WriteString("\n=== CONTENT FROM WORKSPACE (this turn) ===\n")
	prompt.WriteString("The user wants writing or marketing content grounded in the open project. ")
	prompt.WriteString("Use README, docs, and PROJECT DOCS below — do NOT ask them to paste project details. ")
	prompt.WriteString("You may use read_file / semantic_search for additional paths when tools are available.\n")
	prompt.WriteString("Format long copy as proper markdown: blank lines before ### headings, --- on its own line, ")
	prompt.WriteString("one numbered list item per line, and sub-bullets on separate lines starting with -.\n")
	if userRequestsFileExport(msg.Content) {
		prompt.WriteString("If PRIOR ASSISTANT CONTENT appears below, use it verbatim as the file body — do not invent a generic template.\n")
	}
	prompt.WriteString("\n")
}

func appendImplementationDeliveryGuidance(prompt *strings.Builder, a *Agent, msg *protocol.Message, agentType protocol.AgentType) {
	if msg == nil {
		return
	}
	shipChanges := userRequestsImplementationForMessage(a, msg) || userRequestsFileExport(msg.Content)
	if !shipChanges || !agentTypeCanShipFileChanges(agentType) {
		return
	}
	prompt.WriteString("\n=== IMPLEMENTATION DELIVERY (required) ===\n")
	prompt.WriteString("The user wants working changes in the shared workspace, not advice-only.\n")
	prompt.WriteString("You MUST include one or more [FILE_CHANGE] blocks with real file paths and content.\n")
	prompt.WriteString("Each path must be a real relative file (e.g. tailwind.config.js, src/index.css) — never labels like \"File:\" or \"path:\".\n")
	prompt.WriteString("Prefer dependencies already declared in package.json; run npm install or npm ci when modules are missing from node_modules.\n")
	prompt.WriteString("Keep conversational text short (2-4 sentences); put code in [FILE_CHANGE], not long fenced dumps.\n")
	if a != nil && a.hasWorkspaceTools() {
		prompt.WriteString("When workspace tools are available, prefer search_replace or apply_patch for edits; propose_file_edit for creates.\n")
	}
	prompt.WriteString("Do NOT ask the user to paste or share file contents when REFERENCED FILES or WORKSPACE SOURCE FILES appear below — read them and emit [FILE_CHANGE].\n")
	prompt.WriteString("Only ask for a path if a required file is missing from every context section.\n")
	if a != nil {
		history := a.channelHistory(msg.Channel)
		if messageImpliesBootFix(msg.Content, history) || messageHasBootOrBuildError(msg.Content) {
			prompt.WriteString("Boot/build fix: use read_file on paths from the error log, then fix or delete conflicting files. ")
			if messageSuggestsMissingDependencies(msg.Content, history) {
				prompt.WriteString("When errors cite missing npm packages (Cannot find module, TS2307), run npm install or npm ci in the workspace root, then verify with npm run build.\n")
			} else {
				prompt.WriteString("Verify with npm run build (or go test ./...). Use npm install only when package.json lists a dependency that is absent from node_modules.\n")
			}
		}
		applied := channelRecentlyAppliedFilePaths(history, msg.ID, a.Info.ID)
		if len(applied) > 0 {
			prompt.WriteString(fmt.Sprintf(
				"Already applied in this thread (do NOT re-propose identical edits): %s. Ship the NEXT file(s) needed to finish the task.\n",
				strings.Join(applied, ", "),
			))
		}
		if channelHasRecentFileChangeApproval(history, msg.ID, a.Info.ID) {
			prompt.WriteString("The user already approved and applied your file change via the UI — continue with next steps; do NOT ask for approval again.\n")
		}
	}
	if userAffirmsPendingImplementation(msg.Content) {
		prompt.WriteString("The user already approved your plan — do NOT re-plan, re-ask design questions, or request more details. Ship [FILE_CHANGE] blocks now.\n")
	}
}

// messageHasBootOrBuildError is deprecated for routing. Boot/fix turns are stamped
// ActionDebug/Edit by the semantic classifier. Log markers may still appear in history
// scans for playbooks — those should use structured verify outcomes when available.
//
// Deprecated for routing: always returns false.
func messageHasBootOrBuildError(content string) bool {
	_ = content
	return false
}

// userRequestsImplementationStatusCheck is deprecated. Status follow-ups are stamped
// by the classifier (inspect/answer/continue) rather than phrase lists.
//
// Deprecated: always returns false.
func userRequestsImplementationStatusCheck(content string) bool {
	_ = content
	return false
}

// messageSuggestsMissingDependencies reports build output that likely needs npm install.
func messageSuggestsMissingDependencies(content string, history []*protocol.Message) bool {
	check := func(text string) bool {
		lower := strings.ToLower(strings.TrimSpace(text))
		if lower == "" {
			return false
		}
		return strings.Contains(lower, "cannot find module") ||
			strings.Contains(lower, "module not found") ||
			strings.Contains(lower, "are they installed") ||
			strings.Contains(lower, "ts2307") ||
			strings.Contains(lower, "missing dependency") ||
			strings.Contains(lower, "command not allowlisted: npm install")
	}
	if check(content) {
		return true
	}
	for i := len(history) - 1; i >= 0 && i >= len(history)-12; i-- {
		if history[i] == nil {
			continue
		}
		if check(history[i].Content) {
			return true
		}
	}
	return false
}

// tryImplementationStatusCheckShortcut answers "is it fixed?" from recent session context
// without calling the LLM (avoids slow re-runs during boot-fix follow-ups). This is a narrow,
// exact-phrase deterministic reply shortcut (not a general routing decision), so the anchored
// implementationStatusCheckRE stays live here even though the broader
// userRequestsImplementationStatusCheck routing helper is a deprecated stub.
func (a *Agent) tryImplementationStatusCheckShortcut(msg *protocol.Message) (string, bool) {
	if a == nil || msg == nil || !implementationStatusCheckRE.MatchString(strings.TrimSpace(msg.Content)) {
		return "", false
	}
	history := a.channelHistory(msg.Channel)
	if !channelHasRecentImplementationActivity(history, msg.ID, a.Info.ID) {
		return "", false
	}
	var prior string
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m == nil || m.From.ID != a.Info.ID {
			continue
		}
		lower := strings.ToLower(m.Content)
		if strings.Contains(lower, "implementation session complete") ||
			strings.Contains(lower, "implementation session finished") {
			prior = strings.TrimSpace(m.Content)
			break
		}
	}
	if prior == "" {
		return "", false
	}
	lower := strings.ToLower(prior)
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath != "" && strings.Contains(lower, "app.js") {
		if _, err := os.Stat(filepath.Join(wsPath, "src", "App.js")); err != nil {
			return "Yes — src/App.js was removed so Vite resolves ./App to App.tsx. The corrupt App.js diff paste should no longer block boot.", true
		}
	}
	if strings.Contains(lower, "applied and verified") {
		return "Yes — the last implementation session applied and verified successfully.", true
	}
	if strings.Contains(lower, "verification failed") {
		return "Not fully — the last session applied changes but verification failed. Check the prior message for command output.", true
	}
	return "The last implementation session submitted changes; see the prior reply for details.", true
}

// messageImpliesBootFix reports boot/build failure signals in the message or recent history.
func messageImpliesBootFix(content string, history []*protocol.Message) bool {
	if messageHasBootOrBuildError(content) {
		return true
	}
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 12; i-- {
		m := history[i]
		if m == nil {
			continue
		}
		seen++
		if messageHasBootOrBuildError(m.Content) {
			return true
		}
	}
	return false
}

// implementationSeedCandidates returns paths to load from disk for implement turns.
// Stack manifest drives seeds for all agent types; error-log paths are prioritized first.
// exclude holds recently applied or otherwise skipped paths (may use basename keys).
// When researchDocDeliverable is true and taskFocusPaths is non-empty, explicit task
// context paths are seeded instead of generic stack entrypoints (e.g. src/App.tsx).
func implementationSeedCandidates(workspacePath string, content string, history []*protocol.Message, exclude map[string]bool, taskFocusPaths []string, researchDocDeliverable bool) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] || seedPathExcluded(p, exclude) {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, p := range DetectFilePaths(content) {
		add(p)
	}
	if workspaceDirectiveRE.MatchString(content) {
		for _, p := range workspaceDirectiveDocSeeds {
			add(p)
		}
	}
	seenHist := 0
	for i := len(history) - 1; i >= 0 && seenHist < 12; i-- {
		m := history[i]
		if m == nil {
			continue
		}
		seenHist++
		for _, p := range DetectFilePaths(m.Content) {
			add(p)
		}
	}
	if messageImpliesBootFix(content, history) && !researchDocDeliverable {
		for _, p := range []string{
			"Makefile", "scripts/start-all.sh",
			"src/App.js", "src/App.tsx", "src/main.tsx", "src/main.ts", "package.json",
		} {
			add(p)
		}
	}
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath != "" {
		if researchDocDeliverable && len(taskFocusPaths) > 0 {
			for _, p := range taskFocusPaths {
				add(p)
			}
		} else if manifest := DetectStackManifest(workspacePath); manifest != nil {
			for _, p := range manifest.ImplementationSeedPaths() {
				add(p)
			}
		}
	}
	return out
}

func isResearchDocumentationDeliverable(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	task := collaboration.CollaborationTask{Title: content, Description: content}
	if !collaboration.NewDeliverablePolicy(task, "", nil).MarkdownOnly() {
		return false
	}
	lower := strings.ToLower(content)
	return strings.Contains(lower, "findings") ||
		strings.Contains(lower, "summariz") ||
		strings.Contains(lower, "summaris") ||
		strings.Contains(lower, "document") ||
		strings.Contains(lower, "research")
}

func taskContextPathsFromMessage(msg *protocol.Message) []string {
	if msg == nil || msg.Metadata == nil {
		return nil
	}
	raw, ok := msg.Metadata["task_context_paths"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, p := range v {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	case []interface{}:
		var out []string
		for _, item := range v {
			if p, ok := item.(string); ok {
				if p = strings.TrimSpace(p); p != "" {
					out = append(out, p)
				}
			}
		}
		return out
	default:
		return nil
	}
}

// AppendContentDeliverySeedFiles loads README/docs from the workspace for writing tasks.
func AppendContentDeliverySeedFiles(prompt *strings.Builder, workspacePath string, excludePaths map[string]bool) int {
	if workspacePath == "" {
		return 0
	}
	return appendWorkspaceSeedFiles(prompt, workspacePath, contentDeliveryDocSeeds, excludePaths,
		"\n=== PROJECT DOCS (content delivery) ===\n",
		"Loaded from the shared workspace for this writing request. "+
			"Ground the article or copy in these files — do NOT ask the user to paste README or project details.\n\n",
		"=== END PROJECT DOCS ===\n\n",
	)
}

func appendWorkspaceSeedFiles(
	prompt *strings.Builder,
	workspacePath string,
	candidates []string,
	excludePaths map[string]bool,
	header, intro, footer string,
) int {
	var loadedFiles []struct {
		path    string
		lang    string
		content string
	}
	totalSize := 0
	loaded := 0

	for _, p := range candidates {
		if loaded >= maxImplementationSeedFiles {
			break
		}
		if excludePaths != nil && excludePaths[p] {
			continue
		}
		resolved := filepath.Join(workspacePath, p)
		if _, err := os.Stat(resolved); err != nil {
			continue
		}
		content, _, err := ReadFileForPrompt(p, workspacePath)
		if err != nil {
			continue
		}
		if totalSize+len(content) > maxTotalFileSize {
			break
		}
		lang := inferLanguage(p)
		loadedFiles = append(loadedFiles, struct {
			path    string
			lang    string
			content string
		}{p, lang, content})
		totalSize += len(content)
		loaded++
		if excludePaths != nil {
			excludePaths[p] = true
		}
	}

	if len(loadedFiles) == 0 {
		return 0
	}

	prompt.WriteString(header)
	prompt.WriteString(intro)
	for _, f := range loadedFiles {
		prompt.WriteString(fmt.Sprintf("### %s (%s)\n```%s\n%s\n```\n\n", f.path, f.lang, f.lang, f.content))
	}
	prompt.WriteString(footer)
	return len(loadedFiles)
}

// AppendImplementationSeedFiles reads likely edit targets from the workspace on implement turns.
func AppendImplementationSeedFiles(prompt *strings.Builder, a *Agent, msg *protocol.Message, workspacePath string, agentType protocol.AgentType, excludePaths map[string]bool) int {
	if workspacePath == "" || msg == nil {
		return 0
	}
	// This is a prompt-context decision (whether to preload likely edit targets), not turn-action
	// routing, so the content-only themeImplementationRE/workspaceDirectiveRE signals stay live
	// here alongside the stamp-first userRequestsImplementationForMessage check.
	if !userRequestsImplementationForMessage(a, msg) && !workspaceDirectiveRE.MatchString(msg.Content) &&
		!themeImplementationRE.MatchString(msg.Content) {
		return 0
	}

	var history []*protocol.Message
	agentID := ""
	if a != nil {
		history = a.channelHistory(msg.Channel)
		agentID = a.Info.ID
	}
	exclude := mergeAppliedPathsIntoExclude(excludePaths, channelRecentlyAppliedFilePaths(history, msg.ID, agentID))
	focusPaths := taskContextPathsFromMessage(msg)
	researchDoc := isResearchDocumentationDeliverable(msg.Content)
	paths := implementationSeedCandidates(workspacePath, msg.Content, history, exclude, focusPaths, researchDoc)
	if len(paths) == 0 {
		return 0
	}

	return appendWorkspaceSeedFiles(prompt, workspacePath, paths, exclude,
		"\n=== REFERENCED FILES (implementation) ===\n",
		"Loaded from the shared workspace for this implementation request. "+
			"Use this ACTUAL code in [FILE_CHANGE] blocks — do not ask the user to paste these files.\n\n",
		"=== END REFERENCED FILES ===\n\n",
	)
}
