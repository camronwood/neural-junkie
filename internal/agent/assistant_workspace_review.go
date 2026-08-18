package agent

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// turnReviewPlan reads stamp-owned review guidance. Phrase helpers are emergency
// rollback only when no decision is stamped.
func turnReviewPlan(msg *protocol.Message) (plan intent.ContextPlan, ok bool) {
	if msg == nil {
		return intent.ContextPlan{}, false
	}
	decision, ok := protocol.ExtractTurnDecision(msg)
	if !ok {
		return intent.ContextPlan{}, false
	}
	return decision.ContextPlan, true
}

func stampRequestsDocumentReview(plan intent.ContextPlan) bool {
	return plan.ReviewMode == intent.ReviewModeDocument ||
		plan.Subject == intent.ContextSubjectActiveDocument
}

func stampRequestsWorkspaceDocReview(plan intent.ContextPlan) bool {
	return plan.ReviewMode == intent.ReviewModeWorkspace ||
		plan.Subject == intent.ContextSubjectWorkspaceDocuments
}

func stampRequestsCodeReview(plan intent.ContextPlan) bool {
	return plan.ReviewMode == intent.ReviewModeCode
}

// userRequestsEditorDocumentReview is true when the user asks to review or inspect
// editor content without necessarily naming a repo path.
// Deprecated for routing: prefer turnReviewPlan / stampRequestsDocumentReview.
func userRequestsEditorDocumentReview(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	markers := []string{
		"review", "reivew", "proofread", "look at this", "look at the",
		"take a look", "in my editor", "in the editor", "open document",
		"document open", "file open", "active file", "active tab",
		"open tab", "the open tab", "current tab", "what's open", "whats open",
		"can you read", "can you see", "see the file", "see files", "see any issues",
		"image i have open", "file i have open", "have open", "document i have",
		"doc i have",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func workspaceContextActiveOpenFile(msg *protocol.Message) (path string, hasContent bool) {
	if msg == nil || msg.Metadata == nil {
		return "", false
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return "", false
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return "", false
	}
	files, ok := ctxMap["open_files"].([]interface{})
	if !ok {
		return "", false
	}
	var fallbackPath string
	var fallbackHasContent bool
	for _, f := range files {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		p, _ := fm["path"].(string)
		content, _ := fm["content"].(string)
		contentOK := strings.TrimSpace(content) != ""
		isActive, _ := fm["is_active"].(bool)
		if isActive && strings.TrimSpace(p) != "" {
			return p, contentOK
		}
		if fallbackPath == "" && strings.TrimSpace(p) != "" {
			fallbackPath = p
			fallbackHasContent = contentOK
		}
	}
	return fallbackPath, fallbackHasContent
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
	plan, hasPlan := turnReviewPlan(msg)
	reviewIntent := false
	workspaceDocReview := false
	codeReview := false
	if hasPlan {
		reviewIntent = stampRequestsDocumentReview(plan)
		workspaceDocReview = stampRequestsWorkspaceDocReview(plan)
		codeReview = stampRequestsCodeReview(plan) || userRequestsCodeReviewForMessage(msg)
	} else {
		reviewIntent = userRequestsEditorDocumentReview(msg.Content)
		codeReview = userRequestsCodeReviewForMessage(msg)
	}
	hasFiles := workspaceContextHasOpenFiles(msg)
	hasScanSummary := workspaceContextHasScanSummary(msg)
	hasScanAnalysis := workspaceContextHasScanAnalysis(msg)
	activePath, activeHasContent := workspaceContextActiveOpenFile(msg)

	switch {
	case workspaceDocReview && messageHasWorkspaceContext(msg) && scope != ContextScopeNone:
		prompt.WriteString("\n=== WORKSPACE DOCUMENT REVIEW (this turn) ===\n")
		prompt.WriteString("The semantic stamp requested a review of workspace documents (Markdown/docs under the shared roots). ")
		prompt.WriteString("Use WORKSPACE CONTEXT file_tree and open_files bodies that were uploaded for this turn. ")
		prompt.WriteString("Synthesize guidance from those real documents. ")
		prompt.WriteString("Do NOT invent paths like src/index.js. Do NOT ask a tech-stack questionnaire when documents are present. ")
		prompt.WriteString("Ask clarifying questions only when required facts are missing from the retrieved documents.\n")
		appendWorkspaceStackGrounding(prompt)
		prompt.WriteString("\n")
	case (scope == ContextScopeFocus || scope == ContextScopeFull) && (reviewIntent || hasFiles || hasScanSummary || hasScanAnalysis):
		prompt.WriteString("\n=== DOCUMENT / CODE REVIEW (this turn) ===\n")
		prompt.WriteString("The user shared workspace context (see WORKSPACE CONTEXT below). ")
		if activePath != "" {
			prompt.WriteString(fmt.Sprintf(
				"The ACTIVE open file is exactly: %s. ",
				activePath,
			))
			if reviewIntent {
				prompt.WriteString(
					"When they say \"the document/file/tab I have open\", they mean THAT file — " +
						"review its contents from WORKSPACE CONTEXT immediately. ",
				)
			}
			if activeHasContent {
				prompt.WriteString(
					"Do NOT ask which file, do NOT ask for a path, and do NOT claim you cannot see their editor, browser, device, or tab. ",
				)
			} else {
				prompt.WriteString(
					"The active file path is known but its body was empty in context — say that specifically and ask them to re-open or paste it. ",
				)
			}
		} else {
			prompt.WriteString("When they ask whether you can see files or the workspace, answer by naming exactly what is visible: project name/path, file tree, open file metadata, file contents, scan-summary metadata, scan-analysis metadata, or attached images. ")
			prompt.WriteString("Do NOT say you cannot access their editor or files when workspace_context is present. ")
		}
		if hasScanAnalysis || hasScanSummary {
			prompt.WriteString("When they ask to run summarize_scan_analysis or summarize_scan_summary on the open file, use the analysis_dir or summary_dir from workspace context — do NOT ask them to type the path. ")
		}
		prompt.WriteString("If an open file has empty content or image pixels were not attached, say that specifically and ask for the missing file/image rather than denying all file access.\n")
		appendWorkspaceStackGrounding(prompt)
		prompt.WriteString("\n")
	case (scope == ContextScopeOutline || scope == ContextScopeFocus || scope == ContextScopeFull) &&
		messageHasWorkspaceContext(msg) && scope != ContextScopeNone:
		if codeReview {
			prompt.WriteString("\n=== PROJECT CODE REVIEW (this turn) ===\n")
			prompt.WriteString("The user asked for a project-wide code review and shared workspace context (file tree). ")
			prompt.WriteString("Use read_file, grep, glob_file_search, and run_typescript_check on key paths under the project root. ")
			prompt.WriteString("Start from package.json, tsconfig.json, src/, and entry files visible in the tree. ")
			prompt.WriteString("Do NOT ask for a single file path or tell them to enable workspace sharing — review what is visible and read more files as needed.\n")
		}
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
