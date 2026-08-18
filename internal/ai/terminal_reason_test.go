package ai

import "testing"

func TestNormalizeTerminalReason(t *testing.T) {
	cases := map[string]string{
		"length":               TerminalReasonLength,
		"max_tokens":           TerminalReasonLength,
		"MAX_TOKENS":           TerminalReasonLength,
		"stop":                 TerminalReasonStop,
		"end_turn":             TerminalReasonStop,
		"tool_calls":           TerminalReasonToolCalls,
		"timeout":              TerminalReasonTimeout,
		"cancelled":            TerminalReasonCancelled,
		"canceled":             TerminalReasonCancelled,
		"error":                TerminalReasonError,
		"":                     "",
		"unload":               "",
		"something_custom":     "",
	}
	for in, want := range cases {
		if got := NormalizeTerminalReason(in); got != want {
			t.Fatalf("NormalizeTerminalReason(%q)=%q want %q", in, got, want)
		}
	}
}
