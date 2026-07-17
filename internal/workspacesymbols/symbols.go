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
		".py": "python",
	}
)

var (
	rsSym = regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+(\w+)|^\s*(?:pub\s+)?struct\s+(\w+)|^\s*(?:pub\s+)?enum\s+(\w+)|^\s*(?:pub\s+)?trait\s+(\w+)|^\s*(?:pub\s+)?type\s+(\w+)`)
	pySym = regexp.MustCompile(`^\s*(?:async\s+)?def\s+(\w+)|^\s*class\s+(\w+)`)
)

// Search finds symbols whose name contains q (case-insensitive). Uses disk-backed index when possible.
func Search(ctx context.Context, workspaceRoot, q string, limit int) ([]Symbol, error) {
	return SearchIndexed(ctx, workspaceRoot, q, "", limit)
}

// DefinitionLines returns sorted 1-based line numbers where definitions start in content.
// Used by codeindex chunking so embeddings align with graph symbol units.
func DefinitionLines(relPath, content string) []int {
	lang := langForPath(relPath)
	if lang == "" {
		return nil
	}
	syms, err := scanContent(relPath, lang, content)
	if err != nil || len(syms) == 0 {
		return nil
	}
	seen := map[int]bool{}
	var lines []int
	for _, s := range syms {
		if s.Line <= 0 || seen[s.Line] {
			continue
		}
		seen[s.Line] = true
		lines = append(lines, s.Line)
	}
	sort.Ints(lines)
	return lines
}

func langForPath(p string) string {
	ext := strings.ToLower(filepath.Ext(p))
	if l, ok := extLang[ext]; ok {
		return l
	}
	return ""
}

func scanFile(full, rel, lang string) ([]Symbol, error) {
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	return scanContent(rel, lang, string(data))
}

func scanContent(rel, lang, content string) ([]Symbol, error) {
	if treeSitterAvailable() {
		if syms, tsErr := scanFileTreeSitter(context.Background(), rel, content); tsErr == nil && len(syms) > 0 {
			for i := range syms {
				syms[i].Language = lang
			}
			return syms, nil
		}
	}
	f := strings.NewReader(content)
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
		case "rust":
			if m := rsSym.FindStringSubmatch(line); len(m) > 0 {
				for i := 1; i < len(m); i++ {
					if m[i] != "" {
						name = m[i]
						if strings.Contains(line, "struct") {
							kind = "struct"
						} else if strings.Contains(line, "enum") {
							kind = "enum"
						} else if strings.Contains(line, "trait") {
							kind = "trait"
						} else {
							kind = "function"
						}
						break
					}
				}
			}
		case "python":
			if m := pySym.FindStringSubmatch(line); len(m) > 0 {
				for i := 1; i < len(m); i++ {
					if m[i] != "" {
						name = m[i]
						if strings.Contains(line, "class") {
							kind = "class"
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
