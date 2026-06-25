package workspace

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
)

var allowedCommandPrefixes = []string{
	"npm test",
	"npm run lint",
	"npm run test",
	"npm run build",
	"npm run typecheck",
	"npm run dev",
	"npm exec -- tsc",
	"./node_modules/.bin/tsc",
	"go test",
	"go vet",
	"cargo test",
	"pytest",
}

// Broader allowlist for implementation sessions (agent coding loop).
var implementationSessionCommandPrefixes = []string{
	"npm install",
	"npm ci",
	"npm i ",
	"npx --yes",
	"npx tsc",
	"npx vite",
	"npx eslint",
	"npx prettier",
	"make start-all",
	"make build",
	"make build-release",
	"make help",
}

var deniedCommandPatterns = []string{
	"rm -rf",
	"rm -r ",
	"sudo ",
	"curl ",
	"wget ",
	"| sh",
	"| bash",
	">/dev/",
	"chmod ",
	"mkfs",
}

func normalizeCommand(cmd string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(cmd)), " ")
}

func commandDenied(cmd string) bool {
	lower := strings.ToLower(cmd)
	for _, d := range deniedCommandPatterns {
		if strings.Contains(lower, d) {
			return true
		}
	}
	return false
}

func commandHasAllowedPrefix(cmd string, prefixes []string) bool {
	lower := strings.ToLower(cmd)
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// CommandAllowed reports whether cmd is permitted in run_command sandbox.
func CommandAllowed(cmd string) bool {
	cmd = normalizeCommand(cmd)
	if cmd == "" || commandDenied(cmd) {
		return false
	}
	return commandHasAllowedPrefix(cmd, allowedCommandPrefixes)
}

func implementationSessionCommandAllowed(cmd string) bool {
	cmd = normalizeCommand(cmd)
	if cmd == "" || commandDenied(cmd) {
		return false
	}
	if commandHasAllowedPrefix(cmd, allowedCommandPrefixes) {
		return true
	}
	return commandHasAllowedPrefix(cmd, implementationSessionCommandPrefixes)
}

// CommandAllowedForContext applies the implementation-session allowlist when ctx is marked.
func CommandAllowedForContext(ctx context.Context, cmd string) bool {
	if CommandAllowed(cmd) {
		return true
	}
	if shared.ImplementationSessionFromContext(ctx) {
		return implementationSessionCommandAllowed(cmd)
	}
	return false
}
