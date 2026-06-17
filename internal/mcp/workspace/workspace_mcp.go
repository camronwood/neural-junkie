// Package workspace provides in-process MCP tools for reading and searching workspace files.
package workspace

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/codeindex"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
	"github.com/camronwood/neural-junkie/internal/workspacefiles"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const maxReadBytes = 50 * 1024

// RootResolver returns the workspace root for tool calls.
type RootResolver func() string

// AttachTools registers read_file, grep, glob_file_search, and list_dir on an MCP server.
func AttachTools(mcpServer *server.MCPServer, root RootResolver) {
	if mcpServer == nil || root == nil {
		return
	}
	w := &tools{root: root}
	w.register(mcpServer)
}

type tools struct {
	root RootResolver
}

func (w *tools) register(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.CreateTool(
		"read_file",
		"Read a file relative to the workspace root (optional line range)",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"path":        "Relative file path",
			"start_line":  "Optional 1-based start line",
			"end_line":    "Optional 1-based end line (inclusive)",
		}),
		nil,
	), w.handleReadFile)

	mcpServer.AddTool(mcp.CreateTool(
		"grep",
		"Search file contents under the workspace for a pattern (uses ripgrep when available)",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"pattern": "Search pattern",
			"path":    "Optional subdirectory relative to workspace",
		}),
		nil,
	), w.handleGrep)

	mcpServer.AddTool(mcp.CreateTool(
		"glob_file_search",
		"Find files matching a glob pattern under the workspace",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"pattern": "Glob pattern (e.g. **/*.go)",
		}),
		nil,
	), w.handleGlob)

	mcpServer.AddTool(mcp.CreateTool(
		"list_dir",
		"List files and directories under a workspace path",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"path": "Relative directory path (default .)",
		}),
		nil,
	), w.handleListDir)

	mcpServer.AddTool(mcp.CreateTool(
		"semantic_search",
		"Hybrid embedding + keyword search over the workspace codebase",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"query": "Natural language or keyword query",
			"limit": "Max chunks to return (default 8, max 20)",
		}),
		nil,
	), w.handleSemanticSearch)

	mcpServer.AddTool(mcp.CreateTool(
		"run_command",
		"Run an allowlisted verify command in the workspace (npm run build, npm test, go test, etc.). "+
			"Use npm run build to verify boot fixes — npm install is NOT allowlisted.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"command": "Shell command (allowlisted: npm run build/test/lint, go test, cargo test, etc.)",
			"cwd":     "Optional relative subdirectory",
		}),
		nil,
	), w.handleRunCommand)

	log.Printf("Registered workspace MCP tools on server")
}

func (w *tools) workspaceRoot() (string, error) {
	root := strings.TrimSpace(w.root())
	if root == "" {
		return "", fmt.Errorf("workspace root not set")
	}
	abs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("workspace not available: %s", root)
	}
	return abs, nil
}

func (w *tools) resolveRel(root, rel string) (string, error) {
	return pathutil.ResolveRelWithinRoot(root, rel)
}

func (w *tools) handleReadFile(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root, err := w.workspaceRoot()
	if err != nil {
		return mcp.HandleToolError(err, "read_file"), nil
	}
	rel := request.GetString("path", "")
	if rel == "" {
		return mcp.HandleToolError(fmt.Errorf("path is required"), "read_file"), nil
	}
	var content string
	if b := shared.BackendFromContext(ctx); b != nil {
		relPath := strings.TrimPrefix(rel, "/")
		data, err := b.ReadFile(ctx, relPath)
		if err != nil {
			return mcp.HandleToolError(err, "read_file"), nil
		}
		content = string(data)
	} else {
		full, err := w.resolveRel(root, rel)
		if err != nil {
			return mcp.HandleToolError(err, "read_file"), nil
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return mcp.HandleToolError(err, "read_file"), nil
		}
		content = string(data)
	}
	if len(content) > maxReadBytes {
		content = content[:maxReadBytes] + "\n...(truncated)"
	}
	startLine := parseIntDefault(request.GetString("start_line", ""), 0)
	endLine := parseIntDefault(request.GetString("end_line", ""), 0)
	if startLine > 0 || endLine > 0 {
		lines := strings.Split(content, "\n")
		if startLine <= 0 {
			startLine = 1
		}
		if endLine <= 0 || endLine > len(lines) {
			endLine = len(lines)
		}
		if startLine > len(lines) {
			startLine = len(lines)
		}
		content = strings.Join(lines[startLine-1:endLine], "\n")
	}
	return mcp.HandleToolSuccess(fmt.Sprintf("### %s\n```\n%s\n```", rel, content)), nil
}

