package memory

import "time"

// SourceType classifies indexed content.
type SourceType string

const (
	SourceMessage        SourceType = "message"
	SourceCollabArtifact SourceType = "collab_artifact"
)

const (
	DefaultTopK          = 5
	DefaultPromptBudget  = 1536
	DefaultChunkSize     = 600
	DefaultChunkOverlap  = 80
	DefaultSearchPrefilter = 200
)

// Chunk is a searchable unit stored in memory.db.
type Chunk struct {
	ID              string     `json:"id"`
	SourceType      SourceType `json:"source_type"`
	SourceID        string     `json:"source_id"`
	Channel         string     `json:"channel"`
	ThreadID        string     `json:"thread_id"`
	CollaborationID string     `json:"collaboration_id"`
	RelPath         string     `json:"rel_path"`
	SenderName      string     `json:"sender_name"`
	Content         string     `json:"content"`
	ContentHash     string     `json:"content_hash"`
	EmbeddingModel  string     `json:"embedding_model"`
	Vector          []float64  `json:"vector,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// SearchResult is a chunk with relevance score.
type SearchResult struct {
	Chunk Chunk   `json:"chunk"`
	Score float64 `json:"score"`
}

// PromptContext carries retrieval inputs at agent prompt build time.
type PromptContext struct {
	Query             string
	Channel           string
	CollaborationID   string
	ExcludeMessageIDs []string
}

// PromptResult is injection output for debug metadata.
type PromptResult struct {
	Count int
	IDs   []string
}

// Stats summarizes index state.
type Stats struct {
	TotalChunks      int            `json:"total_chunks"`
	BySourceType     map[string]int `json:"by_source_type"`
	ChannelsIndexed  int            `json:"channels_indexed"`
	CollabsIndexed   int            `json:"collabs_indexed"`
}
