package agent

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// File size limits for auto-loaded files
const (
	maxFileSize       = 512 * 1024 // prompt injection cap per referenced file
	maxTotalFileSize  = 2 * 1024 * 1024
	legacyMaxFileSize = 50 * 1024
)

// binaryExtensions are file types we skip (non-text)
var binaryExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".ico": true, ".svg": true, ".bmp": true, ".tiff": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true, ".rar": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".bin": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mov": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".o": true, ".a": true, ".pyc": true, ".class": true,
	".gguf": true, ".safetensors": true, ".pb": true, ".onnx": true,
}

// filePathPattern matches common file paths in text:
//   - path/to/file.ext (relative with directories)
//   - tailwind.config.js (root-level filenames)
//   - ./path/to/file.ext (dot-relative)
//   - /absolute/path/file.ext (absolute)
//
// Requires a file extension to reduce false positives.
var filePathPattern = regexp.MustCompile(`(?:^|\s|["'\x60(])([./]?(?:(?:[a-zA-Z0-9_\-]+/)*[a-zA-Z0-9_\-]+)\.[a-zA-Z0-9]{1,16})`)

// bareFilenamePattern matches standalone config/source filenames (e.g. tailwind.config.js).
var bareFilenamePattern = regexp.MustCompile(`\b([a-zA-Z0-9][a-zA-Z0-9._\-]*\.(?:js|ts|tsx|jsx|mjs|cjs|css|scss|less|html|json|yaml|yml|vue|svelte|go|rs|py|md|toml))\b`)

// esbuildLocationPattern matches compiler error locations like src/App.js:1:7.
var esbuildLocationPattern = regexp.MustCompile(`\b((?:[./]?(?:[a-zA-Z0-9_\-]+/)*[a-zA-Z0-9_\-]+)\.[a-zA-Z0-9]{1,16}):\d+(?::\d+)?\b`)

func appendDetectedPath(paths []string, seen map[string]bool, p string) []string {
	p = strings.TrimSpace(p)
	if p == "" {
		return paths
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return paths
	}
	ext := strings.ToLower(filepath.Ext(p))
	if binaryExtensions[ext] {
		return paths
	}
	if seen[p] {
		return paths
	}
	seen[p] = true
	return append(paths, p)
}

// DetectBareFilenames finds root-level or bare filenames in user text (tailwind.config.js, package.json).
func DetectBareFilenames(content string) []string {
	seen := make(map[string]bool)
	var paths []string
	for _, match := range bareFilenamePattern.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		paths = appendDetectedPath(paths, seen, match[1])
	}
	return paths
}

// DetectFilePaths extracts file paths from a message string (with or without directory prefixes).
func DetectFilePaths(content string) []string {
	seen := make(map[string]bool)
	var paths []string

	for _, match := range filePathPattern.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		paths = appendDetectedPath(paths, seen, match[1])
	}
	for _, p := range DetectBareFilenames(content) {
		paths = appendDetectedPath(paths, seen, p)
	}
	for _, match := range esbuildLocationPattern.FindAllStringSubmatch(content, -1) {
		if len(match) >= 2 {
			paths = appendDetectedPath(paths, seen, match[1])
		}
	}

	return paths
}

// ReadFileForPrompt reads a file and returns its content with line numbers,
// capped to maxFileSize. Returns empty string if file can't be read.
func ReadFileForPrompt(filePath string, workspacePath string) (content string, resolvedPath string, err error) {
	absResolved, err := pathutil.ResolveRelWithinRoot(workspacePath, filePath)
	if err != nil {
		return "", "", fmt.Errorf("file not found: %s", filePath)
	}

	// Check if file exists and get size
	info, err := os.Stat(absResolved)
	if err != nil {
		return "", "", fmt.Errorf("file not found: %s", filePath)
	}
	if info.IsDir() {
		return "", "", fmt.Errorf("path is a directory: %s", filePath)
	}

	// Check file size
	if info.Size() > maxFileSize {
		// Read first and last portions for large files
		data, err := os.ReadFile(absResolved)
		if err != nil {
			return "", "", fmt.Errorf("failed to read file: %w", err)
		}
		lines := strings.Split(string(data), "\n")
		keepLines := 200 // first 200 + last 50 lines
		tailLines := 50
		if len(lines) > keepLines+tailLines {
			head := strings.Join(lines[:keepLines], "\n")
			tail := strings.Join(lines[len(lines)-tailLines:], "\n")
			truncated := fmt.Sprintf("%s\n\n... [TRUNCATED: %d lines omitted - file is %d lines total] ...\n\n%s",
				head, len(lines)-keepLines-tailLines, len(lines), tail)
			return addLineNumbers(truncated), absResolved, nil
		}
		// File is large in bytes but not many lines -- include all
		return addLineNumbers(string(data)), absResolved, nil
	}

	// Read full file
	data, err := os.ReadFile(absResolved)
	if err != nil {
		return "", "", fmt.Errorf("failed to read file: %w", err)
	}

	// Quick binary content check (look for null bytes)
	if len(data) > 0 {
		sniff := data
		if len(sniff) > 512 {
			sniff = sniff[:512]
		}
		contentType := http.DetectContentType(sniff)
		if !strings.HasPrefix(contentType, "text/") && contentType != "application/json" && contentType != "application/xml" {
			return "", "", fmt.Errorf("binary file detected: %s", filePath)
		}
	}

	return addLineNumbers(string(data)), absResolved, nil
}

