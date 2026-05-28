package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type pyrightDiagnostic struct {
	File    string `json:"file"`
	Severity string `json:"severity"`
	Message string `json:"message"`
	Range   struct {
		Start struct {
			Line int `json:"line"`
			Character int `json:"character"`
		} `json:"start"`
	} `json:"range"`
}

// PythonDiagnostics runs pyright when on PATH.
func PythonDiagnostics(ctx context.Context, workspaceRoot string) ([]Diagnostic, error) {
	bin := "pyright"
	if _, err := exec.LookPath(bin); err != nil {
		return nil, nil
	}
	root, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--outputjson", root)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()
	var payload struct {
		GeneralDiagnostics []pyrightDiagnostic `json:"generalDiagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		return nil, nil
	}
	var out []Diagnostic
	for _, d := range payload.GeneralDiagnostics {
		path := d.File
		if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
			path = filepath.ToSlash(rel)
		}
		sev := "error"
		if strings.EqualFold(d.Severity, "warning") {
			sev = "warning"
		}
		out = append(out, Diagnostic{
			Path: path,
			Line: d.Range.Start.Line + 1,
			Column: d.Range.Start.Character + 1,
			Message: d.Message,
			Severity: sev,
		})
	}
	return out, nil
}
