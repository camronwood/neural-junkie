package graph

import (
	"fmt"
	"sort"
	"strings"
)

// Summary returns a UI-ready graph snapshot (trimmed for density).
func Summary(repoPath string) (GraphSummary, error) {
	meta, err := Status(repoPath)
	if err != nil {
		return GraphSummary{}, err
	}
	store, err := OpenStore(repoPath)
	if err != nil {
		return GraphSummary{Meta: meta}, err
	}
	defer store.Close()

	nodes, err := store.AllNodes()
	if err != nil {
		return GraphSummary{Meta: meta}, err
	}
	edges, err := store.AllEdges()
	if err != nil {
		return GraphSummary{Meta: meta}, err
	}

	comms := communitiesFromNodes(nodes)
	gods := godNodes(nodes, godNodeLimit)
	viewNodes, viewEdges := selectDenseView(nodes, edges, maxUINodes, maxUIEdges)

	return GraphSummary{
		Meta:        meta,
		Communities: comms,
		GodNodes:    gods,
		Nodes:       viewNodes,
		Edges:       viewEdges,
	}, nil
}

func communitiesFromNodes(nodes []Node) []Community {
	counts := map[string]int{}
	for _, n := range nodes {
		if n.Kind != NodeSymbol && n.Kind != NodeFile && n.Kind != NodePackage {
			continue
		}
		c := n.Community
		if c == "" {
			c = "root"
		}
		counts[c]++
	}
	type kv struct {
		id    string
		count int
	}
	var list []kv
	for id, c := range counts {
		list = append(list, kv{id, c})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].count == list[j].count {
			return list[i].id < list[j].id
		}
		return list[i].count > list[j].count
	})
	out := make([]Community, 0, len(list))
	for _, item := range list {
		out = append(out, Community{
			ID: item.id, Label: item.id, Count: item.count, Color: communityColor(item.id),
		})
	}
	return out
}

func godNodes(nodes []Node, limit int) []Node {
	var ranked []Node
	for _, n := range nodes {
		if n.Kind == NodeSymbol || n.Kind == NodeFile || n.Kind == NodePackage {
			ranked = append(ranked, n)
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Degree == ranked[j].Degree {
			return ranked[i].Label < ranked[j].Label
		}
		return ranked[i].Degree > ranked[j].Degree
	})
	if limit > len(ranked) {
		limit = len(ranked)
	}
	return ranked[:limit]
}

func selectDenseView(nodes []Node, edges []Edge, maxNodes, maxEdges int) ([]Node, []Edge) {
	// Prefer packages + high-degree symbols + import edges for Graphify-like density.
	keep := map[string]bool{}
	var packages, symbols, files []Node
	for _, n := range nodes {
		switch n.Kind {
		case NodeRepo:
			keep[n.ID] = true
		case NodePackage:
			packages = append(packages, n)
		case NodeSymbol:
			symbols = append(symbols, n)
		case NodeFile:
			files = append(files, n)
		}
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Degree > packages[j].Degree })
	sort.Slice(symbols, func(i, j int) bool { return symbols[i].Degree > symbols[j].Degree })
	sort.Slice(files, func(i, j int) bool { return files[i].Degree > files[j].Degree })

	for _, n := range packages {
		if len(keep) >= maxNodes {
			break
		}
		keep[n.ID] = true
	}
	for _, n := range symbols {
		if len(keep) >= maxNodes {
			break
		}
		if n.Degree >= 1 {
			keep[n.ID] = true
		}
	}
	// Include files that participate in import/resolves edges among kept nodes.
	for _, e := range edges {
		if e.Kind != EdgeImports && e.Kind != EdgeResolvesTo {
			continue
		}
		if keep[e.From] || keep[e.To] {
			keep[e.From] = true
			keep[e.To] = true
		}
	}
	for _, n := range files {
		if len(keep) >= maxNodes {
			break
		}
		if n.Degree >= 3 {
			keep[n.ID] = true
		}
	}

	var outNodes []Node
	for _, n := range nodes {
		if keep[n.ID] {
			outNodes = append(outNodes, n)
		}
	}
	var outEdges []Edge
	for _, e := range edges {
		if keep[e.From] && keep[e.To] {
			outEdges = append(outEdges, e)
			if len(outEdges) >= maxEdges {
				break
			}
		}
	}
	return outNodes, outEdges
}

