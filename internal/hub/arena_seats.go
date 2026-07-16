package hub

import "strings"

const arenaHumanPlayer = "human"

// arenaActiveSeat maps the sidecar turn indicator to the white/black player slots.
func arenaActiveSeat(challenge string, state map[string]any) string {
	if state == nil {
		return ""
	}
	turn := strings.TrimSpace(strings.ToLower(stringField(state["turn"])))
	switch strings.TrimSpace(strings.ToLower(challenge)) {
	case "connect4":
		switch turn {
		case "red":
			return "white"
		case "yellow":
			return "black"
		}
	case "chess":
		switch turn {
		case "white":
			return "white"
		case "black":
			return "black"
		}
	}
	return ""
}

func arenaPlayers(session map[string]any) (white, black string) {
	players := mapStringAny(session["players"])
	if players == nil {
		return "", ""
	}
	return strings.TrimSpace(stringField(players["white"])), strings.TrimSpace(stringField(players["black"]))
}

// arenaModelForSeat returns the model tag for a seat and whether that seat is human-controlled.
func arenaModelForSeat(session map[string]any, seat, fallbackModel string) (model string, humanTurn bool) {
	white, black := arenaPlayers(session)
	player := ""
	switch strings.TrimSpace(strings.ToLower(seat)) {
	case "white":
		player = white
	case "black":
		player = black
	}
	if player == "" {
		player = strings.TrimSpace(fallbackModel)
	}
	if strings.EqualFold(player, arenaHumanPlayer) {
		return "", true
	}
	if player != "" {
		return player, false
	}
	fallback := strings.TrimSpace(fallbackModel)
	if fallback != "" && !strings.EqualFold(fallback, arenaHumanPlayer) {
		return fallback, false
	}
	return "", false
}

// arenaLogicModel picks the solver model from session roster, then request fallback.
func arenaLogicModel(session map[string]any, fallbackModel string) string {
	seat := arenaLogicNextSeat(session)
	if seat != "" {
		if model := arenaLogicModelForSeat(session, seat); model != "" {
			return model
		}
	}
	white, black := arenaPlayers(session)
	if white != "" && !strings.EqualFold(white, arenaHumanPlayer) {
		return white
	}
	if black != "" && !strings.EqualFold(black, arenaHumanPlayer) {
		return black
	}
	return strings.TrimSpace(fallbackModel)
}

// arenaLogicSeats returns model-controlled seats (white/black) that should answer.
func arenaLogicSeats(session map[string]any) []string {
	white, black := arenaPlayers(session)
	var seats []string
	if white != "" && !strings.EqualFold(white, arenaHumanPlayer) {
		seats = append(seats, "white")
	}
	if black != "" && !strings.EqualFold(black, arenaHumanPlayer) {
		seats = append(seats, "black")
	}
	return seats
}

func arenaLogicModelForSeat(session map[string]any, seat string) string {
	white, black := arenaPlayers(session)
	switch strings.TrimSpace(strings.ToLower(seat)) {
	case "white":
		return white
	case "black":
		return black
	default:
		return ""
	}
}

// arenaLogicNextSeat returns the next model seat that has not submitted an answer yet.
func arenaLogicNextSeat(session map[string]any) string {
	answers := mapStringAny(session["answers"])
	for _, seat := range arenaLogicSeats(session) {
		if answers == nil {
			return seat
		}
		if _, ok := answers[seat]; !ok {
			return seat
		}
	}
	return ""
}

func stringField(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func arenaSessionTerminal(st map[string]any) bool {
	if st == nil {
		return false
	}
	s := strings.TrimSpace(strings.ToLower(stringField(st["status"])))
	return s != "" && s != "active"
}
