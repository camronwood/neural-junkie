package collaboration

import (
	"encoding/json"
	"strconv"

	"github.com/camronwood/neural-junkie/internal/artifacts"
)

// SharedArtifactToCanvas exposes a collaboration plan as a read-through Neural
// Canvas document without changing the plan's existing lifecycle or storage.
func SharedArtifactToCanvas(plan *SharedArtifact, links artifacts.ArtifactLinks) (*artifacts.Artifact, error) {
	if plan == nil {
		return nil, nil
	}
	payload, err := json.Marshal(map[string]any{"content": plan.Content})
	if err != nil {
		return nil, err
	}
	fallbackData, err := json.Marshal(plan.Content)
	if err != nil {
		return nil, err
	}
	return &artifacts.Artifact{
		SchemaVersion: artifacts.CurrentSchemaVersion,
		ID:            plan.ID,
		Revision:      uint64(max(plan.Version, 1)),
		Kind:          "collaboration-plan",
		Title:         plan.Title,
		Links:         links,
		Renderer: artifacts.Renderer{
			ID:         "nj.markdown",
			APIVersion: "1",
			MediaType:  "text/markdown",
		},
		Payload: payload,
		Fallback: &artifacts.Fallback{
			MediaType: "text/markdown",
			Data:      fallbackData,
		},
		Provenance: []artifacts.SourceReference{{
			Kind:       "collaboration-plan",
			ArtifactID: plan.ID,
			Revision:   uint64(max(plan.Version, 1)),
			Label:      "Collaboration plan v" + strconv.Itoa(max(plan.Version, 1)),
		}},
		CreatedAt: plan.CreatedAt,
		UpdatedAt: plan.UpdatedAt,
	}, nil
}