func (w *tools) handleGrep(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root, err := w.workspaceRoot()
	if err != nil {
		return mcp.HandleToolError(err, "grep"), nil
	}
	pattern := request.GetString("pattern", "")
	if pattern == "" {
		return mcp.HandleToolError(fmt.Errorf("pattern is required"), "grep"), nil
	}
	sub := request.GetString("path", ".")
	if b := shared.BackendFromContext(ctx); b != nil {
		relCwd := strings.TrimPrefix(strings.TrimSpace(sub), "/")
		if relCwd == "" {
			relCwd = "."
		}
		res, err := b.Exec(ctx, workspacebackend.ExecRequest{
			Command: "rg",
			Args:    []string{"-n", "--no-heading", "--color=never", pattern, "."},
			RelCwd:  relCwd,
			Timeout: 60 * time.Second,
		})
		out := res.Stdout
		if res.Stderr != "" {
			if out != "" {
				out += "\n"
			}
			out += res.Stderr
		}
		if err == nil || strings.TrimSpace(out) != "" {
			if len(out) > maxReadBytes {
				out = out[:maxReadBytes] + "\n...(truncated)"
			}
			if strings.TrimSpace(out) == "" {
				return mcp.HandleToolSuccess("No matches found."), nil
			}
			return mcp.HandleToolSuccess(out), nil
		}
	}
	searchRoot, err := w.resolveRel(root, sub)
	if err != nil {
		return mcp.HandleToolError(err, "grep"), nil
	}
	if out, err := runRipgrep(ctx, searchRoot, pattern); err == nil && out != "" {
		if len(out) > maxReadBytes {
			out = out[:maxReadBytes] + "\n...(truncated)"
		}
		return mcp.HandleToolSuccess(out), nil
	}
	// Fallback: scan indexed paths
	paths, err := workspacefiles.Search(ctx, root, "", 200)
	if err != nil {
		return mcp.HandleToolError(err, "grep"), nil
	}
	var b strings.Builder
	patLower := strings.ToLower(pattern)
	for _, rel := range paths {
		if !strings.HasPrefix(rel, strings.TrimPrefix(sub, "./")) && sub != "." {
			continue
		}
		full, err := w.resolveRel(root, rel)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(strings.ToLower(line), patLower) {
				fmt.Fprintf(&b, "%s:%d:%s\n", rel, i+1, line)
			}
		}
		if b.Len() > maxReadBytes {
			break
		}
	}
	if b.Len() == 0 {
		return mcp.HandleToolSuccess("No matches found."), nil
	}
	out := b.String()
	if len(out) > maxReadBytes {
		out = out[:maxReadBytes] + "\n...(truncated)"
	}
	return mcp.HandleToolSuccess(out), nil
}

func runRipgrep(ctx context.Context, dir, pattern string) (string, error) {
	cmd := exec.CommandContext(ctx, "rg", "-n", "--no-heading", "--color=never", pattern, dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return "", err
		}
	}
	return string(out), nil
}

