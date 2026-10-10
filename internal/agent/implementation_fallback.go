package agent

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	semantic "github.com/camronwood/neural-junkie/internal/intent"
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
	if err := validateConfigJSONContent(path, content); err != nil {
		return err
	}
	if looksLikePlaceholderProposalContent(content) {
		return fmt.Errorf("proposal content looks like a placeholder template, not real deliverable text")
	}
	if LooksLikeCorruptSourceContent(content) {
		return fmt.Errorf("proposal content looks corrupt (git diff debris or stub text), not valid source")
	}
	if isToolCallJSONContent(content) {
		return fmt.Errorf("proposal content looks like a tool-call payload, not file source")
	}
	if looksLikeFileChangeDirectivePayload(content) {
		return fmt.Errorf("proposal content looks like a FILE_CHANGE directive header, not file source")
	}
	path = strings.ToLower(normalizeFileChangeRelPath(path))
	if strings.Contains(path, "tailwind.config") {
		lower := strings.ToLower(content)
		if !strings.Contains(lower, "tailwind") && !strings.Contains(lower, "darkmode") &&
			!strings.Contains(lower, "content:") && !strings.Contains(lower, "module.exports") {
			return fmt.Errorf("tailwind config proposal must contain tailwind/darkMode settings")
		}
	}
	if err := validateWebAssetProposalContent(path, content); err != nil {
		return err
	}
	return nil
}

// looksLikeFileChangeDirectivePayload is true when the body is (mostly) a leaked
// [FILE_CHANGE] header instead of real deliverable text.
func looksLikeFileChangeDirectivePayload(content string) bool {
	trim := strings.TrimSpace(content)
	if trim == "" {
		return false
	}
	lower := strings.ToLower(trim)
	if strings.Contains(lower, "[file_change]") {
		return true
	}
	if strings.HasPrefix(lower, "operation:") && strings.Contains(lower, "path:") {
		return true
	}
	if strings.HasPrefix(lower, "markdown [file_change]") || strings.HasPrefix(lower, "markdown[file_change]") {
		return true
	}
	return false
}

