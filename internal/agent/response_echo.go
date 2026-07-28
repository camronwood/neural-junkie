package agent

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// buildImplementationContinuationRetryPrompt nudges the model to act instead of re-planning after approval.
func (a *Agent) buildImplementationContinuationRetryPrompt(msg *protocol.Message) string {
	system := fmt.Sprintf(
		"You are %s. The user approved your plan or asked you to continue implementing.\n"+
			"Reply in 1-3 sentences confirming what you will change, then emit [FILE_CHANGE] blocks or use propose_file_edit.\n"+
			"Do NOT ask design questions, request error details, or repeat an outline.\n",
		a.Info.Name,
	)
	user := strings.TrimSpace(msg.Content)
	return system + ai.SystemPromptSeparator + user
}

// buildEchoRetryPrompt is a last-resort prompt when the model echoed a prior user turn or summary.
func (a *Agent) buildEchoRetryPrompt(msg *protocol.Message) string {
	system := fmt.Sprintf(
		"You are %s. Reply ONLY to the user's latest message in 2-5 sentences.\n"+
			"Do not quote or repeat earlier user messages. Do not claim work is finished unless you did it now.\n",
		a.Info.Name,
	)
	user := strings.TrimSpace(msg.Content)
	return system + ai.SystemPromptSeparator + user
}

// looksLikeEchoOfPriorUserTurn reports replies that repeat an earlier user line or session-summary phrasing.
func looksLikeEchoOfPriorUserTurn(msg *protocol.Message, response string, history []*protocol.Message) bool {
	if msg == nil {
		return false
	}
	r := strings.TrimSpace(strings.ToLower(response))
	if r == "" {
		return false
	}
	if strings.HasPrefix(r, "the user wants") || strings.HasPrefix(r, "user wants to") {
		return true
	}
	if strings.Contains(r, "successfully added") && !strings.Contains(strings.ToLower(msg.Content), "successfully") {
		return true
	}
	cur := strings.TrimSpace(strings.ToLower(msg.Content))
	for _, h := range history {
		if h == nil || h.ID == msg.ID {
			continue
		}
		if !protocol.IsUserLikeSender(h.From) {
			continue
		}
		u := strings.TrimSpace(strings.ToLower(h.Content))
		if len(u) < 12 {
			continue
		}
		if u == cur {
			continue
		}
		if r == u {
			return true
		}
		if len(u) >= 20 && (strings.HasPrefix(r, u) || strings.Contains(r, u)) {
			return true
		}
	}
	return false
}

var reAskAfterAffirmationMarkers = []string{
	"could you provide",
	"could you share",
	"can you share",
	"would you like",
	"should we",
	"how should",
	"do you have any",
	"can you describe",
	"please provide",
	"please share",
	"share the",
	"share the current",
	"current implementation",
	"here's what we'll",
	"here's a brief",
	"let's proceed with defining",
	"let's define",
	"let's move forward with implementing",
	"brief plan",
	"brief outline",
}

// looksLikeReAskAfterAffirmation reports planning/question replies after the user approved or
// asked for code action. This is a response-quality self-check on the model's own generated
// output (retry gating), not a turn-action routing decision, so it stays out of scope for the
// stamp-first router; it now uses structural signals (a stamped implementation action, or a
// short non-question reply in a channel with recent implementation activity) instead of the
// deprecated userAffirmsPendingImplementation / userRequestsImplementation phrase heuristics.
func looksLikeReAskAfterAffirmation(msg *protocol.Message, response string, history []*protocol.Message) bool {
	if msg == nil {
		return false
	}
	content := strings.TrimSpace(msg.Content)
	r := strings.ToLower(strings.TrimSpace(response))
	if userRequestsCodeReview(content) {
		if strings.Contains(r, "specific file path") || strings.Contains(r, "path to the file") ||
			strings.Contains(r, "provide the path") || strings.Contains(r, "path you'd like me to review") {
			return true
		}
	}
	stampedImpl := messageStampedImplAction(msg)
	shortAffirmation := content != "" && len(strings.Fields(content)) <= 6 && !strings.Contains(content, "?")
	if !shortAffirmation && !stampedImpl {
		return false
	}
	if shortAffirmation {
		if !channelHasRecentImplementationActivity(history, msg.ID, "") {
			return false
		}
	}
	if r == "" || !strings.Contains(r, "?") {
		if !shortAffirmation {
			return false
		}
		for _, marker := range reAskAfterAffirmationMarkers {
			if strings.Contains(r, marker) {
				return true
			}
		}
		return false
	}
	for _, marker := range reAskAfterAffirmationMarkers {
		if strings.Contains(r, marker) {
			return true
		}
	}
	return false
}

