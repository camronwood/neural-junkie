package runbooklibrary

import (
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

// DefinitionSource identifies where a runbook definition was loaded from.
type DefinitionSource string

const (
	SourceBundled DefinitionSource = "bundled"
	SourceUser    DefinitionSource = "user"
	SourcePack    DefinitionSource = "pack"
)

// RunInputSpec describes a run-time parameter for a definition.
type RunInputSpec struct {
	Key      string `json:"key"`
	Type     string `json:"type"` // string | number | bool | slack_channel
	Label    string `json:"label,omitempty"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// RunbookDefinition is a versioned, reusable runbook workflow.
type RunbookDefinition struct {
	ID              string                          `json:"id"`
	Version         int                             `json:"version"`
	Title           string                          `json:"title"`
	Description     string                          `json:"description"`
	Source          DefinitionSource                `json:"source,omitempty"`
	PackID          string                          `json:"pack_id,omitempty"`
	AgentIDs        []string                        `json:"agent_ids,omitempty"`
	ExecutionPolicy collaboration.ExecutionPolicy   `json:"execution_policy,omitempty"`
	GraphLayout     collaboration.GraphLayout       `json:"graph_layout,omitempty"`
	Inputs          []RunInputSpec                  `json:"inputs,omitempty"`
	Tasks           []collaboration.CollaborationTask `json:"tasks"`
	UpdatedAt       time.Time                       `json:"updated_at,omitempty"`
}

// DefinitionSummary is a lightweight list entry.
type DefinitionSummary struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description,omitempty"`
	Version     int              `json:"version"`
	Source      DefinitionSource `json:"source"`
	PackID      string           `json:"pack_id,omitempty"`
	UpdatedAt   time.Time        `json:"updated_at,omitempty"`
}

// Manifest tracks versions for a user definition directory.
type Manifest struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	LatestVersion  int       `json:"latest_version"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ToSummary returns a list entry for this definition.
func (d *RunbookDefinition) ToSummary() DefinitionSummary {
	if d == nil {
		return DefinitionSummary{}
	}
	title := d.Title
	if title == "" {
		title = d.ID
	}
	return DefinitionSummary{
		ID:          d.ID,
		Title:       title,
		Description: d.Description,
		Version:     d.Version,
		Source:      d.Source,
		PackID:      d.PackID,
		UpdatedAt:   d.UpdatedAt,
	}
}

// FromTemplate converts a legacy RunbookTemplate to a definition.
func FromTemplate(t collaboration.RunbookTemplate, source DefinitionSource) RunbookDefinition {
	id := t.Name
	if id == "" {
		id = t.Title
	}
	inputs := make([]RunInputSpec, 0, len(t.Inputs))
	for _, in := range t.Inputs {
		inputs = append(inputs, RunInputSpec{
			Key: in.Key, Type: in.Type, Label: in.Label,
			Default: in.Default, Required: in.Required,
		})
	}
	return RunbookDefinition{
		ID:              id,
		Version:         1,
		Title:           t.Title,
		Description:     t.Description,
		Source:          source,
		ExecutionPolicy: t.ExecutionPolicy,
		Inputs:          inputs,
		Tasks:           t.Tasks,
	}
}
