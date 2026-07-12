package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/arenasidecar"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const capModelArena = "model-arena"

// ArenaAvailable reports whether the hub can use Model Arena via the pack sidecar.
func (h *Hub) ArenaAvailable() bool {
	if h == nil || h.commandHandler == nil || h.commandHandler.appConfig == nil {
		return false
	}
	return h.commandHandler.appConfig.AnyPackCapability(capModelArena)
}

// ArenaEnabled implements agent.HubClient.
func (h *Hub) ArenaEnabled() bool {
	return h.ArenaAvailable()
}

func arenaClient() *arenasidecar.Client {
	if arenasidecar.DefaultSidecarClient != nil {
		return arenasidecar.DefaultSidecarClient
	}
	return arenasidecar.NewSidecarClient(arenasidecar.SidecarBaseURL)
}

// ArenaSidecarGet proxies GET to the pack sidecar.
func (h *Hub) ArenaSidecarGet(ctx context.Context, path string) (map[string]any, error) {
	if !h.ArenaAvailable() {
		return nil, fmt.Errorf("model arena disabled (install and enable the Model Arena pack)")
	}
	return arenaClient().Get(ctx, path)
}

// ArenaSidecarPost proxies POST to the pack sidecar.
func (h *Hub) ArenaSidecarPost(ctx context.Context, path string, body map[string]any) (map[string]any, error) {
	if !h.ArenaAvailable() {
		return nil, fmt.Errorf("model arena disabled (install and enable the Model Arena pack)")
	}
	return arenaClient().Post(ctx, path, body)
}

// ArenaMatchStepRequest selects a model move for one turn.
type ArenaMatchStepRequest struct {
	SessionID  string `json:"session_id"`
	ProviderID string `json:"provider_id"`
	Model      string `json:"model"`
	Seat       string `json:"seat,omitempty"`
}

// ArenaMatchRunRequest auto-runs up to max_steps model moves.
type ArenaMatchRunRequest struct {
	SessionID  string `json:"session_id"`
	ProviderID string `json:"provider_id"`
	Model      string `json:"model"`
	MaxSteps   int    `json:"max_steps"`
}

// ArenaRunMatchStep prompts the configured model for one legal move or logic answer.
func (h *Hub) ArenaRunMatchStep(ctx context.Context, req ArenaMatchStepRequest) (map[string]any, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id required")
	}
	session, err := h.ArenaSidecarGet(ctx, "/api/arena/sessions/"+sessionID)
	if err != nil {
		return nil, err
	}
	challenge, _ := session["challenge"].(string)
	status, _ := session["state"].(map[string]any)
	if status == nil {
		status, _ = session["state"].(map[string]interface{})
	}
	st := mapStringAny(status)
	if st != nil {
		if s, _ := st["status"].(string); s != "" && s != "active" {
			return session, nil
		}
	}
	if challenge == "logic" {
		return h.arenaLogicStep(ctx, sessionID, req, session)
	}
	return h.arenaBoardStep(ctx, sessionID, req, session)
}

// ArenaRunMatchAuto runs multiple model moves until terminal or max_steps.
func (h *Hub) ArenaRunMatchAuto(ctx context.Context, req ArenaMatchRunRequest) (map[string]any, error) {
	max := req.MaxSteps
	if max <= 0 {
		max = 20
	}
	if max > 200 {
		max = 200
	}
	var last map[string]any
	for i := 0; i < max; i++ {
		out, err := h.ArenaRunMatchStep(ctx, ArenaMatchStepRequest{
			SessionID:  req.SessionID,
			ProviderID: req.ProviderID,
			Model:      req.Model,
		})
		if err != nil {
			return last, err
		}
		last = out
		st := mapStringAny(out["state"])
		if st == nil {
			break
		}
		s, _ := st["status"].(string)
		if s != "" && s != "active" {
			break
		}
	}
	if last == nil {
		return nil, fmt.Errorf("no moves applied")
	}
	return last, nil
}

