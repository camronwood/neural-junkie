package agent

import (
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// tryDenyDestructiveImplementationSession declines agent-mode implementation turns that
// ask for destructive shell cleanup (rm -rf, etc.) before the bounded edit loop runs.
// This guarantees implementation_session_outcome metadata even when the full session
// gate (shouldRunImplementationSession) would skip the loop.
func (a *Agent) tryDenyDestructiveImplementationSession(msg *protocol.Message) (string, map[string]interface{}, bool) {
	if a == nil || msg == nil || !userRequestsDestructiveCommand(msg.Content) {
		return "", nil, false
	}
	if !msg.ImplementationSession() && msg.IdeEditorMode() != "agent" && !msg.IdeEditorModeIsExport() {
		return "", nil, false
	}
	if a.Info.Type == protocol.AgentTypeCodeReview {
		return "", nil, false
	}
	state := &ImplementationSessionState{}
	summary := "I can't run destructive cleanup commands against your workspace. No file changes were made."
	outcome := a.buildImplementationSessionOutcome(msg, state, false)
	return summary, outcome, true
}
