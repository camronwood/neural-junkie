package protocol

import "testing"

func TestNormalizeAgentNamePreservesCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"GuitarCoach", "GuitarCoach"},
		{"RustExpert", "RustExpert"},
		{"Day One Expert", "Day-One-Expert"},
		{"my-app-expert", "my-app-expert"},
	}
	for _, tc := range tests {
		if got := NormalizeAgentName(tc.in); got != tc.want {
			t.Errorf("NormalizeAgentName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
