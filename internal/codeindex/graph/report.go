package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func writeReport(dir string, nodes []Node, edges []Edge, meta Meta) error {
	comms := communitiesFromNodes(nodes)
	gods := godNodes(nodes, 15)
	var b strings.Builder
	b.WriteString("# Knowledge Graph Report\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Repo: `%s`\n", meta.RepoPath))
	b.WriteString(fmt.Sprintf("- Nodes: %d\n", meta.NodeCount))
	b.WriteString(fmt.Sprintf("- Edges: %d\n", meta.EdgeCount))
	if meta.GitHEAD != "" {
		b.WriteString(fmt.Sprintf("- Git HEAD: `%s`\n", meta.GitHEAD))
	}
	b.WriteString("\n## God nodes\n\n")
	for _, n := range gods {
		b.WriteString(fmt.Sprintf("- **%s** (%s) degree=%d community=`%s`", n.Label, n.Kind, n.Degree, n.Community))
		if n.Path != "" {
			b.WriteString(fmt.Sprintf(" `%s`", n.Path))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Communities\n\n")
	limit := 30
	if len(comms) < limit {
		limit = len(comms)
	}
	for i := 0; i < limit; i++ {
		c := comms[i]
		b.WriteString(fmt.Sprintf("- `%s` (%d)\n", c.Label, c.Count))
	}
	b.WriteString("\n## Edge provenance\n\n")
	extracted, inferred := 0, 0
	for _, e := range edges {
		switch e.Provenance {
		case ProvenanceExtracted:
			extracted++
		case ProvenanceInferred:
			inferred++
		}
	}
	b.WriteString(fmt.Sprintf("- EXTRACTED: %d\n", extracted))
	b.WriteString(fmt.Sprintf("- INFERRED: %d\n", inferred))

	// Top import targets
	type kv struct {
		label string
		n     int
	}
	impCounts := map[string]int{}
	for _, e := range edges {
		if e.Kind == EdgeImports {
			impCounts[e.To]++
		}
	}
	var imps []kv
	for id, n := range impCounts {
		label := id
		for _, node := range nodes {
			if node.ID == id {
				label = node.Label
				break
			}
		}
		imps = append(imps, kv{label, n})
	}
	sort.Slice(imps, func(i, j int) bool { return imps[i].n > imps[j].n })
	b.WriteString("\n## Top import targets\n\n")
	for i := 0; i < len(imps) && i < 20; i++ {
		b.WriteString(fmt.Sprintf("- `%s` (%d)\n", imps[i].label, imps[i].n))
	}

	path := filepath.Join(dir, "GRAPH_REPORT.md")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