// inferLanguage guesses the language from a file extension for code block syntax highlighting.
func inferLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".jsx":
		return "jsx"
	case ".css":
		return "css"
	case ".html":
		return "html"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".sql":
		return "sql"
	case ".sh", ".bash":
		return "bash"
	case ".md":
		return "markdown"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h", ".hpp":
		return "cpp"
	case ".proto":
		return "protobuf"
	case ".tf":
		return "hcl"
	case ".dockerfile":
		return "dockerfile"
	default:
		if strings.Contains(filepath.Base(filePath), "Dockerfile") {
			return "dockerfile"
		}
		if strings.Contains(filepath.Base(filePath), "Makefile") {
			return "makefile"
		}
		return ""
	}
}

// AppendReferencedFiles detects file paths in the user's message, reads them
// from disk, and appends their content to the prompt with line numbers.
// This allows agents to access files even when workspace sharing is off.
// It returns the number of referenced files successfully loaded.
func AppendReferencedFiles(prompt *strings.Builder, messageContent string, workspacePath string) int {
	if workspacePath == "" {
		return 0
	}

	paths := DetectFilePaths(messageContent)
	if len(paths) == 0 {
		return 0
	}

	var loadedFiles []struct {
		path    string
		lang    string
		content string
	}
	totalSize := 0

	for _, p := range paths {
		content, _, err := ReadFileForPrompt(p, workspacePath)
		if err != nil {
			log.Printf("[workspace] Skipping referenced file %s: %v", p, err)
			continue
		}

		if totalSize+len(content) > maxTotalFileSize {
			log.Printf("[workspace] Skipping %s: would exceed total file size limit", p)
			break
		}

		lang := inferLanguage(p)
		loadedFiles = append(loadedFiles, struct {
			path    string
			lang    string
			content string
		}{p, lang, content})
		totalSize += len(content)
	}

	if len(loadedFiles) == 0 {
		return 0
	}

	prompt.WriteString("\n=== REFERENCED FILES ===\n")
	prompt.WriteString("The following files were referenced in the user's message and loaded from disk.\n")
	prompt.WriteString("Each line is prefixed with its real line number. Use THESE line numbers when referencing code.\n")
	prompt.WriteString("Analyze the ACTUAL code below -- do NOT give generic advice.\n\n")

	for _, f := range loadedFiles {
		prompt.WriteString(fmt.Sprintf("### %s (%s)\n```%s\n%s\n```\n\n", f.path, f.lang, f.lang, f.content))
	}

	prompt.WriteString("=== END REFERENCED FILES ===\n\n")
	return len(loadedFiles)
}

// messagePromotesWorkspaceFocus reports when a message should receive file content, not tree-only outline.
func messagePromotesWorkspaceFocus(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if isWeakImplementationAffirmation(msg.Content) {
		return false
	}
	if userRequestsImplementation(msg.Content) || workspaceDirectiveRE.MatchString(msg.Content) {
		return true
	}
	if ConversationModeFromMessage(msg) == ConversationModeCode {
		return true
	}
	return hasCodeTaskSignals(msg.Content)
}

