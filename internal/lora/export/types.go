package export

// SourceKind selects where training rows are collected from.
type SourceKind string

const (
	SourceChannel      SourceKind = "channel"
	SourceCollaboration SourceKind = "collaboration"
	SourceRepo         SourceKind = "repo"
)

const (
	DefaultMaxRows = 2000
	MinRows        = 10
)

// Row is one Alpaca-style training example.
type Row struct {
	Instruction string `json:"instruction"`
	Input       string `json:"input"`
	Output      string `json:"output"`
}

// Request describes a dataset export.
type Request struct {
	Source     SourceKind
	SourceID   string
	ThreadID   string
	AgentName  string
	MaxRows    int
}
