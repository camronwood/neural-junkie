package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var fenceBlockRE = regexp.MustCompile("(?s)```([a-zA-Z0-9_-]*)\\s*\\n(.*?)```")

// extractCodeFenceForPath picks fenced code that plausibly belongs to targetPath.
func extractCodeFenceForPath(response, targetPath string) string {
	targetPath = normalizeFileChangeRelPath(targetPath)
	if targetPath == "" {
		return extractAnyCodeFenceContent(response)
	}
	var best string
	for _, m := range fenceBlockRE.FindAllStringSubmatch(response, -1) {
		if len(m) < 3 {
			continue
		}
		lang := strings.ToLower(strings.TrimSpace(m[1]))
		body := strings.TrimSpace(m[2])
		if body == "" {
			continue
		}
		if !fencedContentPlausibleForPath(targetPath, lang, body) {
			continue
		}
		if len(body) > len(best) {
			best = body
		}
	}
	if best != "" {
		return best
	}
	body := extractAnyCodeFenceContent(response)
	if fencedContentPlausibleForPath(targetPath, "", body) {
		return body
	}
	return ""
}

func validateProposalContent(path, content string) error {
	if looksLikePlaceholderProposalContent(content) {
		return fmt.Errorf("proposal content looks like a placeholder template, not real deliverable text")
	}
	if LooksLikeCorruptSourceContent(content) {
		return fmt.Errorf("proposal content looks corrupt (git diff debris or stub text), not valid source")
	}
	if isToolCallJSONContent(content) {
		return fmt.Errorf("proposal content looks like a tool-call payload, not file source")
	}
	path = strings.ToLower(normalizeFileChangeRelPath(path))
	if strings.Contains(path, "tailwind.config") {
		lower := strings.ToLower(content)
		if !strings.Contains(lower, "tailwind") && !strings.Contains(lower, "darkmode") &&
			!strings.Contains(lower, "content:") && !strings.Contains(lower, "module.exports") {
			return fmt.Errorf("tailwind config proposal must contain tailwind/darkMode settings")
		}
	}
	return nil
}

func isToolCallJSONContent(content string) bool {
	trim := strings.TrimSpace(content)
	return strings.HasPrefix(trim, "{") && strings.Contains(trim, `"name"`) &&
		(strings.Contains(trim, "propose_file_edit") || strings.Contains(trim, "arguments"))
}

func fencedContentPlausibleForPath(path, lang, content string) bool {
	if isToolCallJSONContent(content) {
		return false
	}
	path = strings.ToLower(normalizeFileChangeRelPath(path))
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(path, "tailwind.config"):
		return strings.Contains(lower, "tailwind") || strings.Contains(lower, "darkmode") ||
			strings.Contains(lower, "content:") || strings.Contains(lower, "module.exports")
	case strings.HasSuffix(path, ".go"):
		return strings.Contains(lower, "package ") || strings.HasPrefix(lower, "func ") || lang == "go"
	case strings.HasSuffix(path, ".css"):
		return strings.Contains(lower, "{") && !strings.Contains(lower, "import ")
	case strings.HasSuffix(path, ".tsx"), strings.HasSuffix(path, ".jsx"):
		return strings.Contains(lower, "export ") || strings.Contains(lower, "import ") || lang == "tsx" || lang == "jsx"
	default:
		return true
	}
}