// ResolveContextScopeForChannel caps workspace injection for DMs so open-file payloads
// do not overwhelm local models. Legacy full scope in a DM becomes outline (tree only).
// Implementation turns promote to focus so seed paths and open tabs can reach the model.
func ResolveContextScopeForChannel(msg *protocol.Message, channelType protocol.ChannelType) string {
	scope := ResolveContextScope(msg)
	if channelType != protocol.ChannelTypeDM {
		return scope
	}
	impl := messagePromotesWorkspaceFocus(msg)
	switch scope {
	case ContextScopeFull:
		if impl {
			return ContextScopeFocus
		}
		return ContextScopeOutline
	case ContextScopeOutline:
		if impl {
			return ContextScopeFocus
		}
		return ContextScopeOutline
	case ContextScopeFocus:
		return ContextScopeFocus
	default:
		return scope
	}
}

// ResolveContextScope returns the effective workspace context tier for a message.
// If workspace_context is absent, returns none. Legacy messages with workspace_context
// but no context_scope default to full.
func ResolveContextScope(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ContextScopeNone
	}
	if raw, ok := msg.Metadata[MetadataContextScope]; ok {
		if s, ok := raw.(string); ok {
			switch strings.TrimSpace(strings.ToLower(s)) {
			case ContextScopeNone, ContextScopeHint, ContextScopeOutline, ContextScopeFocus, ContextScopeFull:
				return strings.TrimSpace(strings.ToLower(s))
			}
		}
	}
	if _, ok := msg.Metadata["workspace_context"]; ok {
		return ContextScopeFull
	}
	return ContextScopeNone
}

func workspaceOpenFileMatchesReference(filePath, reference string) bool {
	filePath = filepath.ToSlash(strings.TrimSpace(filePath))
	reference = filepath.ToSlash(strings.TrimSpace(reference))
	if filePath == "" || reference == "" {
		return false
	}
	if filePath == reference || strings.HasSuffix(filePath, "/"+strings.TrimPrefix(reference, "./")) {
		return true
	}
	return filepath.Base(filePath) == filepath.Base(reference)
}

// retrimWorkspaceOpenFiles enforces the effective scope on the actual context map
// consumed by prompt construction. This is a server-side defense against clients
// sending a full open_files payload with a narrower resolved scope.
func retrimWorkspaceOpenFiles(ctxMap map[string]interface{}, scope string, msg *protocol.Message) map[string]interface{} {
	trimmed := make(map[string]interface{}, len(ctxMap))
	for key, value := range ctxMap {
		trimmed[key] = value
	}

	rawFiles, ok := ctxMap["open_files"].([]interface{})
	if !ok {
		return trimmed
	}
	switch scope {
	case ContextScopeFull:
		return trimmed
	case ContextScopeFocus:
		references := []string(nil)
		if msg != nil {
			references = DetectFilePaths(msg.Content)
		}
		focused := make([]interface{}, 0, len(rawFiles))
		for _, rawFile := range rawFiles {
			fileMap, ok := rawFile.(map[string]interface{})
			if !ok {
				continue
			}
			isActive, _ := fileMap["is_active"].(bool)
			filePath, _ := fileMap["path"].(string)
			matchesReference := false
			for _, reference := range references {
				if workspaceOpenFileMatchesReference(filePath, reference) {
					matchesReference = true
					break
				}
			}
			if isActive || matchesReference {
				focused = append(focused, rawFile)
			}
		}
		trimmed["open_files"] = focused
	default:
		trimmed["open_files"] = []interface{}{}
	}
	return trimmed
}

// AppendWorkspaceContext checks for workspace_context in message metadata
// and appends a scope-appropriate section to the prompt builder.
func AppendWorkspaceContext(prompt *strings.Builder, msg *protocol.Message) {
	AppendWorkspaceContextForChannel(prompt, msg, "")
}

// AppendWorkspaceContextForChannel applies DM-aware scope limits before appending workspace data.
func AppendWorkspaceContextForChannel(prompt *strings.Builder, msg *protocol.Message, channelType protocol.ChannelType) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	wsCtx, ok := msg.Metadata["workspace_context"]
	if !ok {
		return
	}
	ctxMap, ok := wsCtx.(map[string]interface{})
	if !ok {
		return
	}
	scope := ResolveContextScope(msg)
	if channelType != "" {
		scope = ResolveContextScopeForChannel(msg, channelType)
	}
	ctxMap = retrimWorkspaceOpenFiles(ctxMap, scope, msg)
	if scope == ContextScopeNone {
		return
	}
	appendWorkspacePromptSection(prompt, scope, ctxMap)
	appendLinkedWorkspacesPromptSection(prompt, msg, scope)
}

