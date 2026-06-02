package agent

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

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
