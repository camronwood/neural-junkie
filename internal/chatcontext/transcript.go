package chatcontext

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// FormatTranscript renders recent channel messages for session-summary LLM prompts.
func FormatTranscript(messages []*protocol.Message, max int) string {
	filtered := FilterForLLM(messages, "", max)
	if len(filtered) == 0 {
		return ""
	}
	var b strings.Builder
	for _, m := range filtered {
		role := "Agent"
		if protocol.IsUserLikeSender(m.From) {
			role = "User"
		} else if m.From.Name != "" {
			role = m.From.Name
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if len(content) > 800 {
			content = content[:800] + "…"
		}
		fmt.Fprintf(&b, "%s: %s\n", role, content)
	}
	return strings.TrimSpace(b.String())
}