func appendLinkedWorkspacesPromptSection(prompt *strings.Builder, msg *protocol.Message, scope string) {
	if scope == ContextScopeNone || scope == ContextScopeHint {
		return
	}
	linked := linkedWorkspacesFromMetadata(msg)
	if len(linked) == 0 {
		return
	}
	prompt.WriteString("\n=== LINKED WORKSPACES ===\n")
	prompt.WriteString("Additional repositories in scope for this turn. Tag answers with the correct repo name.\n\n")
	for _, ref := range linked {
		label := ref.Name
		if label == "" {
			label = filepath.Base(ref.Path)
		}
		prompt.WriteString(fmt.Sprintf("## %s\nPath: %s\n", label, ref.Path))
		if raw, ok := msg.Metadata[MetadataLinkedWorkspaces].([]interface{}); ok {
			for _, item := range raw {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				p, _ := m["workspace_path"].(string)
				if normalizeScopePath(p) != normalizeScopePath(ref.Path) {
					continue
				}
				if tree, ok := m["file_tree"].(string); ok && tree != "" {
					prompt.WriteString(fmt.Sprintf("File tree:\n%s\n", tree))
				}
				if scope == ContextScopeFocus || scope == ContextScopeFull {
					if files, ok := m["open_files"].([]interface{}); ok && len(files) > 0 {
						for _, f := range files {
							fm, ok := f.(map[string]interface{})
							if !ok {
								continue
							}
							filePath, _ := fm["path"].(string)
							lang, _ := fm["language"].(string)
							content, _ := fm["content"].(string)
							if filePath == "" {
								continue
							}
							prompt.WriteString(fmt.Sprintf("\n### %s (%s)\n```%s\n%s\n```\n", filePath, lang, lang, addLineNumbers(content)))
						}
					}
				}
				break
			}
		}
		prompt.WriteString("\n")
	}
	prompt.WriteString("=== END LINKED WORKSPACES ===\n\n")
}

func appendWorkspacePromptSection(prompt *strings.Builder, scope string, ctxMap map[string]interface{}) {
	prompt.WriteString("\n=== WORKSPACE CONTEXT ===\n")
	switch scope {
	case ContextScopeHint:
		prompt.WriteString("The user has a project open in their editor, but has NOT shared file contents. ")
		prompt.WriteString("Do NOT assume their question is about this project unless they explicitly refer to it, paths, or code. ")
		prompt.WriteString("Answer general questions directly without inventing repo-specific details.\n\n")
	case ContextScopeOutline:
		prompt.WriteString("The user shared a high-level view of their project (name, path, file tree). ")
		prompt.WriteString("This workspace is available on disk at the Path below — use read_file tools or REFERENCED FILES sections; ")
		prompt.WriteString("NEVER ask the user to paste or share file contents from this project. ")
		prompt.WriteString("When REFERENCED FILES or WORKSPACE SOURCE FILES sections appear below, those are real disk content — use them.\n\n")
	case ContextScopeFocus:
		prompt.WriteString("The user shared limited code context (referenced paths and/or active file). ")
		prompt.WriteString("Stay within the files shown; do not assume other parts of the repo. ")
		prompt.WriteString("Line numbers in code blocks match the user's editor.\n\n")
	default:
		prompt.WriteString("The user has shared their active codebase with you. ")
		prompt.WriteString("Use the code context below when the user's request is code-specific (review/debug/explain/edit paths/files). ")
		prompt.WriteString("For capability or planning questions, answer directly first and only reference code when relevant. ")
		prompt.WriteString("Each line of code is prefixed with its real line number (e.g., '  42 | code here'). ")
		prompt.WriteString("When referencing code, use THESE line numbers. ")
		prompt.WriteString("When suggesting code changes, submit a file-change proposal that the user can approve.\n\n")
		prompt.WriteString("Important: you can propose create/edit/delete changes for the currently shared workspace, but the user must approve before changes are applied.\n\n")
	}

	if name, ok := ctxMap["workspace_name"].(string); ok && name != "" {
		prompt.WriteString(fmt.Sprintf("Project: %s\n", name))
	}
	if path, ok := ctxMap["workspace_path"].(string); ok && path != "" {
		prompt.WriteString(fmt.Sprintf("Path: %s\n", path))
	}
	appendScanSummaryContext(prompt, ctxMap)
	appendScanAnalysisContext(prompt, ctxMap)
	appendCadContext(prompt, ctxMap)

	includeTree := scope == ContextScopeOutline || scope == ContextScopeFocus || scope == ContextScopeFull
	if includeTree {
		if tree, ok := ctxMap["file_tree"].(string); ok && tree != "" {
			prompt.WriteString(fmt.Sprintf("\nProject file tree:\n%s\n", tree))
		}
	}

	includeFiles := scope == ContextScopeFocus || scope == ContextScopeFull
	if includeFiles {
		if files, ok := ctxMap["open_files"].([]interface{}); ok && len(files) > 0 {
			prompt.WriteString(fmt.Sprintf("\nOpen files (%d):\n", len(files)))
			for _, f := range files {
				fm, ok := f.(map[string]interface{})
				if !ok {
					continue
				}
				filePath, _ := fm["path"].(string)
				lang, _ := fm["language"].(string)
				content, _ := fm["content"].(string)
				isActive, _ := fm["is_active"].(bool)
				activeMarker := ""
				if isActive {
					activeMarker = " [ACTIVE - user is viewing this file]"
				}
				numberedContent := addLineNumbers(content)
				prompt.WriteString(fmt.Sprintf(
					"\n### %s (%s)%s\n```%s\n%s\n```\n",
					filePath, lang, activeMarker, lang, numberedContent,
				))
				if isActive {
					if start, ok := fm["selection_start_line"].(float64); ok {
						if end, ok2 := fm["selection_end_line"].(float64); ok2 {
							selText, _ := fm["selected_text"].(string)
							if strings.TrimSpace(selText) != "" {
								prompt.WriteString(fmt.Sprintf(
									"\nUser selection in %s (lines %d–%d):\n```\n%s\n```\n",
									filePath, int(start), int(end), strings.TrimSpace(selText),
								))
							}
						}
					}
				}
			}
		}
	}

	prompt.WriteString("=== END WORKSPACE CONTEXT ===\n\n")
}

