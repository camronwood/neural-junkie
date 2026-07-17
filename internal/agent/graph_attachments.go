package agent

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/codeindex/graph"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

var pathBetweenRE = regexp.MustCompile(`(?i)path\s+between\s+([A-Za-z0-9_./:-]+)\s+and\s+([A-Za-z0-9_./:-]+)`)

// MergeGraphNeighborhoodForRoute injects graph path/neighborhood context into prompt attachments.
func MergeGraphNeighborhoodForRoute(msg *protocol.Message, plan routing.KnowledgePlan) bool {
	if msg == nil || !plan.Has(routing.RouteCodeGraph) {
		return false
	}
	repoPath := workspacePathFromMetadata(msg)
	if repoPath == "" {
		return false
	}
	meta, err := graph.Status(repoPath)
	if err != nil {
		return false
	}
	if !meta.Ready {
		if !meta.Building {
			graph.BuildIndexAsync(repoPath)
		}
		return false
	}

	content := strings.TrimSpace(msg.Content)
	var injected []map[string]interface{}

	if m := pathBetweenRE.FindStringSubmatch(content); len(m) == 3 {
		pr, err := graph.ShortestPath(repoPath, m[1], m[2])
		if err == nil && pr.Found {
			text := graph.FormatPathForPrompt(pr)
			if text != "" {
				injected = append(injected, map[string]interface{}{
					"type":    "code_graph_path",
					"content": text,
					"from":    m[1],
					"to":      m[2],
				})
			}
		}
	}

	seeds := codebaseIdentifierRE.FindAllString(content, -1)
	if len(seeds) == 0 {
		seeds = codebaseSymbolRE.FindAllString(content, -1)
	}
	for i, seed := range seeds {
		if i >= 3 {
			break
		}
		sg, err := graph.QuerySubgraph(repoPath, seed, 1, 40)
		if err != nil || len(sg.Nodes) == 0 {
			continue
		}
		text := graph.FormatNeighborhoodForPrompt(sg)
		if text == "" {
			continue
		}
		injected = append(injected, map[string]interface{}{
			"type":    "code_graph_neighborhood",
			"content": text,
			"seed":    seed,
		})
	}

	// Path between first two CamelCase symbols when no explicit path cue.
	if len(injected) == 0 && len(seeds) >= 2 {
		pr, err := graph.ShortestPath(repoPath, seeds[0], seeds[1])
		if err == nil && pr.Found {
			text := graph.FormatPathForPrompt(pr)
			if text != "" {
				injected = append(injected, map[string]interface{}{
					"type":    "code_graph_path",
					"content": text,
					"from":    seeds[0],
					"to":      seeds[1],
				})
			}
		}
	}

	if len(injected) == 0 {
		return false
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	existing := promptAttachmentsSlice(msg.Metadata[MetadataPromptAttachments])
	for _, att := range injected {
		existing = append(existing, att)
	}
	msg.Metadata[MetadataPromptAttachments] = existing
	msg.Metadata["injected_code_graph"] = true
	return true
}
