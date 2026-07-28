package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/artifacts"
	mapssideecar "github.com/camronwood/neural-junkie/internal/maps"
	mapsmcp "github.com/camronwood/neural-junkie/internal/mcp/maps"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	mapsCreateToolName = "maps_create"
	mapsUpdateToolName = "maps_update"
)

var mapsCreateToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "title": {"type": "string"},
    "center": {"type": "object", "description": "{lat, lon} map center"},
    "zoom": {"type": "number"},
    "markers": {"type": "array", "items": {"type": "object"}},
    "routes": {"type": "array", "items": {"type": "object"}},
    "waypoints": {
      "type": "array",
      "description": "If routes omitted and >=2 waypoints, computes a route via OSRM",
      "items": {"type": "object"}
    },
    "mode": {"type": "string", "description": "walking or driving when computing route from waypoints"},
    "tile_url_template": {"type": "string"},
    "workspace_id": {"type": "string"},
    "project_id": {"type": "string"}
  }
}`)

var mapsUpdateToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "artifact_id": {"type": "string"},
    "expected_revision": {"type": "integer", "minimum": 1},
    "title": {"type": "string"},
    "center": {"type": "object"},
    "zoom": {"type": "number"},
    "markers": {"type": "array", "items": {"type": "object"}},
    "routes": {"type": "array", "items": {"type": "object"}},
    "waypoints": {"type": "array", "items": {"type": "object"}},
    "mode": {"type": "string"},
    "tile_url_template": {"type": "string"}
  },
  "required": ["artifact_id", "expected_revision"]
}`)

func mapsCreateToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        mapsCreateToolName,
		Description: "Create an interactive Neural Canvas map artifact (nj.map). Optionally geocode-driven waypoints compute a walking/driving route via the maps sidecar.",
		InputSchema: mapsCreateToolSchema,
	}
}

func mapsUpdateToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        mapsUpdateToolName,
		Description: "Update an existing Neural Canvas map artifact (optimistic revision). Pass markers, routes, waypoints, center, or zoom to revise.",
		InputSchema: mapsUpdateToolSchema,
	}
}

func (a *Agent) mapsToolsEnabledForMessage(msg *protocol.Message) bool {
	if a == nil || !a.agentSupportsMapsTools() {
		return false
	}
	if msg != nil && msg.Type == protocol.MessageTypeCollabDiscussion {
		phase := strings.ToLower(strings.TrimSpace(msg.GetCollaborationPhase()))
		if phase == "planning" || phase == "reviewing" {
			return false
		}
	}
	return true
}

func appendMapsPrompt(system *strings.Builder) {
	system.WriteString("MAPS (Neural Canvas):\n")
	system.WriteString("Never call generate_image for maps. Call maps_geocode / maps_route / maps_create in this turn.\n")
	system.WriteString("Prefer maps_create with waypoints + mode (walking|driving) so the sidecar computes route geometry and opens nj.map.\n")
	system.WriteString("Use maps_update with artifact_id and expected_revision to revise an open map.\n")
	system.WriteString("Always show distance/duration from the tool result; do not invent coordinates.\n\n")
}

func rawArgsToMap(input json.RawMessage) (map[string]any, error) {
	if len(input) == 0 {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}
	if args == nil {
		args = map[string]any{}
	}
	return args, nil
}

func (a *Agent) executeMapsCreateTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	if !a.mapsToolsEnabledForMessage(msg) {
		return "", fmt.Errorf("%s requires the Maps pack (Assistant or a granted custom expert)", mapsCreateToolName)
	}
	args, err := rawArgsToMap(input)
	if err != nil {
		return "", fmt.Errorf("invalid maps_create input: %w", err)
	}
	payload, err := mapsmcp.BuildMapPayload(ctx, mapssideecar.DefaultSidecarClient, args)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	title, _ := payload["title"].(string)
	store, err := getAgentArtifactStore()
	if err != nil {
		return "", err
	}
	workspaceID, _ := args["workspace_id"].(string)
	projectID, _ := args["project_id"].(string)
	artifact := artifacts.Artifact{
		Kind:  "map",
		Title: strings.TrimSpace(title),
		Links: artifacts.ArtifactLinks{
			WorkspaceID: strings.TrimSpace(workspaceID),
			ProjectID:   strings.TrimSpace(projectID),
			ChannelID:   messageChannel(msg),
		},
		Renderer: artifacts.Renderer{
			ID:         mapsmcp.RendererID,
			APIVersion: "1",
			MediaType:  mapsmcp.MediaType,
		},
		Payload: raw,
		Provenance: []artifacts.SourceReference{{
			Kind:  "agent",
			Label: a.Info.Name,
			Metadata: map[string]string{
				"agent_id": a.Info.ID,
			},
		}},
	}
	created, err := store.Create(artifact)
	if err != nil {
		return "", err
	}
	a.postArtifactReference(msg, created, "created")
	if ledger := actionEvidenceFromContext(ctx); ledger != nil {
		ledger.Record(ActionEvidence{
			Kind:   EvidenceArtifactCreated,
			Tool:   mapsCreateToolName,
			Status: "succeeded",
			Detail: created.ID,
		})
	}
	return fmt.Sprintf("Created Neural Canvas map `%s` at revision %d.", created.ID, created.Revision), nil
}

func (a *Agent) executeMapsUpdateTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	if !a.mapsToolsEnabledForMessage(msg) {
		return "", fmt.Errorf("%s requires the Maps pack (Assistant or a granted custom expert)", mapsUpdateToolName)
	}
	args, err := rawArgsToMap(input)
	if err != nil {
		return "", fmt.Errorf("invalid maps_update input: %w", err)
	}
	artifactID, _ := args["artifact_id"].(string)
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return "", fmt.Errorf("artifact_id is required")
	}
	revFloat, ok := args["expected_revision"].(float64)
	if !ok || revFloat < 1 {
		return "", fmt.Errorf("expected_revision is required (integer >= 1)")
	}
	expectedRevision := uint64(revFloat)

	store, err := getAgentArtifactStore()
	if err != nil {
		return "", err
	}
	current, err := store.Get(artifactID)
	if err != nil {
		return "", err
	}

	mergeArgs := mapsmcp.StripMetaArgs(args)
	// Seed from existing payload so partial updates keep center/markers.
	if len(current.Payload) > 0 {
		var existing map[string]any
		if err := json.Unmarshal(current.Payload, &existing); err == nil {
			seeded := map[string]any{}
			for k, v := range existing {
				seeded[k] = v
			}
			for k, v := range mergeArgs {
				seeded[k] = v
			}
			mergeArgs = seeded
		}
	}

	payload, err := mapsmcp.BuildMapPayload(ctx, mapssideecar.DefaultSidecarClient, mergeArgs)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if title, _ := payload["title"].(string); strings.TrimSpace(title) != "" {
		current.Title = strings.TrimSpace(title)
	}
	current.Renderer.ID = mapsmcp.RendererID
	current.Renderer.MediaType = mapsmcp.MediaType
	current.Payload = raw
	updated, err := store.Update(*current, expectedRevision)
	if err != nil {
		return "", err
	}
	a.postArtifactReference(msg, updated, "updated")
	if ledger := actionEvidenceFromContext(ctx); ledger != nil {
		ledger.Record(ActionEvidence{
			Kind:   EvidenceArtifactCreated,
			Tool:   mapsUpdateToolName,
			Status: "succeeded",
			Detail: updated.ID,
		})
	}
	return fmt.Sprintf("Updated Neural Canvas map `%s` to revision %d.", updated.ID, updated.Revision), nil
}
