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
		"what's open", "whats open", "can you read", "can you see",
		"see the file", "see files", "see any issues", "image i have open",
		"file i have open", "have open",
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

func workspaceContextHasScanSummary(msg *protocol.Message) bool {
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
	scan, ok := ctxMap["scan_summary"].(map[string]interface{})
	if ok && len(scan) > 0 {
		if dir, _ := scan["summary_dir"].(string); strings.TrimSpace(dir) != "" {
			return true
		}
	}
	return openFilesHaveScanDir(ctxMap, "scan_summary_dir")
}

// appendWorkspaceReviewGuidance steers any agent to use shared editor context
// precisely instead of claiming it cannot access files that were shared.
func appendWorkspaceReviewGuidance(prompt *strings.Builder, msg *protocol.Message) {
	if msg == nil {
		return
	}
	scope := ResolveContextScope(msg)
	reviewIntent := userRequestsEditorDocumentReview(msg.Content)
	hasFiles := workspaceContextHasOpenFiles(msg)
	hasScanSummary := workspaceContextHasScanSummary(msg)
	hasScanAnalysis := workspaceContextHasScanAnalysis(msg)

	switch {
	case (scope == ContextScopeFocus || scope == ContextScopeFull) && (reviewIntent || hasFiles || hasScanSummary || hasScanAnalysis):
		prompt.WriteString("\n=== DOCUMENT / CODE REVIEW (this turn) ===\n")
		prompt.WriteString("The user shared workspace context (see WORKSPACE CONTEXT). ")
		prompt.WriteString("When they ask whether you can see files or the workspace, answer by naming exactly what is visible: project name/path, file tree, open file metadata, file contents, scan-summary metadata, scan-analysis metadata, or attached images. ")
		prompt.WriteString("Do NOT say you cannot access their editor or files when workspace_context is present. ")
		if hasScanAnalysis || hasScanSummary {
			prompt.WriteString("When they ask to run summarize_scan_analysis or summarize_scan_summary on the open file, use the analysis_dir or summary_dir from workspace context — do NOT ask them to type the path. ")
		}
		prompt.WriteString("If an open file has empty content or image pixels were not attached, say that specifically and ask for the missing file/image rather than denying all file access.\n")
		appendWorkspaceStackGrounding(prompt)
		prompt.WriteString("\n")
	case (scope == ContextScopeOutline || scope == ContextScopeFocus || scope == ContextScopeFull) &&
		messageHasWorkspaceContext(msg) && ResolveContextScope(msg) != ContextScopeNone:
		appendWorkspaceStackGrounding(prompt)
		prompt.WriteString("\n")
	case scope == ContextScopeHint && reviewIntent:
		prompt.WriteString("\n=== EDITOR CONTEXT (limited) ===\n")
		prompt.WriteString("The user asked to review something in their editor, but only a project hint was shared (no file bodies). ")
		prompt.WriteString("Ask them to mention a file path, enable workspace focus, or paste the content — do not invent file contents.\n\n")
	}
}

func appendWorkspaceStackGrounding(prompt *strings.Builder) {
	prompt.WriteString("**Stack grounding:** Infer languages and frameworks from the file tree and open files. ")
	prompt.WriteString("Do NOT invent packages (e.g. `golang.org/x/themes`), generic Gin/Bootstrap tutorials, or dependencies that are not in WORKSPACE CONTEXT.\n")
}
