package collaboration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Path-like tokens that must cite the focus allowlist (no fixture-specific stack names).
var focusPathTokenRE = regexp.MustCompile(`(?i)(?:\b[\w.-]+(?:/[\w.-]+)+\b|\b[\w.-]+\.(?:md|markdown|go|ts|tsx|js|jsx|yml|yaml|json|py|rs|html|css|vue|svelte)\b)`)

// Deliverable kind stamped on collaboration_task messages (hub dispatch metadata).
const (
	DeliverableKindNone     = "none"
	DeliverableKindMarkdown = "markdown"
	DeliverableKindFile     = "file"
)

// DeliverablePolicy answers deliverable questions for a collaboration task in one place.
type DeliverablePolicy struct {
	Task             CollaborationTask
	CollabGoal       string
	AllowedReadPaths []string
}

// NewDeliverablePolicy builds a policy for task (goal optional for research classification).
func NewDeliverablePolicy(task CollaborationTask, collabGoal string, allowedReadPaths []string) DeliverablePolicy {
	return DeliverablePolicy{
		Task:             task,
		CollabGoal:       collabGoal,
		AllowedReadPaths: allowedReadPaths,
	}
}

func (p DeliverablePolicy) RequiresFile() bool {
	return TaskRequiresFileDeliverable(p.Task)
}

func (p DeliverablePolicy) MarkdownOnly() bool {
	return TaskLooksLikeMarkdownDeliverable(p.Task)
}

// Kind is markdown | file | none for structured dispatch metadata.
func (p DeliverablePolicy) Kind() string {
	if !p.RequiresFile() {
		return DeliverableKindNone
	}
	if p.MarkdownOnly() {
		return DeliverableKindMarkdown
	}
	return DeliverableKindFile
}

// RequiresImplementationSession is true for coding file deliverables (not markdown-only).
func (p DeliverablePolicy) RequiresImplementationSession() bool {
	return p.Kind() == DeliverableKindFile
}

func (p DeliverablePolicy) ResearchFindings() bool {
	return taskLooksLikeResearchFindingsDeliverable(p.Task, p.CollabGoal)
}

// InventoryHit returns the first out-of-allowlist path-like token in content when focus-scoped.
func (p DeliverablePolicy) InventoryHit(content string) string {
	return FocusScopedDeliverableInventoryHit(content, p.AllowedReadPaths)
}

// FocusScopedDeliverableInventoryHit returns the first path-like token not covered by allowedPaths.
func FocusScopedDeliverableInventoryHit(content string, allowedPaths []string) string {
	content = strings.TrimSpace(content)
	if content == "" || len(allowedPaths) == 0 {
		return ""
	}
	for _, m := range focusPathTokenRE.FindAllString(content, -1) {
		tok := strings.TrimSpace(m)
		if tok == "" {
			continue
		}
		if focusTokenAllowed(tok, allowedPaths) {
			continue
		}
		return tok
	}
	return ""
}

func focusTokenAllowed(tok string, allowedPaths []string) bool {
	lower := strings.ToLower(filepath.ToSlash(strings.TrimSuffix(tok, "/")))
	base := filepath.Base(lower)
	hasSlash := strings.Contains(lower, "/")
	for _, p := range allowedPaths {
		p = filepath.ToSlash(strings.ToLower(strings.TrimSpace(p)))
		if p == "" {
			continue
		}
		if p == lower || strings.HasSuffix(p, "/"+lower) {
			return true
		}
		if !hasSlash && (filepath.Base(p) == base || filepath.Base(p) == lower) {
			return true
		}
		if hasSlash && (strings.HasPrefix(p, lower+"/") || strings.HasPrefix(lower, p+"/")) {
			return true
		}
	}
	return false
}

// FocusScopedDeliverableInventoryError builds a repair-friendly rejection for scope inventory.
func FocusScopedDeliverableInventoryError(hit string, allowedPaths []string) error {
	return fmt.Errorf(
		"focus-scoped deliverable mentions out-of-scope %q — rewrite using only these sources: %s (cite allowlisted paths only)",
		hit,
		strings.Join(allowedPaths, ", "),
	)
}