func appendScanSummaryContext(prompt *strings.Builder, ctxMap map[string]interface{}) {
	raw, ok := ctxMap["scan_summary"]
	if !ok || raw == nil {
		return
	}
	scan, ok := raw.(map[string]interface{})
	if !ok {
		return
	}
	prompt.WriteString("\nPhoenix scan summary context:\n")
	if dir, ok := scan["summary_dir"].(string); ok && dir != "" {
		prompt.WriteString(fmt.Sprintf("- Summary directory: %s\n", dir))
	}
	if count, ok := scan["wells_count"].(float64); ok {
		prompt.WriteString(fmt.Sprintf("- Wells with metadata: %d\n", int(count)))
	}
	if analytes, ok := scan["analytes"].([]interface{}); ok && len(analytes) > 0 {
		names := make([]string, 0, len(analytes))
		for _, a := range analytes {
			if s, ok := a.(string); ok && strings.TrimSpace(s) != "" {
				names = append(names, s)
			}
		}
		if len(names) > 0 {
			prompt.WriteString(fmt.Sprintf("- Analytes/controls: %s\n", strings.Join(names, ", ")))
		}
	}
	if note, ok := scan["note"].(string); ok && note != "" {
		prompt.WriteString(fmt.Sprintf("- Note: %s\n", note))
	}
	if active, ok := scan["active_well"].(map[string]interface{}); ok && len(active) > 0 {
		well, _ := active["well"].(string)
		if well != "" {
			prompt.WriteString(fmt.Sprintf("- Active well: %s", well))
			if spotCount, ok := active["spot_count"].(float64); ok {
				prompt.WriteString(fmt.Sprintf(" (%d spots)", int(spotCount)))
			}
			if t, ok := active["time"].(string); ok && t != "" {
				prompt.WriteString(fmt.Sprintf(", acquired %s", t))
			}
			prompt.WriteString("\n")
		}
		if spots, ok := active["spots"].([]interface{}); ok && len(spots) > 0 {
			prompt.WriteString("  Active well spot layout (first 64):\n")
			for _, rawSpot := range spots {
				spot, ok := rawSpot.(map[string]interface{})
				if !ok {
					continue
				}
				analyte, _ := spot["analyte"].(string)
				row, _ := spot["row"].(string)
				col, _ := spot["column"].(string)
				x, _ := spot["x_px"].(float64)
				y, _ := spot["y_px"].(float64)
				prompt.WriteString(fmt.Sprintf("  - %s r%s c%s at (%.0f, %.0f)\n", analyte, row, col, x, y))
			}
		}
	}
}

