// Package graph provides a local knowledge graph over code structure (files, symbols, imports).
// Edges are tagged EXTRACTED or INFERRED. No external Graphify dependency.
package graph

import (
	"context"
	"strings"
)

// Reference is a symbol reference / neighborhood hit site.
type Reference struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column,omitempty"`
	Symbol  string `json:"symbol"`
	Context string `json:"context,omitempty"`
}

// FindReferences returns graph neighbors for a symbol (imports / defines / resolves).
func FindReferences(ctx context.Context, repoPath, symbol string, limit int) ([]Reference, error) {
	_ = ctx
	symbol = strings.TrimSpace(symbol)
	if symbol == "" || limit <= 0 {
		return nil, nil
	}
	meta, err := Status(repoPath)
	if err != nil || !meta.Ready {
		return nil, nil
	}
	sg, err := QuerySubgraph(repoPath, symbol, 1, limit*3)
	if err != nil {
		return nil, err
	}
	var out []Reference
	for _, n := range sg.Nodes {
		if n.Kind != NodeSymbol && n.Kind != NodeFile {
			continue
		}
		if !strings.Contains(strings.ToLower(n.Label), strings.ToLower(symbol)) &&
			!strings.Contains(strings.ToLower(n.Path), strings.ToLower(symbol)) {
			// Keep import-linked neighbors even when label differs.
			if n.Kind == NodeSymbol {
				continue
			}
		}
		out = append(out, Reference{
			Path:    n.Path,
			Line:    n.Line,
			Symbol:  n.Label,
			Context: string(n.Kind) + " community=" + n.Community,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// ExpandNeighborhoodBFS expands from a concrete node id.
func ExpandNeighborhoodBFS(repoPath, nodeID string, depth, limit int) (Subgraph, error) {
	if depth <= 0 {
		depth = 1
	}
	if limit <= 0 {
		limit = 120
	}
	store, err := OpenStore(repoPath)
	if err != nil {
		return Subgraph{}, err
	}
	defer store.Close()

	if _, ok := store.GetNode(nodeID); !ok {
		hits, _ := store.FindNodesByLabel(nodeID, 1)
		if len(hits) == 0 {
			return Subgraph{Query: nodeID}, nil
		}
		nodeID = hits[0].ID
	}

	adj, _, err := store.Adjacency()
	if err != nil {
		return Subgraph{}, err
	}
	keep := map[string]bool{nodeID: true}
	frontier := []string{nodeID}
	for d := 0; d < depth; d++ {
		var next []string
		for _, id := range frontier {
			for _, nb := range adj[id] {
				if keep[nb] {
					continue
				}
				keep[nb] = true
				next = append(next, nb)
				if len(keep) >= limit {
					break
				}
			}
			if len(keep) >= limit {
				break
			}
		}
		frontier = next
		if len(frontier) == 0 || len(keep) >= limit {
			break
		}
	}
	allNodes, _ := store.AllNodes()
	allEdges, _ := store.AllEdges()
	var nodes []Node
	for _, n := range allNodes {
		if keep[n.ID] {
			nodes = append(nodes, n)
		}
	}
	var edges []Edge
	for _, e := range allEdges {
		if keep[e.From] && keep[e.To] {
			edges = append(edges, e)
		}
	}
	return Subgraph{Query: nodeID, Nodes: nodes, Edges: edges}, nil
}
