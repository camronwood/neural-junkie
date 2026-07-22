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
	"go build",
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

var readOnlyGitSubcommands = map[string]bool{
	"status":    true,
	"diff":      true,
	"show":      true,
	"log":       true,
	"rev-parse": true,
	"ls-files":  true,
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

func readOnlyGitCommandAllowed(cmd string) bool {
	cmd = normalizeCommand(cmd)
	if cmd == "" || commandDenied(cmd) || strings.ContainsAny(cmd, "\n;&|`$><") {
		return false
	}
	fields := strings.Fields(cmd)
	if len(fields) < 2 || strings.ToLower(fields[0]) != "git" ||
		!readOnlyGitSubcommands[strings.ToLower(fields[1])] {
		return false
	}
	for _, field := range fields[2:] {
		lower := strings.ToLower(strings.Trim(field, `"'`))
		if strings.HasPrefix(lower, "--output") || lower == "--ext-diff" ||
			lower == "--textconv" || strings.HasPrefix(lower, "--exec") {
			return false
		}
	}
	return true
}

// CommandAllowed reports whether cmd is permitted in run_command sandbox.
func CommandAllowed(cmd string) bool {
	cmd = normalizeCommand(cmd)
	if cmd == "" || commandDenied(cmd) {
		return false
	}
	return readOnlyGitCommandAllowed(cmd) || commandHasAllowedPrefix(cmd, allowedCommandPrefixes)
}

// CommandHardDenied reports whether cmd is blocked even with user approval
// (destructive / network exfil patterns).
func CommandHardDenied(cmd string) bool {
	return commandDenied(normalizeCommand(cmd))
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

func userAllowMatches(ctx context.Context, cmd string) bool {
	cmd = strings.ToLower(normalizeCommand(cmd))
	if cmd == "" {
		return false
	}
	if allow := shared.RunCommandUserAllowFromContext(ctx); allow != "" &&
		(cmd == allow || strings.HasPrefix(cmd, allow+" ")) {
		return true
	}
	return commandHasAllowedPrefix(cmd, shared.RunCommandExtraAllowsFromContext(ctx))
}

// CommandAllowedForContext applies the implementation-session allowlist when ctx is marked.
func CommandAllowedForContext(ctx context.Context, cmd string) bool {
	if CommandAllowed(cmd) {
		return true
	}
	if userAllowMatches(ctx, cmd) {
		return true
	}
	if shared.ImplementationSessionFromContext(ctx) {
		return implementationSessionCommandAllowed(cmd)
	}
	return false
}

// CommandNeedsUserApproval is true when the command is safe-to-prompt but not yet allowlisted.
func CommandNeedsUserApproval(ctx context.Context, cmd string) bool {
	cmd = normalizeCommand(cmd)
	if cmd == "" || CommandHardDenied(cmd) {
		return false
	}
	return !CommandAllowedForContext(ctx, cmd)
}
