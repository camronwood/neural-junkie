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

// RustDiagnostics runs cargo check when cargo is on PATH.
func RustDiagnostics(ctx context.Context, workspaceRoot string) ([]Diagnostic, error) {
	if _, err := exec.LookPath("cargo"); err != nil {
		return nil, nil
	}
	root, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cargo", "check", "--message-format=json")
	cmd.Dir = root
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()
	return parseCargoJSON(root, stdout.Bytes()), nil
}

type cargoMessage struct {
	Reason string `json:"reason"`
	Message struct {
		Level   string `json:"level"`
		Message string `json:"message"`
		Spans   []struct {
			FileName string `json:"file_name"`
			LineStart int    `json:"line_start"`
			ColumnStart int  `json:"column_start"`
		} `json:"spans"`
	} `json:"message"`
}

func parseCargoJSON(workspaceRoot string, data []byte) []Diagnostic {
	var out []Diagnostic
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg cargoMessage
		if json.Unmarshal([]byte(line), &msg) != nil || msg.Reason != "compiler-message" {
			continue
		}
		if msg.Message.Level != "error" && msg.Message.Level != "warning" {
			continue
		}
		if len(msg.Message.Spans) == 0 {
			continue
		}
		sp := msg.Message.Spans[0]
		path := sp.FileName
		if rel, err := filepath.Rel(workspaceRoot, path); err == nil && !strings.HasPrefix(rel, "..") {
			path = filepath.ToSlash(rel)
		}
		sev := "error"
		if msg.Message.Level == "warning" {
			sev = "warning"
		}
		out = append(out, Diagnostic{
			Path: path, Line: sp.LineStart, Column: sp.ColumnStart,
			Message: msg.Message.Message, Severity: sev,
		})
	}
	return out
}
