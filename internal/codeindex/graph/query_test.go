package graph

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestCommunitiesAndGodNodesFromSeededStore(t *testing.T) {
	repo := t.TempDir()
	dir, err := graphDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	nodes := []Node{
		{ID: "pkg:hub", Kind: NodePackage, Label: "hub", Community: "internal/hub", Degree: 10},
		{ID: "sym:A", Kind: NodeSymbol, Label: "Handle", Community: "internal/hub", Degree: 8, Path: "internal/hub/a.go"},
		{ID: "sym:B", Kind: NodeSymbol, Label: "Helper", Community: "internal/util", Degree: 2, Path: "internal/util/b.go"},
		{ID: "file:a", Kind: NodeFile, Label: "a.go", Community: "internal/hub", Degree: 3, Path: "internal/hub/a.go"},
	}
	edges := []Edge{
		{ID: "e1", From: "pkg:hub", To: "sym:A", Kind: EdgeContains, Provenance: ProvenanceExtracted},
		{ID: "e2", From: "sym:A", To: "sym:B", Kind: EdgeImports, Provenance: ProvenanceInferred},
	}
	meta := Meta{
		RepoPath: repo, RepoHash: RepoHash(repo), Ready: true,
		NodeCount: len(nodes), EdgeCount: len(edges),
	}
	if err := store.ReplaceAll(nodes, edges, meta); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	summary, err := Summary(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Meta.Ready {
		t.Fatalf("meta not ready: %+v", summary.Meta)
	}
	if len(summary.Communities) < 2 {
		t.Fatalf("communities = %#v", summary.Communities)
	}
	if len(summary.GodNodes) == 0 || summary.GodNodes[0].ID != "pkg:hub" {
		t.Fatalf("god nodes = %#v", summary.GodNodes)
	}

	sg, err := QuerySubgraph(repo, "Handle", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(sg.Nodes) < 2 {
		t.Fatalf("subgraph nodes = %#v", sg.Nodes)
	}
	_ = filepath.Base(repo)
}

func TestCommunitiesFromNodesEmpty(t *testing.T) {
	if got := communitiesFromNodes(nil); len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestSelectDenseViewRespectsMaxNodesWithImportExpansion(t *testing.T) {
	nodes := []Node{
		{ID: "repo:x", Kind: NodeRepo, Label: "x", Degree: 1},
		{ID: "pkg:core", Kind: NodePackage, Label: "core", Degree: 100},
	}
	edges := []Edge{}
	for i := 0; i < 500; i++ {
		id := "file:f" + strconv.Itoa(i)
		nodes = append(nodes, Node{ID: id, Kind: NodeFile, Label: id, Degree: 5})
		edges = append(edges, Edge{
			ID: "e" + strconv.Itoa(i), From: "pkg:core", To: id, Kind: EdgeImports, Provenance: ProvenanceExtracted,
		})
	}
	const maxNodes = 50
	outNodes, outEdges := selectDenseView(nodes, edges, maxNodes, 100)
	if len(outNodes) > maxNodes {
		t.Fatalf("selectDenseView returned %d nodes, want <= %d", len(outNodes), maxNodes)
	}
	if len(outEdges) > 100 {
		t.Fatalf("selectDenseView returned %d edges, want <= 100", len(outEdges))
	}
	keep := map[string]bool{}
	for _, n := range outNodes {
		keep[n.ID] = true
	}
	if !keep["repo:x"] || !keep["pkg:core"] {
		t.Fatalf("expected repo + package kept, got nodes=%d", len(outNodes))
	}
}

