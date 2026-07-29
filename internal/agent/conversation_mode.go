package agent

import (
	"regexp"
	"strings"

	semantic "github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	MetadataConversationMode = "conversation_mode"

	ConversationModeChat    = "chat"
	ConversationModeCode    = "code"
	ConversationModeCollab  = "collab"
	ConversationModeClarify = "clarify"
)

var (
	taskVerbRE = regexp.MustCompile(`(?i)\b(review|refactor|debug|fix|implement|compile|lint|test|patch|edit|change|update|add|remove|rewrite|optimize|trace|diff|analyze|analyse)\b`)
	strongTaskVerbRE = regexp.MustCompile(`(?i)\b(refactor|debug|implement|compile|lint|patch|rewrite|trace|diff)\b`)
	filePathRE = regexp.MustCompile("(?:^|[\\s\"'`(])([./]?(?:[a-zA-Z0-9_-]+/)+[a-zA-Z0-9_-]+\\.[a-zA-Z0-9]+)")
	greetingRE = regexp.MustCompile(`(?i)^(?:@\w+\s+)?(?:hi|hello|hey|yo|sup|what'?s up(?:\s+(?:every\s+)?(?:body|one|folks|all|y'?all|everyone))?|howdy|good (?:morning|afternoon|evening)|thanks|thank you|ok|okay|nice|cool)[!.?\s]*$`)
	hereMentionRE = regexp.MustCompile(`(?i)(?:^|[\s])@(?:here|channel|everyone)\b`)
	socialPingRE = regexp.MustCompile(`(?i)^(?:@(?:here|channel|everyone|\w+)\s+)*(?:what'?s\s+going\s+on|what\s+is\s+going\s+on|whats?\s+up(?:\s+(?:every\s+)?(?:body|one|folks|all|y'?all|everyone|\w+))?|how'?s\s+it\s+going|how\s+are\s+things|anyone\s+around|you\s+there|status\??|ping)[?!.\s]*$`)
	leadingMentionsRE = regexp.MustCompile(`(?i)^(?:\s*@(?:here|channel|everyone|\w+)\b)+\s*`)
	scanToolRE = regexp.MustCompile(`(?i)\b(summarize_scan_summary|summarize_scan_analysis|scan analysis|scan summary)\b`)
	editorOpenRE = regexp.MustCompile(`(?i)\b(file i have open|open in my editor|in my editor|editor open|active tab|active file|have open)\b`)
)

func stripLeadingMentions(content string) string {
	return strings.TrimSpace(leadingMentionsRE.ReplaceAllString(content, ""))
}

func hasHereOrChannelMention(content string) bool {
	return hereMentionRE.MatchString(content)
}

func hasStrongCodeTaskSignals(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if strongTaskVerbRE.MatchString(content) {
		return true
	}
	if filePathRE.MatchString(content) {
		return true
	}
	if strings.Contains(strings.ToLower(content), "@codebase") {
		return true
	}
	return hasScanOrEditorTaskSignals(content)
}

// isSocialOrStatusPing reports casual @here / vibe-check messages that should stay in chat mode.
func isSocialOrStatusPing(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if hasStrongCodeTaskSignals(content) {
		return false
	}
	stripped := stripLeadingMentions(content)
	if stripped == "" {
		return hasHereOrChannelMention(content)
	}
	if greetingRE.MatchString(content) || greetingRE.MatchString(stripped) {
		return true
	}
	if socialPingRE.MatchString(content) || socialPingRE.MatchString(stripped) {
		return true
	}
	if hasHereOrChannelMention(content) && len(stripped) <= 80 && !hasCodeTaskSignals(stripped) {
		return true
	}
	if semantic.LooksLikePresenceCheck(content) || semantic.LooksLikePresenceCheck(stripped) {
		return true
	}
	return false
}

func appendHereOrSocialPingPrompt(system *strings.Builder) {
	system.WriteString("=== CHANNEL PING ===\n")
	system.WriteString("This is a casual @here/@channel (or short status) ping. Reply in 1-3 sentences.\n")
	system.WriteString("Do not inventory the repo, invent file tours, or use tools unless the user explicitly asks for work.\n\n")
}

// ConversationModeFromMessage returns chat/code/collab from outbound metadata (empty if unset).
func ConversationModeFromMessage(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	mode, _ := msg.Metadata[MetadataConversationMode].(string)
	return strings.TrimSpace(strings.ToLower(mode))
}

func hasScanOrEditorTaskSignals(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if scanToolRE.MatchString(content) {
		return true
	}
	if editorOpenRE.MatchString(content) {
		return true
	}
	return userRequestsEditorDocumentReview(content)
}

