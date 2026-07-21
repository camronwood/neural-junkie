package agent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const implSessionMaxFiles = 5

// implementationTargets returns ordered file paths still relevant for a multi-file session.
func implementationTargets(manifest *StackManifest, userContent string) []string {
	if manifest == nil {
		return nil
	}
	lower := strings.ToLower(strings.TrimSpace(userContent))
	themeTask := strings.Contains(lower, "theme") ||
		strings.Contains(lower, "dark") ||
		strings.Contains(lower, "light") ||
		strings.Contains(lower, "toggle")

	var out []string
	seen := make(map[string]bool)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}

	if themeTask && manifest.HasTailwind && manifest.TailwindConfig != "" {
		add(manifest.TailwindConfig)
	}
	if manifest.EntryPoint != "" && (themeTask || manifest.HasReact || manifest.HasVue) {
		add(manifest.EntryPoint)
	}
	return out
}

// implementationTargetSatisfied reports whether a target file already meets the task on disk.
func implementationTargetSatisfied(wsPath, rel, userContent string) bool {
	wsPath = strings.TrimSpace(wsPath)
	rel = strings.TrimSpace(rel)
	if wsPath == "" || rel == "" {
		return false
	}
	b, err := os.ReadFile(filepath.Join(wsPath, rel))
	if err != nil {
		return false
	}
	body := string(b)
	lower := strings.ToLower(strings.TrimSpace(userContent))
	themeTask := strings.Contains(lower, "theme") ||
		strings.Contains(lower, "dark") ||
		strings.Contains(lower, "light") ||
		strings.Contains(lower, "toggle")

	base := strings.ToLower(filepath.Base(rel))
	if strings.HasPrefix(base, "tailwind.config") && themeTask {
		return strings.Contains(body, "darkMode")
	}
	if strings.HasSuffix(strings.ToLower(rel), "app.tsx") ||
		strings.HasSuffix(strings.ToLower(rel), "app.jsx") {
		if themeTask {
			return strings.Contains(body, "toggleTheme") ||
				strings.Contains(body, "setTheme") ||
				(strings.Contains(body, "useState") && strings.Contains(strings.ToLower(body), "theme"))
		}
	}
	if strings.HasSuffix(strings.ToLower(rel), "theme.css") && themeTask {
		return strings.Contains(strings.ToLower(body), "dark")
	}
	return false
}

// remainingImplementationTargets lists manifest targets not yet satisfied on disk.
func remainingImplementationTargets(wsPath string, manifest *StackManifest, userContent string) []string {
	targets := implementationTargets(manifest, userContent)
	if len(targets) == 0 {
		return nil
	}
	var remaining []string
	for _, rel := range targets {
		if !implementationTargetSatisfied(wsPath, rel, userContent) {
			remaining = append(remaining, rel)
		}
	}
	return remaining
}

func shouldContinueImplementationSession(a *Agent, msg *protocol.Message, state *ImplementationSessionState) (bool, string) {
	if a == nil || msg == nil || state == nil {
		return false, ""
	}
	if msg.IdeEditorModeIsExport() || userRequestsFileExportForMessage(msg) {
		return false, ""
	}
	if a.Hub != nil && agentsDeferred(a.Hub, msg.Channel) {
		return false, ""
	}
	if msg.EditorAgentTrust() != editorTrustAutoApply && state.TrustMode != editorTrustAutoApply {
		history := a.channelHistory(msg.Channel)
		if !channelHasRecentFileChangeApproval(history, msg.ID, a.Info.ID) {
			return false, ""
		}
	}
	_, _, maxFiles := implSessionLimits(msg)
	if len(state.FilesChanged) >= maxFiles {
		return false, ""
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath == "" {
		return false, ""
	}
	seedMsg := msg
	if userAffirmsPendingImplementation(msg.Content) {
		history := a.channelHistory(msg.Channel)
		for i := len(history) - 1; i >= 0; i-- {
			m := history[i]
			if m == nil || m.ID == msg.ID {
				continue
			}
			if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
				seedMsg = m
				break
			}
		}
	}
	remaining := remainingImplementationTargets(wsPath, state.StackManifest, seedMsg.Content)
	if len(remaining) == 0 {
		return false, ""
	}
	note := "Continue the same implementation task in this session — ship the NEXT file change now.\n"
	note += "Next target(s): " + strings.Join(remaining, ", ") + "\n"
	if len(state.FilesChanged) > 0 {
		note += "Already applied (do NOT re-propose): " + strings.Join(state.FilesChanged, ", ") + "\n"
	}
	return true, note
}