func (h *Hub) arenaBoardStep(ctx context.Context, sessionID string, req ArenaMatchStepRequest, session map[string]any) (map[string]any, error) {
	challenge, _ := session["challenge"].(string)
	state := mapStringAny(session["state"])
	if state == nil {
		return nil, fmt.Errorf("session missing state")
	}
	legal := stringSlice(state["legal_moves"])
	if len(legal) == 0 {
		return session, nil
	}
	prompt := arenaMovePrompt(challenge, state, legal)
	reply, err := h.arenaGenerate(ctx, req.ProviderID, req.Model, prompt)
	if err != nil {
		return nil, err
	}
	moveBody, parseErr := parseArenaMove(challenge, reply, legal)
	if parseErr != nil {
		reply2, err2 := h.arenaGenerate(ctx, req.ProviderID, req.Model, prompt+"\n\nYour previous reply was invalid. Reply with ONLY one legal move from the list.")
		if err2 == nil {
			if mb2, pe2 := parseArenaMove(challenge, reply2, legal); pe2 == nil {
				moveBody = mb2
				parseErr = nil
			}
		}
	}
	if parseErr != nil {
		return nil, fmt.Errorf("could not parse model move: %w", parseErr)
	}
	moveBody["by"] = req.Model
	out, err := h.ArenaSidecarPost(ctx, "/api/arena/sessions/"+sessionID+"/move", moveBody)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (h *Hub) arenaLogicStep(ctx context.Context, sessionID string, req ArenaMatchStepRequest, session map[string]any) (map[string]any, error) {
	state := mapStringAny(session["state"])
	promptText, _ := state["prompt"].(string)
	if promptText == "" {
		if puzzle, ok := session["puzzle"].(map[string]any); ok {
			promptText, _ = puzzle["prompt"].(string)
		}
	}
	prompt := "Solve this logic puzzle. Reply with ONLY the final answer (e.g. knight, knave, or a name).\n\n" + promptText
	reply, err := h.arenaGenerate(ctx, req.ProviderID, req.Model, prompt)
	if err != nil {
		return nil, err
	}
	answer := strings.TrimSpace(strings.Split(reply, "\n")[0])
	out, err := h.ArenaSidecarPost(ctx, "/api/arena/sessions/"+sessionID+"/answer", map[string]any{
		"answer": answer,
		"by":     req.Model,
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (h *Hub) arenaGenerate(ctx context.Context, providerID, model, prompt string) (string, error) {
	if h.commandHandler == nil || h.commandHandler.appConfig == nil {
		return "", fmt.Errorf("hub config not loaded")
	}
	cfg := h.commandHandler.appConfig
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		for _, p := range cfg.ListProvidersSnapshot() {
			if strings.TrimSpace(p.ID) != "" {
				providerID = p.ID
				break
			}
		}
	}
	if providerID == "" {
		return "", fmt.Errorf("no provider configured")
	}
	pcfg := cfg.GetProvider(providerID)
	if pcfg == nil {
		return "", fmt.Errorf("provider %q not found", providerID)
	}
	copy := *pcfg
	if strings.TrimSpace(model) != "" {
		copy.Model = strings.TrimSpace(model)
	}
	prov, err := ai.ProviderFromConfig(&copy)
	if err != nil {
		return "", err
	}
	return prov.GenerateResponse(ctx, prompt, nil)
}

func arenaMovePrompt(challenge string, state map[string]any, legal []string) string {
	var b strings.Builder
	b.WriteString("You are playing ")
	b.WriteString(challenge)
	b.WriteString(". Choose exactly ONE legal move from the list. Reply with ONLY the move, no explanation.\n\n")
	if fen, ok := state["fen"].(string); ok && fen != "" {
		b.WriteString("FEN: ")
		b.WriteString(fen)
		b.WriteString("\n")
	}
	if ascii, ok := state["ascii"].(string); ok && ascii != "" {
		b.WriteString(ascii)
		b.WriteString("\n")
	}
	if turn, ok := state["turn"].(string); ok && turn != "" {
		b.WriteString("Turn: ")
		b.WriteString(turn)
		b.WriteString("\n")
	}
	if board, ok := state["board"].([]any); ok && len(board) > 0 {
		b.WriteString("Board:\n")
		for _, row := range board {
			if cols, ok := row.([]any); ok {
				parts := make([]string, 0, len(cols))
				for _, c := range cols {
					parts = append(parts, fmt.Sprint(c))
				}
				b.WriteString(strings.Join(parts, " "))
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\nLegal moves:\n")
	for _, m := range legal {
		b.WriteString("- ")
		b.WriteString(m)
		b.WriteString("\n")
	}
	return b.String()
}

var uciMoveRe = regexp.MustCompile(`\b([a-h][1-8][a-h][1-8][qrbn]?)\b`)
var colMoveRe = regexp.MustCompile(`\b([0-6])\b`)

func parseArenaMove(challenge, reply string, legal []string) (map[string]any, error) {
	reply = strings.TrimSpace(strings.ToLower(reply))
	legalSet := make(map[string]struct{}, len(legal))
	for _, m := range legal {
		legalSet[strings.ToLower(m)] = struct{}{}
	}
	if challenge == "connect4" {
		if m := colMoveRe.FindString(reply); m != "" {
			col, _ := strconv.Atoi(m)
			for _, lm := range legal {
				if lm == m || lm == strconv.Itoa(col) {
					return map[string]any{"column": col}, nil
				}
			}
		}
		for _, lm := range legal {
			if reply == strings.ToLower(lm) {
				c, _ := strconv.Atoi(lm)
				return map[string]any{"column": c}, nil
			}
		}
		return nil, fmt.Errorf("no legal column in reply")
	}
	if m := uciMoveRe.FindString(reply); m != "" {
		if _, ok := legalSet[m]; ok {
			return map[string]any{"move": m, "uci": m}, nil
		}
	}
	for _, lm := range legal {
		if strings.Contains(reply, strings.ToLower(lm)) {
			return map[string]any{"move": lm, "uci": lm}, nil
		}
	}
	return nil, fmt.Errorf("no legal move in reply")
}

func mapStringAny(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out map[string]any
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			} else if f, ok := item.(float64); ok {
				out = append(out, strconv.Itoa(int(f)))
			}
		}
		return out
	default:
		return nil
	}
}

func agentTypeSupportsArena(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeArena, protocol.AgentTypeAssistant:
		return true
	default:
		return false
	}
}

// enrichAgentArena sets SupportsArena when the model-arena pack is active.
func enrichAgentArena(h *Hub, agent *protocol.AgentInfo) {
	if h == nil || agent == nil {
		return
	}
	agent.SupportsArena = h.ArenaAvailable() && agentTypeSupportsArena(agent.Type)
}