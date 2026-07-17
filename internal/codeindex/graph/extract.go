package graph

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type rawImport struct {
	Target string
	Line   int
	Raw    string
}

var (
	goImportBlock = regexp.MustCompile(`(?m)^import\s*\(([\s\S]*?)\)`)
	goImportSingle = regexp.MustCompile(`(?m)^import\s+(?:[a-zA-Z0-9_]+\s+)?"([^"]+)"`)
	goImportLine   = regexp.MustCompile(`^\s*(?:[a-zA-Z0-9_]+\s+)?"([^"]+)"`)
	tsImportFrom   = regexp.MustCompile(`(?m)(?:import|export)\s+(?:type\s+)?(?:[\w*{}\s,]+\s+from\s+)?['"]([^'"]+)['"]`)
	tsRequire      = regexp.MustCompile(`(?m)require\(\s*['"]([^'"]+)['"]\s*\)`)
	pyImport       = regexp.MustCompile(`(?m)^\s*(?:from\s+([\w.]+)\s+import|import\s+([\w.]+))`)
	rsUse          = regexp.MustCompile(`(?m)^\s*(?:pub\s+)?use\s+((?:crate|super|self)::)?([\w:]+)`)
	rsMod          = regexp.MustCompile(`(?m)^\s*(?:pub\s+)?mod\s+(\w+)`)
)

func extractImports(relPath, content string) []rawImport {
	ext := strings.ToLower(filepath.Ext(relPath))
	switch ext {
	case ".go":
		return extractGoImports(content)
	case ".ts", ".tsx", ".js", ".jsx":
		return extractTSImports(content)
	case ".py":
		return extractPyImports(content)
	case ".rs":
		return extractRustImports(content)
	default:
		return nil
	}
}

func extractGoImports(content string) []rawImport {
	var out []rawImport
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if m := goImportSingle.FindStringSubmatch(line); len(m) > 1 {
			out = append(out, rawImport{Target: m[1], Line: i + 1, Raw: line})
		}
	}
	for _, block := range goImportBlock.FindAllStringSubmatch(content, -1) {
		if len(block) < 2 {
			continue
		}
		blockStart := strings.Index(content, block[0])
		prefix := content[:blockStart]
		baseLine := strings.Count(prefix, "\n") + 1
		for j, line := range strings.Split(block[1], "\n") {
			if m := goImportLine.FindStringSubmatch(line); len(m) > 1 {
				out = append(out, rawImport{Target: m[1], Line: baseLine + j + 1, Raw: line})
			}
		}
	}
	return dedupeImports(out)
}

func extractTSImports(content string) []rawImport {
	var out []rawImport
	for _, re := range []*regexp.Regexp{tsImportFrom, tsRequire} {
		for _, m := range re.FindAllStringSubmatchIndex(content, -1) {
			if len(m) < 4 {
				continue
			}
			target := content[m[2]:m[3]]
			line := strings.Count(content[:m[0]], "\n") + 1
			out = append(out, rawImport{Target: target, Line: line, Raw: target})
		}
	}
	return dedupeImports(out)
}

func extractPyImports(content string) []rawImport {
	var out []rawImport
	for _, m := range pyImport.FindAllStringSubmatchIndex(content, -1) {
		line := strings.Count(content[:m[0]], "\n") + 1
		target := ""
		if m[2] >= 0 {
			target = content[m[2]:m[3]]
		} else if m[4] >= 0 {
			target = content[m[4]:m[5]]
		}
		if target != "" {
			out = append(out, rawImport{Target: target, Line: line, Raw: target})
		}
	}
	return dedupeImports(out)
}

func extractRustImports(content string) []rawImport {
	var out []rawImport
	for _, m := range rsUse.FindAllStringSubmatchIndex(content, -1) {
		line := strings.Count(content[:m[0]], "\n") + 1
		prefix := ""
		if m[2] >= 0 {
			prefix = content[m[2]:m[3]]
		}
		path := content[m[4]:m[5]]
		target := prefix + path
		out = append(out, rawImport{Target: target, Line: line, Raw: target})
	}
	for _, m := range rsMod.FindAllStringSubmatchIndex(content, -1) {
		line := strings.Count(content[:m[0]], "\n") + 1
		name := content[m[2]:m[3]]
		out = append(out, rawImport{Target: "mod::" + name, Line: line, Raw: name})
	}
	return dedupeImports(out)
}

func dedupeImports(in []rawImport) []rawImport {
	seen := map[string]bool{}
	var out []rawImport
	for _, im := range in {
		key := im.Target + "@" + itoa(im.Line)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, im)
	}
	return out
}

func packageCommunity(relPath string) string {
	rel := filepath.ToSlash(relPath)
	dir := path.Dir(rel)
	if dir == "." || dir == "" {
		return "root"
	}
	parts := strings.Split(dir, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return parts[0]
}

func communityColor(id string) string {
	palette := []string{
		"#e94560", "#0f3460", "#533483", "#16a34a", "#ca8a04",
		"#0891b2", "#db2777", "#ea580c", "#4f46e5", "#059669",
	}
	h := 0
	for _, c := range id {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return palette[h%len(palette)]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// resolveImportTarget maps an import string to a file node id within the workspace when possible.
func resolveImportTarget(fromRel, target string, fileIDs map[string]string, moduleRoots []string) (string, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", false
	}
	ext := strings.ToLower(filepath.Ext(fromRel))

	switch ext {
	case ".ts", ".tsx", ".js", ".jsx":
		if strings.HasPrefix(target, ".") {
			base := path.Clean(path.Join(path.Dir(filepath.ToSlash(fromRel)), target))
			candidates := []string{
				base,
				base + ".ts", base + ".tsx", base + ".js", base + ".jsx",
				base + "/index.ts", base + "/index.tsx", base + "/index.js",
			}
			for _, c := range candidates {
				if id, ok := fileIDs[c]; ok {
					return id, true
				}
			}
		}
	case ".py":
		modPath := strings.ReplaceAll(target, ".", "/")
		candidates := []string{modPath + ".py", modPath + "/__init__.py"}
		for _, c := range candidates {
			if id, ok := fileIDs[c]; ok {
				return id, true
			}
		}
	case ".go":
		for _, root := range moduleRoots {
			if root != "" && strings.HasPrefix(target, root+"/") {
				rel := strings.TrimPrefix(target, root+"/")
				// Prefer package directory match via any file under that dir.
				for p, id := range fileIDs {
					if path.Dir(p) == rel || strings.HasPrefix(p, rel+"/") {
						return id, true
					}
				}
			}
		}
	case ".rs":
		if strings.HasPrefix(target, "mod::") {
			name := strings.TrimPrefix(target, "mod::")
			base := path.Join(path.Dir(filepath.ToSlash(fromRel)), name+".rs")
			if id, ok := fileIDs[base]; ok {
				return id, true
			}
			base = path.Join(path.Dir(filepath.ToSlash(fromRel)), name, "mod.rs")
			if id, ok := fileIDs[base]; ok {
				return id, true
			}
		}
		if strings.HasPrefix(target, "crate::") {
			rest := strings.TrimPrefix(target, "crate::")
			parts := strings.Split(rest, "::")
			if len(parts) > 0 {
				cand := "src/" + parts[0] + ".rs"
				if id, ok := fileIDs[cand]; ok {
					return id, true
				}
			}
		}
	}
	return "", false
}

func readGoModulePath(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