func hasCodeTaskSignals(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if isWeakImplementationAffirmation(content) {
		return false
	}
	if userAffirmsPendingImplementation(content) {
		return true
	}
	if hasScanOrEditorTaskSignals(content) {
		return true
	}
	if strings.Contains(strings.ToLower(content), "@codebase") {
		return true
	}
	if taskVerbRE.MatchString(content) {
		return true
	}
	if filePathRE.MatchString(content) {
		return true
	}
	if strings.Contains(content, "`") {
		return true
	}
	return false
}

func inferConversationModeFromMessage(msg *protocol.Message, channelType protocol.ChannelType) string {
	// Phrase/inference path is emergency rollback when no stamped turn decision exists.
	// Callers must prefer EffectiveConversationMode, which short-circuits on ExtractTurnDecision.
	if msg == nil {
		return ConversationModeChat
	}
	if channelType == protocol.ChannelTypeCollaboration {
		return ConversationModeCollab
	}
	content := strings.TrimSpace(msg.Content)
	if isSocialOrStatusPing(content) {
		return ConversationModeChat
	}
	if hasCodeTaskSignals(content) {
		return ConversationModeCode
	}
	if greetingRE.MatchString(content) {
		return ConversationModeChat
	}
	if strings.Contains(content, "?") && len(content) >= 20 && !hasCodeTaskSignals(content) {
		return ConversationModeChat
	}
	if scope := ContextScopeFromMessage(msg); scope == ContextScopeNone && !hasCodeTaskSignals(content) {
		return ConversationModeChat
	}
	return ConversationModeCode
}

// EffectiveConversationMode returns metadata mode or server-side inference.
func EffectiveConversationMode(msg *protocol.Message, channelType protocol.ChannelType) string {
	if channelType == protocol.ChannelTypeCollaboration {
		return ConversationModeCollab
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		switch decision.Action {
		case semantic.ActionInspect, semantic.ActionDebug, semantic.ActionEdit, semantic.ActionRun, semantic.ActionContinue:
			return ConversationModeCode
		default:
			return ConversationModeChat
		}
	}
	if mode := ConversationModeFromMessage(msg); mode != "" {
		return mode
	}
	return inferConversationModeFromMessage(msg, channelType)
}

// ToolingConversationMode maps clarify → chat so agents ask before using tools.
func ToolingConversationMode(msg *protocol.Message, channelType protocol.ChannelType) string {
	mode := EffectiveConversationMode(msg, channelType)
	if mode == ConversationModeClarify {
		return ConversationModeChat
	}
	return mode
}

func appendConversationModeClarifyPrompt(system *strings.Builder) {
	system.WriteString("=== INTENT CLARIFICATION REQUIRED ===\n")
	system.WriteString("It is unclear whether the user wants a conversational answer or hands-on code/workspace work.\n")
	system.WriteString("Ask one short clarifying question in chat before using tools, editing files, or running commands.\n")
	system.WriteString("Offer clear choices, for example: discuss/explain vs edit/run in the workspace.\n")
	system.WriteString("Do not call tools or propose file edits until they choose.\n\n")
}

func appendConversationModeChatPrompt(system *strings.Builder) {
	system.WriteString("=== CHAT MODE (ADVISORY) ===\n")
	system.WriteString("Answer in conversation only. Do not edit files, run workspace tools, or dump repo findings.\n")
	system.WriteString("Retain named constraints from earlier turns (component names, section placement, accessibility rules).\n")
	system.WriteString("When summarizing a multi-turn design, restate those constraints explicitly in your reply.\n")
	system.WriteString("When the user critiques code you previously showed, provide a corrected snippet that fixes every bug they named — shallow relabeling or partial fixes fail the request.\n\n")
}

var codeCritiqueFollowUpRE = regexp.MustCompile(`(?i)(?:\bhold on\b|\bdead code\b|\bdead ref\b|\bnever assigned\b|\bconditional hook\b|\bhook ordering\b|\byou're calling\b|\byou never\b|\byou didn't\b|\byou haven't\b|\bfights itself\b|\bstill just prose\b|\bstrategy memo\b|\bshow me the actual\b|\bconflicting\b.{0,48}\b(?:handler|listener|click)\b|\bshow me the corrected\b|\bfix the\b.{0,64}\b(?:logic|handler|bug|effect|code|ref|hook|ordering)\b)`)

var concreteCodeRequestRE = regexp.MustCompile(`(?i)(?:\bactual implementation\b|\bactual mechanics\b|\bshow me the actual\b|\breal code\b|\bthe works\b|\bstill just prose\b|\bstrategy memo\b|\bnot just\b.{0,40}\b(?:library|prose|strategy|memo|advice|cheerleading|punt)\b|\bnot an answer\b|\bcode or specifics\b|\bconcrete steps/code\b|\bwhat(?:'s| is) the actual implementation\b)`)

