// Package workspacesymbols provides lightweight workspace symbol search without LSP.
package workspacesymbols

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacefiles"
)

// Symbol is a navigable definition in a source file.
type Symbol struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Kind     string `json:"kind"`
	Language string `json:"language"`
}

var (
	tsSym = regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+(\w+)|^\s*(?:export\s+)?class\s+(\w+)|^\s*(?:export\s+)?interface\s+(\w+)|^\s*(?:export\s+)?(?:const|let|var)\s+(\w+)`)
	extLang = map[string]string{
		".go": "go", ".ts": "typescript", ".tsx": "typescript",
		".js": "javascript", ".jsx": "javascript", ".rs": "rust",
	}
)

// Search finds symbols whose name contains q (case-insensitive).
func Search(ctx context.Context, workspaceRoot, q string, limit int) ([]Symbol, error) {
	if limit <= 0 {
		limit = 50
	}
	q = strings.ToLower(strings.TrimSpace(q))
	paths, err := workspacefiles.Search(ctx, workspaceRoot, "", 5000)
	if err != nil {
		return nil, err
	}
	var out []Symbol
	root, _ := filepath.Abs(filepath.Clean(workspaceRoot))
	for _, rel := range paths {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		lang := langForPath(rel)
		if lang == "" {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(rel))
		syms, err := scanFile(full, rel, lang)
		if err != nil {
			continue
		}
		for _, s := range syms {
			if q == "" || strings.Contains(strings.ToLower(s.Name), q) {
				out = append(out, s)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func langForPath(p string) string {
	ext := strings.ToLower(filepath.Ext(p))
	if l, ok := extLang[ext]; ok {
		return l
	}
	return ""
}

func scanFile(full, rel, lang string) ([]Symbol, error) {
	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Symbol
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		var name, kind string
		switch lang {
		case "go":
			if m := regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?(\w+)`).FindStringSubmatch(line); len(m) > 1 {
				name, kind = m[1], "function"
			} else if m := regexp.MustCompile(`^type\s+(\w+)`).FindStringSubmatch(line); len(m) > 1 {
				name, kind = m[1], "type"
			}
		case "typescript", "javascript":
			if m := tsSym.FindStringSubmatch(line); len(m) > 0 {
				for i := 1; i < len(m); i++ {
					if m[i] != "" {
						name = m[i]
						if strings.Contains(line, "class") {
							kind = "class"
						} else if strings.Contains(line, "interface") {
							kind = "interface"
						} else {
							kind = "function"
						}
						break
					}
				}
			}
		}
		if name != "" {
			out = append(out, Symbol{Name: name, Path: rel, Line: lineNo, Kind: kind, Language: lang})
		}
	}
	return out, sc.Err()
}
