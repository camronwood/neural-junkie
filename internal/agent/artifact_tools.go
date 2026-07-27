package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/artifacts"
	semantic "github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	createArtifactToolName = "create_artifact"
	updateArtifactToolName = "update_artifact"
)

var (
	agentArtifactStoreOnce sync.Once
	agentArtifactStore     *artifacts.Store
	agentArtifactStoreErr  error
)

var (
	explicitCanvasRequestRE = regexp.MustCompile(`(?i)(?:\b(?:create|make|generate|show|give|build|produce|open|render)\b.{0,64}\b(?:neural canvas|canvas|artifact)\b|\b(?:neural canvas|canvas|artifact)\b.{0,64}\b(?:of|for|showing|about)\b)`)
	explicitVisualRequestRE = regexp.MustCompile(`(?i)\b(?:create|make|generate|show|give|produce|render)\b.{0,64}\b(?:report|chart|timeline|mermaid diagram|diagram|table)\b`)
	artifactCodeTargetRE    = regexp.MustCompile(`(?i)\b(?:component|page|screen|endpoint|function|class|service|widget|html canvas|canvas element|source file)\b`)
)

// UserRequestsArtifact reports explicit requests for a standalone Neural Canvas
// deliverable. Generic visuals with a code target remain implementation work.
// Canvas *revisions* are routed via semantic ActionArtifact + open-canvas features,
// not additional update/style phrase lists.
func UserRequestsArtifact(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if explicitCanvasRequestRE.MatchString(content) {
		return !artifactCodeTargetRE.MatchString(content) ||
			strings.Contains(strings.ToLower(content), "neural canvas") ||
			strings.Contains(strings.ToLower(content), "artifact")
	}
	return explicitVisualRequestRE.MatchString(content) && !artifactCodeTargetRE.MatchString(content)
}

// neuralCanvasIsSecondaryToCodeChange reports mixed turns where a canvas is asked
// only after a code fix/impl (e.g. "create a canvas after you fix the typo").
// Those keep ActionEdit authority; pure canvas creates must not.
func neuralCanvasIsSecondaryToCodeChange(content string) bool {
	if !UserRequestsArtifact(content) {
		return false
	}
	lower := strings.ToLower(content)
	cues := []string{
		"after you fix", "after fixing", "fix the typo", "fix the bug",
		"fix it then", "then create a canvas", "then show a canvas",
		"then make a canvas", "then generate a canvas",
	}
	for _, cue := range cues {
		if strings.Contains(lower, cue) {
			return true
		}
	}
	return false
}

func channelHasPendingArtifactRequest(history []*protocol.Message, skipMsgID string) bool {
	return pendingArtifactRequestID(history, skipMsgID) != ""
}

func pendingArtifactRequestID(history []*protocol.Message, skipMsgID string) string {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 20; i-- {
		msg := history[i]
		if msg == nil || msg.ID == skipMsgID {
			continue
		}
		seen++
		if msg.Type == protocol.MessageTypeArtifactChanged {
			return ""
		}
		if protocol.IsUserLikeSender(msg.From) {
			if decision, ok := protocol.ExtractTurnDecision(msg); ok {
				switch decision.Action {
				case semantic.ActionArtifact:
					return msg.ID
				case semantic.ActionDebug, semantic.ActionEdit, semantic.ActionContinue, semantic.ActionRun:
					return ""
				default:
					// Canonical non-artifact decisions must not re-enter via phrases.
					continue
				}
			}
			if UserRequestsArtifact(msg.Content) {
				return msg.ID
			}
			if userRequestsImplementation(msg.Content) {
				return ""
			}
		}
	}
	return ""
}

func messageHasArtifactAction(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	action, _ := msg.Metadata["turn_action"].(string)
	return ActionIntent(strings.TrimSpace(action)) == ActionArtifact
}

// neuralCanvasDeliverableTurn reports turns that must ship create/update_artifact rather
// than FILE_CHANGE or shell work. Mixed "canvas after you fix…" stays implementation-capable.
// Stamped Answer/AskUser/Plan keep conversational authority (no artifact tool force).
func neuralCanvasDeliverableTurn(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		switch decision.Action {
		case semantic.ActionArtifact:
			return true
		case semantic.ActionAnswer, semantic.ActionAskUser, semantic.ActionPlan:
			return false
		}
	}
	if messageHasArtifactAction(msg) {
		return true
	}
	if !UserRequestsArtifact(msg.Content) {
		return false
	}
	return !neuralCanvasIsSecondaryToCodeChange(msg.Content)
}

var createArtifactToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "title": {"type": "string", "description": "Short descriptive artifact title"},
    "renderer_id": {"type": "string", "enum": ["nj.markdown","nj.mermaid","nj.code","nj.table","nj.chart","nj.timeline","nj.image","nj.graph","nj.map"]},
    "media_type": {"type": "string", "description": "Renderer media type"},
    "kind": {"type": "string"},
    "data": {"description": "Declarative JSON payload for the renderer"},
    "fallback": {"type": "string", "description": "Markdown or plain-text fallback"},
    "workspace_id": {"type": "string"},
    "project_id": {"type": "string"}
  },
  "required": ["title", "renderer_id", "media_type", "data"]
}`)

var updateArtifactToolSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "artifact_id": {"type": "string"},
    "expected_revision": {"type": "integer", "minimum": 1},
    "title": {"type": "string"},
    "renderer_id": {"type": "string"},
    "media_type": {"type": "string"},
    "data": {"description": "Complete replacement declarative JSON payload"},
    "fallback": {"type": "string"}
  },
  "required": ["artifact_id", "expected_revision", "data"]
}`)

func artifactToolDefinitions() []ai.ClaudeToolDefinition {
	return []ai.ClaudeToolDefinition{
		{
			Name:        createArtifactToolName,
			Description: "Create a durable Neural Canvas artifact for a report, chart, table, timeline, Mermaid diagram, or graph. Use when the output benefits from a standalone visual artifact.",
			InputSchema: createArtifactToolSchema,
		},
		{
			Name:        updateArtifactToolName,
			Description: "Create a new revision of an existing Neural Canvas artifact using optimistic revision control.",
			InputSchema: updateArtifactToolSchema,
		},
	}
}

func artifactToolsEnabledForMessage(msg *protocol.Message) bool {
	if msg == nil {
		return true
	}
	if neuralCanvasDeliverableTurn(msg) {
		return true
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		if decision.Action == semantic.ActionArtifact {
			return true
		}
		// Stamped answer/edit/debug stay authoritative. Run/inspect are frequent
		// misroutes for Neural Canvas create asks — re-enable create_artifact when
		// create phrases match. Revisions rely on ActionArtifact (open-canvas policy).
		if UserRequestsArtifact(msg.Content) {
			switch decision.Action {
			case semantic.ActionRun, semantic.ActionInspect:
				return true
			}
		}
		return false
	}
	if msg.ImplementationSession() && !UserRequestsArtifact(msg.Content) && !messageHasArtifactAction(msg) {
		return false
	}
	phase := strings.ToLower(strings.TrimSpace(msg.GetCollaborationPhase()))
	return msg.Type != protocol.MessageTypeCollabDiscussion && phase != "planning" && phase != "reviewing"
}

func appendArtifactPrompt(system *strings.Builder) {
	system.WriteString("NEURAL CANVAS:\n")
	system.WriteString("Use create_artifact for substantial standalone analytical deliverables such as reports, charts, tables, timelines, Mermaid diagrams, and graph explorations. Keep short factual answers in chat.\n")
	system.WriteString("When the user explicitly requests a Neural Canvas or standalone artifact, call create_artifact in this turn. Do not promise to create it later and do not start a file implementation session.\n")
	system.WriteString("When the user asks to update/revise an existing canvas (colors, layout, black-and-white, content), call update_artifact with artifact_id, expected_revision, and the full new data payload. Do not edit repo files (tauri.conf.json, CSS, themes) for canvas style changes.\n")
	system.WriteString("Never emit [FILE_CHANGE], propose_file_edit, or workspace file edits for Neural Canvas requests — the canvas is app-managed, not a repo file.\n")
	system.WriteString("Never call generate_image for Neural Canvas / Mermaid / chart / table requests — call create_artifact instead.\n")
	system.WriteString("Artifact payloads must be declarative JSON. Never place executable JavaScript, React code, or arbitrary HTML in an artifact.\n\n")
}

func getAgentArtifactStore() (*artifacts.Store, error) {
	agentArtifactStoreOnce.Do(func() {
		agentArtifactStore, agentArtifactStoreErr = artifacts.NewStore("")
	})
	return agentArtifactStore, agentArtifactStoreErr
}