// synthesizeGoMainEdit builds a minimal main.go when the model returned no proposal.
func synthesizeGoMainEdit(userContent, existing string) (string, bool) {
	lower := strings.ToLower(userContent)
	wantHello := strings.Contains(lower, "helloworld")
	wantPrint := strings.Contains(lower, "printversion")
	if !wantHello && !wantPrint {
		return "", false
	}

	var b strings.Builder
	if strings.Contains(existing, "package main") {
		b.WriteString(strings.TrimSpace(existing))
		if !strings.HasSuffix(strings.TrimSpace(existing), "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	} else {
		b.WriteString("package main\n\nimport \"fmt\"\n\n")
	}
	if wantHello && !strings.Contains(existing, "func HelloWorld") {
		b.WriteString("func HelloWorld() {\n\tfmt.Println(\"HelloWorld\")\n}\n\n")
	}
	if wantPrint && !strings.Contains(existing, "func PrintVersion") {
		b.WriteString("func PrintVersion() {\n\tfmt.Println(\"v0.1.0\")\n}\n\n")
	}
	mainBody := "func main() {\n"
	if wantHello {
		mainBody += "\tHelloWorld()\n"
	}
	if wantPrint {
		mainBody += "\tPrintVersion()\n"
	}
	mainBody += "}\n"
	if strings.Contains(existing, "func main()") {
		out := b.String()
		out = regexp.MustCompile(`(?s)func main\(\)\s*\{[^}]*\}`).ReplaceAllString(out, mainBody)
		if !strings.Contains(out, "import \"fmt\"") && (wantHello || wantPrint) {
			out = strings.Replace(out, "package main\n", "package main\n\nimport \"fmt\"\n", 1)
		}
		return out, true
	}
	b.WriteString(mainBody)
	out := b.String()
	if !strings.Contains(out, "import \"fmt\"") {
		out = strings.Replace(out, "package main\n", "package main\n\nimport \"fmt\"\n", 1)
	}
	return out, true
}

// synthesizeThemeCSS builds src/theme.css when the user asked for theme variables.
func synthesizeThemeCSS(userContent string) (string, bool) {
	lower := strings.ToLower(userContent)
	if !strings.Contains(lower, "theme.css") && !strings.Contains(lower, "theme variables") {
		return "", false
	}
	return `:root {
  --bg: #f8fafc;
  --text: #0f172a;
}

[data-theme="dark"] {
  --bg: #0f172a;
  --text: #f8fafc;
}
`, true
}

// synthesizeTailwindDarkMode patches tailwind.config.js with darkMode when missing.
func synthesizeTailwindDarkMode(existing string) (string, bool) {
	if strings.Contains(strings.ToLower(existing), "darkmode") {
		return "", false
	}
	trim := strings.TrimSpace(existing)
	if trim == "" {
		return `/** @type {import('tailwindcss').Config} */
export default {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: { extend: {} },
  plugins: [],
};
`, true
	}
	if strings.Contains(trim, "export default {") {
		return strings.Replace(trim, "export default {", "export default {\n  darkMode: \"class\",", 1), true
	}
	if strings.Contains(trim, "module.exports = {") {
		return strings.Replace(trim, "module.exports = {", "module.exports = {\n  darkMode: 'class',", 1), true
	}
	if strings.Contains(trim, "module.exports={") {
		return strings.Replace(trim, "module.exports={", "module.exports={\n  darkMode: 'class',", 1), true
	}
	return "", false
}

// synthesizeAppThemeToggle adds local theme state and a sidebar toggle for React App entry files.
func synthesizeAppThemeToggle(userContent, existing string) (string, bool) {
	lower := strings.ToLower(userContent)
	if !strings.Contains(lower, "theme") && !strings.Contains(lower, "dark") &&
		!strings.Contains(lower, "light") && !strings.Contains(lower, "toggle") {
		return "", false
	}
	body := strings.TrimSpace(existing)
	if body == "" {
		return "", false
	}
	if strings.Contains(body, "toggleTheme") || strings.Contains(body, "setTheme") {
		return "", false
	}
	var b strings.Builder
	b.WriteString(`import "./index.css";
import { useState, useEffect } from "react";

export default function App() {
  const [theme, setTheme] = useState("dark");

  useEffect(() => {
    document.documentElement.classList.toggle("dark", theme === "dark");
  }, [theme]);

  const toggleTheme = () => {
    setTheme((prevTheme) => (prevTheme === "dark" ? "light" : "dark"));
  };

  return (
    <div className="min-h-screen bg-slate-900 text-slate-100 p-4 dark:bg-white dark:text-slate-900">
      <aside className="w-48 border border-slate-700 rounded p-3 dark:border-slate-300">
        <p className="text-sm text-slate-400 dark:text-slate-600">Sidebar</p>
        <button className="mt-4 px-3 py-1 bg-slate-800 text-slate-100 rounded dark:bg-slate-200 dark:text-slate-900" onClick={toggleTheme}>
          Toggle Theme
        </button>
      </aside>
    </div>
  );
}
`)
	return b.String(), true
}

func resolveImplementationFallbackTarget(ctx context.Context, a *Agent, msg *protocol.Message, wsPath, userContent string) string {
	if st := implementationSessionStateFromContext(ctx); st != nil && st.StackManifest != nil {
		if rem := remainingImplementationTargets(wsPath, st.StackManifest, userContent); len(rem) > 0 {
			return rem[0]
		}
	}
	target := preferImplementationTargetPathForMessage(a, msg)
	if target == "" {
		target = preferImplementationTargetPath(a.resolveWorkspacePath(msg), userContent, "")
	}
	return target
}

func (a *Agent) attemptDeterministicImplementationFallback(ctx context.Context, msg *protocol.Message) (bool, []string) {
	if a == nil || msg == nil {
		return false, nil
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath == "" {
		return false, nil
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	userContent := msg.Content
	if userAffirmsPendingImplementation(msg.Content) {
		for i := len(a.channelHistory(msg.Channel)) - 1; i >= 0; i-- {
			m := a.channelHistory(msg.Channel)[i]
			if m == nil || m.ID == msg.ID {
				continue
			}
			if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
				userContent = m.Content
				break
			}
		}
	}
	var paths []string

	if a.attemptCorruptAppJSBootFix(ctx, msg, wsPath, channel, userContent, implementationSessionStateFromContext(ctx)) {
		return true, []string{"src/App.js"}
	}

	target := resolveImplementationFallbackTarget(ctx, a, msg, wsPath, userContent)
	if target == "" {
		return false, nil
	}

	switch {
	case strings.HasSuffix(strings.ToLower(target), ".go"):
		existing, err := os.ReadFile(filepath.Join(wsPath, target))
		if err != nil {
			return false, nil
		}
		body, ok := synthesizeGoMainEdit(userContent, string(existing))
		if !ok {
			return false, nil
		}
		if err := a.validateProposalForSession(ctx, msg, target, ProposalOpEdit); err != nil {
			return false, nil
		}
		if err := a.proposeFileEditInChannel(channel, target, string(existing), body, msg); err != nil {
			return false, nil
		}
		paths = []string{target}
	case target == "src/theme.css" || strings.HasSuffix(target, "theme.css"):
		body, ok := synthesizeThemeCSS(userContent)
		if !ok {
			return false, nil
		}
		if err := a.validateProposalForSession(ctx, msg, "src/theme.css", ProposalOpCreate); err != nil {
			return false, nil
		}
		if err := a.proposeFileCreateInChannel(channel, "src/theme.css", body, msg); err != nil {
			return false, nil
		}
		paths = []string{"src/theme.css"}
	case strings.Contains(strings.ToLower(target), "tailwind.config"):
		existing, err := os.ReadFile(filepath.Join(wsPath, target))
		if err != nil {
			return false, nil
		}
		body, ok := synthesizeTailwindDarkMode(string(existing))
		if !ok {
			return false, nil
		}
		target = a.ResolveProposalPath(ctx, msg, target)
		if err := a.validateProposalForSession(ctx, msg, target, ProposalOpEdit); err != nil {
			return false, nil
		}
		if err := a.proposeFileEditInChannel(channel, target, string(existing), body, msg); err != nil {
			return false, nil
		}
		paths = []string{target}
	case strings.HasSuffix(strings.ToLower(target), "app.tsx"), strings.HasSuffix(strings.ToLower(target), "app.jsx"):
		existing, err := os.ReadFile(filepath.Join(wsPath, target))
		if err != nil {
			return false, nil
		}
		body, ok := synthesizeAppThemeToggle(userContent, string(existing))
		if !ok {
			return false, nil
		}
		target = a.ResolveProposalPath(ctx, msg, target)
		if err := a.validateProposalForSession(ctx, msg, target, ProposalOpEdit); err != nil {
			return false, nil
		}
		if err := a.proposeFileEditInChannel(channel, target, string(existing), body, msg); err != nil {
			return false, nil
		}
		paths = []string{target}
	default:
		return false, nil
	}
	if st := implementationSessionStateFromContext(ctx); st != nil {
		st.ProposedCount++
	}
	log.Printf("[%s] deterministic_impl_fallback(paths=%v)", a.Info.Name, paths)
	return true, paths
}

func (a *Agent) repairTailwindDarkModeIfNeeded(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState) {
	if a == nil || msg == nil || state == nil {
		return
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath == "" {
		return
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	userContent := msg.Content
	if userAffirmsPendingImplementation(msg.Content) {
		for i := len(a.channelHistory(msg.Channel)) - 1; i >= 0; i-- {
			m := a.channelHistory(msg.Channel)[i]
			if m == nil || m.ID == msg.ID {
				continue
			}
			if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
				userContent = m.Content
				break
			}
		}
	}
	var tailwindTargets []string
	if themeImplementationRE.MatchString(userContent) || themeImplementationRE.MatchString(msg.Content) {
		if st := state.StackManifest; st != nil && st.TailwindConfig != "" {
			tailwindTargets = append(tailwindTargets, st.TailwindConfig)
		} else {
			tailwindTargets = append(tailwindTargets, "tailwind.config.js")
		}
	}
	seen := make(map[string]bool)
	for _, rel := range append(state.FilesChanged, tailwindTargets...) {
		if seen[rel] {
			continue
		}
		seen[rel] = true
		if !strings.Contains(strings.ToLower(rel), "tailwind.config") {
			continue
		}
		rel = a.ResolveProposalPath(ctx, msg, rel)
		full := filepath.Join(wsPath, rel)
		existing, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(existing)), "darkmode") {
			continue
		}
		body, ok := synthesizeTailwindDarkMode(string(existing))
		if !ok || validateProposalContent(rel, body) != nil {
			continue
		}
		if err := a.validateProposalForSession(ctx, msg, rel, ProposalOpEdit); err != nil {
			continue
		}
		if err := a.proposeFileEditInChannel(channel, rel, string(existing), body, msg); err != nil {
			continue
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
		log.Printf("[%s] tailwind_darkmode_repair(path=%s)", a.Info.Name, rel)
	}
}

func (a *Agent) repairAppThemeIfNeeded(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState) {
	if a == nil || msg == nil || state == nil || state.StackManifest == nil {
		return
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath == "" {
		return
	}
	userContent := msg.Content
	if userAffirmsPendingImplementation(msg.Content) {
		for i := len(a.channelHistory(msg.Channel)) - 1; i >= 0; i-- {
			m := a.channelHistory(msg.Channel)[i]
			if m == nil || m.ID == msg.ID {
				continue
			}
			if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
				userContent = m.Content
				break
			}
		}
	}
	entry := state.StackManifest.EntryPoint
	lowerEntry := strings.ToLower(entry)
	if entry == "" || (!strings.HasSuffix(lowerEntry, "app.tsx") && !strings.HasSuffix(lowerEntry, "app.jsx")) {
		return
	}
	if implementationTargetSatisfied(wsPath, entry, userContent) {
		return
	}
	existing, err := os.ReadFile(filepath.Join(wsPath, entry))
	if err != nil {
		return
	}
	body, ok := synthesizeAppThemeToggle(userContent, string(existing))
	if !ok || validateProposalContent(entry, body) != nil {
		return
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	rel := a.ResolveProposalPath(ctx, msg, entry)
	if err := a.validateProposalForSession(ctx, msg, rel, ProposalOpEdit); err != nil {
		return
	}
	if err := a.proposeFileEditInChannel(channel, rel, string(existing), body, msg); err != nil {
		return
	}
	state.ProposedCount++
	state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
	log.Printf("[%s] app_theme_repair(path=%s)", a.Info.Name, rel)
}

func corruptAppJSEntryConflict(wsPath string, manifest *StackManifest) bool {
	return DetectEntryConflicts(wsPath, manifest) != ""
}

func (a *Agent) shouldRepairCorruptAppJSEntry(msg *protocol.Message, wsPath, userContent string, manifest *StackManifest) bool {
	if wsPath == "" || manifest == nil || !corruptAppJSEntryConflict(wsPath, manifest) {
		return false
	}
	content := userContent
	if msg != nil {
		content = msg.Content + "\n" + userContent
	}
	return messageHasBootOrBuildError(content) || messageImpliesBootFix(content, a.channelHistory(msg.Channel))
}

func (a *Agent) attemptCorruptAppJSBootFix(ctx context.Context, msg *protocol.Message, wsPath, channel, userContent string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil {
		return false
	}
	manifest := a.manifestForProposal(ctx, msg)
	if !a.shouldRepairCorruptAppJSEntry(msg, wsPath, userContent, manifest) {
		return false
	}
	rel := "src/App.js"
	if _, err := os.Stat(filepath.Join(wsPath, rel)); err != nil {
		return false
	}
	if err := a.proposeFileDeleteInChannel(channel, rel, msg); err != nil {
		return false
	}
	if state != nil {
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
	}
	log.Printf("[%s] corrupt_appjs_entry_repair(path=%s)", a.Info.Name, rel)
	return true
}

// tryEarlyCorruptAppJSBootFix deletes corrupt src/App.js before LLM rounds when boot-fix
// intent is set. BootFixIntent keeps groundingSatisfied() false, so the round-0 fallback
// would not run without this preflight.
func (a *Agent) tryEarlyCorruptAppJSBootFix(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || !state.BootFixIntent || wsPath == "" || state.StackManifest == nil {
		return false
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	return a.attemptCorruptAppJSBootFix(ctx, msg, wsPath, channel, msg.Content, state)
}

func (a *Agent) repairCorruptAppJSEntryIfNeeded(ctx context.Context, msg *protocol.Message, state *ImplementationSessionState) {
	if a == nil || msg == nil || state == nil || state.StackManifest == nil {
		return
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath == "" {
		return
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	userContent := msg.Content
	if userAffirmsPendingImplementation(msg.Content) {
		for i := len(a.channelHistory(msg.Channel)) - 1; i >= 0; i-- {
			m := a.channelHistory(msg.Channel)[i]
			if m == nil || m.ID == msg.ID {
				continue
			}
			if protocol.IsUserLikeSender(m.From) && (userRequestsImplementation(m.Content) || messageHasBootOrBuildError(m.Content)) {
				userContent = m.Content
				break
			}
		}
	}
	if !a.shouldRepairCorruptAppJSEntry(msg, wsPath, userContent, state.StackManifest) {
		return
	}
	rel := "src/App.js"
	if _, err := os.Stat(filepath.Join(wsPath, rel)); err != nil {
		return
	}
	for _, changed := range state.FilesChanged {
		if normalizeFileChangeRelPath(changed) == rel {
			return
		}
	}
	if err := a.proposeFileDeleteInChannel(channel, rel, msg); err != nil {
		return
	}
	state.ProposedCount++
	state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
	log.Printf("[%s] corrupt_appjs_entry_repair(path=%s)", a.Info.Name, rel)
}
