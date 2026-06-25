package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// resolveImplementationTrustMode maps composer mode to file-change trust.
// Agent/export sessions auto-apply; ask/plan stay interactive.
func resolveImplementationTrustMode(msg *protocol.Message) string {
	if msg == nil {
		return ""
	}
	if msg.IdeEditorModeIsAsk() || msg.IdeEditorMode() == "plan" {
		return ""
	}
	if msg.IdeEditorMode() == "agent" || msg.IdeEditorModeIsExport() || msg.ImplementationSession() {
		return editorTrustAutoApply
	}
	trust := strings.TrimSpace(msg.EditorAgentTrust())
	if trust == "" {
		return editorTrustAutoApply
	}
	return trust
}

// EffectiveEditorTrustForAutoApprove is used by the hub when applying file changes.
func EffectiveEditorTrustForAutoApprove(msg *protocol.Message) string {
	if trust := resolveImplementationTrustMode(msg); trust != "" {
		return trust
	}
	if msg == nil {
		return ""
	}
	return strings.TrimSpace(msg.EditorAgentTrust())
}
