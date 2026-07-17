package graph

import "time"

// Provenance tags how an edge was obtained.
type Provenance string

const (
	ProvenanceExtracted Provenance = "EXTRACTED"
	ProvenanceInferred  Provenance = "INFERRED"
)

// NodeKind classifies graph nodes.
type NodeKind string

const (
	NodeRepo    NodeKind = "repo"
	NodePackage NodeKind = "package"
	NodeFile    NodeKind = "file"
	NodeSymbol  NodeKind = "symbol"
)

// EdgeKind classifies graph edges.
type EdgeKind string

const (
	EdgeContains   EdgeKind = "contains"
	EdgeImports    EdgeKind = "imports"
	EdgeDefines    EdgeKind = "defines"
	EdgeResolvesTo EdgeKind = "resolves_to"
)

// Node is a graph entity.
type Node struct {
	ID         string   `json:"id"`
	Kind       NodeKind `json:"kind"`
	Label      string   `json:"label"`
	Path       string   `json:"path,omitempty"`
	Line       int      `json:"line,omitempty"`
	Language   string   `json:"language,omitempty"`
	Community  string   `json:"community,omitempty"`
	Degree     int      `json:"degree,omitempty"`
	SymbolKind string   `json:"symbol_kind,omitempty"`
}

// Edge is a directed relationship between nodes.
type Edge struct {
	ID         string     `json:"id"`
	From       string     `json:"from"`
	To         string     `json:"to"`
	Kind       EdgeKind   `json:"kind"`
	Provenance Provenance `json:"provenance"`
	Path       string     `json:"path,omitempty"`
	Line       int        `json:"line,omitempty"`
}

// Community groups nodes (package/directory clustering in v1).
type Community struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Count int    `json:"count"`
	Color string `json:"color,omitempty"`
}

// Meta describes graph build status.
type Meta struct {
	RepoPath    string    `json:"repo_path"`
	RepoHash    string    `json:"repo_hash"`
	NodeCount   int       `json:"node_count"`
	EdgeCount   int       `json:"edge_count"`
	GitHEAD     string    `json:"git_head,omitempty"`
	LastBuiltAt time.Time `json:"last_built_at,omitempty"`
	Ready       bool      `json:"ready"`
	Building    bool      `json:"building,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// GraphSummary is the UI bootstrap payload.
type GraphSummary struct {
	Meta        Meta        `json:"meta"`
	Communities []Community `json:"communities"`
	GodNodes    []Node      `json:"god_nodes"`
	Nodes       []Node      `json:"nodes"`
	Edges       []Edge      `json:"edges"`
}

// ExplainResult describes a single node.
type ExplainResult struct {
	Node        Node     `json:"node"`
	Neighbors   []Node   `json:"neighbors"`
	Edges       []Edge   `json:"edges"`
	Community   string   `json:"community"`
	Degree      int      `json:"degree"`
	Provenance  []string `json:"provenance_summary"`
	Description string   `json:"description"`
}

// PathResult is a shortest path between two nodes.
type PathResult struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
	Found bool   `json:"found"`
}

// Subgraph is a scoped view for query/focus.
type Subgraph struct {
	Query string `json:"query,omitempty"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
