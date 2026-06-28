package shared

import (
	"context"
	"strings"
	"time"
)

const (
	DefaultRunCommandTimeout = 60 * time.Second
	BootFixRunCommandTimeout = 30 * time.Second
)

type runCommandTimeoutKey struct{}
type runCommandProgressKey struct{}
type bootFixDiagnosticKey struct{}

// RunCommandProgress receives streamed stdout/stderr lines during run_command.
type RunCommandProgress func(line string)

// ContextWithRunCommandTimeout overrides the default run_command timeout.
func ContextWithRunCommandTimeout(ctx context.Context, d time.Duration) context.Context {
	if ctx == nil || d <= 0 {
		return ctx
	}
	return context.WithValue(ctx, runCommandTimeoutKey{}, d)
}

// RunCommandTimeoutFromContext returns the run_command timeout, or DefaultRunCommandTimeout.
func RunCommandTimeoutFromContext(ctx context.Context) time.Duration {
	if ctx == nil {
		return DefaultRunCommandTimeout
	}
	if d, ok := ctx.Value(runCommandTimeoutKey{}).(time.Duration); ok && d > 0 {
		return d
	}
	return DefaultRunCommandTimeout
}

// ContextWithRunCommandProgress attaches a callback for streamed command output lines.
func ContextWithRunCommandProgress(ctx context.Context, fn RunCommandProgress) context.Context {
	if ctx == nil || fn == nil {
		return ctx
	}
	return context.WithValue(ctx, runCommandProgressKey{}, fn)
}

// RunCommandProgressFromContext returns the progress callback when set.
func RunCommandProgressFromContext(ctx context.Context) RunCommandProgress {
	if ctx == nil {
		return nil
	}
	fn, _ := ctx.Value(runCommandProgressKey{}).(RunCommandProgress)
	return fn
}

// ContextWithBootFixDiagnostic marks bootstrap diagnostic run_command calls.
func ContextWithBootFixDiagnostic(ctx context.Context, enabled bool) context.Context {
	if !enabled {
		return ctx
	}
	return context.WithValue(ctx, bootFixDiagnosticKey{}, true)
}

// BootFixDiagnosticFromContext reports whether the call is a boot-fix bootstrap diagnostic.
func BootFixDiagnosticFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(bootFixDiagnosticKey{}).(bool)
	return v
}

// IsDevServerCommand reports whether cmd starts a long-running dev server.
func IsDevServerCommand(cmd string) bool {
	cmd = strings.ToLower(strings.TrimSpace(cmd))
	if cmd == "" {
		return false
	}
	prefixes := []string{
		"make start-all",
		"make dev",
		"npm run dev",
		"npm start",
		"yarn dev",
		"pnpm dev",
		"pnpm start",
		"tauri dev",
		"vite",
		"./scripts/start-all.sh",
		"bash scripts/start-all.sh",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(cmd, p) {
			return true
		}
	}
	return false
}
