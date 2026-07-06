package biology

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/camronwood/neural-junkie/internal/biologysidecar"
)

var (
	sharedBiologyOnce sync.Once
	sharedBiologyMCP  *BiologyMCP
	sharedBiologyErr  error
)

// SharedBiologyMCP returns a process-wide Biology MCP server (one HTTP listener).
func SharedBiologyMCP() (*BiologyMCP, error) {
	sharedBiologyOnce.Do(func() {
		sharedBiologyMCP, sharedBiologyErr = NewBiologyMCP()
	})
	return sharedBiologyMCP, sharedBiologyErr
}

func sidecarPostJSON(ctx context.Context, path string, body map[string]any) (string, error) {
	client := biologysidecar.DefaultSidecarClient
	if client == nil || !client.Available() {
		return "", fmt.Errorf("biology sidecar not available (enable Life sciences pack)")
	}
	out, err := client.Post(ctx, path, body)
	if err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (b *BiologyMCP) registerSidecarTools() {
	b.mcpServer.AddTool(mcp.CreateTool(
		"blast_search",
		"Submit a BLAST query via the biology sidecar (NCBI REST). Research use only.",
		mcp.CreateStringInputSchema("query", "Protein or nucleotide query sequence"),
		nil,
	), b.handleBlastSearch)

	b.mcpServer.AddTool(mcp.CreateTool(
		"pathway_lookup",
		"Look up pathways for a gene name or symbol via Reactome (read-only).",
		mcp.CreateStringInputSchema("gene", "Gene name or symbol"),
		nil,
	), b.handlePathwayLookup)

	b.mcpServer.AddTool(mcp.CreateTool(
		"structure_metadata",
		"Summarize a PDB/mmCIF file: atom count, chains, mean B-factor.",
		mcp.CreateStringInputSchema("path", "Absolute or workspace path to structure file"),
		nil,
	), b.handleStructureMetadata)

	b.mcpServer.AddTool(mcp.CreateTool(
		"validate_smiles",
		"Validate and canonicalize a SMILES string (requires RDKit in biology sidecar).",
		mcp.CreateStringInputSchema("smiles", "SMILES string"),
		nil,
	), b.handleValidateSMILES)

	b.mcpServer.AddTool(mcp.CreateTool(
		"mol_descriptors",
		"Compute basic molecular descriptors for a SMILES string (requires RDKit).",
		mcp.CreateStringInputSchema("smiles", "SMILES string"),
		nil,
	), b.handleMolDescriptors)
}

func (b *BiologyMCP) handleBlastSearch(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"query"}); err != nil {
		return mcp.HandleToolError(err, "blast_search"), nil
	}
	out, err := sidecarPostJSON(ctx, "/api/biology/blast", map[string]any{
		"query": request.GetString("query", ""),
	})
	if err != nil {
		return mcp.HandleToolError(err, "blast_search"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handlePathwayLookup(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	gene := request.GetString("gene", "")
	if strings.TrimSpace(gene) == "" {
		gene = request.GetString("query", "")
	}
	if strings.TrimSpace(gene) == "" {
		return mcp.HandleToolError(fmt.Errorf("gene required"), "pathway_lookup"), nil
	}
	out, err := sidecarPostJSON(ctx, "/api/biology/pathway", map[string]any{"gene": gene})
	if err != nil {
		return mcp.HandleToolError(err, "pathway_lookup"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handleStructureMetadata(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"path"}); err != nil {
		return mcp.HandleToolError(err, "structure_metadata"), nil
	}
	out, err := sidecarPostJSON(ctx, "/api/biology/structure-metadata", map[string]any{
		"path": request.GetString("path", ""),
	})
	if err != nil {
		return mcp.HandleToolError(err, "structure_metadata"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handleValidateSMILES(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"smiles"}); err != nil {
		return mcp.HandleToolError(err, "validate_smiles"), nil
	}
	out, err := sidecarPostJSON(ctx, "/api/biology/validate-smiles", map[string]any{
		"smiles": request.GetString("smiles", ""),
	})
	if err != nil {
		return mcp.HandleToolError(err, "validate_smiles"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (b *BiologyMCP) handleMolDescriptors(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"smiles"}); err != nil {
		return mcp.HandleToolError(err, "mol_descriptors"), nil
	}
	out, err := sidecarPostJSON(ctx, "/api/biology/mol-descriptors", map[string]any{
		"smiles": request.GetString("smiles", ""),
	})
	if err != nil {
		return mcp.HandleToolError(err, "mol_descriptors"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

// ToolAllowlistForAgentType returns MCP tool names for a life-sciences specialist.
func ToolAllowlistForAgentType(agentType string) []string {
	switch strings.ToLower(strings.TrimSpace(agentType)) {
	case "genomics":
		return []string{
			"analyze_sequence", "blast_search", "pathway_lookup",
			"summarize_scan_summary", "summarize_scan_analysis", "run_12plex_qc",
			"summarize_panel_qc", "summarize_comparator_output", "run_secondary_analysis",
		}
	case "structural-biology":
		return []string{"fold_protein", "structure_metadata"}
	case "cheminformatics":
		return []string{"validate_smiles", "mol_descriptors"}
	default:
		return nil
	}
}
