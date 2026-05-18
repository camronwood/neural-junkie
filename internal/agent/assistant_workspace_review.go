package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// userRequestsEditorDocumentReview is true when the user asks to review or inspect
// editor content without necessarily naming a repo path.
func userRequestsEditorDocumentReview(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	markers := []string{
		"review", "reivew", "proofread", "look at this", "look at the",
		"take a look", "in my editor", "in the editor", "open document",
		"document open", "file open", "active file", "active tab",
		"what's open", "whats open", "can you read",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func workspaceContextHasOpenFiles(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return false
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return false
	}
	files, ok := ctxMap["open_files"].([]interface{})
	return ok && len(files) > 0
}

// appendAssistantWorkspaceReviewGuidance steers the assistant to use shared editor
// context for document/code review instead of claiming it has no file access.
func appendAssistantWorkspaceReviewGuidance(prompt *strings.Builder, msg *protocol.Message) {
	if msg == nil {
		return
	}
	scope := ResolveContextScope(msg)
	reviewIntent := userRequestsEditorDocumentReview(msg.Content)
	hasFiles := workspaceContextHasOpenFiles(msg)

	switch {
	case (scope == ContextScopeFocus || scope == ContextScopeFull) && (reviewIntent || hasFiles):
		prompt.WriteString("\n=== DOCUMENT / CODE REVIEW (this turn) ===\n")
		prompt.WriteString("The user shared workspace context including open editor files (see WORKSPACE CONTEXT). ")
		prompt.WriteString("When they ask to review a document, RFC, or code, answer using those file contents and line numbers. ")
		prompt.WriteString("Do NOT say you cannot access their editor or files.\n\n")
	case scope == ContextScopeHint && reviewIntent:
		prompt.WriteString("\n=== EDITOR CONTEXT (limited) ===\n")
		prompt.WriteString("The user asked to review something in their editor, but only a project hint was shared (no file bodies). ")
		prompt.WriteString("Ask them to mention a file path, enable workspace focus, or paste the content — do not invent file contents.\n\n")
	}
}
