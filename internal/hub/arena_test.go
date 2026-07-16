package hub

import "testing"

func TestArenaActiveSeat(t *testing.T) {
	tests := []struct {
		challenge string
		turn      string
		want      string
	}{
		{"connect4", "red", "white"},
		{"connect4", "yellow", "black"},
		{"chess", "white", "white"},
		{"chess", "black", "black"},
		{"connect4", "", ""},
	}
	for _, tc := range tests {
		got := arenaActiveSeat(tc.challenge, map[string]any{"turn": tc.turn})
		if got != tc.want {
			t.Fatalf("arenaActiveSeat(%q, %q) = %q, want %q", tc.challenge, tc.turn, got, tc.want)
		}
	}
}

func TestArenaModelForSeat(t *testing.T) {
	session := map[string]any{
		"players": map[string]any{
			"white": "qwen3.5:9b",
			"black": "qwen2.5-coder:14b",
		},
	}
	model, human := arenaModelForSeat(session, "white", "")
	if human || model != "qwen3.5:9b" {
		t.Fatalf("white seat: model=%q human=%v", model, human)
	}
	model, human = arenaModelForSeat(session, "black", "")
	if human || model != "qwen2.5-coder:14b" {
		t.Fatalf("black seat: model=%q human=%v", model, human)
	}

	humanSession := map[string]any{
		"players": map[string]any{
			"white": "human",
			"black": "qwen3.5:9b",
		},
	}
	model, human = arenaModelForSeat(humanSession, "white", "")
	if !human || model != "" {
		t.Fatalf("human white: model=%q human=%v", model, human)
	}
	model, human = arenaModelForSeat(humanSession, "black", "")
	if human || model != "qwen3.5:9b" {
		t.Fatalf("model black: model=%q human=%v", model, human)
	}
}

func TestArenaLogicModel(t *testing.T) {
	session := map[string]any{
		"players": map[string]any{
			"white": "human",
			"black": "qwen3.5:9b",
		},
	}
	if got := arenaLogicModel(session, ""); got != "qwen3.5:9b" {
		t.Fatalf("arenaLogicModel = %q, want qwen3.5:9b", got)
	}
}

func TestArenaLogicNextSeat(t *testing.T) {
	session := map[string]any{
		"players": map[string]any{
			"white": "qwen3.5:9b",
			"black": "qwen2.5-coder:14b",
		},
	}
	if got := arenaLogicNextSeat(session); got != "white" {
		t.Fatalf("first seat = %q, want white", got)
	}
	session["answers"] = map[string]any{
		"white": map[string]any{"answer": "knave"},
	}
	if got := arenaLogicNextSeat(session); got != "black" {
		t.Fatalf("second seat = %q, want black", got)
	}
	session["answers"] = map[string]any{
		"white": map[string]any{"answer": "knave"},
		"black": map[string]any{"answer": "knight"},
	}
	if got := arenaLogicNextSeat(session); got != "" {
		t.Fatalf("done seat = %q, want empty", got)
	}
}

func TestParseArenaMove(t *testing.T) {
	move, err := parseArenaMove("connect4", "I choose column 3", []string{"0", "1", "2", "3"})
	if err != nil || move["column"] != 3 {
		t.Fatalf("connect4 parse: %#v err=%v", move, err)
	}
	move, err = parseArenaMove("connect4", "Center control looks strong.\nI'll block their threat.\n3", []string{"0", "1", "2", "3"})
	if err != nil || move["column"] != 3 {
		t.Fatalf("connect4 multiline parse: %#v err=%v", move, err)
	}
	move, err = parseArenaMove("chess", "e2e4", []string{"e2e4", "d2d4"})
	if err != nil || move["move"] != "e2e4" {
		t.Fatalf("chess parse: %#v err=%v", move, err)
	}
}