func (a *Agent) executeArtifactTool(ctx context.Context, msg *protocol.Message, name string, input json.RawMessage) (string, error) {
	if !artifactToolsEnabledForMessage(msg) {
		return "", fmt.Errorf("%s is unavailable during implementation or collaboration planning", name)
	}
	store, err := getAgentArtifactStore()
	if err != nil {
		return "", err
	}
	switch name {
	case createArtifactToolName:
		var args struct {
			Title       string          `json:"title"`
			RendererID  string          `json:"renderer_id"`
			MediaType   string          `json:"media_type"`
			Kind        string          `json:"kind"`
			Data        json.RawMessage `json:"data"`
			Fallback    string          `json:"fallback"`
			WorkspaceID string          `json:"workspace_id"`
			ProjectID   string          `json:"project_id"`
		}
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid create_artifact input: %w", err)
		}
		if strings.TrimSpace(args.Title) == "" || strings.TrimSpace(args.RendererID) == "" || strings.TrimSpace(args.MediaType) == "" || len(args.Data) == 0 {
			return "", fmt.Errorf("title, renderer_id, media_type, and data are required")
		}
		artifact := artifacts.Artifact{
			Kind:  strings.TrimSpace(args.Kind),
			Title: strings.TrimSpace(args.Title),
			Links: artifacts.ArtifactLinks{
				WorkspaceID: args.WorkspaceID,
				ProjectID:   args.ProjectID,
				ChannelID:   messageChannel(msg),
			},
			Renderer: artifacts.Renderer{
				ID:         strings.TrimSpace(args.RendererID),
				APIVersion: "1",
				MediaType:  strings.TrimSpace(args.MediaType),
			},
			Payload: args.Data,
			Provenance: []artifacts.SourceReference{{
				Kind:  "agent",
				Label: a.Info.Name,
				Metadata: map[string]string{
					"agent_id": a.Info.ID,
				},
			}},
		}
		if args.Fallback != "" {
			data, _ := json.Marshal(args.Fallback)
			artifact.Fallback = &artifacts.Fallback{MediaType: "text/markdown", Data: data}
		}
		created, err := store.Create(artifact)
		if err != nil {
			return "", err
		}
		a.postArtifactReference(msg, created, "created")
		if ledger := actionEvidenceFromContext(ctx); ledger != nil {
			ledger.Record(ActionEvidence{
				Kind:   EvidenceArtifactCreated,
				Tool:   createArtifactToolName,
				Status: "succeeded",
				Detail: created.ID,
			})
		}
		return fmt.Sprintf("Created Neural Canvas artifact `%s` at revision %d.", created.ID, created.Revision), nil
	case updateArtifactToolName:
		var args struct {
			ArtifactID       string          `json:"artifact_id"`
			ExpectedRevision uint64          `json:"expected_revision"`
			Title            string          `json:"title"`
			RendererID       string          `json:"renderer_id"`
			MediaType        string          `json:"media_type"`
			Data             json.RawMessage `json:"data"`
			Fallback         string          `json:"fallback"`
		}
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid update_artifact input: %w", err)
		}
		current, err := store.Get(strings.TrimSpace(args.ArtifactID))
		if err != nil {
			return "", err
		}
		if args.Title != "" {
			current.Title = strings.TrimSpace(args.Title)
		}
		if args.RendererID != "" {
			current.Renderer.ID = strings.TrimSpace(args.RendererID)
		}
		if args.MediaType != "" {
			current.Renderer.MediaType = strings.TrimSpace(args.MediaType)
		}
		current.Payload = args.Data
		if args.Fallback != "" {
			data, _ := json.Marshal(args.Fallback)
			current.Fallback = &artifacts.Fallback{MediaType: "text/markdown", Data: data}
		}
		updated, err := store.Update(*current, args.ExpectedRevision)
		if err != nil {
			return "", err
		}
		a.postArtifactReference(msg, updated, "updated")
		if ledger := actionEvidenceFromContext(ctx); ledger != nil {
			ledger.Record(ActionEvidence{
				Kind:   EvidenceArtifactCreated,
				Tool:   updateArtifactToolName,
				Status: "succeeded",
				Detail: updated.ID,
			})
		}
		return fmt.Sprintf("Updated Neural Canvas artifact `%s` to revision %d.", updated.ID, updated.Revision), nil
	default:
		return "", fmt.Errorf("unknown artifact tool %q", name)
	}
}

func (a *Agent) postArtifactReference(source *protocol.Message, artifact *artifacts.Artifact, action string) {
	if a.Hub == nil || source == nil || artifact == nil {
		return
	}
	message := protocol.NewMessage(protocol.MessageTypeArtifactChanged, source.Channel, a.Info, artifact.Title)
	message.ThreadID = source.ThreadID
	message.SetArtifactReference(protocol.ArtifactReference{
		ID:          artifact.ID,
		Title:       artifact.Title,
		RendererID:  artifact.Renderer.ID,
		MediaType:   artifact.Renderer.MediaType,
		Revision:    int64(artifact.Revision),
		WorkspaceID: artifact.Links.WorkspaceID,
		Action:      action,
	})
	_ = a.Hub.SendMessage(message)
}

func messageChannel(msg *protocol.Message) string {
	if msg == nil {
		return ""
	}
	return msg.Channel
}