// looksLikeAsksUserToPasteWorkspaceFiles reports replies that request pasted file content
// or falsely claim the shared workspace is unavailable.
func looksLikeAsksUserToPasteWorkspaceFiles(msg *protocol.Message, response string) bool {
	if msg == nil || !messageHasWorkspaceContext(msg) {
		return false
	}
	r := strings.ToLower(strings.TrimSpace(response))
	if r == "" {
		return false
	}
	for _, marker := range reAskAfterAffirmationMarkers {
		if strings.Contains(r, marker) {
			return true
		}
	}
	if strings.Contains(r, "paste") && (strings.Contains(r, "content") || strings.Contains(r, "file")) {
		return true
	}
	if strings.Contains(r, "share the content") || strings.Contains(r, "provide the content") {
		return true
	}
	// Follow-ups after a grounded turn sometimes claim the context window is empty.
	denyMarkers := []string{
		"context window is empty",
		"don't have visibility",
		"do not have visibility",
		"don't have your specific codebase",
		"do not have your specific codebase",
		"cannot give you a tailored",
		"can't give you a tailored",
		"codebase mounted",
		"not the actual file tree",
		"unless they are explicitly",
		"enable workspace sharing",
		"aren't visible in this",
		"are not visible in this",
		"files aren't visible",
		"files are not visible",
		"project files aren't visible",
		"project files are not visible",
		"specific project files aren't",
		"immediate context",
		"i'll assume we are building",
		"i will assume we are building",
		"don't have any specific project details",
		"do not have any specific project details",
		"don't have any specific project",
		"do not have any specific project",
		"no specific project details",
		"don't have the project details",
		"do not have the project details",
		"provide more information or context about the project",
		"provide more information or context",
		"if you could provide more information",
		"if you could provide more context",
	}
	for _, m := range denyMarkers {
		if strings.Contains(r, m) {
			return true
		}
	}
	return false
}