func appendScanAnalysisContext(prompt *strings.Builder, ctxMap map[string]interface{}) {
	raw, ok := ctxMap["scan_analysis"]
	if !ok || raw == nil {
		return
	}
	scan, ok := raw.(map[string]interface{})
	if !ok {
		return
	}
	prompt.WriteString("\nPhoenix scan analysis context:\n")
	if dir, ok := scan["analysis_dir"].(string); ok && dir != "" {
		prompt.WriteString(fmt.Sprintf("- Analysis directory: %s\n", dir))
		prompt.WriteString(fmt.Sprintf("- Use summarize_scan_analysis with path %q when the user asks for scan QC on the open file.\n", dir))
	}
	if product, ok := scan["product_name"].(string); ok && product != "" {
		prompt.WriteString(fmt.Sprintf("- Product: %s\n", product))
	}
	if df, ok := scan["dilution_factor"].(float64); ok && df > 0 && df != 1 {
		prompt.WriteString(fmt.Sprintf("- Dilution factor: %.1f (multiply unknown concentrations)\n", df))
	}
	if linked, ok := scan["linked_scan_dir"].(string); ok && linked != "" {
		prompt.WriteString(fmt.Sprintf("- Linked scan directory: %s\n", linked))
	}
	if analytes, ok := scan["analytes"].([]interface{}); ok && len(analytes) > 0 {
		names := make([]string, 0, len(analytes))
		for _, a := range analytes {
			if s, ok := a.(string); ok && strings.TrimSpace(s) != "" {
				names = append(names, s)
			}
		}
		if len(names) > 0 {
			prompt.WriteString(fmt.Sprintf("- Analytes: %s\n", strings.Join(names, ", ")))
		}
	}
	if note, ok := scan["note"].(string); ok && note != "" {
		prompt.WriteString(fmt.Sprintf("- Note: %s\n", note))
	}
	if activeAnalyte, ok := scan["active_analyte"].(string); ok && activeAnalyte != "" {
		prompt.WriteString(fmt.Sprintf("- Active analyte: %s\n", activeAnalyte))
	}
	if active, ok := scan["active_well"].(map[string]interface{}); ok && len(active) > 0 {
		well, _ := active["well"].(string)
		if well != "" {
			prompt.WriteString(fmt.Sprintf("- Active well: %s", well))
			if conc, ok := active["concentration"].(float64); ok {
				prompt.WriteString(fmt.Sprintf(", concentration %.4f pg/ml", conc))
			}
			if loq, ok := active["within_loq"].(bool); ok {
				prompt.WriteString(fmt.Sprintf(", within LOQ: %v", loq))
			}
			prompt.WriteString("\n")
		}
	}
}

func appendCadContext(prompt *strings.Builder, ctxMap map[string]interface{}) {
	raw, ok := ctxMap["cad"]
	if !ok || raw == nil {
		return
	}
	cad, ok := raw.(map[string]interface{})
	if !ok {
		return
	}
	prompt.WriteString("\nCAD workbench context:\n")
	if path, ok := cad["scad_path"].(string); ok && path != "" {
		prompt.WriteString(fmt.Sprintf("- OpenSCAD file: %s\n", path))
		prompt.WriteString(fmt.Sprintf("- Use write_openscad / render_openscad with path %q when the user asks to model or update the part.\n", path))
	}
	if pid, ok := cad["project_id"].(string); ok && pid != "" {
		prompt.WriteString(fmt.Sprintf("- Project id: %s\n", pid))
	}
	if note, ok := cad["note"].(string); ok && note != "" {
		prompt.WriteString(fmt.Sprintf("- Note: %s\n", note))
	}
}

// addLineNumbers prepends each line with its 1-based line number.
// The format matches common editor gutter style: right-aligned number, pipe separator.
// Example: "   1 | package main"
func addLineNumbers(content string) string {
	if content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Determine padding width based on total line count
	width := 4 // default for files up to 9999 lines
	if totalLines >= 10000 {
		width = 5
	} else if totalLines < 100 {
		width = 3
	}

	var sb strings.Builder
	// Pre-allocate roughly: each line gets ~(width+3) chars of prefix + original content + newline
	sb.Grow(len(content) + totalLines*(width+4))

	for i, line := range lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%*d | %s", width, i+1, line)
	}

	return sb.String()
}
