package repomcp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/codeintel"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/repo"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RepoMCP provides in-process MCP tools scoped to a repository index.
type RepoMCP struct {
	mcpServer *server.MCPServer
	repoPath  string
	mu        sync.RWMutex
	index     *repo.RepositoryIndex
}

// NewRepoMCP creates an in-process MCP server for a repo agent.
func NewRepoMCP(repoPath string) (*RepoMCP, error) {
	mcpServer, err := mcp.NewInProcessMCPServer("repo-agent-mcp", "1.0.0")
	if err != nil {
		return nil, fmt.Errorf("create repo MCP: %w", err)
	}
	r := &RepoMCP{mcpServer: mcpServer, repoPath: repoPath}
	r.registerTools()
	return r, nil
}

func (r *RepoMCP) GetMCPServer() *server.MCPServer {
	return r.mcpServer
}

func (r *RepoMCP) Start() error {
	return nil
}

func (r *RepoMCP) SetIndex(index *repo.RepositoryIndex) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.index = index
}

func (r *RepoMCP) registerTools() {
	r.mcpServer.AddTool(mcp.CreateTool(
		"search_codebase",
		"Search the indexed repository for files relevant to a query",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"query":      "Search query",
			"max_files":  "Optional max files to return (default 5)",
		}),
		nil,
	), r.handleSearchCodebase)

	r.mcpServer.AddTool(mcp.CreateTool(
		"get_file_content",
		"Get indexed source file content by path",
		mcp.CreateStringInputSchema("file_path", "Relative file path in the repository"),
		nil,
	), r.handleGetFileContent)

	r.mcpServer.AddTool(mcp.CreateTool(
		"search_by_path",
		"Find indexed files matching a path pattern",
		mcp.CreateStringInputSchema("path_pattern", "Path substring or pattern"),
		nil,
	), r.handleSearchByPath)

	r.mcpServer.AddTool(mcp.CreateTool(
		"list_key_files",
		"List README, config, and other key files from the repository index",
		mcp.CreateStringInputSchema("unused", "Leave empty"),
		nil,
	), r.handleListKeyFiles)

	r.mcpServer.AddTool(mcp.CreateTool(
		"git_log",
		"Show recent git commit history for the repository",
		mcp.CreateStringInputSchema("unused", "Leave empty"),
		nil,
	), r.handleGitLog)

	log.Printf("Registered %d Repo MCP tools", len(r.mcpServer.ListTools()))
}

func (r *RepoMCP) getIndex() *repo.RepositoryIndex {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.index
}

func (r *RepoMCP) handleSearchCodebase(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	query := request.GetString("query", "")
	if query == "" {
		return mcp.HandleToolError(fmt.Errorf("query is required"), "search_codebase"), nil
	}
	maxFiles := 5
	if r.repoPath != "" {
		results, err := codeintel.SemanticSearch(ctx, r.repoPath, query, maxFiles)
		if err == nil && len(results) > 0 {
			var b strings.Builder
			for _, hit := range results {
				snippet := hit.Content
				if len(snippet) > 500 {
					snippet = snippet[:500] + "..."
				}
				fmt.Fprintf(&b, "### %s\n%s\n\n", hit.Path, snippet)
			}
			return mcp.HandleToolSuccess(b.String()), nil
		}
	}
	index := r.getIndex()
	if index == nil {
		return mcp.HandleToolError(fmt.Errorf("repository index not ready"), "search_codebase"), nil
	}
	files := repo.SearchRelevantFiles(query, index, maxFiles)
	if len(files) == 0 {
		return mcp.HandleToolSuccess("No matching files found."), nil
	}
	var b strings.Builder
	for _, f := range files {
		content, err := repo.DecompressContent(f.Content)
		if err != nil {
			continue
		}
		snippet := content
		if len(snippet) > 500 {
			snippet = snippet[:500] + "..."
		}
		fmt.Fprintf(&b, "### %s (%s)\n%s\n\n", f.Path, f.Language, snippet)
	}
	return mcp.HandleToolSuccess(b.String()), nil
}

func (r *RepoMCP) handleGetFileContent(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	filePath := request.GetString("file_path", "")
	index := r.getIndex()
	if index == nil {
		return mcp.HandleToolError(fmt.Errorf("index not ready"), "get_file_content"), nil
	}
	f, ok := index.SourceFiles[filePath]
	if !ok {
		matches := repo.SearchByPath(filePath, index)
		if len(matches) == 0 {
			return mcp.HandleToolError(fmt.Errorf("file not in index: %s", filePath), "get_file_content"), nil
		}
		f = matches[0]
	}
	content, err := repo.DecompressContent(f.Content)
	if err != nil {
		return mcp.HandleToolError(err, "get_file_content"), nil
	}
	return mcp.HandleToolSuccess(fmt.Sprintf("### %s\n```\n%s\n```", f.Path, content)), nil
}

func (r *RepoMCP) handleSearchByPath(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	pattern := request.GetString("path_pattern", "")
	index := r.getIndex()
	if index == nil {
		return mcp.HandleToolError(fmt.Errorf("index not ready"), "search_by_path"), nil
	}
	files := repo.SearchByPath(pattern, index)
	if len(files) == 0 {
		return mcp.HandleToolSuccess("No files matched."), nil
	}
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	return mcp.HandleToolSuccess(strings.Join(paths, "\n")), nil
}

func (r *RepoMCP) handleListKeyFiles(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	index := r.getIndex()
	if index == nil {
		return mcp.HandleToolError(fmt.Errorf("index not ready"), "list_key_files"), nil
	}
	var b strings.Builder
	for path, content := range index.KeyFiles {
		summary := content
		if len(summary) > 120 {
			summary = summary[:120] + "..."
		}
		fmt.Fprintf(&b, "%s: %s\n", path, summary)
	}
	if b.Len() == 0 {
		return mcp.HandleToolSuccess("No key files indexed."), nil
	}
	return mcp.HandleToolSuccess(b.String()), nil
}

func (r *RepoMCP) handleGitLog(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	out, err := shared.RunCommand(ctx, r.repoPath, "git", "log", "-n", "20", "--oneline")
	if err != nil {
		return mcp.HandleToolSuccess(mcp.MissingBinaryMessage("git", "Ensure the repository is a git checkout.") + "\n" + out), nil
	}
	return mcp.HandleToolSuccess(out), nil
}
