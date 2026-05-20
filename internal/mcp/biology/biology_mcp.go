package biology

import (
	"context"
	"fmt"
	"log"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// BiologyMCP provides MCP tools for life-sciences workflows.
type BiologyMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewBiologyMCP creates a new Biology MCP server.
func NewBiologyMCP() (*BiologyMCP, error) {
	config := mcp.GetMCPServerConfig("BIOLOGY")

	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	b := &BiologyMCP{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		config:     config,
	}

	b.registerTools()
	return b, nil
}

// Start starts the Biology MCP server.
func (b *BiologyMCP) Start() error {
	if b.httpServer == nil {
		return fmt.Errorf("MCP server not configured")
	}
	return mcp.StartMCPServer(b.httpServer, b.config.Port)
}

// GetMCPServer returns the underlying MCP server.
func (b *BiologyMCP) GetMCPServer() *server.MCPServer {
	return b.mcpServer
}

func (b *BiologyMCP) registerTools() {
	b.mcpServer.AddTool(mcp.CreateTool(
		"analyze_sequence",
		"Analyze a DNA, RNA, or protein sequence (length, type, validity, reverse complement for DNA). Research use only.",
		mcp.CreateStringInputSchema("sequence", "Raw sequence or FASTA text"),
		nil,
	), b.handleAnalyzeSequence)

	b.mcpServer.AddTool(mcp.CreateTool(
		"fold_protein",
		"Predict 3D protein structure from amino acid sequence using ESMFold (requires Hugging Face token in Settings). Writes PDB under the configured biology artifacts folder.",
		mcp.CreateStringInputSchema("sequence", "Amino acid sequence or FASTA (protein only)"),
		nil,
	), b.handleFoldProtein)

	log.Printf("Registered %d Biology MCP tools", len(b.mcpServer.ListTools()))
}

func (b *BiologyMCP) handleAnalyzeSequence(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"sequence"}); err != nil {
		return mcp.HandleToolError(err, "analyze_sequence"), nil
	}
	seq := request.GetString("sequence", "")
	out, err := analyzeSequenceText(seq)
	if err != nil {
		return mcp.HandleToolError(err, "analyze_sequence"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handleFoldProtein(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"sequence"}); err != nil {
		return mcp.HandleToolError(err, "fold_protein"), nil
	}
	seq := request.GetString("sequence", "")
	out, err := foldProteinSequence(ctx, seq)
	if err != nil {
		return mcp.HandleToolError(err, "fold_protein"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}
