package shared

import (
	"context"
	"fmt"
	"strings"
)

// CommandPolicy tracks run_command history during implementation sessions.
type CommandPolicy interface {
	BootFixGatingEnabled() bool
	BootFixReadsSatisfied() bool
	RecordCommandRun(cmd string, exitCode int, output string)
	RecordReadPath(path string)
	RecordEdit(path string)
	ShouldBlockRunCommand(cmd string) error
	LastCommandOutput() string
	CircuitBreakerTriggered() bool
	PlaybookUsed() string
	SetPlaybookUsed(name string)
	CommandFailureSummary() []CommandFailureCount
}

// CommandFailureCount summarizes repeated command failures for session outcomes.
type CommandFailureCount struct {
	Command string
	Count   int
}

type commandPolicyKey struct{}

// ContextWithCommandPolicy attaches command policy state for MCP tool guards.
func ContextWithCommandPolicy(ctx context.Context, p CommandPolicy) context.Context {
	if p == nil {
		return ctx
	}
	return context.WithValue(ctx, commandPolicyKey{}, p)
}

// CommandPolicyFromContext returns command policy when present.
func CommandPolicyFromContext(ctx context.Context) CommandPolicy {
	if ctx == nil {
		return nil
	}
	p, _ := ctx.Value(commandPolicyKey{}).(CommandPolicy)
	return p
}

// BootFixBootCommand reports whether cmd is a boot/dev command gated during boot-fix.
func BootFixBootCommand(cmd string) bool {
	cmd = strings.ToLower(strings.TrimSpace(cmd))
	if cmd == "" {
		return false
	}
	prefixes := []string{
		"make start-all",
		"make build",
		"npm run dev",
		"npm start",
		"npm run build",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(cmd, p) {
			return true
		}
	}
	return false
}

// BootFixRequiredReadPaths returns stack paths that satisfy boot-fix grounding.
func BootFixRequiredReadPaths() []string {
	return []string{
		"Makefile",
		"package.json",
		"scripts/start-all.sh",
		"src-tauri/tauri.conf.json",
	}
}

// NormalizeCommandPath normalizes a workspace-relative path for read tracking.
func NormalizeCommandPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	return strings.ReplaceAll(path, "\\", "/")
}

// CommandMatchesRequiredBootRead reports whether path satisfies boot-fix read gate.
func CommandMatchesRequiredBootRead(path string) bool {
	path = strings.ToLower(NormalizeCommandPath(path))
	if path == "" {
		return false
	}
	for _, req := range BootFixRequiredReadPaths() {
		if path == strings.ToLower(req) || strings.HasSuffix(path, "/"+strings.ToLower(req)) {
			return true
		}
	}
	return false
}

// BootFixReadGateError is returned when boot commands run before required reads.
func BootFixReadGateError() error {
	return fmt.Errorf("boot-fix grounding: read Makefile, package.json, or scripts/start-all.sh with read_file before running make/npm dev commands")
}

// RepeatedCommandFailureError is returned when the same command failed repeatedly without edits.
func RepeatedCommandFailureError(cmd string) error {
	return fmt.Errorf("repeated command failure — read Makefile/package.json and propose an edit before re-running: %s", strings.TrimSpace(cmd))
}
