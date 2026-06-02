package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	implementationAffirmRE = regexp.MustCompile(`(?i)\b(go ahead|do it( now)?|yes please|please do|proceed|make (the |those )?changes|apply (that|it|your plan)|do that now|ok please|sure,?\s*please|let's do it|please implement|sounds good[,!]?\s*(go|do)|you can (start|begin))\b`)
	themeImplementationRE  = regexp.MustCompile(`(?i)(?:\b(theme|themes|dark mode|light mode|ui theme)\b.{0,48}\b(add|implement|build|wire|toggle)\b|\b(add|implement|build|wire)\b.{0,48}\b(theme|themes|ui theme|dark mode|light mode)\b)`)
	implementTypoRE        = regexp.MustCompile(`(?i)\bimpl[e]?ment\b`)
)

const maxImplementationSeedFiles = 6

// userRequestsImplementation reports coding/build asks (themes, features, fixes) where the
// user expects [FILE_CHANGE] deliverables, not a codebase overview.
func userRequestsImplementation(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	if implementTypoRE.MatchString(lower) || themeImplementationRE.MatchString(lower) {
		return true
	}
	if userAffirmsPendingImplementation(content) {
		return false // continuation alone is not enough without channel history
	}
	phrases := []string{
		"please implement", "implement that", "implement the",
		"implement this", "build this", "build out", "code this", "code it",
		"ship ", "apply the plan", "apply your plan", "make the changes",
		"make the change", "do the implementation", "actually implement",
		"write the code", "add the code", "add light", "add dark",
		"theme support", "light/dark", "dark mode", "light mode",
		"wire up", "hook up", "under settings", "settings page",
	}
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// userAffirmsPendingImplementation reports short follow-ups after an implementation ask.
func userAffirmsPendingImplementation(content string) bool {
	return implementationAffirmRE.MatchString(strings.TrimSpace(content))
}

// channelHasRecentImplementationAsk scans recent user turns for an implementation request.
func channelHasRecentImplementationAsk(history []*protocol.Message, skipMsgID string) bool {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 12; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID {
			continue
		}
		if !protocol.IsUserLikeSender(m.From) {
			continue
		}
		seen++
		if userRequestsImplementation(m.Content) {
			return true
		}
	}
	return false
}

// userRequestsImplementationForMessage includes affirmation follow-ups in the same channel thread.
func userRequestsImplementationForMessage(a *Agent, msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if userRequestsImplementation(msg.Content) {
		return true
	}
	if a == nil || !userAffirmsPendingImplementation(msg.Content) {
		return false
	}
	return channelHasRecentImplementationAsk(a.channelHistory(msg.Channel), msg.ID)
}

func agentTypeCanShipFileChanges(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeFrontend, protocol.AgentTypeBackend, protocol.AgentTypeDatabase,
		protocol.AgentTypeSecurity, protocol.AgentTypeArchitecture, protocol.AgentTypeCodeReview,
		protocol.AgentTypeDevOps, protocol.AgentTypeExpert, protocol.AgentTypeRust:
		return true
	default:
		return false
	}
}

// shouldProactiveScanWorkspace limits bulk workspace scans on implementation turns so
// models do not reply with multi-file architecture tours (e.g. after "package.json" appears
// in constraint text).
func shouldProactiveScanWorkspace(content string) bool {
	if userRequestsImplementation(content) {
		for _, p := range DetectFilePaths(content) {
			if strings.Contains(p, "/") {
				return true
			}
		}
		return false
	}
	return shouldInjectWorkspaceCode(content)
}

func workspaceGroundingRequirement(totalLoaded int, content string, implementation ...bool) string {
	if totalLoaded <= 0 {
		return ""
	}
	impl := len(implementation) > 0 && implementation[0]
	if !impl {
		impl = userRequestsImplementation(content)
	}
	if impl {
		return fmt.Sprintf(
			"\nGrounding requirement: Start your answer with exactly this one line:\n"+
				"\"Grounding: I loaded %d file(s) from the workspace context for this answer.\"\n"+
				"Then ship concrete changes via [FILE_CHANGE] blocks (one per file you modify). "+
				"Do not write a codebase tour or architecture summary unless the user asked for one.\n\n",
			totalLoaded,
		)
	}
	return fmt.Sprintf(
		"\nGrounding requirement: Start your answer with exactly this one line:\n"+
			"\"Grounding: I loaded %d file(s) from the workspace context for this answer.\"\nThen continue with your analysis.\n\n",
		totalLoaded,
	)
}