// looksLikeCodeCritiqueFollowUp reports when the user is correcting bugs in prior assistant code.
func looksLikeCodeCritiqueFollowUp(content string) bool {
	return codeCritiqueFollowUpRE.MatchString(strings.TrimSpace(content))
}

// looksLikeConcreteCodeRequest reports when the user wants copy-paste code, not a strategy overview.
func looksLikeConcreteCodeRequest(content string) bool {
	return concreteCodeRequestRE.MatchString(strings.TrimSpace(content))
}

func appendCodeCritiqueFollowUpPrompt(system *strings.Builder, msg *protocol.Message) {
	if msg == nil || !looksLikeCodeCritiqueFollowUp(msg.Content) {
		return
	}
	system.WriteString("=== CODE CORRECTION FOLLOW-UP ===\n")
	system.WriteString("The user is pointing out bugs in code you previously showed. Address EVERY specific issue they named.\n")
	system.WriteString("Do not claim a fix unless the code demonstrably implements it: assign refs before reading them in cleanup, ")
	system.WriteString("avoid event-target guards that break bubbling (keydown from focused children still reaches the modal listener), ")
	system.WriteString("and use a single intentional focus target (first focusable, not modal shell then child).\n")
	system.WriteString("Call hooks unconditionally before any `if (!isOpen) return null` — capture `document.activeElement` for focus restore; ")
	system.WriteString("do not use `getElementById` hacks for trigger refs.\n")
	system.WriteString("Keep Tab-cycling focus-trap logic when the user asked for a focus trap — do not drop it for comments.\n")
	system.WriteString("In React hooks, never assign to a boolean parameter inside a handler (e.g. `isOpen = false`); ")
	system.WriteString("call an `onClose` callback or parent setter on Escape — never `console.log` placeholders.\n")
	system.WriteString("Show the complete corrected snippet they asked for.\n\n")
}

func appendModalAccessibilityGapFollowUpPrompt(system *strings.Builder, msg *protocol.Message) {
	if msg == nil || !modalAccessibilityGapFollowUp(msg.Content) {
		return
	}
	system.WriteString("=== MODAL ACCESSIBILITY GAP-FILL ===\n")
	system.WriteString("The user is filling gaps in a modal implementation you showed. Address every named issue in working code.\n")
	system.WriteString("querySelectorAll does NOT filter by computed style — a display:none button still matches; filter afterward with ")
	system.WriteString("`.filter(el => el.offsetParent !== null)` (or `getClientRects().length`).\n")
	system.WriteString("Put `tabIndex={-1}` on the dialog element in JSX so `.focus()` works when no focusable children exist — do not use setAttribute at runtime.\n")
	system.WriteString("When using a document-level keydown listener, guard the ENTIRE handler (Escape AND Tab) with ")
	system.WriteString("`if (!dialogRef.current.contains(document.activeElement)) return` so nested modals/popovers receive Escape/Tab first.\n")
	system.WriteString("Add `aria-labelledby` pointing at a visible title element (`<h2 id=\"modal-title\">`).\n\n")
}

func appendConcreteCodeRequestPrompt(system *strings.Builder, msg *protocol.Message) {
	if msg == nil || !looksLikeConcreteCodeRequest(msg.Content) {
		return
	}
	system.WriteString("=== CONCRETE CODE REQUEST ===\n")
	system.WriteString("The user asked for working code, not a strategy memo or library recommendation.\n")
	system.WriteString("Lead with a complete, copy-paste-ready code block (hook/component + usage). ")
	system.WriteString("Keep intro prose to 1-2 sentences max.\n")
	system.WriteString("For modal focus traps: accept an onClose callback, call it on Escape (never console.log placeholders), ")
	system.WriteString("intercept Tab/Shift+Tab ONLY at the first/last focusable boundary (`document.activeElement === first|last`) — let Tab move normally between inner elements, ")
	system.WriteString("restore focus in effect cleanup via captured `document.activeElement` (assign to a ref, then `ref.current?.focus()` in the effect cleanup — do not claim restore without that), ")
	system.WriteString("guard the effect with `if (!isOpen) return` before querying `modalRef.current` when the component returns null while closed, ")
	system.WriteString("call hooks before early returns, and do not reassign hook boolean parameters.\n\n")
}

