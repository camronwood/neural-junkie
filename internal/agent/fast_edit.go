package agent

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// RunFastEdit runs one specialist turn and submits file-change proposals when present.
func RunFastEdit(ctx context.Context, a *Agent, channel string, userMsg *protocol.Message) (response string, proposed bool, err error) {
	resp, err := a.GenerateResponse(ctx, userMsg)
	if err != nil {
		return "", false, err
	}
	cleaned, ok, err := a.maybeSubmitFileChangeFromResponse(resp, channel, userMsg)
	return cleaned, ok, err
}
