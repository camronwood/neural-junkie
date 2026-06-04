package cad

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RenderOptions configures an OpenSCAD render run.
type RenderOptions struct {
	OpenSCADPath string
	Timeout      time.Duration
	Params       map[string]string
}

// RenderSCADToSTL runs OpenSCAD on scadPath and writes stlPath.
func RenderSCADToSTL(ctx context.Context, scadPath, stlPath string, opts RenderOptions) error {
	scadPath = strings.TrimSpace(scadPath)
	stlPath = strings.TrimSpace(stlPath)
	if scadPath == "" || stlPath == "" {
		return fmt.Errorf("scad and stl paths required")
	}
	if _, err := os.Stat(scadPath); err != nil {
		return fmt.Errorf("scad file: %w", err)
	}
	bin := strings.TrimSpace(opts.OpenSCADPath)
	if bin == "" {
		bin = "openscad"
	}
	if err := os.MkdirAll(filepath.Dir(stlPath), 0755); err != nil {
		return err
	}
	args := []string{"-o", stlPath}
	for k, v := range opts.Params {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		args = append(args, "-D", fmt.Sprintf("%s=%s", k, formatOpenSCADValue(v)))
	}
	args = append(args, scadPath)

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		if runCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("openscad timed out after %s: %s", timeout, msg)
		}
		if errorsIsNotFound(err) {
			return fmt.Errorf("openscad not found (%q): install from https://openscad.org or set path in Settings → CAD tools", bin)
		}
		return fmt.Errorf("openscad failed: %s", msg)
	}
	if _, err := os.Stat(stlPath); err != nil {
		return fmt.Errorf("openscad did not produce output: %w", err)
	}
	return nil
}

func formatOpenSCADValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return `""`
	}
	if strings.HasPrefix(v, "[") || strings.HasPrefix(v, "(") {
		return v
	}
	lower := strings.ToLower(v)
	if lower == "true" || lower == "false" {
		return lower
	}
	if isNumericLiteral(v) {
		return v
	}
	return fmt.Sprintf("%q", v)
}

func isNumericLiteral(s string) bool {
	if s == "" {
		return false
	}
	dot := false
	for i, r := range s {
		if r == '.' {
			if dot {
				return false
			}
			dot = true
			continue
		}
		if r == '-' && i == 0 {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func errorsIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errorsIsExec(err) {
		return true
	}
	return os.IsNotExist(err)
}

func errorsIsExec(err error) bool {
	var pathErr *exec.Error
	if err == nil {
		return false
	}
	// exec.Error is deprecated but still returned for not found on some platforms
	if strings.Contains(err.Error(), "executable file not found") {
		return true
	}
	_ = pathErr
	return false
}

// TestOpenSCAD runs `openscad --version` and returns combined output.
func TestOpenSCAD(ctx context.Context, bin string) (string, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		bin = "openscad"
	}
	runCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, bin, "--version")
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if errorsIsNotFound(err) {
			return "", fmt.Errorf("openscad not found at %q", bin)
		}
		return text, fmt.Errorf("%s", text)
	}
	return text, nil
}