// QuerySubgraph seeds from label/path matches and expands hops.
func QuerySubgraph(repoPath, query string, hops, limit int) (Subgraph, error) {
	query = strings.TrimSpace(query)
	if hops <= 0 {
		hops = 1
	}
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	store, err := OpenStore(repoPath)
	if err != nil {
		return Subgraph{}, err
	}
	defer store.Close()

	var seeds []Node
	if n, ok := store.GetNode(query); ok {
		seeds = []Node{n}
	} else {
		var err error
		seeds, err = store.FindNodesByLabel(query, 30)
		if err != nil {
			return Subgraph{}, err
		}
	}
	if len(seeds) == 0 {
		return Subgraph{Query: query}, nil
	}

	adj, _, err := store.Adjacency()
	if err != nil {
		return Subgraph{}, err
	}
	keep := map[string]bool{}
	frontier := []string{}
	for _, s := range seeds {
		keep[s.ID] = true
		frontier = append(frontier, s.ID)
	}
	for depth := 0; depth < hops; depth++ {
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
		if len(keep) >= limit || len(frontier) == 0 {
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
	return Subgraph{Query: query, Nodes: nodes, Edges: edges}, nil
}

// Neighborhood returns nodes within depth of a seed id or label.
func Neighborhood(repoPath, seed string, depth, limit int) (Subgraph, error) {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return Subgraph{}, fmt.Errorf("seed required")
	}
	sg, err := ExpandNeighborhoodBFS(repoPath, seed, depth, limit)
	if err != nil {
		return Subgraph{}, err
	}
	if len(sg.Nodes) == 0 {
		return Subgraph{}, fmt.Errorf("node not found: %s", seed)
	}
	return sg, nil
}

// ShortestPath finds an undirected path between two nodes (id or label).
func ShortestPath(repoPath, from, to string) (PathResult, error) {
	store, err := OpenStore(repoPath)
	if err != nil {
		return PathResult{}, err
	}
	defer store.Close()

	resolve := func(s string) (string, error) {
		if _, ok := store.GetNode(s); ok {
			return s, nil
		}
		hits, err := store.FindNodesByLabel(s, 1)
		if err != nil {
			return "", err
		}
		if len(hits) == 0 {
			return "", fmt.Errorf("node not found: %s", s)
		}
		return hits[0].ID, nil
	}
	fromID, err := resolve(from)
	if err != nil {
		return PathResult{}, err
	}
	toID, err := resolve(to)
	if err != nil {
		return PathResult{}, err
	}

	adj, edgeByPair, err := store.Adjacency()
	if err != nil {
		return PathResult{}, err
	}

	type step struct {
		id   string
		prev string
	}
	queue := []string{fromID}
	prev := map[string]string{fromID: ""}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == toID {
			break
		}
		for _, nb := range adj[cur] {
			if _, seen := prev[nb]; seen {
				continue
			}
			prev[nb] = cur
			queue = append(queue, nb)
		}
	}
	if _, ok := prev[toID]; !ok && fromID != toID {
		return PathResult{From: fromID, To: toID, Found: false}, nil
	}

	var pathIDs []string
	for cur := toID; cur != ""; cur = prev[cur] {
		pathIDs = append([]string{cur}, pathIDs...)
		if cur == fromID {
			break
		}
	}
	var nodes []Node
	var edges []Edge
	for i, id := range pathIDs {
		if n, ok := store.GetNode(id); ok {
			nodes = append(nodes, n)
		}
		if i > 0 {
			if e, ok := edgeByPair[pathIDs[i-1]+"→"+id]; ok {
				edges = append(edges, e)
			}
		}
	}
	return PathResult{From: fromID, To: toID, Nodes: nodes, Edges: edges, Found: true}, nil
}

