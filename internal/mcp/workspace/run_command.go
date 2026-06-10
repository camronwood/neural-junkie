package workspace

import (
	"strings"
)

var allowedCommandPrefixes = []string{
	"npm test",
	"npm run lint",
	"npm run test",
	"npm run build",
	"npm run typecheck",
	"npm exec -- tsc",
	"./node_modules/.bin/tsc",
	"go test",
	"go vet",
	"cargo test",
	"pytest",
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

// CommandAllowed reports whether cmd is permitted in run_command sandbox.
func CommandAllowed(cmd string) bool {
	cmd = normalizeCommand(cmd)
	if cmd == "" {
		return false
	}
	lower := strings.ToLower(cmd)
	for _, d := range deniedCommandPatterns {
		if strings.Contains(lower, d) {
			return false
		}
	}
	for _, p := range allowedCommandPrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}
