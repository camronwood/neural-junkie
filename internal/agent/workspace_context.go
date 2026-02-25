package agent

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// File size limits for auto-loaded files
const (
	maxFileSize      = 50 * 1024  // 50KB per file
	maxTotalFileSize = 200 * 1024 // 200KB total across all referenced files
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
//   - path/to/file.ext (relative with at least one directory separator)
//   - ./path/to/file.ext (dot-relative)
//   - /absolute/path/file.ext (absolute)
//
// Requires a file extension to reduce false positives.
var filePathPattern = regexp.MustCompile(`(?:^|\s|["'\x60(])([./]?(?:[a-zA-Z0-9_\-]+/)+[a-zA-Z0-9_\-]+\.[a-zA-Z0-9]+)`)

// DetectFilePaths extracts file paths from a message string.
// It returns deduplicated paths that look like real files (have extensions, directory separators).
func DetectFilePaths(content string) []string {
	matches := filePathPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var paths []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		p := match[1]

		// Skip URLs
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			continue
		}

		// Skip binary files
		ext := strings.ToLower(filepath.Ext(p))
		if binaryExtensions[ext] {
			continue
		}

		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}

	return paths
}

// ReadFileForPrompt reads a file and returns its content with line numbers,
// capped to maxFileSize. Returns empty string if file can't be read.
func ReadFileForPrompt(filePath string, workspacePath string) (content string, resolvedPath string, err error) {
	// Resolve relative paths against workspace
	resolved := filePath
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(workspacePath, resolved)
	}

	// Clean the path to prevent traversal
	resolved = filepath.Clean(resolved)

	// Safety: ensure the resolved path is within the workspace
	absWorkspace, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", "", fmt.Errorf("invalid workspace path: %w", err)
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", "", fmt.Errorf("invalid file path: %w", err)
	}
	if !strings.HasPrefix(absResolved, absWorkspace+string(filepath.Separator)) && absResolved != absWorkspace {
		return "", "", fmt.Errorf("path %s is outside workspace", filePath)
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

// AppendWorkspaceContext checks for workspace_context in message metadata
// and appends it to the prompt builder so the agent can reference project files.
//
// The framing instructions are placed BEFORE the code so the model reads the
// directive first, then processes the code with that directive in mind.
//
// Line numbers are prepended to every line of code so the model can reference
// exact line numbers that match what the user sees in their editor.
func AppendWorkspaceContext(prompt *strings.Builder, msg *protocol.Message) {
	if msg.Metadata == nil {
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

	prompt.WriteString("\n=== WORKSPACE CONTEXT ===\n")
	prompt.WriteString("The user has shared their active codebase with you. ")
	prompt.WriteString("Use the code context below when the user's request is code-specific (review/debug/explain/edit paths/files). ")
	prompt.WriteString("For capability or planning questions, answer directly first and only reference code when relevant. ")
	prompt.WriteString("Each line of code is prefixed with its real line number (e.g., '  42 | code here'). ")
	prompt.WriteString("When referencing code, use THESE line numbers -- they match what the user sees in their editor. ")
	prompt.WriteString("When suggesting code changes, submit a file-change proposal that the user can approve.\n\n")
	prompt.WriteString("Important: you can propose create/edit/delete changes for the currently shared workspace, but the user must approve before changes are applied.\n\n")

	if name, ok := ctxMap["workspace_name"].(string); ok && name != "" {
		prompt.WriteString(fmt.Sprintf("Project: %s\n", name))
	}
	if path, ok := ctxMap["workspace_path"].(string); ok && path != "" {
		prompt.WriteString(fmt.Sprintf("Path: %s\n", path))
	}
	if tree, ok := ctxMap["file_tree"].(string); ok && tree != "" {
		prompt.WriteString(fmt.Sprintf("\nProject file tree:\n%s\n", tree))
	}

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

			// Add line numbers to the code content so the model can reference
			// exact lines that match the user's editor.
			numberedContent := addLineNumbers(content)

			prompt.WriteString(fmt.Sprintf(
				"\n### %s (%s)%s\n```%s\n%s\n```\n",
				filePath, lang, activeMarker, lang, numberedContent,
			))
		}
	}

	prompt.WriteString("=== END WORKSPACE CONTEXT ===\n\n")
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
