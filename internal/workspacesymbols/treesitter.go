package workspacesymbols

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// treeSitterAvailable reports whether the tree-sitter CLI is on PATH.
func treeSitterAvailable() bool {
	_, err := exec.LookPath("tree-sitter")
	return err == nil
}

// scanFileTreeSitter extracts symbols using tree-sitter parse + query when available.
func scanFileTreeSitter(ctx context.Context, path, content string) ([]Symbol, error) {
	if !treeSitterAvailable() {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// tree-sitter parse --quiet prints S-expression; we extract def-like nodes heuristically.
	cmd := exec.CommandContext(ctx, "tree-sitter", "parse", "--quiet", "-")
	cmd.Stdin = strings.NewReader(content)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return extractSymbolsFromTree(path, stdout.String()), nil
}

func extractSymbolsFromTree(path, tree string) []Symbol {
	var out []Symbol
	lines := strings.Split(tree, "\n")
	lineNum := 1
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		for _, kind := range []struct {
			node string
			kind string
		}{
			{"function_definition", "function"},
			{"function_declaration", "function"},
			{"method_definition", "method"},
			{"class_definition", "class"},
			{"type_specifier", "type"},
		} {
			if strings.Contains(trim, "("+kind.node) {
				name := extractTreeSitterName(trim)
				if name != "" {
					out = append(out, Symbol{Name: name, Kind: kind.kind, Path: path, Line: lineNum})
				}
			}
		}
		lineNum++
	}
	return out
}

func extractTreeSitterName(s string) string {
	// (function_definition name: (identifier) @name) — grab first quoted or identifier token
	if i := strings.Index(s, `"`); i >= 0 {
		rest := s[i+1:]
		if j := strings.Index(rest, `"`); j > 0 {
			return rest[:j]
		}
	}
	parts := strings.Fields(strings.Trim(s, "()"))
	for _, p := range parts {
		if len(p) > 1 && p[0] != '@' && !strings.Contains(p, ":") {
			return strings.Trim(p, `"`)
		}
	}
	return ""
}