// looksLikeGroundingOnlyStub reports replies that only echoed the forced grounding
// opener (and maybe a hollow "Changes:" heading) without answering the user.
func looksLikeGroundingOnlyStub(response string) bool {
	r := strings.TrimSpace(response)
	if r == "" {
		return false
	}
	lower := strings.ToLower(r)
	if !strings.HasPrefix(lower, "grounding: i loaded") {
		return false
	}
	rest := ""
	if i := strings.IndexByte(r, '\n'); i >= 0 {
		rest = strings.TrimSpace(r[i+1:])
	}
	if rest == "" {
		return true
	}
	cleaned := strings.ToLower(rest)
	cleaned = strings.TrimSpace(strings.Trim(cleaned, "#*:-\t "))
	cleaned = strings.ReplaceAll(cleaned, "#", "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" || cleaned == "changes" || cleaned == "changes:" {
		return true
	}
	// "Changes: ###" / tiny leftovers after stripping markdown noise.
	if len(cleaned) < 48 && strings.HasPrefix(cleaned, "changes") {
		return true
	}
	return false
}

var prematureFileApplyClaimRE = regexp.MustCompile(`(?i)\b(created and saved|has been saved|has been created|written to disk|file is ready|successfully saved|saved to disk|file has been written)\b`)

// looksLikePrematureFileApplyClaim reports chat text that claims a file was applied before approval.
func looksLikePrematureFileApplyClaim(msg *protocol.Message, response string, history []*protocol.Message) bool {
	if msg == nil {
		return false
	}
	r := strings.ToLower(strings.TrimSpace(response))
	if r == "" || !prematureFileApplyClaimRE.MatchString(r) {
		return false
	}
	if strings.Contains(r, "proposal for your approval") ||
		strings.Contains(r, "submitted a file change proposal") ||
		strings.Contains(r, "file change proposal") {
		return false
	}
	targetPath := longestValidPathIn(DetectFilePaths(msg.Content))
	if targetPath == "" {
		targetPath = longestValidPathIn(DetectFilePaths(response))
	}
	scanned := 0
	for i := len(history) - 1; i >= 0 && scanned < 12; i-- {
		m := history[i]
		if m == nil || m.ID == msg.ID {
			continue
		}
		scanned++
		if m.FileChangeApproved() {
			if targetPath == "" {
				return false
			}
			if p, _ := m.Metadata[protocol.MetaFileChangePath].(string); p != "" &&
				(normalizeFileChangeRelPath(p) == targetPath || strings.Contains(p, targetPath)) {
				return false
			}
		}
		if m.Type == protocol.MessageTypeSystemInfo &&
			strings.Contains(m.Content, "Applied change") {
			if targetPath == "" {
				return false
			}
			if strings.Contains(m.Content, targetPath) {
				return false
			}
		}
	}
	return true
}

func (a *Agent) buildPrematureFileApplyRetryPrompt(msg *protocol.Message) string {
	system := fmt.Sprintf(
		"You are %s. You proposed a file change; do NOT claim it is saved or applied until the user approves.\n"+
			"Say a proposal was submitted and wait for approval.\n",
		a.Info.Name,
	)
	user := strings.TrimSpace(msg.Content)
	return system + ai.SystemPromptSeparator + user
}

// maybeRetryConversationalQuality retries replies that echo prior turns or re-ask after approval.
func (a *Agent) maybeRetryConversationalQuality(ctx context.Context, msg *protocol.Message, response string, history []*protocol.Message, eff ai.AIProvider) string {
	if msg == nil || eff == nil {
		return response
	}
	approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
	if looksLikeAsksUserToPasteWorkspaceFiles(msg, response) ||
		(messageHasWorkspaceContext(msg) && looksLikeGroundingOnlyStub(response)) {
		// Always retry when workspace is shared — including Assistant. Claiming the
		// project is unavailable after injection is a grounding failure, not an
		// implementation-session gate.
		retry, err := eff.GenerateResponse(approvalCtx, a.buildWorkspaceGroundedRetryPrompt(msg), nil)
		if err == nil && strings.TrimSpace(retry) != "" &&
			!looksLikeAsksUserToPasteWorkspaceFiles(msg, retry) &&
			!looksLikeGroundingOnlyStub(retry) {
			log.Printf("[%s] Workspace denial/hollow grounding detected; used grounded retry", a.Info.Name)
			return retry
		}
	}
	if looksLikeEchoOfPriorUserTurn(msg, response, history) {
		retry, err := eff.GenerateResponse(approvalCtx, a.buildEchoRetryPrompt(msg), historyToMessages(shortenedConversationWindow(history, 4)))
		if err == nil && strings.TrimSpace(retry) != "" &&
			!looksLikeEchoOfPriorUserTurn(msg, retry, nil) {
			log.Printf("[%s] Prior-turn echo detected; used echo retry", a.Info.Name)
			return retry
		}
	}
	if looksLikeReAskAfterAffirmation(msg, response, history) {
		retry, err := eff.GenerateResponse(approvalCtx, a.buildImplementationContinuationRetryPrompt(msg), nil)
		if err == nil && strings.TrimSpace(retry) != "" &&
			!looksLikeReAskAfterAffirmation(msg, retry, history) {
			log.Printf("[%s] Re-ask after approval detected; used continuation retry", a.Info.Name)
			return retry
		}
	}
	if looksLikePrematureFileApplyClaim(msg, response, history) {
		retry, err := eff.GenerateResponse(approvalCtx, a.buildPrematureFileApplyRetryPrompt(msg), nil)
		if err == nil && strings.TrimSpace(retry) != "" &&
			!looksLikePrematureFileApplyClaim(msg, retry, history) {
			log.Printf("[%s] Premature file-apply claim detected; used proposal retry", a.Info.Name)
			return retry
		}
	}
	if looksLikeIgnoresCodebaseAttachments(msg, response) {
		retry, err := eff.GenerateResponse(approvalCtx, a.buildCodebaseAttachmentRetryPrompt(msg), nil)
		if err == nil && strings.TrimSpace(retry) != "" &&
			!looksLikeIgnoresCodebaseAttachments(msg, retry) {
			log.Printf("[%s] @codebase reply ignored attachments; used grounded retry", a.Info.Name)
			return retry
		}
		if ans, ok := tryCodebaseReturnLiteralAnswer(msg); ok {
			log.Printf("[%s] @codebase return-literal fallback after ungrounded reply", a.Info.Name)
			return ans
		}
	}
	return response
}

var codebaseReturnLitRE = regexp.MustCompile(`(?m)\breturn\s+(-?\d+|true|false|"[^"\n]{1,64}"|'[^'\n]{1,64}')`)

func codebaseChunkContents(msg *protocol.Message) []string {
	if msg == nil || msg.Metadata == nil {
		return nil
	}
	raw, ok := msg.Metadata[MetadataPromptAttachments]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		fm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		typ, _ := fm["type"].(string)
		content, _ := fm["content"].(string)
		if strings.TrimSpace(content) == "" {
			continue
		}
		if typ != "" && typ != "codebase_chunk" {
			continue
		}
		out = append(out, content)
	}
	return out
}

