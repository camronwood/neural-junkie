// Package lsp provides optional language-server helpers (gopls diagnostics).
package lsp

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"path/filepath"
	"strings"
	"time"
)

// Diagnostic is a single issue from gopls or similar.
type Diagnostic struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// GoDiagnostics runs `gopls check` in workspaceRoot when gopls is on PATH.
func GoDiagnostics(ctx context.Context, workspaceRoot string) ([]Diagnostic, error) {
	if _, err := exec.LookPath("gopls"); err != nil {
		return nil, nil
	}
	root, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gopls", "check", "./...")
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// gopls check exits non-zero when issues exist; parse stdout/stderr anyway.
		combined := stdout.String() + "\n" + stderr.String()
		if strings.TrimSpace(combined) == "" {
			return nil, nil
		}
	}
	return parseGoplsCheckOutput(root, stdout.String()+"\n"+stderr.String()), nil
}

func parseGoplsCheckOutput(workspaceRoot, text string) []Diagnostic {
	var out []Diagnostic
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// file.go:12:3: message
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		path := parts[0]
		if !filepath.IsAbs(path) {
			path = filepath.Join(workspaceRoot, path)
		}
		rel, _ := filepath.Rel(workspaceRoot, path)
		if rel != "" && !strings.HasPrefix(rel, "..") {
			path = filepath.ToSlash(rel)
		}
		lineNum := 1
		if n, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			lineNum = n
		}
		col := 1
		if n, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
			col = n
		}
		msg := strings.TrimSpace(parts[3])
		out = append(out, Diagnostic{
			Path: path, Line: lineNum, Column: col,
			Message: msg, Severity: "error",
		})
	}
	return out
}
