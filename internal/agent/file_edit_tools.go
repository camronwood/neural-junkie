package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/fileedit"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	searchReplaceToolName = "search_replace"
	applyPatchToolName    = "apply_patch"
)

func fileEditToolDefinitions() []ai.ClaudeToolDefinition {
	return []ai.ClaudeToolDefinition{
		{
			Name: searchReplaceToolName,
			Description: "Replace an exact unique snippet in an existing file (preferred for edits). " +
				"old_string must match exactly once unless replace_all is true. " +
				"When the user has a selection, old_string must appear inside it.",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"path":{"type":"string","description":"Relative file path under workspace root"},
					"old_string":{"type":"string","description":"Exact text to find"},
					"new_string":{"type":"string","description":"Replacement text"},
					"replace_all":{"type":"boolean","description":"Replace every occurrence (default false)"}
				},
				"required":["path","old_string","new_string"]
			}`),
		},
		{
			Name: applyPatchToolName,
			Description: "Apply a unified diff patch to an existing file (multi-hunk edits in one file). " +
				"Use standard unified diff format with @@ hunk headers.",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"properties":{
					"path":{"type":"string","description":"Relative file path under workspace root"},
					"patch":{"type":"string","description":"Unified diff patch body (may include ---/+++ headers)"}
				},
				"required":["path","patch"]
			}`),
		},
		proposeFileEditToolDefinition(),
	}
}

func appendFileEditToolsPrompt(system *strings.Builder) {
	system.WriteString("FILE EDITING (Cursor-style):\n")
	system.WriteString("1. Prefer search_replace for surgical edits to existing files (exact unique old_string).\n")
	system.WriteString("2. Use apply_patch for multi-hunk changes in one file.\n")
	system.WriteString("3. Use propose_file_edit only for new files or when replacing an entire small file (<120 lines).\n")
	system.WriteString("4. When the user selected code, keep edits inside that selection (add imports above if needed).\n")
	system.WriteString("5. Read the file first when unsure of exact content — do not guess old_string.\n\n")
}

func (a *Agent) executeSearchReplaceTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	if isAskModeReadOnly(msg) {
		return "", fmt.Errorf("ask mode is read-only")
	}
	var args struct {
		Path       string `json:"path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid search_replace input: %w", err)
		}
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	path = a.ResolveProposalPath(ctx, msg, path)
	if !isValidFileChangeRelPath(path) {
		return "", fmt.Errorf("invalid path: %q", path)
	}

	scope := selectionScopeFromMessage(msg, path)
	if err := fileedit.RequireOldStringInSelection(scope, args.OldString); err != nil {
		return "", err
	}

	oldContent, err := a.readWorkspaceFileForEdit(ctx, msg, path)
	if err != nil {
		return "", err
	}

	newContent, strategy, err := fileedit.SearchReplaceWithFallback(oldContent, args.OldString, args.NewString, args.ReplaceAll)
	if err != nil {
		if pe, ok := err.(*fileedit.PatchError); ok && pe.Code == fileedit.ErrNotFound {
			return a.searchReplaceSmartFallback(ctx, msg, path, oldContent, args.OldString, args.NewString, args.ReplaceAll, scope)
		}
		return "", err
	}
	if err := fileedit.ValidateSelectionScope(scope, oldContent, newContent); err != nil {
		return "", err
	}
	if err := a.validateProposalForSession(ctx, msg, path, "edit"); err != nil {
		return "", err
	}

	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	if err := a.proposeFileEditInChannel(ctx, channel, path, oldContent, newContent, msg); err != nil {
		return "", err
	}
	a.trackFileEditProposal(ctx, msg, path)
	return fmt.Sprintf(`{"status":"proposed","path":%q,"strategy":%q}`, path, strategy), nil
}

func (a *Agent) executeApplyPatchTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	if isAskModeReadOnly(msg) {
		return "", fmt.Errorf("ask mode is read-only")
	}
	var args struct {
		Path  string `json:"path"`
		Patch string `json:"patch"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid apply_patch input: %w", err)
		}
	}
	path := strings.TrimSpace(args.Path)
	patch := strings.TrimSpace(args.Patch)
	if path == "" || patch == "" {
		return "", fmt.Errorf("path and patch are required")
	}
	path = a.ResolveProposalPath(ctx, msg, path)
	if !isValidFileChangeRelPath(path) {
		return "", fmt.Errorf("invalid path: %q", path)
	}

	oldContent, err := a.readWorkspaceFileForEdit(ctx, msg, path)
	if err != nil {
		return "", err
	}
	newContent, err := fileedit.ApplyUnifiedPatch(oldContent, patch)
	if err != nil {
		return "", err
	}
	scope := selectionScopeFromMessage(msg, path)
	if err := fileedit.ValidateSelectionScope(scope, oldContent, newContent); err != nil {
		return "", err
	}
	if err := a.validateProposalForSession(ctx, msg, path, "edit"); err != nil {
		return "", err
	}

	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	if err := a.proposeFileEditInChannel(ctx, channel, path, oldContent, newContent, msg); err != nil {
		return "", err
	}
	a.trackFileEditProposal(ctx, msg, path)
	return fmt.Sprintf(`{"status":"proposed","path":%q}`, path), nil
}