// Explain returns a plain-language summary of a node.
func Explain(repoPath, nodeRef string) (ExplainResult, error) {
	store, err := OpenStore(repoPath)
	if err != nil {
		return ExplainResult{}, err
	}
	defer store.Close()

	n, ok := store.GetNode(nodeRef)
	if !ok {
		hits, _ := store.FindNodesByLabel(nodeRef, 1)
		if len(hits) == 0 {
			return ExplainResult{}, fmt.Errorf("node not found: %s", nodeRef)
		}
		n = hits[0]
	}
	edges, err := store.EdgesForNode(n.ID)
	if err != nil {
		return ExplainResult{}, err
	}
	neighborIDs := map[string]bool{}
	provSet := map[string]bool{}
	for _, e := range edges {
		provSet[string(e.Provenance)] = true
		if e.From == n.ID {
			neighborIDs[e.To] = true
		} else {
			neighborIDs[e.From] = true
		}
	}
	var neighbors []Node
	for id := range neighborIDs {
		if nb, ok := store.GetNode(id); ok {
			neighbors = append(neighbors, nb)
		}
	}
	sort.Slice(neighbors, func(i, j int) bool { return neighbors[i].Label < neighbors[j].Label })
	var prov []string
	for p := range provSet {
		prov = append(prov, p)
	}
	sort.Strings(prov)

	desc := fmt.Sprintf("%s (%s) in community %q — degree %d, %d neighbors",
		n.Label, n.Kind, n.Community, n.Degree, len(neighbors))
	if n.Path != "" {
		desc += fmt.Sprintf("; path %s", n.Path)
		if n.Line > 0 {
			desc += fmt.Sprintf(":%d", n.Line)
		}
	}

	return ExplainResult{
		Node:        n,
		Neighbors:   neighbors,
		Edges:       edges,
		Community:   n.Community,
		Degree:      n.Degree,
		Provenance:  prov,
		Description: desc,
	}, nil
}

// FormatPathForPrompt renders a path for agent context injection.
func FormatPathForPrompt(p PathResult) string {
	if !p.Found || len(p.Nodes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Knowledge graph path:\n")
	for i, n := range p.Nodes {
		loc := n.Path
		if n.Line > 0 {
			loc = fmt.Sprintf("%s:%d", n.Path, n.Line)
		}
		b.WriteString(fmt.Sprintf("%d. %s [%s]", i+1, n.Label, n.Kind))
		if loc != "" {
			b.WriteString(" @ " + loc)
		}
		b.WriteByte('\n')
		if i < len(p.Edges) {
			e := p.Edges[i]
			b.WriteString(fmt.Sprintf("   --%s--> (%s)\n", e.Kind, e.Provenance))
		}
	}
	return b.String()
}

// FormatNeighborhoodForPrompt renders a subgraph for agent context.
func FormatNeighborhoodForPrompt(sg Subgraph) string {
	if len(sg.Nodes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Knowledge graph neighborhood")
	if sg.Query != "" {
		b.WriteString(" for " + sg.Query)
	}
	b.WriteString(":\n")
	limit := 40
	if len(sg.Nodes) < limit {
		limit = len(sg.Nodes)
	}
	for i := 0; i < limit; i++ {
		n := sg.Nodes[i]
		b.WriteString(fmt.Sprintf("- %s (%s)", n.Label, n.Kind))
		if n.Path != "" {
			b.WriteString(" @ " + n.Path)
			if n.Line > 0 {
				b.WriteString(fmt.Sprintf(":%d", n.Line))
			}
		}
		b.WriteString(fmt.Sprintf(" community=%s degree=%d\n", n.Community, n.Degree))
	}
	edgeLimit := 30
	if len(sg.Edges) < edgeLimit {
		edgeLimit = len(sg.Edges)
	}
	if edgeLimit > 0 {
		b.WriteString("Edges:\n")
		for i := 0; i < edgeLimit; i++ {
			e := sg.Edges[i]
			b.WriteString(fmt.Sprintf("- %s -[%s/%s]-> %s\n", e.From, e.Kind, e.Provenance, e.To))
		}
	}
	return b.String()
}
