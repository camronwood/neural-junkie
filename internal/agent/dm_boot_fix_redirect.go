package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func bootFixImplementerType(a *Agent, msg *protocol.Message) string {
	wsPath := ""
	if a != nil {
		wsPath = a.resolveWorkspacePath(msg)
	}
	if wsPath != "" {
		if _, err := os.Stat(filepath.Join(wsPath, "go.mod")); err == nil {
			if m := DetectStackManifest(wsPath); m == nil || !m.HasReact {
				return string(protocol.AgentTypeBackend)
			}
		}
	}
	return string(protocol.AgentTypeFrontend)
}

func bootFixImplementerDisplayName(agentType string) string {
	switch strings.ToLower(strings.TrimSpace(agentType)) {
	case string(protocol.AgentTypeBackend):
		return "BackendEngineer"
	case string(protocol.AgentTypeFrontend):
		return "FrontendEngineer"
	default:
		return agentType
	}
}

// tryBootFixImplementerRedirect declines boot-fix implementation sessions in specialist DMs
// when the channel partner is not the stack implementer (e.g. SoftwareArchitect vs FrontendEngineer).
// ide_route_agent_type only routes on team/IDE channels; DMs always talk to the channel partner.
func (a *Agent) tryBootFixImplementerRedirect(msg *protocol.Message) (string, map[string]interface{}, bool) {
	if a == nil || msg == nil || !a.isDMChannel(msg.Channel) {
		return "", nil, false
	}
	// Assistant chat must never be aborted into a coding specialist redirect.
	if a.Info.Type == protocol.AgentTypeAssistant {
		return "", nil, false
	}
	// Presence / workspace visibility questions must not be swallowed by wrong_route.
	if userAsksAboutWorkspaceVisibility(msg.Content) {
		return "", nil, false
	}
	if msg.IdeEditorModeIsAsk() || msg.IdeEditorModeIsPlan() {
		return "", nil, false
	}
	// Require an actual implementation/export session — conversation_mode=code alone is
	// used for workspace/@codebase Q&A and must not trigger specialist redirects.
	if !msg.ImplementationSession() && !msg.IdeEditorModeIsExport() {
		return "", nil, false
	}
	if !messageStampedBootFailure(msg) {
		return "", nil, false
	}
	want := bootFixImplementerType(a, msg)
	if strings.EqualFold(string(a.Info.Type), want) {
		return "", nil, false
	}
	target := bootFixImplementerDisplayName(want)
	text := fmt.Sprintf(
		"Boot-fix and build repair run in **Agent** mode with **%s** — they're set up for file edits and verification on this stack.\n\n"+
			"You're in a DM with **%s**. For boot-fix work, open a DM with **%s**, or use the main IDE channel in Agent mode (boot-fix routes to %s there automatically).\n\n"+
			"I can still help with architecture, tradeoffs, and design questions here.",
		target, a.Info.Name, target, target,
	)
	outcome := map[string]interface{}{
		"outcome":          "wrong_route",
		"failure_type":     "wrong_route",
		"suggested_agent":  target,
		"repair_used":      false,
		"verify_failed":    false,
		"verify_skipped":   false,
		"files_changed":    []string{},
	}
	return text, outcome, true
}
