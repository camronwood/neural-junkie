package shared

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunCommand runs a command in dir and returns combined output.
func RunCommand(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// PathExists reports whether path exists on disk.
func PathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// FindProjectRoot walks up from path looking for marker files.
func FindProjectRoot(start string, markers ...string) string {
	dir := start
	if info, err := os.Stat(start); err == nil && !info.IsDir() {
		dir = filepath.Dir(start)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	for {
		for _, m := range markers {
			if PathExists(filepath.Join(abs, m)) {
				return abs
			}
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			break
		}
		abs = parent
	}
	return dir
}

// FormatCommandResult formats command output for MCP tools.
func FormatCommandResult(label string, output string, err error) string {
	var b strings.Builder
	b.WriteString(label)
	b.WriteString("\n")
	if output != "" {
		b.WriteString(output)
		if !strings.HasSuffix(output, "\n") {
			b.WriteString("\n")
		}
	}
	if err != nil {
		b.WriteString(fmt.Sprintf("Exit error: %v\n", err))
	}
	return b.String()
}