func appendImplementationDeliveryGuidance(prompt *strings.Builder, a *Agent, msg *protocol.Message, agentType protocol.AgentType) {
	if msg == nil || !userRequestsImplementationForMessage(a, msg) || !agentTypeCanShipFileChanges(agentType) {
		return
	}
	prompt.WriteString("\n=== IMPLEMENTATION DELIVERY (required) ===\n")
	prompt.WriteString("The user wants working changes in the shared workspace, not advice-only.\n")
	prompt.WriteString("You MUST include one or more [FILE_CHANGE] blocks with real file paths and content.\n")
	prompt.WriteString("Each path must be a real relative file (e.g. tailwind.config.js, src/index.css) — never labels like \"File:\" or \"path:\".\n")
	prompt.WriteString("Use only dependencies already in package.json / the repo — do not invent packages.\n")
	prompt.WriteString("Keep conversational text short (2-4 sentences); put code in [FILE_CHANGE], not long fenced dumps.\n")
	prompt.WriteString("Do NOT ask the user to paste or share file contents when REFERENCED FILES or WORKSPACE SOURCE FILES appear below — read them and emit [FILE_CHANGE].\n")
	prompt.WriteString("Only ask for a path if a required file is missing from every context section.\n")
}

var frontendImplementationSeeds = []string{
	"tailwind.config.js",
	"tailwind.config.ts",
	"postcss.config.js",
	"package.json",
	"src/index.css",
	"src/App.tsx",
	"src/App.jsx",
	"src/main.tsx",
	"src/main.ts",
	"index.html",
}

// implementationSeedCandidates returns paths to load from disk for implement turns.
func implementationSeedCandidates(agentType protocol.AgentType, content string) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, p := range DetectFilePaths(content) {
		add(p)
	}
	if agentType == protocol.AgentTypeFrontend {
		for _, p := range frontendImplementationSeeds {
			add(p)
		}
	}
	if agentType == protocol.AgentTypeBackend {
		for _, p := range []string{"go.mod", "main.go", "cmd/main.go", "package.json"} {
			add(p)
		}
	}
	return out
}

// AppendImplementationSeedFiles reads likely edit targets from the workspace on implement turns.
func AppendImplementationSeedFiles(prompt *strings.Builder, a *Agent, msg *protocol.Message, workspacePath string, agentType protocol.AgentType, excludePaths map[string]bool) int {
	if workspacePath == "" || msg == nil || !userRequestsImplementationForMessage(a, msg) {
		return 0
	}

	paths := implementationSeedCandidates(agentType, msg.Content)
	if len(paths) == 0 {
		return 0
	}

	var loadedFiles []struct {
		path    string
		lang    string
		content string
	}
	totalSize := 0
	loaded := 0

	for _, p := range paths {
		if loaded >= maxImplementationSeedFiles {
			break
		}
		if excludePaths != nil && excludePaths[p] {
			continue
		}
		resolved := filepath.Join(workspacePath, p)
		if _, err := os.Stat(resolved); err != nil {
			continue
		}
		content, _, err := ReadFileForPrompt(p, workspacePath)
		if err != nil {
			continue
		}
		if totalSize+len(content) > maxTotalFileSize {
			break
		}
		lang := inferLanguage(p)
		loadedFiles = append(loadedFiles, struct {
			path    string
			lang    string
			content string
		}{p, lang, content})
		totalSize += len(content)
		loaded++
		if excludePaths != nil {
			excludePaths[p] = true
		}
	}

	if len(loadedFiles) == 0 {
		return 0
	}

	prompt.WriteString("\n=== REFERENCED FILES (implementation) ===\n")
	prompt.WriteString("Loaded from the shared workspace for this implementation request. ")
	prompt.WriteString("Use this ACTUAL code in [FILE_CHANGE] blocks — do not ask the user to paste these files.\n\n")
	for _, f := range loadedFiles {
		prompt.WriteString(fmt.Sprintf("### %s (%s)\n```%s\n%s\n```\n\n", f.path, f.lang, f.lang, f.content))
	}
	prompt.WriteString("=== END REFERENCED FILES ===\n\n")
	return len(loadedFiles)
}
