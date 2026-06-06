package rust

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RustMCP provides MCP tools for Rust development.
type RustMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewRustMCP creates a new Rust MCP server.
func NewRustMCP() (*RustMCP, error) {
	config := mcp.GetMCPServerConfig("rust")
	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	r := &RustMCP{mcpServer: mcpServer, httpServer: httpServer, config: config}
	r.registerTools()
	return r, nil
}

func (r *RustMCP) Start() error {
	return mcp.StartMCPServer(r.httpServer, r.config.Port)
}

func (r *RustMCP) GetMCPServer() *server.MCPServer {
	return r.mcpServer
}

func (r *RustMCP) registerTools() {
	r.mcpServer.AddTool(mcp.CreateTool(
		"cargo_clippy",
		"Run cargo clippy lints on a Rust crate",
		mcp.CreateStringInputSchema("crate_path", "Path to Cargo.toml directory"),
		nil,
	), r.handleCargoClippy)

	r.mcpServer.AddTool(mcp.CreateTool(
		"cargo_test",
		"Run cargo test in a Rust crate",
		mcp.CreateStringInputSchema("crate_path", "Path to Cargo.toml directory"),
		nil,
	), r.handleCargoTest)

	r.mcpServer.AddTool(mcp.CreateTool(
		"cargo_audit",
		"Run cargo audit for crate vulnerability advisories",
		mcp.CreateStringInputSchema("crate_path", "Path to Cargo.toml directory"),
		nil,
	), r.handleCargoAudit)

	r.mcpServer.AddTool(mcp.CreateTool(
		"check_cargo_toml",
		"Parse Cargo.toml and summarize package metadata and dependencies",
		mcp.CreateStringInputSchema("crate_path", "Path to Cargo.toml directory"),
		nil,
	), r.handleCheckCargoToml)

	log.Printf("Registered %d Rust MCP tools", len(r.mcpServer.ListTools()))
}

func (r *RustMCP) crateRoot(cratePath string) string {
	return shared.FindProjectRoot(cratePath, "Cargo.toml")
}

func (r *RustMCP) handleCargoClippy(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root := r.crateRoot(request.GetString("crate_path", "."))
	out, err := shared.RunCommand(ctx, root, "cargo", "clippy", "--message-format=short")
	return mcp.HandleToolSuccess(shared.FormatCommandResult("cargo clippy:", out, err)), nil
}

func (r *RustMCP) handleCargoTest(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root := r.crateRoot(request.GetString("crate_path", "."))
	out, err := shared.RunCommand(ctx, root, "cargo", "test")
	return mcp.HandleToolSuccess(shared.FormatCommandResult("cargo test:", out, err)), nil
}

func (r *RustMCP) handleCargoAudit(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root := r.crateRoot(request.GetString("crate_path", "."))
	out, err := shared.RunCommand(ctx, root, "cargo", "audit")
	if err != nil && strings.Contains(out, "no such subcommand") {
		return mcp.HandleToolSuccess(mcp.MissingBinaryMessage("cargo-audit", "Install: cargo install cargo-audit")), nil
	}
	return mcp.HandleToolSuccess(shared.FormatCommandResult("cargo audit:", out, err)), nil
}

func (r *RustMCP) handleCheckCargoToml(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root := r.crateRoot(request.GetString("crate_path", "."))
	path := filepath.Join(root, "Cargo.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("Cargo.toml not found: %w", err), "check_cargo_toml"), nil
	}
	var manifest struct {
		Package struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Edition string `json:"edition"`
		} `json:"package"`
		Dependencies map[string]any `json:"dependencies"`
		Features     map[string]any `json:"features"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return mcp.HandleToolSuccess(fmt.Sprintf("Cargo.toml (%d bytes):\n%s", len(data), string(data))), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Crate: %s v%s (edition %s)\n", manifest.Package.Name, manifest.Package.Version, manifest.Package.Edition)
	fmt.Fprintf(&b, "Dependencies: %d\nFeatures: %d\n", len(manifest.Dependencies), len(manifest.Features))
	return mcp.HandleToolSuccess(b.String()), nil
}
