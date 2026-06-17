package lsp

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// DiagnosticsViaBackend runs language-server check commands on a workspace backend (local or remote).
func DiagnosticsViaBackend(ctx context.Context, b workspacebackend.Backend, lang string) ([]Diagnostic, error) {
	if b == nil {
		return nil, nil
	}
	switch lang {
	case "go":
		res, err := b.Exec(ctx, workspacebackend.ExecRequest{
			Command: "gopls",
			Args:    []string{"check", "./..."},
		})
		if err != nil {
			return nil, err
		}
		combined := res.Stdout + "\n" + res.Stderr
		if strings.TrimSpace(combined) == "" {
			return nil, nil
		}
		return parseGoplsCheckOutput(b.Root(), combined), nil
	case "rust":
		res, err := b.Exec(ctx, workspacebackend.ExecRequest{
			Command: "cargo",
			Args:    []string{"check", "--message-format=short"},
		})
		if err != nil {
			return nil, err
		}
		return parseCargoShortOutput(b.Root(), res.Stdout+"\n"+res.Stderr), nil
	case "python":
		check, checkErr := b.Exec(ctx, workspacebackend.ExecRequest{
			Command: "pyright",
			Args:    []string{"."},
		})
		combined := check.Stdout + "\n" + check.Stderr
		if checkErr != nil && strings.TrimSpace(combined) == "" {
			return nil, checkErr
		}
		return parsePyrightOutput(b.Root(), combined), nil
	default:
		return nil, nil
	}
}

func parseCargoShortOutput(workspaceRoot, text string) []Diagnostic {
	var out []Diagnostic
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		lineNum := atoiTrim(parts[1])
		col := atoiTrim(parts[2])
		out = append(out, Diagnostic{
			Path:     parts[0],
			Line:     lineNum,
			Column:   col,
			Message:  strings.TrimSpace(parts[3]),
			Severity: "error",
		})
	}
	return out
}

func atoiTrim(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func parsePyrightOutput(workspaceRoot, text string) []Diagnostic {
	var out []Diagnostic
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, " - ") {
			continue
		}
		// file:line:col - error: message
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) != 2 {
			continue
		}
		loc := strings.Split(parts[0], ":")
		if len(loc) < 3 {
			continue
		}
		lineNum := atoiTrim(loc[1])
		col := atoiTrim(loc[2])
		msg := strings.TrimSpace(parts[1])
		sev := "warning"
		if strings.Contains(strings.ToLower(msg), "error") {
			sev = "error"
		}
		out = append(out, Diagnostic{
			Path:     loc[0],
			Line:     lineNum,
			Column:   col,
			Message:  msg,
			Severity: sev,
		})
	}
	return out
}
