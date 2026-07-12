package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	arenaCreateSessionToolName = "arena_create_session"
	arenaGetStateToolName      = "arena_get_state"
	arenaMakeMoveToolName      = "arena_make_move"
	arenaSubmitAnswerToolName  = "arena_submit_answer"
	arenaListChallengesTool    = "arena_list_challenges"
)

var arenaCreateSessionSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "challenge": {"type": "string", "description": "chess, connect4, or logic"},
    "white": {"type": "string", "description": "White/red player model tag or human"},
    "black": {"type": "string", "description": "Black/yellow player model tag or human"},
    "puzzle_id": {"type": "string", "description": "Logic puzzle id"},
    "fen": {"type": "string", "description": "Optional chess starting FEN"}
  },
  "required": ["challenge"]
}`)

var arenaGetStateSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "session_id": {"type": "string"}
  },
  "required": ["session_id"]
}`)

var arenaMakeMoveSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "session_id": {"type": "string"},
    "move": {"type": "string", "description": "UCI move for chess"},
    "column": {"type": "integer", "description": "Column 0-6 for Connect Four"}
  },
  "required": ["session_id"]
}`)

var arenaSubmitAnswerSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "session_id": {"type": "string"},
    "answer": {"type": "string"}
  },
  "required": ["session_id", "answer"]
}`)

var arenaListChallengesSchema = json.RawMessage(`{"type": "object", "properties": {}}`)

func arenaCreateSessionToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        arenaCreateSessionToolName,
		Description: "Create a Model Arena session for chess, connect4, or logic puzzles.",
		InputSchema: arenaCreateSessionSchema,
	}
}

func arenaGetStateToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        arenaGetStateToolName,
		Description: "Get the current Model Arena session state including board, legal moves, and status.",
		InputSchema: arenaGetStateSchema,
	}
}

func arenaMakeMoveToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        arenaMakeMoveToolName,
		Description: "Apply a validated move to a chess or Connect Four arena session.",
		InputSchema: arenaMakeMoveSchema,
	}
}

func arenaSubmitAnswerToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        arenaSubmitAnswerToolName,
		Description: "Submit an answer for a logic puzzle arena session.",
		InputSchema: arenaSubmitAnswerSchema,
	}
}

func arenaListChallengesToolDefinition() ai.ClaudeToolDefinition {
	return ai.ClaudeToolDefinition{
		Name:        arenaListChallengesTool,
		Description: "List available Model Arena challenges and logic puzzles.",
		InputSchema: arenaListChallengesSchema,
	}
}

func (a *Agent) arenaToolsEnabledForMessage(msg *protocol.Message) bool {
	if a.Hub == nil || !a.Hub.ArenaEnabled() {
		return false
	}
	if !agentTypeSupportsArenaTools(a.Info.Type) {
		return false
	}
	if messageSuppressesImageGeneration(msg) {
		return false
	}
	return true
}

func agentTypeSupportsArenaTools(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeArena, protocol.AgentTypeAssistant:
		return true
	default:
		return false
	}
}

func appendArenaPrompt(system *strings.Builder) {
	system.WriteString("MODEL ARENA:\n")
	system.WriteString("Always use arena tools for game state — never invent board positions or legal moves.\n")
	system.WriteString("Flow: arena_list_challenges → arena_create_session → arena_get_state → arena_make_move or arena_submit_answer.\n")
	system.WriteString("For chess use UCI moves from legal_moves. For Connect Four use column numbers 0-6.\n\n")
}

func (a *Agent) executeArenaCreateSessionTool(ctx context.Context, _ *protocol.Message, input json.RawMessage) (string, error) {
	var args map[string]any
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", err
		}
	}
	out, err := a.Hub.ArenaSidecarPost(ctx, "/api/arena/sessions", args)
	if err != nil {
		return "", err
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	return string(raw), nil
}

func (a *Agent) executeArenaGetStateTool(ctx context.Context, _ *protocol.Message, input json.RawMessage) (string, error) {
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	args.SessionID = strings.TrimSpace(args.SessionID)
	if args.SessionID == "" {
		return "", fmt.Errorf("session_id required")
	}
	out, err := a.Hub.ArenaSidecarGet(ctx, "/api/arena/sessions/"+args.SessionID)
	if err != nil {
		return "", err
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	return string(raw), nil
}

func (a *Agent) executeArenaMakeMoveTool(ctx context.Context, _ *protocol.Message, input json.RawMessage) (string, error) {
	var args struct {
		SessionID string `json:"session_id"`
		Move      string `json:"move"`
		Column    *int   `json:"column"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	body := map[string]any{"by": a.Info.Name}
	if args.Column != nil {
		body["column"] = *args.Column
	}
	if strings.TrimSpace(args.Move) != "" {
		body["move"] = strings.TrimSpace(args.Move)
	}
	out, err := a.Hub.ArenaSidecarPost(ctx, "/api/arena/sessions/"+strings.TrimSpace(args.SessionID)+"/move", body)
	if err != nil {
		return "", err
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	return string(raw), nil
}

func (a *Agent) executeArenaSubmitAnswerTool(ctx context.Context, _ *protocol.Message, input json.RawMessage) (string, error) {
	var args struct {
		SessionID string `json:"session_id"`
		Answer    string `json:"answer"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	out, err := a.Hub.ArenaSidecarPost(ctx, "/api/arena/sessions/"+strings.TrimSpace(args.SessionID)+"/answer", map[string]any{
		"answer": strings.TrimSpace(args.Answer),
		"by":     a.Info.Name,
	})
	if err != nil {
		return "", err
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	return string(raw), nil
}

func (a *Agent) executeArenaListChallengesTool(ctx context.Context, _ *protocol.Message, _ json.RawMessage) (string, error) {
	out, err := a.Hub.ArenaSidecarGet(ctx, "/api/arena/challenges")
	if err != nil {
		return "", err
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	return string(raw), nil
}
