package biology

import (
	"strings"
	"sync"
)

var (
	sharedBiologyOnce sync.Once
	sharedBiologyMCP  *BiologyMCP
	sharedBiologyErr  error
)

// SharedBiologyMCP returns a process-wide scan/QC Biology MCP server (one HTTP listener).
func SharedBiologyMCP() (*BiologyMCP, error) {
	sharedBiologyOnce.Do(func() {
		sharedBiologyMCP, sharedBiologyErr = NewBiologyMCP()
	})
	return sharedBiologyMCP, sharedBiologyErr
}

// ToolAllowlistForAgentType returns MCP tool names for a life-sciences specialist.
// Official research tools come from the pack hub catalog; scan tools stay host-gated.
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