func (w *tools) handleGlob(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root, err := w.workspaceRoot()
	if err != nil {
		return mcp.HandleToolError(err, "glob_file_search"), nil
	}
	pattern := request.GetString("pattern", "")
	if pattern == "" {
		return mcp.HandleToolError(fmt.Errorf("pattern is required"), "glob_file_search"), nil
	}
	paths, err := workspacefiles.Search(ctx, root, "", 5000)
	if err != nil {
		return mcp.HandleToolError(err, "glob_file_search"), nil
	}
	var matches []string
	for _, p := range paths {
		ok, _ := filepath.Match(pattern, filepath.Base(p))
		if ok {
			matches = append(matches, p)
			continue
		}
		if ok, _ := doublestarMatch(pattern, p); ok {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		return mcp.HandleToolSuccess("No files matched."), nil
	}
	if len(matches) > 100 {
		matches = matches[:100]
	}
	return mcp.HandleToolSuccess(strings.Join(matches, "\n")), nil
}

func doublestarMatch(pattern, path string) (bool, error) {
	pattern = strings.ReplaceAll(pattern, "**", "*")
	return filepath.Match(pattern, path)
}

func (w *tools) handleListDir(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root, err := w.workspaceRoot()
	if err != nil {
		return mcp.HandleToolError(err, "list_dir"), nil
	}
	rel := request.GetString("path", ".")
	if b := shared.BackendFromContext(ctx); b != nil {
		entries, err := b.ReadDir(ctx, strings.TrimPrefix(rel, "/"))
		if err != nil {
			return mcp.HandleToolError(err, "list_dir"), nil
		}
		var lines []string
		for _, e := range entries {
			kind := "file"
			if e.IsDir {
				kind = "dir"
			}
			lines = append(lines, fmt.Sprintf("%s (%s)", e.Name, kind))
		}
		if len(lines) == 0 {
			return mcp.HandleToolSuccess("(empty directory)"), nil
		}
		return mcp.HandleToolSuccess(strings.Join(lines, "\n")), nil
	}
	dir, err := w.resolveRel(root, rel)
	if err != nil {
		return mcp.HandleToolError(err, "list_dir"), nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return mcp.HandleToolError(err, "list_dir"), nil
	}
	var lines []string
	for _, e := range entries {
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		lines = append(lines, fmt.Sprintf("%s (%s)", e.Name(), kind))
	}
	if len(lines) == 0 {
		return mcp.HandleToolSuccess("(empty directory)"), nil
	}
	return mcp.HandleToolSuccess(strings.Join(lines, "\n")), nil
}

func (w *tools) handleSemanticSearch(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root, err := w.workspaceRoot()
	if err != nil {
		return mcp.HandleToolError(err, "semantic_search"), nil
	}
	query := strings.TrimSpace(request.GetString("query", ""))
	if query == "" {
		return mcp.HandleToolError(fmt.Errorf("query is required"), "semantic_search"), nil
	}
	limit := parseIntDefault(request.GetString("limit", ""), 8)
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	codeindex.BuildIndexAsync(root)
	results, err := codeindex.Search(ctx, root, query, limit)
	if err != nil {
		return mcp.HandleToolError(err, "semantic_search"), nil
	}
	if len(results) == 0 {
		return mcp.HandleToolSuccess("No matching chunks found."), nil
	}
	var b strings.Builder
	for _, r := range results {
		content := r.Content
		if len(content) > 2000 {
			content = content[:2000] + "\n...(truncated)"
		}
		fmt.Fprintf(&b, "### %s\n```\n%s\n```\n\n", r.Path, content)
	}
	return mcp.HandleToolSuccess(b.String()), nil
}

func (w *tools) handleRunCommand(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root, err := w.workspaceRoot()
	if err != nil {
		return mcp.HandleToolError(err, "run_command"), nil
	}
	cmdStr := normalizeCommand(request.GetString("command", ""))
	if cmdStr == "" {
		return mcp.HandleToolError(fmt.Errorf("command is required"), "run_command"), nil
	}
	if !CommandAllowed(cmdStr) {
		return mcp.HandleToolError(fmt.Errorf("command not allowlisted: %s", cmdStr), "run_command"), nil
	}
	relCwd := strings.TrimSpace(request.GetString("cwd", ""))
	if b := shared.BackendFromContext(ctx); b != nil {
		if relCwd == "" {
			relCwd = "."
		}
		runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		res, err := b.Exec(runCtx, workspacebackend.ExecRequest{
			Command: "sh",
			Args:    []string{"-c", cmdStr},
			RelCwd:  strings.TrimPrefix(relCwd, "/"),
			Timeout: 60 * time.Second,
		})
		text := res.Stdout
		if res.Stderr != "" {
			if text != "" {
				text += "\n"
			}
			text += res.Stderr
		}
		if len(text) > maxReadBytes {
			text = text[:maxReadBytes] + "\n...(truncated)"
		}
		exitCode := res.ExitCode
		if err != nil && exitCode == 0 {
			exitCode = 1
		}
		return mcp.HandleToolSuccess(fmt.Sprintf("exit_code=%d\n%s", exitCode, text)), nil
	}
	cwd := root
	if sub := strings.TrimSpace(request.GetString("cwd", "")); sub != "" && sub != "." {
		resolved, err := w.resolveRel(root, sub)
		if err != nil {
			return mcp.HandleToolError(err, "run_command"), nil
		}
		cwd = resolved
	}
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "sh", "-c", cmdStr)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	text := string(out)
	if len(text) > maxReadBytes {
		text = text[:maxReadBytes] + "\n...(truncated)"
	}
	exitCode := 0
	if err != nil {
		exitCode = 1
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		}
	}
	summary := fmt.Sprintf("exit_code=%d\n%s", exitCode, text)
	if err != nil {
		return mcp.HandleToolSuccess(summary), nil
	}
	return mcp.HandleToolSuccess(summary), nil
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
