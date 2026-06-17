package shared

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

type backendKey struct{}

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
