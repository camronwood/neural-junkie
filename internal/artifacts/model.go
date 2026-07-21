// Package artifacts provides the durable backend model and file store for
// Neural Canvas artifacts.
package artifacts

import (
	"encoding/json"
	"time"
)

const CurrentSchemaVersion = 1

// Artifact is the current, versioned representation of a Canvas artifact.
type Artifact struct {
	SchemaVersion int               `json:"schemaVersion"`
	ID            string            `json:"id"`
	Revision      uint64            `json:"revision"`
	Kind          string            `json:"kind,omitempty"`
	Title         string            `json:"title,omitempty"`
	Description   string            `json:"description,omitempty"`
	Provenance    []SourceReference `json:"provenance,omitempty"`
	Links         ArtifactLinks     `json:"links,omitempty"`
	Renderer      Renderer          `json:"renderer"`
	Payload       json.RawMessage   `json:"payload"`
	Fallback      *Fallback         `json:"fallback,omitempty"`
	Capabilities  []string          `json:"capabilities,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

// ArtifactRevision is an immutable snapshot written for every artifact revision.
type ArtifactRevision struct {
	ArtifactID string    `json:"artifactId"`
	Revision   uint64    `json:"revision"`
	CreatedAt  time.Time `json:"createdAt"`
	Artifact   Artifact  `json:"artifact"`
}

// SourceReference records where artifact content originated.
type SourceReference struct {
	Kind       string            `json:"kind"`
	URI        string            `json:"uri,omitempty"`
	ArtifactID string            `json:"artifactId,omitempty"`
	Revision   uint64            `json:"revision,omitempty"`
	Label      string            `json:"label,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ArtifactLinks connects an artifact to the surrounding Neural Junkie context.
type ArtifactLinks struct {
	WorkspaceID     string `json:"workspaceId,omitempty"`
	ProjectID       string `json:"projectId,omitempty"`
	ChannelID       string `json:"channelId,omitempty"`
	CollaborationID string `json:"collaborationId,omitempty"`
}

// Renderer identifies the renderer contract required by an artifact.
type Renderer struct {
	ID         string `json:"id"`
	APIVersion string `json:"apiVersion"`
	MediaType  string `json:"mediaType"`
}

// Fallback is renderer-independent content for clients that cannot render the payload.
type Fallback struct {
	MediaType string          `json:"mediaType"`
	Data      json.RawMessage `json:"data"`
}

// Filter restricts List results. Empty fields match all artifacts.
type Filter struct {
	Kind            string
	WorkspaceID     string
	ProjectID       string
	ChannelID       string
	CollaborationID string
	RendererID      string
	Capability      string
}
