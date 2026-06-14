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

// looksLikeReAskAfterAffirmation reports planning/question replies after the user approved or asked for code action.
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
	if !userAffirmsPendingImplementation(content) && !userRequestsImplementation(content) {
		return false
	}
	if userAffirmsPendingImplementation(content) {
		if !channelHasRecentImplementationActivity(history, msg.ID, "") {
			return false
		}
	}
	if r == "" || !strings.Contains(r, "?") {
		if !userAffirmsPendingImplementation(content) {
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
// when the user already shared a workspace.
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
	if looksLikeAsksUserToPasteWorkspaceFiles(msg, response) {
		if a.Info.Type == protocol.AgentTypeAssistant && !assistantAllowsImplementationSession(a, msg) {
			return response
		}
		retry, err := eff.GenerateResponse(approvalCtx, a.buildWorkspaceGroundedRetryPrompt(msg), nil)
		if err == nil && strings.TrimSpace(retry) != "" &&
			!looksLikeAsksUserToPasteWorkspaceFiles(msg, retry) {
			log.Printf("[%s] Paste-request detected with shared workspace; used grounded retry", a.Info.Name)
			return retry
		}
	}
	if looksLikeEchoOfPriorUserTurn(msg, response, history) {
		retry, err := eff.GenerateResponse(approvalCtx, a.buildEchoRetryPrompt(msg), nil)
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
	return response
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
