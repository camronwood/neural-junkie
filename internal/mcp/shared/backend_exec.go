package shared

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

type backendKey struct{}
type implementationSessionKey struct{}
type runCommandUserAllowKey struct{}
type runCommandExtraAllowsKey struct{}

// ContextWithImplementationSession marks tool calls that may use broader run_command allowlists.
func ContextWithImplementationSession(ctx context.Context, enabled bool) context.Context {
	if !enabled {
		return ctx
	}
	return context.WithValue(ctx, implementationSessionKey{}, true)
}

// ImplementationSessionFromContext reports whether implementation-session commands are allowed.
func ImplementationSessionFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(implementationSessionKey{}).(bool)
	return v
}

// ContextWithRunCommandUserAllow grants a one-shot allow for a normalized command after user approval.
func ContextWithRunCommandUserAllow(ctx context.Context, command string) context.Context {
	command = strings.Join(strings.Fields(strings.TrimSpace(command)), " ")
	if command == "" {
		return ctx
	}
	return context.WithValue(ctx, runCommandUserAllowKey{}, strings.ToLower(command))
}

// RunCommandUserAllowFromContext returns a one-shot user-approved command, if any.
func RunCommandUserAllowFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(runCommandUserAllowKey{}).(string)
	return v
}

// ContextWithRunCommandExtraAllows attaches persisted user allowlist prefixes.
func ContextWithRunCommandExtraAllows(ctx context.Context, prefixes []string) context.Context {
	if len(prefixes) == 0 {
		return ctx
	}
	out := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		p = strings.Join(strings.Fields(strings.TrimSpace(p)), " ")
		if p != "" {
			out = append(out, strings.ToLower(p))
		}
	}
	if len(out) == 0 {
		return ctx
	}
	return context.WithValue(ctx, runCommandExtraAllowsKey{}, out)
}

// RunCommandExtraAllowsFromContext returns persisted user allowlist prefixes.
func RunCommandExtraAllowsFromContext(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(runCommandExtraAllowsKey{}).([]string)
	return v
}

// ContextWithBackend attaches a workspace backend for remote command execution.
func ContextWithBackend(ctx context.Context, b workspacebackend.Backend) context.Context {
	if b == nil {
		return ctx
	}
	return context.WithValue(ctx, backendKey{}, b)
}

// BackendFromContext returns workspace backend from context if set.
func BackendFromContext(ctx context.Context) workspacebackend.Backend {
	if ctx == nil {
		return nil
	}
	b, _ := ctx.Value(backendKey{}).(workspacebackend.Backend)
	return b
}

// RunCommandViaBackend runs a command through nj-remote sidecar exec.
func RunCommandViaBackend(ctx context.Context, b workspacebackend.Backend, relCwd, name string, args ...string) (string, error) {
	if b == nil {
		return "", fmt.Errorf("backend required")
	}
	res, err := b.Exec(ctx, workspacebackend.ExecRequest{
		Command: name,
		Args:    args,
		RelCwd:  relCwd,
	})
	out := res.Stdout
	if res.Stderr != "" {
		if out != "" && !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		out += res.Stderr
	}
	if err != nil {
		return out, err
	}
	if res.ExitCode != 0 {
		return out, fmt.Errorf("exit %d", res.ExitCode)
	}
	return out, nil
}

// RunCommandMaybeRemote uses backend from context when present, else local exec.
func RunCommandMaybeRemote(ctx context.Context, dir, name string, args ...string) (string, error) {
	if b := BackendFromContext(ctx); b != nil {
		relCwd := "."
		if dir != "" && dir != b.Root() {
			relCwd = strings.TrimPrefix(dir, b.Root())
			relCwd = strings.TrimPrefix(relCwd, "/")
		}
		return RunCommandViaBackend(ctx, b, relCwd, name, args...)
	}
	return RunCommand(ctx, dir, name, args...)
}