var (
	repoFactAskRE = regexp.MustCompile(`(?i)\b(?:health\s*check|liveness|readiness|heartbeat|http\s+path|api\s+path|endpoint|what\s+path|which\s+path|hit\s+to\s+check|route\s+(?:is|to|for))\b`)
	repoFactChallengeRE = regexp.MustCompile(`(?i)(?:\bmade-up\b|\bmaking\s+up\b|\bgeneric\s+guess\b|\bactually\s+a\s+real\s+path\b|\bin\s+this\s+repo\b|\bfake\b|\bwould\s+that\s+be\s+correct\b|\bor\s+were\s+you\b|\binvent(?:ed|ing)\b)`)

	repoFactSeedFilePaths = []string{
		"cmd/server/health_settings_handlers.go",
		"cmd/server/routes.go",
	}
)

// looksLikeRepoFactAsk reports questions about repo-specific routes, health checks, or HTTP paths.
func looksLikeRepoFactAsk(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if !repoFactAskRE.MatchString(content) {
		return false
	}
	return strings.Contains(content, "/") || strings.Contains(content, "health") ||
		strings.Contains(content, "path") || strings.Contains(content, "endpoint") ||
		strings.Contains(content, "route")
}

// looksLikeRepoFactChallengeFollowUp reports when the user challenges invented repo paths/endpoints.
func looksLikeRepoFactChallengeFollowUp(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if !repoFactChallengeRE.MatchString(content) {
		return false
	}
	return strings.Contains(content, "/") || strings.Contains(content, "path") ||
		strings.Contains(content, "endpoint") || strings.Contains(content, "health")
}

func appendRepoFactGroundingPrompt(system *strings.Builder, msg *protocol.Message) {
	if msg == nil || (!looksLikeRepoFactAsk(msg.Content) && !looksLikeRepoFactChallengeFollowUp(msg.Content)) {
		return
	}
	system.WriteString("=== REPO FACT GROUNDING ===\n")
	system.WriteString("The user is asking about HTTP paths/routes/endpoints in THIS repository.\n")
	system.WriteString("Do NOT invent paths, handler file locations, or code samples from REST conventions or memory.\n")
	system.WriteString("If you have not verified a route in workspace context or tool output, say you do not know yet — do not guess or dress guesses up as confirmed facts.\n")
	system.WriteString("When the user proposes an arbitrary or joke path (e.g. /api/v9/quantum-health), do NOT call it \"correct\" just because it could be registered — health checks must match this repo's existing probes, tests, monitoring configs, and docs.\n")
	system.WriteString("Prefer honest uncertainty, or grep/read_file on cmd/ and internal/ when tools or workspace files are available.\n\n")
}

// appendRepoFactSeedFiles loads health/route source files when the user asks about repo HTTP paths.
func appendRepoFactSeedFiles(prompt *strings.Builder, workspacePath string) int {
	if prompt == nil || strings.TrimSpace(workspacePath) == "" {
		return 0
	}
	return appendWorkspaceSeedFiles(prompt, workspacePath, repoFactSeedFilePaths, nil,
		"\n=== REPO ROUTE SOURCE (health/endpoints) ===\n",
		"Loaded from the shared workspace. Cite only paths and handlers present here — do not invent routes from REST conventions.\n\n",
		"=== END REPO ROUTE SOURCE ===\n\n",
	)
}

func appendGitInspectPrompt(system *strings.Builder, msg *protocol.Message) {
	if msg == nil || !semantic.LooksLikeGitInspectRequest(msg.Content) {
		return
	}
	decision, ok := protocol.ExtractTurnDecision(msg)
	if ok && decision.Action != semantic.ActionInspect && decision.Action != semantic.ActionDebug {
		return
	}
	system.WriteString("=== GIT INSPECTION (required this turn) ===\n")
	system.WriteString("The user asked to use git (status/history/diff/known-good). Do NOT speculate from memory.\n")
	system.WriteString("Before answering, call run_command with read-only git (start with `git status --short`, then `git log --oneline -n 20` and/or `git diff` / `git show` as needed).\n")
	system.WriteString("Ground the reply in that command output. Chat-only guesses about working configs or history are a failure.\n\n")
}

func ContextScopeFromMessage(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	scope, _ := msg.Metadata[MetadataContextScope].(string)
	return strings.TrimSpace(strings.ToLower(scope))
}

func shouldMaintainSessionSummary(chType protocol.ChannelType, channel string) bool {
	if chType == protocol.ChannelTypeDM || chType == protocol.ChannelTypeCustom ||
		chType == protocol.ChannelTypePublic || chType == protocol.ChannelTypeCollaboration {
		return true
	}
	channel = strings.TrimSpace(strings.ToLower(channel))
	return strings.HasPrefix(channel, "dm-")
}
