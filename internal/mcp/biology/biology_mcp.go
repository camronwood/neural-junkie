package biology

import (
	"context"
	"fmt"
	"log"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// BiologyMCP provides customer-gated Phoenix/scan QC MCP tools.
// Official research tools (sequence, fold, BLAST, SMILES) live in the
// life-sciences pack hub (assets/mcp/tools.json + POST /mcp/call).
type BiologyMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewBiologyMCP creates a scan/QC-only Biology MCP server.
func NewBiologyMCP() (*BiologyMCP, error) {
	config := mcp.GetMCPServerConfig("biology")
	if cfg := mcp.AppConfig(); cfg != nil && !config.Enabled {
		for _, t := range []string{"genomics", "structural-biology", "cheminformatics"} {
			if cfg.MCPEnabledForAgent(t) {
				config.Enabled = true
				break
			}
		}
	}

	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	b := &BiologyMCP{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		config:     config,
	}

	b.registerScanTools()
	return b, nil
}

// Start starts the Biology MCP server.
func (b *BiologyMCP) Start() error {
	return mcp.StartMCPServer(b.httpServer, b.config.Port)
}

// GetMCPServer returns the underlying MCP server.
func (b *BiologyMCP) GetMCPServer() *server.MCPServer {
	return b.mcpServer
}

// AttachCustomerScanTools registers Phoenix/scan QC tools on an existing MCP server
// (e.g. pack hub MCP for life-sciences specialists).
func AttachCustomerScanTools(mcpServer *server.MCPServer) {
	if mcpServer == nil {
		return
	}
	b := &BiologyMCP{mcpServer: mcpServer}
	b.registerScanTools()
}

func (b *BiologyMCP) registerScanTools() {
	if b == nil || b.mcpServer == nil {
		return
	}
	b.mcpServer.AddTool(mcp.CreateTool(
		"summarize_scan_summary",
		"Summarize a Phoenix-style scan summary folder (imageMetadata.json + well TIFFs): well counts, analyte spot distribution, and QC flags. Path may be the summary directory, scan-export/, or imageMetadata.json.",
		mcp.CreateStringInputSchema("path", "Absolute or workspace path to scan summary directory, scan-export/, or imageMetadata.json"),
		nil,
	), b.handleSummarizeScanSummary)

	b.mcpServer.AddTool(mcp.CreateTool(
		"summarize_scan_analysis",
		"Summarize a Phoenix-style scan analysis export: reports/results.json and/or reports/{analyte}_summary_report.csv, plus optional process_report.txt. Returns QC stats (LOQ counts, dilution factor, analyte list) and process report excerpt. Path may be the analysis directory, reports/results.json, or a summary CSV file.",
		mcp.CreateStringInputSchema("path", "Absolute or workspace path to scan analysis directory, reports/results.json, or reports/{analyte}_summary_report.csv"),
		nil,
	), b.handleSummarizeScanAnalysis)

	b.mcpServer.AddTool(mcp.CreateTool(
		"run_12plex_qc",
		"Run Human Inflammatory 12-Plex SOP QC on a Phoenix-style scan analysis export. Returns per-analyte pass/fail for LLOQ, ULOQ, intraplate CV, column/row deviation, and spike recovery.",
		mcp.CreateStringInputSchema("path", "Absolute or workspace path to scan analysis directory or reports/results.json"),
		nil,
	), b.handleRun12PlexQC)

	b.mcpServer.AddTool(mcp.CreateTool(
		"summarize_panel_qc",
		"Alias for run_12plex_qc — returns 12-Plex SOP QC pass/fail markdown for a scan analysis export.",
		mcp.CreateStringInputSchema("path", "Absolute or workspace path to scan analysis directory"),
		nil,
	), b.handleSummarizePanelQC)

	b.mcpServer.AddTool(mcp.CreateTool(
		"summarize_comparator_output",
		"Summarize a Plate Comparator Analysis output folder (Summary Statistics/LLOQs_and_ULOQs.csv and per-plate stats).",
		mcp.CreateStringInputSchema("path", "Absolute or workspace path to Comparator Analysis folder"),
		nil,
	), b.handleSummarizeComparatorOutput)

	b.mcpServer.AddTool(mcp.CreateTool(
		"run_secondary_analysis",
		"Run a secondary analysis workflow. Supports 12plex_qc and summarize_comparator inline; other workflows (comparator, endogenous, std_curves, print_order) use the Secondary Analysis panel.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"workflow": map[string]interface{}{
				"type":        "string",
				"description": "Workflow name: 12plex_qc, summarize_comparator, comparator, endogenous, std_curves, print_order",
			},
			"config_json": map[string]interface{}{
				"type":        "string",
				"description": "JSON object with workflow-specific paths and options",
			},
		}, []string{"workflow"}),
		nil,
	), b.handleRunSecondaryAnalysis)

	log.Printf("Registered Biology scan/QC MCP tools")
}

