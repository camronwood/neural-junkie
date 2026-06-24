package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const maxRecentlyAppliedFilePaths = 8

var (
	implementationAffirmRE = regexp.MustCompile(`(?i)\b(approved|approve(d| it)?|keep going|please continue|continue(?: with (it|this|that|the work))?|looks good|that sounds good|sounds good|go[- ]?ahead|goadhead|do it( now)?|yes please|please do|proceed|make (the |those )?(changes|them)|apply (that|it|your plan)|do that now|ok please|sure,?\s*please|let's do it|please implement|sounds good[,!]?\s*(go|do)|that works[,!]?\s*(go|do)?|you can (start|begin|proceed)|yeah go ahead|yes[,!]?\s*(keep going|that sounds good|use that|please))\b`)
	weakImplementationAffirmRE = regexp.MustCompile(`(?i)^(?:@\w+\s+)?(?:ok|okay|looks good|that works|sounds good|nice|great|cool|perfect)[!.?\s]*$`)
	themeImplementationRE  = regexp.MustCompile(`(?i)(?:\b(theme|themes|dark[/ ]?light|dark mode|light mode|ui theme)\b.{0,64}\b(add(?:ing)?|implement(?:ing)?|build(?:ing)?|wire|toggle|finish)\b|\b(add(?:ing)?|implement(?:ing)?|build(?:ing)?|wire|finish)\b.{0,64}\b(theme|themes|ui theme|dark mode|light mode|font size)\b)`)
	implementTypoRE        = regexp.MustCompile(`(?i)\bimpl[e]?ment\b`)
	workspaceDirectiveRE   = regexp.MustCompile(`(?i)\b(use|read|from)\s+(the\s+)?(open\s+)?workspace\b`)
	bootErrorIntentRE           = regexp.MustCompile(`(?i)(not booting|won't boot|does not boot|failed to scan|esbuild|✘\s*\[ERROR\]|\[ERROR\].*Expected|make start-all|vite dev|syntax error|white screen|blank screen|exit_code=)`)
	implementationStatusCheckRE = regexp.MustCompile(`(?i)^(?:@\w+\s+)?(?:is it fixed|did (?:that|it) fix|does it work(?: now)?|is it working(?: now)?|still broken|still not (?:booting|working)|working now)\??[!.?\s]*$`)
	destructiveCommandRE       = regexp.MustCompile(`(?i)\brm\s+-rf\b|\brm\s+-r\b|\brmdir\s+/\b|>\s*/dev/`)
	contentDeliveryRE      = regexp.MustCompile(`(?i)\b(linkedin|blog post|blog article|article about|write (?:me )?(?:a |an )?article|marketing copy|press release|social media post|whitepaper|writeup|newsletter)\b`)
	fileExportRE           = regexp.MustCompile(`(?i)\b(store (?:that|it|in|the)|save (?:it|as|in|the)|fill (?:the file|.* with)|create (?:that |the )?file|please create (?:that |the )?file|write (?:it |that ).*(?:file|\.md)|markdown file)\b`)
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

// userRequestsImplementation reports coding/build asks (themes, features, fixes) where the
// user expects [FILE_CHANGE] deliverables, not a codebase overview.
func userRequestsImplementation(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	if userRequestsCodeReview(content) {
		return false
	}
	if implementTypoRE.MatchString(lower) || themeImplementationRE.MatchString(lower) {
		return true
	}
	if userAffirmsPendingImplementation(content) {
		return false // continuation alone is not enough without channel history
	}
	if workspaceDirectiveRE.MatchString(lower) {
		return true
	}
	phrases := []string{
		"please implement", "implement that", "implement the",
		"implement this", "build this", "build out", "code this", "code it",
		"ship ", "apply the plan", "apply your plan", "make the changes",
		"make the change", "do the implementation", "actually implement",
		"write the code", "add the code", "add light", "add dark",
		"theme support", "light/dark", "dark/light", "dark mode", "light mode",
		"wire up", "hook up", "under settings", "settings page", "settings modal",
		"font size", "pick up where", "finish that work", "finish the work",
		"review the code for issues", "review the code for bugs", "review and fix",
		"code for issues", "not working", "doesn't work",
		"does not work", "does not seem to be working", "broken", "fix the app", "debug this", "troubleshoot",
		"blank screen", "white screen", "can you fix", "not booting", "won't boot", "will not boot",
	}
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// userRequestsContentDelivery reports writing/marketing tasks that should not trigger file-edit fallback.
func userRequestsContentDelivery(content string) bool {
	return contentDeliveryRE.MatchString(strings.TrimSpace(content))
}

// userRequestsFileExportForMessage includes explicit composer export mode.
func userRequestsFileExportForMessage(msg *protocol.Message) bool {
	if msg != nil && msg.IdeEditorModeIsExport() {
		return true
	}
	if msg == nil {
		return false
	}
	return userRequestsFileExport(msg.Content)
}

// userRequestsFileExport reports save/store/create/fill markdown file asks.
// Deprecated: prefer explicit composer export mode (IdeEditorModeIsExport).
func userRequestsFileExport(content string) bool {
	text := strings.TrimSpace(content)
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	if fileExportRE.MatchString(lower) {
		return true
	}
	hasFileTarget := strings.Contains(lower, ".md") || strings.Contains(lower, "markdown file") ||
		strings.Contains(lower, "the file")
	hasExportVerb := strings.Contains(lower, "store") || strings.Contains(lower, "save") ||
		strings.Contains(lower, "create") || strings.Contains(lower, "fill")
	return hasFileTarget && hasExportVerb
}

// isBareWorkspaceDirective reports short "use the workspace" style messages without a code deliverable.
func isBareWorkspaceDirective(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if !workspaceDirectiveRE.MatchString(lower) {
		return false
	}
	stripped := workspaceDirectiveRE.ReplaceAllString(lower, "")
	stripped = bareWorkspaceWrapperRE.ReplaceAllString(stripped, "")
	stripped = strings.TrimSpace(strings.Trim(stripped, "?.!,"))
	if stripped == "" {
		return true
	}
	if implementTypoRE.MatchString(stripped) || themeImplementationRE.MatchString(stripped) {
		return false
	}
	codeVerbs := []string{
		"implement", "fix", "debug", "build", "theme", "settings", "modal",
		"add ", "wire", "patch", "refactor", "broken", "not working",
	}
	for _, v := range codeVerbs {
		if strings.Contains(stripped, v) {
			return false
		}
	}
	return len(stripped) < 40
}

// userAffirmsPendingImplementation reports short follow-ups after an implementation ask.
func userAffirmsPendingImplementation(content string) bool {
	return implementationAffirmRE.MatchString(strings.TrimSpace(content))
}

// isWeakImplementationAffirmation reports bare acknowledgements ("ok", "looks good") that are
// not explicit approval to ship file changes.
func isWeakImplementationAffirmation(content string) bool {
	return weakImplementationAffirmRE.MatchString(strings.TrimSpace(content))
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
			strings.Contains(body, "i will ") ||
			strings.Contains(body, "i'll ") ||
			strings.Contains(body, "plan:") {
			return true
		}
		return false
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
			body := strings.ToLower(m.Content)
			if strings.Contains(body, "implementation session complete") ||
				strings.Contains(body, "proposals submitted for approval") ||
				strings.Contains(body, "finished without file changes") {
				return true
			}
		}
		if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
			return true
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
		if userRequestsImplementation(m.Content) {
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
		if userRequestsImplementation(m.Content) {
			return true
		}
	}
	return false
}

// userRequestsImplementationForMessage includes affirmation follow-ups in the same channel thread.
func userRequestsImplementationForMessage(a *Agent, msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if userRequestsImplementation(msg.Content) || userRequestsFileExportForMessage(msg) {
		return true
	}
	if userRequestsImplementationStatusCheck(msg.Content) && a != nil &&
		channelHasRecentImplementationActivity(a.channelHistory(msg.Channel), msg.ID, a.Info.ID) {
		return true
	}
	if a != nil {
		history := a.channelHistory(msg.Channel)
		if msg.FileChangeApproved() && shouldSkipAgentResponseOnFileExportApproval(a, msg) {
			return false
		}
		if channelHasRecentFileChangeApproval(history, msg.ID, a.Info.ID) &&
			(channelHasRecentImplementationAsk(history, msg.ID) ||
				channelHasRecentImplementationActivity(history, msg.ID, a.Info.ID)) {
			if channelHasRecentFileExportAsk(history, msg.ID) &&
				!channelHasRecentCodeImplementationAsk(history, msg.ID) {
				return false
			}
			return true
		}
	}
	if a == nil || !userAffirmsPendingImplementation(msg.Content) {
		return false
	}
	history := a.channelHistory(msg.Channel)
	return affirmationContinuesImplementation(history, msg.ID, a.Info.ID, msg.Content)
}

// ShouldForceSessionSummaryRefresh reports user turns that should refresh a stale session summary.
func ShouldForceSessionSummaryRefresh(content string) bool {
	if userAffirmsPendingImplementation(content) {
		return true
	}
	if userRequestsCodeReview(content) {
		return true
	}
	if userRequestsImplementation(content) {
		return true
	}
	if messageHasBootOrBuildError(content) {
		return true
	}
	if userRequestsImplementationStatusCheck(content) {
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
	return ShouldForceSessionSummaryRefresh(msg.Content)
}

// ScrubStaleSessionSummary removes summary bullets contradicted by a boot/error log in the transcript.
func ScrubStaleSessionSummary(summary, transcript string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" || !messageHasBootOrBuildError(transcript) {
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
// in constraint text).
func shouldProactiveScanWorkspace(content string) bool {
	if userRequestsImplementation(content) {
		return true
	}
	return shouldInjectWorkspaceCode(content)
}

// shouldProactiveScanWorkspaceForMessage includes affirmation follow-ups in the same thread.
func shouldProactiveScanWorkspaceForMessage(a *Agent, msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if a != nil && userRequestsImplementationForMessage(a, msg) {
		return true
	}
	return shouldProactiveScanWorkspace(msg.Content)
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
	prompt.WriteString("Use only dependencies already in package.json / the repo — do not invent packages.\n")
	prompt.WriteString("Keep conversational text short (2-4 sentences); put code in [FILE_CHANGE], not long fenced dumps.\n")
	if a != nil && a.hasWorkspaceTools() {
		prompt.WriteString("When workspace tools are available, prefer propose_file_edit over [FILE_CHANGE] text blocks.\n")
	}
	prompt.WriteString("Do NOT ask the user to paste or share file contents when REFERENCED FILES or WORKSPACE SOURCE FILES appear below — read them and emit [FILE_CHANGE].\n")
	prompt.WriteString("Only ask for a path if a required file is missing from every context section.\n")
	if a != nil {
		history := a.channelHistory(msg.Channel)
		if messageImpliesBootFix(msg.Content, history) || messageHasBootOrBuildError(msg.Content) {
			prompt.WriteString("Boot/build fix: use read_file on paths from the error log, then fix or delete conflicting files. ")
			prompt.WriteString("Verify with npm run build (or go test ./...) — do NOT run npm install; dependencies are already in the repo.\n")
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

// messageHasBootOrBuildError reports Vite/esbuild/boot failure logs in message text.
func messageHasBootOrBuildError(content string) bool {
	return bootErrorIntentRE.MatchString(strings.TrimSpace(content))
}

// userRequestsImplementationStatusCheck reports short follow-ups after a fix attempt.
func userRequestsImplementationStatusCheck(content string) bool {
	return implementationStatusCheckRE.MatchString(strings.TrimSpace(content))
}

// tryImplementationStatusCheckShortcut answers "is it fixed?" from recent session context
// without calling the LLM (avoids slow re-runs during boot-fix follow-ups).
func (a *Agent) tryImplementationStatusCheckShortcut(msg *protocol.Message) (string, bool) {
	if a == nil || msg == nil || !userRequestsImplementationStatusCheck(msg.Content) {
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
func implementationSeedCandidates(workspacePath string, content string, history []*protocol.Message, exclude map[string]bool) []string {
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
	if messageImpliesBootFix(content, history) {
		for _, p := range []string{"src/App.js", "src/App.tsx", "src/main.tsx", "src/main.ts", "package.json"} {
			add(p)
		}
	}
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath != "" {
		if manifest := DetectStackManifest(workspacePath); manifest != nil {
			for _, p := range manifest.ImplementationSeedPaths() {
				add(p)
			}
		}
	}
	return out
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
	if !userRequestsImplementationForMessage(a, msg) && !workspaceDirectiveRE.MatchString(msg.Content) {
		return 0
	}

	var history []*protocol.Message
	agentID := ""
	if a != nil {
		history = a.channelHistory(msg.Channel)
		agentID = a.Info.ID
	}
	exclude := mergeAppliedPathsIntoExclude(excludePaths, channelRecentlyAppliedFilePaths(history, msg.ID, agentID))
	paths := implementationSeedCandidates(workspacePath, msg.Content, history, exclude)
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
