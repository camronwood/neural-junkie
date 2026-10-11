package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const implSessionMaxFiles = 5

// swiftDirTokenRE matches folder tokens like TriviaGame/ in user prompts.
var swiftDirTokenRE = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_\-]*)/`)

// implementationTargets returns ordered file paths still relevant for a multi-file session.
func implementationTargets(manifest *StackManifest, userContent string) []string {
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

	if manifest != nil {
		if themeTask && manifest.HasTailwind && manifest.TailwindConfig != "" {
			add(manifest.TailwindConfig)
		}
		if manifest.EntryPoint != "" && (themeTask || manifest.HasReact || manifest.HasVue) {
			add(manifest.EntryPoint)
		}
	}
	for _, p := range swiftImplementationTargetsFromPrompt(userContent) {
		add(p)
	}
	return out
}

// swiftImplementationTargetsFromPrompt extracts .swift deliverable paths from the user ask.
// Bare names are qualified with a mentioned folder token (e.g. TriviaGame/).
// ContentView.swift and GameView.swift are treated as alternatives (prefer ContentView).
func swiftImplementationTargetsFromPrompt(userContent string) []string {
	raw := DetectFilePaths(userContent)
	if len(raw) == 0 {
		return nil
	}
	dir := firstSwiftDirToken(userContent)
	var swift []string
	seen := make(map[string]bool)
	for _, p := range raw {
		p = strings.TrimSpace(strings.TrimPrefix(p, "./"))
		if p == "" || !strings.HasSuffix(strings.ToLower(p), ".swift") {
			continue
		}
		p = filepath.ToSlash(p)
		if !strings.Contains(p, "/") && dir != "" {
			p = dir + "/" + p
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		swift = append(swift, p)
	}
	return dedupeSwiftUIAlternatives(swift)
}

func firstSwiftDirToken(userContent string) string {
	for _, m := range swiftDirTokenRE.FindAllStringSubmatch(userContent, -1) {
		if len(m) < 2 {
			continue
		}
		tok := strings.TrimSpace(m[1])
		if tok == "" {
			continue
		}
		// Skip common relative noise.
		switch strings.ToLower(tok) {
		case ".", "..", "src", "app", "lib", "pkg", "cmd", "internal":
			continue
		}
		return tok
	}
	return ""
}

func dedupeSwiftUIAlternatives(paths []string) []string {
	hasContentView := false
	for _, p := range paths {
		if strings.EqualFold(filepath.Base(p), "ContentView.swift") {
			hasContentView = true
			break
		}
	}
	if !hasContentView {
		return paths
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if strings.EqualFold(filepath.Base(p), "GameView.swift") {
			continue
		}
		out = append(out, p)
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
	if strings.HasSuffix(strings.ToLower(rel), ".swift") {
		return swiftImplementationTargetSatisfied(wsPath, rel)
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

func swiftImplementationTargetSatisfied(wsPath, rel string) bool {
	if fileNonEmpty(wsPath, rel) {
		return true
	}
	if !strings.EqualFold(filepath.Base(rel), "ContentView.swift") {
		return false
	}
	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." {
		dir = ""
	}
	alts := []string{
		filepath.ToSlash(filepath.Join(dir, "GameView.swift")),
		filepath.ToSlash(filepath.Join(dir, "Views", "ContentView.swift")),
	}
	for _, alt := range alts {
		alt = strings.TrimPrefix(alt, "/")
		if fileNonEmpty(wsPath, alt) {
			return true
		}
	}
	return false
}

func fileNonEmpty(wsPath, rel string) bool {
	rel = strings.TrimSpace(strings.TrimPrefix(filepath.ToSlash(rel), "./"))
	if rel == "" {
		return false
	}
	b, err := os.ReadFile(filepath.Join(wsPath, filepath.FromSlash(rel)))
	return err == nil && len(bytes.TrimSpace(b)) > 0
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
	if state.CircuitBreakerFired || state.ConsecutiveNoVerifyProgress > 0 {
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
			if protocol.IsUserLikeSender(m.From) && messageStampedImplAction(m) {
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
