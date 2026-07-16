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
	Seat       string `json:"seat,omitempty"`
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
	st := mapStringAny(session["state"])
	if arenaSessionTerminal(st) {
		return session, nil
	}
	if challenge == "logic" {
		return h.arenaLogicStep(ctx, sessionID, req, session)
	}
	return h.arenaBoardStep(ctx, sessionID, req, session)
}

// ArenaRunMatchAuto runs model moves until terminal, a human turn, or max_steps.
func (h *Hub) ArenaRunMatchAuto(ctx context.Context, req ArenaMatchRunRequest) (map[string]any, error) {
	max := req.MaxSteps
	if max <= 0 {
		max = 20
	}
	if max > 200 {
		max = 200
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id required")
	}

	var last map[string]any
	applied := 0
	for applied < max {
		session, err := h.ArenaSidecarGet(ctx, "/api/arena/sessions/"+sessionID)
		if err != nil {
			return last, err
		}
		last = session
		challenge, _ := session["challenge"].(string)
		st := mapStringAny(session["state"])
		if arenaSessionTerminal(st) {
			break
		}
		if challenge == "logic" {
			seat := arenaLogicNextSeat(session)
			if seat == "" {
				break
			}
			out, err := h.arenaLogicStep(ctx, sessionID, ArenaMatchStepRequest{
				SessionID:  sessionID,
				ProviderID: req.ProviderID,
				Model:      req.Model,
			}, session)
			if err != nil {
				return last, err
			}
			last = out
			applied++
			st = mapStringAny(out["state"])
			if arenaSessionTerminal(st) || arenaSessionTerminal(mapStringAny(out)) {
				break
			}
			continue
		}
		seat := strings.TrimSpace(req.Seat)
		if seat == "" {
			seat = arenaActiveSeat(challenge, st)
		}
		model, humanTurn := arenaModelForSeat(session, seat, req.Model)
		if humanTurn {
			break
		}
		if model == "" {
			return last, fmt.Errorf("no model assigned for %s to move", seat)
		}
		out, err := h.arenaBoardStep(ctx, sessionID, ArenaMatchStepRequest{
			SessionID:  sessionID,
			ProviderID: req.ProviderID,
			Model:      model,
			Seat:       seat,
		}, session)
		if err != nil {
			return last, err
		}
		last = out
		applied++
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
	seat := strings.TrimSpace(req.Seat)
	if seat == "" {
		seat = arenaActiveSeat(challenge, state)
	}
	model, humanTurn := arenaModelForSeat(session, seat, req.Model)
	if humanTurn {
		return arenaSkippedStep(session, seat, "human_turn"), nil
	}
	if model == "" {
		return nil, fmt.Errorf("no model assigned for %s to move", seat)
	}

	const maxAttempts = 3
	var (
		moveBody map[string]any
		reply    string
		parseErr error
		attempts int
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attempts = attempt
		prompt := arenaMovePrompt(challenge, state, legal)
		if attempt > 1 {
			prompt = arenaMoveRepairPrompt(challenge, legal, reply, attempt)
		}
		nextReply, err := h.arenaGenerate(ctx, req.ProviderID, model, prompt)
		if err != nil {
			if attempt == 1 {
				return nil, err
			}
			// Keep trying with the last bad reply context when later attempts fail to generate.
			continue
		}
		reply = nextReply
		moveBody, parseErr = parseArenaMove(challenge, reply, legal)
		if parseErr == nil {
			break
		}
	}
	if parseErr != nil {
		// Soft-fail after re-prompts: keep the match alive so the user can retry.
		return arenaWithStep(session, map[string]any{
			"skipped":  true,
			"reason":   "invalid_model_move",
			"model":    model,
			"seat":     seat,
			"reply":    reply,
			"error":    parseErr.Error(),
			"attempts": attempts,
		}), nil
	}
	moveBody["by"] = model
	out, err := h.ArenaSidecarPost(ctx, "/api/arena/sessions/"+sessionID+"/move", moveBody)
	if err != nil {
		return nil, err
	}
	step := map[string]any{
		"model":       model,
		"seat":        seat,
		"reply":       reply,
		"parsed_move": moveBody,
	}
	if attempts > 1 {
		step["attempts"] = attempts
	}
	return arenaWithStep(out, step), nil
}

func (h *Hub) arenaLogicStep(ctx context.Context, sessionID string, req ArenaMatchStepRequest, session map[string]any) (map[string]any, error) {
	state := mapStringAny(session["state"])
	if arenaSessionTerminal(state) || arenaSessionTerminal(mapStringAny(session)) {
		return session, nil
	}
	seat := arenaLogicNextSeat(session)
	if seat == "" {
		return session, nil
	}
	promptText, _ := state["prompt"].(string)
	if promptText == "" {
		if puzzle, ok := session["puzzle"].(map[string]any); ok {
			promptText, _ = puzzle["prompt"].(string)
		}
	}
	model := arenaLogicModelForSeat(session, seat)
	if model == "" || strings.EqualFold(model, arenaHumanPlayer) {
		model = strings.TrimSpace(req.Model)
	}
	if model == "" {
		return nil, fmt.Errorf("no model assigned for logic seat %s", seat)
	}
	prompt := "Solve this logic puzzle. Explain your reasoning briefly, then put the final answer alone on the last line (e.g. knight, knave, or a name).\n\n" + promptText
	reply, err := h.arenaGenerate(ctx, req.ProviderID, model, prompt)
	if err != nil {
		return nil, err
	}
	answer := parseLogicAnswer(reply)
	out, err := h.ArenaSidecarPost(ctx, "/api/arena/sessions/"+sessionID+"/answer", map[string]any{
		"answer": answer,
		"by":     model,
		"seat":   seat,
	})
	if err != nil {
		return nil, err
	}
	return arenaWithStep(out, map[string]any{
		"model":         model,
		"seat":          seat,
		"reply":         reply,
		"parsed_answer": answer,
	}), nil
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
	if challenge == "chess" {
		b.WriteString(". Briefly explain in 1-2 sentences, then put ONE legal UCI move alone on the final line (format e2e4 / g1f3 / e7e8q). Never use SAN like Nf3 or O-O.\n\n")
	} else {
		b.WriteString(". Briefly explain your strategy in 1-3 sentences, then put your chosen legal move alone on the final line.\n\n")
	}
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

// arenaMoveRepairPrompt asks again after an invalid model reply, getting stricter each attempt.
func arenaMoveRepairPrompt(challenge string, legal []string, previousReply string, attempt int) string {
	var b strings.Builder
	prev := strings.TrimSpace(previousReply)
	if len(prev) > 400 {
		prev = prev[:400] + "…"
	}
	if challenge == "chess" {
		if attempt >= 3 {
			b.WriteString("Your last chess reply had no legal UCI move. Reply with ONLY one move token and nothing else.\n")
			b.WriteString("Valid examples look like: e2e4, g1f3, e7e8q. Never write Nf3, O-O, or sentences.\n\n")
			b.WriteString("Pick exactly one of these legal moves:\n")
			limit := 16
			if len(legal) < limit {
				limit = len(legal)
			}
			for _, m := range legal[:limit] {
				b.WriteString(m)
				b.WriteString("\n")
			}
			if len(legal) > limit {
				b.WriteString("…\n")
			}
		} else {
			b.WriteString("Your previous chess reply was invalid (no legal UCI move).\n")
			if prev != "" {
				b.WriteString("Previous reply:\n")
				b.WriteString(prev)
				b.WriteString("\n\n")
			}
			b.WriteString("Try again. Put EXACTLY one legal UCI move alone on the final line (e2e4 / g1f3 / e7e8q). No SAN (Nf3), no castling notation (O-O), no commentary on that last line.\n\n")
			b.WriteString("Legal moves:\n")
			for _, m := range legal {
				b.WriteString("- ")
				b.WriteString(m)
				b.WriteString("\n")
			}
		}
		return b.String()
	}

	b.WriteString("Your previous reply was invalid. Choose one legal move from the list and put it alone on the final line.\n")
	if prev != "" {
		b.WriteString("Previous reply:\n")
		b.WriteString(prev)
		b.WriteString("\n\n")
	}
	b.WriteString("Legal moves:\n")
	for _, m := range legal {
		b.WriteString("- ")
		b.WriteString(m)
		b.WriteString("\n")
	}
	return b.String()
}

var uciMoveRe = regexp.MustCompile(`(?i)\b([a-h][1-8][a-h][1-8][qrbn]?)\b`)
var uciLooseRe = regexp.MustCompile(`(?i)([a-h][1-8])\s*[-–—]?\s*([a-h][1-8])([qrbn])?`)
var colMoveRe = regexp.MustCompile(`\b([0-6])\b`)

func normalizeArenaReply(reply string) string {
	s := strings.TrimSpace(reply)
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

func lastNonEmptyLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return strings.TrimSpace(text)
}

// parseLogicAnswer pulls a compact final answer from a model reply (last line, strip markdown / Answer:).
func parseLogicAnswer(reply string) string {
	line := lastNonEmptyLine(reply)
	line = strings.TrimSpace(line)
	line = strings.ReplaceAll(line, "*", "")
	line = strings.ReplaceAll(line, "`", "")
	line = strings.Trim(line, " \"'")
	lower := strings.ToLower(line)
	for _, prefix := range []string{
		"final answer:",
		"final answer",
		"answer:",
		"answer",
		"thus:",
		"therefore:",
		"conclusion:",
	} {
		if strings.HasPrefix(lower, prefix) {
			line = strings.TrimSpace(line[len(prefix):])
			line = strings.TrimLeft(line, ":-–— ")
			lower = strings.ToLower(line)
			break
		}
	}
	return strings.TrimSpace(strings.Trim(line, " .;:!"))
}

func parseArenaMove(challenge, reply string, legal []string) (map[string]any, error) {
	normalized := normalizeArenaReply(reply)
	lines := strings.Split(strings.TrimSpace(normalized), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if move, err := parseArenaMoveLine(challenge, line, legal); err == nil {
			return move, nil
		}
	}
	// Whole reply: pick the last legal UCI token anywhere in the text.
	if challenge != "connect4" {
		if move, err := parseChessUCIFromText(normalized, legal); err == nil {
			return move, nil
		}
	}
	return parseArenaMoveLine(challenge, normalized, legal)
}

func parseChessUCIFromText(reply string, legal []string) (map[string]any, error) {
	legalSet := make(map[string]struct{}, len(legal))
	for _, m := range legal {
		legalSet[strings.ToLower(m)] = struct{}{}
	}
	candidates := uciMoveRe.FindAllString(strings.ToLower(reply), -1)
	for _, m := range uciLooseRe.FindAllStringSubmatch(strings.ToLower(reply), -1) {
		if len(m) >= 3 {
			cand := m[1] + m[2]
			if len(m) > 3 {
				cand += m[3]
			}
			candidates = append(candidates, cand)
		}
	}
	for i := len(candidates) - 1; i >= 0; i-- {
		m := candidates[i]
		if _, ok := legalSet[m]; ok {
			return map[string]any{"move": m, "uci": m}, nil
		}
	}
	return nil, fmt.Errorf("no legal move in reply")
}

func parseArenaMoveLine(challenge, reply string, legal []string) (map[string]any, error) {
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
	if move, err := parseChessUCIFromText(reply, legal); err == nil {
		return move, nil
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