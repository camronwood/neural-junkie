package agent

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// RunDevAgentTurn runs one editor-agent turn (tools + optional file proposal).
func RunDevAgentTurn(
	ctx context.Context,
	a *Agent,
	channel string,
	userMsg *protocol.Message,
) (response string, proposed bool, err error) {
	resp, err := a.GenerateResponse(ctx, userMsg)
	if err != nil {
		return "", false, err
	}
	cleaned, ok, err := a.maybeSubmitFileChangeFromResponse(ctx, resp, channel, userMsg)
	return cleaned, ok, err
}

// EditorAgentChannelID returns the hub channel for a workspace editor agent.
func EditorAgentChannelID(workspaceID string) string {
	var b strings.Builder
	b.WriteString("editor-")
	for _, r := range workspaceID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}
