package agent

import (
	"strings"
	"testing"
)

func TestUserRequestsImplementation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"Please implement the light/dark theme plan", true},
		{"ok please implement that", true},
		{"I want to add UI themes under settings", true},
		{"can you implment that for me", true},
		{"can you see my workspace?", false},
		{"review src/App.tsx", false},
	}
	for _, tc := range cases {
		if got := userRequestsImplementation(tc.in); got != tc.want {
			t.Errorf("userRequestsImplementation(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestShouldProactiveScanWorkspace_skipsBulkScanOnImplement(t *testing.T) {
	t.Parallel()
	msg := "Please implement themes. use only packages in package.json."
	if shouldProactiveScanWorkspace(msg) {
		t.Fatal("implementation + package.json mention should not trigger proactive scan")
	}
	if !shouldProactiveScanWorkspace("review internal/hub/hub.go for bugs") {
		t.Fatal("review with path should still scan")
	}
}

func TestIsUserRequestingFileWrite_implement(t *testing.T) {
	t.Parallel()
	if !isUserRequestingFileWrite("please implement the theme plan") {
		t.Fatal("expected implement to count as file-write intent for fallback")
	}
}

func TestWorkspaceGroundingRequirement_implement(t *testing.T) {
	t.Parallel()
	g := workspaceGroundingRequirement(3, "please implement themes")
	if g == "" || !strings.Contains(g, "FILE_CHANGE") || !strings.Contains(g, "codebase tour") {
		t.Fatalf("expected implementation grounding, got %q", g)
	}
}