// validateWebAssetProposalContent rejects HTML/CSS proposals with the wrong language shape
// (e.g. CSS dumped into about.html, or HTML dumped into style.css).
func validateWebAssetProposalContent(path, content string) error {
	trim := strings.TrimSpace(content)
	if trim == "" {
		return nil
	}
	// Strip accidental FILE_CHANGE field wrappers: `content: /* css */` or `content: "<!DOCTYPE…"`
	if strings.HasPrefix(strings.ToLower(trim), "content:") {
		trim = strings.TrimSpace(trim[len("content:"):])
		trim = strings.Trim(trim, "\"'`")
	}
	lower := strings.ToLower(trim)
	switch {
	case strings.HasSuffix(path, ".html"), strings.HasSuffix(path, ".htm"):
		hasHTML := strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<html") ||
			strings.Contains(lower, "<body") || strings.Contains(lower, "<head")
		looksCSS := !hasHTML && strings.Contains(lower, "{") &&
			(strings.Contains(lower, "margin:") || strings.Contains(lower, "padding:") ||
				strings.Contains(lower, "background") || strings.Contains(lower, "font-family"))
		if looksCSS || !hasHTML {
			return fmt.Errorf("html proposal must contain HTML markup, not CSS-only content")
		}
	case strings.HasSuffix(path, ".css"):
		if strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<html") {
			return fmt.Errorf("css proposal must not contain HTML markup")
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
	looksLikeCargoToml := strings.HasPrefix(lower, "[package]") ||
		strings.Contains(lower, "\n[package]\n") ||
		(strings.Contains(lower, "[dependencies]") && !strings.Contains(lower, "fn ") && !strings.Contains(lower, "struct "))
	switch {
	case strings.Contains(path, "tailwind.config"):
		return strings.Contains(lower, "tailwind") || strings.Contains(lower, "darkmode") ||
			strings.Contains(lower, "content:") || strings.Contains(lower, "module.exports")
	case strings.HasSuffix(path, "cargo.toml"):
		return looksLikeCargoToml || strings.Contains(lower, "edition") || lang == "toml"
	case strings.HasSuffix(path, ".rs"):
		// Never land a Cargo.toml body on a Rust source path (blackjack greenfield burn).
		if looksLikeCargoToml {
			return false
		}
		return strings.Contains(lower, "fn ") || strings.Contains(lower, "struct ") ||
			strings.Contains(lower, "enum ") || strings.Contains(lower, "mod ") ||
			lang == "rust" || lang == "rs"
	case strings.HasSuffix(path, ".go"):
		return strings.Contains(lower, "package ") || strings.HasPrefix(lower, "func ") || lang == "go"
	case strings.HasSuffix(path, ".rs"):
		// Reject Cargo.toml bodies written into .rs (blackjack user-flow corruption mode).
		if strings.HasPrefix(lower, "[package]") || strings.Contains(lower, "\n[dependencies]") && !strings.Contains(lower, "fn ") {
			return false
		}
		return strings.Contains(lower, "fn ") || strings.Contains(lower, "struct ") ||
			strings.Contains(lower, "enum ") || strings.Contains(lower, "mod ") || lang == "rust" || lang == "rs"
	case strings.HasSuffix(path, ".css"):
		return strings.Contains(lower, "{") && !strings.Contains(lower, "import ") &&
			!strings.Contains(lower, "<!doctype") && !strings.Contains(lower, "<html")
	case strings.HasSuffix(path, ".html"), strings.HasSuffix(path, ".htm"):
		return strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<html") ||
			strings.Contains(lower, "<body") || lang == "html"
	case strings.HasSuffix(path, ".tsx"), strings.HasSuffix(path, ".jsx"):
		return strings.Contains(lower, "export ") || strings.Contains(lower, "import ") || lang == "tsx" || lang == "jsx"
	default:
		return true
	}
}

// retargetFencePathForContent remaps fenced bodies that clearly belong to another path
// (e.g. Cargo.toml text while the user/session target is src/main.rs).
func retargetFencePathForContent(namedPath, content string) string {
	path := normalizeFileChangeRelPath(namedPath)
	body := strings.TrimSpace(content)
	if body == "" {
		return path
	}
	lower := strings.ToLower(body)
	looksLikeCargoToml := strings.HasPrefix(lower, "[package]") ||
		strings.Contains(lower, "\n[package]\n") ||
		(strings.Contains(lower, "[dependencies]") && !strings.Contains(lower, "fn ") && !strings.Contains(lower, "struct "))
	if looksLikeCargoToml && !strings.EqualFold(filepath.Base(path), "Cargo.toml") {
		return "Cargo.toml"
	}
	return path
}
var (
	themeCSSAffirmRE = regexp.MustCompile(`(?i)\b(?:write|create|implement|add|make|fix|update|edit|ship)\b[^.\n]{0,100}\btheme\.css\b|\btheme\.css\b[^.\n]{0,60}\b(?:with|using)\s+(?:light|dark|theme)`)
	themeCSSNegateRE = regexp.MustCompile(`(?i)\b(?:do not|don't|dont|never|ignore|without|not\s+(?:reference|mention|cite|use|touch|edit|change))\b[^.\n]{0,100}\btheme\.css\b`)
)

// userRequestsThemeCSSDeliverable is true for affirmative theme.css work, not "do not reference theme.css".
func userRequestsThemeCSSDeliverable(content string) bool {
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "theme.css") && !strings.Contains(lower, "theme variables") {
		return false
	}
	if themeCSSNegateRE.MatchString(content) && !themeCSSAffirmRE.MatchString(content) {
		return false
	}
	if strings.Contains(lower, "theme variables") && themeCSSAffirmRE.MatchString(content) {
		return true
	}
	return themeCSSAffirmRE.MatchString(content) ||
		(strings.Contains(lower, "theme variables") &&
			(strings.Contains(lower, "create") || strings.Contains(lower, "add") || strings.Contains(lower, "implement")))
}

func corruptAppJSEntryConflict(wsPath string, manifest *StackManifest) bool {
	return DetectEntryConflicts(wsPath, manifest) != ""
}

func (a *Agent) shouldRepairCorruptAppJSEntry(msg *protocol.Message, wsPath, userContent string, manifest *StackManifest) bool {
	if wsPath == "" || manifest == nil || !corruptAppJSEntryConflict(wsPath, manifest) {
		return false
	}
	// Prefer structural gates after phrase-heuristic cutover: impl session or
	// stamped Debug/Edit. Deprecated messageHasBootOrBuildError always returns false.
	if msg != nil && msg.ImplementationSession() {
		return true
	}
	if msg != nil {
		if d, ok := protocol.ExtractTurnDecision(msg); ok &&
			(d.Action == semantic.ActionDebug || d.Action == semantic.ActionEdit) {
			return true
		}
	}
	content := userContent
	if msg != nil {
		content = msg.Content + "\n" + userContent
	}
	var history []*protocol.Message
	if a != nil && msg != nil {
		history = a.channelHistory(msg.Channel)
	}
	return messageHasBootOrBuildError(content) || messageImpliesBootFix(content, history)
}