// searchReplaceSmartFallback retries within an active selection when exact match fails.
func (a *Agent) searchReplaceSmartFallback(
	ctx context.Context,
	msg *protocol.Message,
	path, fileContent, oldString, newString string,
	replaceAll bool,
	scope *fileedit.SelectionScope,
) (string, error) {
	if scope == nil || strings.TrimSpace(scope.Text) == "" {
		return "", &fileedit.PatchError{Code: fileedit.ErrNotFound, Message: "old_string not found in file"}
	}
	// Apply inside selection text only, then splice back into file.
	selOld := scope.Text
	selNew, _, err := fileedit.SearchReplaceWithFallback(selOld, oldString, newString, replaceAll)
	if err != nil {
		return "", &fileedit.PatchError{
			Code:    fileedit.ErrNotFound,
			Message: "old_string not found in file or selection; read_file and copy exact content",
		}
	}
	// Locate selection in file (first occurrence).
	normFile := strings.ReplaceAll(fileContent, "\r\n", "\n")
	normSel := strings.ReplaceAll(selOld, "\r\n", "\n")
	idx := strings.Index(normFile, normSel)
	if idx < 0 {
		return "", &fileedit.PatchError{
			Code:    fileedit.ErrNotFound,
			Message: "selection text not found in file; read_file for current content",
		}
	}
	newContent := normFile[:idx] + selNew + normFile[idx+len(normSel):]
	if err := fileedit.ValidateSelectionScope(scope, fileContent, newContent); err != nil {
		return "", err
	}
	if err := a.validateProposalForSession(ctx, msg, path, "edit"); err != nil {
		return "", err
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	if err := a.proposeFileEditInChannel(ctx, channel, path, fileContent, newContent, msg); err != nil {
		return "", err
	}
	a.trackFileEditProposal(ctx, msg, path)
	return fmt.Sprintf(`{"status":"proposed","path":%q,"strategy":"selection_fallback"}`, path), nil
}

func (a *Agent) readWorkspaceFileForEdit(ctx context.Context, msg *protocol.Message, relPath string) (string, error) {
	relPath = strings.TrimPrefix(relPath, "/")
	if b := shared.BackendFromContext(ctx); b != nil {
		data, err := b.ReadFile(ctx, relPath)
		if err != nil {
			return "", fmt.Errorf("read file %s: %w", relPath, err)
		}
		if st := implementationSessionStateFromContext(ctx); st != nil {
			st.RecordReadPath(relPath)
		}
		return string(data), nil
	}
	ws := a.resolveWorkspacePath(msg)
	if ws == "" {
		return "", fmt.Errorf("workspace not available")
	}
	abs, err := pathutil.ResolveRelWithinRoot(ws, relPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", relPath, err)
	}
	if st := implementationSessionStateFromContext(ctx); st != nil {
		st.RecordReadPath(relPath)
	}
	return string(data), nil
}

func selectionScopeFromMessage(msg *protocol.Message, targetPath string) *fileedit.SelectionScope {
	if msg == nil || msg.Metadata == nil {
		return nil
	}
	wsCtx, ok := msg.Metadata["workspace_context"].(map[string]interface{})
	if !ok {
		return nil
	}
	openFiles, ok := wsCtx["open_files"].([]interface{})
	if !ok {
		return nil
	}
	targetPath = filepath.ToSlash(strings.TrimPrefix(targetPath, "/"))
	for _, f := range openFiles {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		fp, _ := fm["path"].(string)
		fp = filepath.ToSlash(strings.TrimPrefix(fp, "/"))
		if fp != targetPath {
			continue
		}
		isActive, _ := fm["is_active"].(bool)
		if !isActive {
			continue
		}
		start, _ := fm["selection_start_line"].(float64)
		end, _ := fm["selection_end_line"].(float64)
		selText, _ := fm["selected_text"].(string)
		if start <= 0 || end <= 0 || strings.TrimSpace(selText) == "" {
			return nil
		}
		return &fileedit.SelectionScope{
			Path:      fp,
			StartLine: int(start),
			EndLine:   int(end),
			Text:      selText,
		}
	}
	return nil
}

func (a *Agent) trackFileEditProposal(ctx context.Context, msg *protocol.Message, path string) {
	if st := implementationSessionStateFromContext(ctx); st != nil {
		st.ProposedCount++
		st.FilesChanged = appendUnique(st.FilesChanged, []string{path})
	}
	_ = msg
}