func codebaseReturnLiterals(contents []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, c := range contents {
		for _, m := range codebaseReturnLitRE.FindAllStringSubmatch(c, -1) {
			if len(m) < 2 {
				continue
			}
			lit := strings.Trim(m[1], `"'`)
			if lit == "" || seen[lit] {
				continue
			}
			seen[lit] = true
			out = append(out, lit)
		}
	}
	return out
}

// looksLikeIgnoresCodebaseAttachments reports @codebase replies that invent behavior
// instead of citing return values / literals present in injected source chunks.
func looksLikeIgnoresCodebaseAttachments(msg *protocol.Message, response string) bool {
	if msg == nil || !codebaseMentionRE.MatchString(msg.Content) {
		return false
	}
	ensureCodebaseAttachments(msg)
	contents := codebaseChunkContents(msg)
	if len(contents) == 0 {
		return false
	}
	lits := codebaseReturnLiterals(contents)
	if len(lits) == 0 {
		return false
	}
	r := strings.TrimSpace(response)
	if r == "" {
		return true
	}
	for _, lit := range lits {
		if strings.Contains(r, lit) {
			return false
		}
	}
	return true
}

func ensureCodebaseAttachments(msg *protocol.Message) {
	if msg == nil || !codebaseMentionRE.MatchString(msg.Content) {
		return
	}
	if len(codebaseChunkContents(msg)) > 0 {
		return
	}
	MergeCodebaseAttachments(msg)
}

// tryCodebaseReturnLiteralAnswer answers "@codebase … return?" from attached return literals
// without another model call — used when the LLM invents behavior despite chunks.
func tryCodebaseReturnLiteralAnswer(msg *protocol.Message) (string, bool) {
	if msg == nil || !codebaseMentionRE.MatchString(msg.Content) {
		return "", false
	}
	q := strings.ToLower(msg.Content)
	if !strings.Contains(q, "return") {
		return "", false
	}
	ensureCodebaseAttachments(msg)
	contents := codebaseChunkContents(msg)
	lits := codebaseReturnLiterals(contents)
	if len(lits) == 0 {
		return "", false
	}
	lit := lits[0]
	sym := "it"
	if ids := codebaseIdentifierRE.FindAllString(msg.Content, -1); len(ids) > 0 {
		sym = ids[0]
	}
	path := ""
	if raw, ok := msg.Metadata[MetadataPromptAttachments].([]interface{}); ok {
		for _, item := range raw {
			fm, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			content, _ := fm["content"].(string)
			if strings.Contains(content, lit) {
				path, _ = fm["path"].(string)
				break
			}
		}
	}
	if path != "" {
		return fmt.Sprintf("`%s` returns %s (see `%s`).", sym, lit, path), true
	}
	return fmt.Sprintf("`%s` returns %s.", sym, lit), true
}

func (a *Agent) buildCodebaseAttachmentRetryPrompt(msg *protocol.Message) string {
	var chunks strings.Builder
	for i, c := range codebaseChunkContents(msg) {
		if i >= 4 {
			break
		}
		if len(c) > 1200 {
			c = c[:1200]
		}
		chunks.WriteString(c)
		chunks.WriteString("\n---\n")
	}
	system := fmt.Sprintf(
		"You are %s. This is an @codebase lookup. Answer ONLY from the source chunks below.\n"+
			"If a function returns a literal (number, bool, or string), state that exact value.\n"+
			"Do not invent UI/layout behavior that is not in the chunks.\n\n"+
			"SOURCE CHUNKS:\n%s",
		a.Info.Name,
		chunks.String(),
	)
	user := strings.TrimSpace(msg.Content)
	return system + ai.SystemPromptSeparator + user
}

// looksLikeIgnoresWorkspaceVisibility reports generic coding advice when the user asked what workspace the agent can see.
func looksLikeIgnoresWorkspaceVisibility(msg *protocol.Message, response string) bool {
	if msg == nil || !userAsksAboutWorkspaceVisibility(msg.Content) {
		return false
	}
	r := strings.ToLower(strings.TrimSpace(response))
	if r == "" {
		return false
	}
	visibilityMarkers := []string{
		"workspace context", "file tree", "open files", "i can see", "i have context",
		"project:", "context scope", "do not have workspace",
	}
	for _, m := range visibilityMarkers {
		if strings.Contains(r, m) {
			return false
		}
	}
	// Fell through to unrelated implementation spam.
	fakeMarkers := []string{
		"gin-gonic", "golang.org/x/themes", "go get ", "bootstrap", "install gin",
		"theme package", "steps to implement",
	}
	for _, m := range fakeMarkers {
		if strings.Contains(r, m) {
			return true
		}
	}
	return len(r) > 120
}