func (b *BiologyMCP) requireScanTool(toolName string) error {
	cfg := mcp.AppConfig()
	if cfg == nil || !cfg.ScanMCPToolAllowed(toolName) {
		return fmt.Errorf("%s requires an enabled custom pack with the appropriate capability (see capability_defs in your lab pack)", toolName)
	}
	return nil
}

func (b *BiologyMCP) handleSummarizeScanSummary(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := b.requireScanTool("summarize_scan_summary"); err != nil {
		return mcp.HandleToolError(err, "summarize_scan_summary"), nil
	}
	if err := mcp.ValidateToolInput(request, []string{"path"}); err != nil {
		return mcp.HandleToolError(err, "summarize_scan_summary"), nil
	}
	path := request.GetString("path", "")
	out, err := summarizeScanSummaryPath(path)
	if err != nil {
		return mcp.HandleToolError(err, "summarize_scan_summary"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handleSummarizeScanAnalysis(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := b.requireScanTool("summarize_scan_analysis"); err != nil {
		return mcp.HandleToolError(err, "summarize_scan_analysis"), nil
	}
	if err := mcp.ValidateToolInput(request, []string{"path"}); err != nil {
		return mcp.HandleToolError(err, "summarize_scan_analysis"), nil
	}
	path := request.GetString("path", "")
	out, err := summarizeScanAnalysisPath(path)
	if err != nil {
		return mcp.HandleToolError(err, "summarize_scan_analysis"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handleRun12PlexQC(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := b.requireScanTool("run_12plex_qc"); err != nil {
		return mcp.HandleToolError(err, "run_12plex_qc"), nil
	}
	if err := mcp.ValidateToolInput(request, []string{"path"}); err != nil {
		return mcp.HandleToolError(err, "run_12plex_qc"), nil
	}
	out, err := run12PlexQCPath(request.GetString("path", ""), false)
	if err != nil {
		return mcp.HandleToolError(err, "run_12plex_qc"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handleSummarizePanelQC(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := b.requireScanTool("summarize_panel_qc"); err != nil {
		return mcp.HandleToolError(err, "summarize_panel_qc"), nil
	}
	if err := mcp.ValidateToolInput(request, []string{"path"}); err != nil {
		return mcp.HandleToolError(err, "summarize_panel_qc"), nil
	}
	out, err := summarizePanelQCPath(request.GetString("path", ""))
	if err != nil {
		return mcp.HandleToolError(err, "summarize_panel_qc"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handleSummarizeComparatorOutput(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := b.requireScanTool("summarize_comparator_output"); err != nil {
		return mcp.HandleToolError(err, "summarize_comparator_output"), nil
	}
	if err := mcp.ValidateToolInput(request, []string{"path"}); err != nil {
		return mcp.HandleToolError(err, "summarize_comparator_output"), nil
	}
	out, err := summarizeComparatorOutputPath(request.GetString("path", ""))
	if err != nil {
		return mcp.HandleToolError(err, "summarize_comparator_output"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handleRunSecondaryAnalysis(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := b.requireScanTool("run_secondary_analysis"); err != nil {
		return mcp.HandleToolError(err, "run_secondary_analysis"), nil
	}
	if err := mcp.ValidateToolInput(request, []string{"workflow"}); err != nil {
		return mcp.HandleToolError(err, "run_secondary_analysis"), nil
	}
	out, err := runSecondaryAnalysisWorkflow(
		request.GetString("workflow", ""),
		request.GetString("config_json", "{}"),
	)
	if err != nil {
		return mcp.HandleToolError(err, "run_secondary_analysis"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}
