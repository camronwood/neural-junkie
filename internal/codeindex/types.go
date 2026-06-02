package codeindex

import "time"

// Chunk is a searchable slice of a source file.
type Chunk struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Start   int    `json:"start_line"`
	End     int    `json:"end_line"`
	Content string `json:"content"`
}

// IndexMeta tracks build state for a repository.
type IndexMeta struct {
	RepoPath       string    `json:"repo_path"`
	RepoHash       string    `json:"repo_hash"`
	ChunkCount     int       `json:"chunk_count"`
	EmbeddingModel string    `json:"embedding_model"`
	LastBuiltAt    time.Time `json:"last_built_at"`
	GitHEAD        string    `json:"git_head,omitempty"`
	Ready          bool      `json:"ready"`
	Building       bool      `json:"building"`
}

// SearchResult is returned to callers and API handlers.
type SearchResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Score   float64 `json:"score,omitempty"`
}
