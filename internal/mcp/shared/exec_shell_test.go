package shared

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestIsDevServerCommand(t *testing.T) {
	cases := map[string]bool{
		"make start-all":                     true,
		"npm run build":                      false,
		"npm run dev":                        true,
		"npm run tauri:dev -- --verbose":     true,
		"npx tauri dev":                      true,
		"./node_modules/.bin/tsc --noEmit":   false,
	}
	for cmd, want := range cases {
		if got := IsDevServerCommand(cmd); got != want {
			t.Errorf("IsDevServerCommand(%q) = %v, want %v", cmd, got, want)
		}
	}
}

func TestRunShellCommand_streamsProgress(t *testing.T) {
	ctx := ContextWithRunCommandProgress(context.Background(), func(line string) {
		// progress hook wired
	})
	ctx = ContextWithRunCommandTimeout(ctx, 5*time.Second)
	res, err := RunShellCommand(ctx, "", "echo hello-stream")
	if err != nil {
		t.Fatalf("RunShellCommand: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit = %d", res.ExitCode)
	}
	if !strings.Contains(res.Output, "hello-stream") {
		t.Fatalf("output = %q", res.Output)
	}
}

func TestRunShellCommand_devServerTimeoutAnnotation(t *testing.T) {
	ctx := ContextWithRunCommandTimeout(context.Background(), 50*time.Millisecond)
	res, _ := RunShellCommand(ctx, "", "sleep 2")
	if !res.TimedOut {
		t.Fatalf("expected timeout, exit=%d output=%q", res.ExitCode, res.Output)
	}
}

func TestRunShellCommand_devServerTimeoutHintOnlyForDevCommands(t *testing.T) {
	ctx := ContextWithRunCommandTimeout(context.Background(), 50*time.Millisecond)
	res, _ := RunShellCommand(ctx, "", "sleep 2")
	if strings.Contains(res.Output, "bootfix_hint=dev_server_timeout") {
		t.Fatalf("non-dev command should not get dev server hint: %q", res.Output)
	}
}

func TestRunCommandTimeoutFromContext(t *testing.T) {
	ctx := ContextWithRunCommandTimeout(context.Background(), 12*time.Second)
	if got := RunCommandTimeoutFromContext(ctx); got != 12*time.Second {
		t.Fatalf("got %v", got)
	}
	if got := RunCommandTimeoutFromContext(context.Background()); got != DefaultRunCommandTimeout {
		t.Fatalf("default got %v", got)
	}
}

func TestRunShellCommand_annotatesDevServerTimeout(t *testing.T) {
	ctx := ContextWithRunCommandTimeout(context.Background(), 50*time.Millisecond)
	res, _ := RunShellCommand(ctx, "", "npm run dev -- sleep 2")
	if !strings.Contains(res.Output, "bootfix_hint=dev_server_timeout") {
		t.Fatalf("expected dev server timeout hint, output=%q", res.Output)
	}
}

func TestAnnotateDevServerTimeout(t *testing.T) {
	out := annotateDevServerTimeout("vite starting")
	if !strings.Contains(out, "bootfix_hint=dev_server_timeout") {
		t.Fatalf("missing hint: %q", out)
	}
	if !strings.Contains(strings.ToLower(out), "dev server command timed out") {
		t.Fatalf("missing guidance: %q", out)
	}
}
