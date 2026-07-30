//go:build darwin || linux

package ollama

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLinuxElevator_returnsKnownMethod(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	got := linuxElevator()
	if got != "pkexec" && got != "bash" {
		t.Fatalf("linuxElevator() = %q", got)
	}
}

func TestElevatedInstallCmd_producesRunnableBash(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "install.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd, method, err := elevatedInstallCmd(context.Background(), script)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("nil cmd")
	}
	switch runtime.GOOS {
	case "linux":
		if method != "pkexec" && method != "bash" {
			t.Fatalf("method = %q", method)
		}
		if method == "pkexec" && (len(cmd.Args) < 3 || !strings.Contains(cmd.Path, "pkexec") && cmd.Args[0] != "pkexec" && !strings.HasSuffix(cmd.Path, "pkexec")) {
			// CommandContext may set Path to absolute pkexec
			if !strings.Contains(cmd.Path, "pkexec") && (len(cmd.Args) == 0 || cmd.Args[0] != "pkexec") {
				t.Fatalf("expected pkexec command, path=%q args=%v", cmd.Path, cmd.Args)
			}
		}
	case "darwin":
		if method != "osascript" && method != "bash" {
			t.Fatalf("method = %q", method)
		}
	}
}

func TestDarwinAdminBashCmd_embedsScriptPath(t *testing.T) {
	cmd := darwinAdminBashCmd(context.Background(), "/bin/bash", `/tmp/nj ollama.sh`)
	if len(cmd.Args) < 3 {
		t.Fatalf("args = %v", cmd.Args)
	}
	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, `/tmp/nj ollama.sh`) {
		t.Fatalf("script path missing from AppleScript: %s", joined)
	}
	if !strings.Contains(joined, "administrator privileges") {
		t.Fatalf("missing admin privileges: %s", joined)
	}
}
