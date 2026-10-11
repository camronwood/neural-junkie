package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	rustUnresolvedImportRE = regexp.MustCompile("(?i)error\\[e0432\\][^\\n]*`([a-z][a-z0-9_-]*)`")
	rustUndeclaredCrateRE  = regexp.MustCompile("(?i)error\\[e0433\\][^\\n]*`([a-z][a-z0-9_-]*)`")
	rustCannotFindCrateRE  = regexp.MustCompile("(?i)(?:cannot find crate|could not find) `([a-z][a-z0-9_-]*)`")
	invalidRustCrateNameRE = regexp.MustCompile(`[^a-z0-9_-]+`)
	rustDebugNotImplForRE  = regexp.MustCompile("(?i)trait `Debug` is not implemented for `([A-Za-z_][A-Za-z0-9_]*)`")
	rustDebugAnnotateRE    = regexp.MustCompile("(?i)annotating `([A-Za-z_][A-Za-z0-9_]*)` with `#\\[derive\\(Debug\\)\\]`")
	rustPartialEqNotImplRE  = regexp.MustCompile("(?i)trait `PartialEq` is not implemented for `([A-Za-z_][A-Za-z0-9_]*)`")
	rustPartialEqBoundRE    = regexp.MustCompile("(?i)`([A-Za-z_][A-Za-z0-9_]*): PartialEq` is not satisfied")
	rustPartialEqAnnotateRE = regexp.MustCompile("(?i)annotating `([A-Za-z_][A-Za-z0-9_]*)` with `#\\[derive\\(PartialEq\\)\\]`")
	rustEqNotImplForRE      = regexp.MustCompile("(?i)trait `Eq` is not implemented for `([A-Za-z_][A-Za-z0-9_]*)`")
	rustCloneHintTypeRE    = regexp.MustCompile("(?i)if `([A-Za-z_][A-Za-z0-9_]*)` implemented `Clone`")
	rustCopyHintTypeRE     = regexp.MustCompile("(?i)if `([A-Za-z_][A-Za-z0-9_]*)` implemented `Copy`")
	rustMoveValueTypeRE    = regexp.MustCompile("(?i)move occurs because `[^`]*` has type `([A-Za-z_][A-Za-z0-9_]*)`")
)

var rustStdPseudoCrates = map[string]bool{
	"std": true, "core": true, "alloc": true, "test": true, "proc_macro": true,
}

func extractMissingRustCrates(output string) []string {
	seen := make(map[string]bool)
	var crates []string
	add := func(crate string) {
		crate = strings.TrimSpace(crate)
		if crate == "" || rustStdPseudoCrates[crate] {
			return
		}
		if seen[crate] {
			return
		}
		seen[crate] = true
		crates = append(crates, crate)
	}
	for _, re := range []*regexp.Regexp{rustUnresolvedImportRE, rustUndeclaredCrateRE, rustCannotFindCrateRE} {
		for _, m := range re.FindAllStringSubmatch(output, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}
	return crates
}


func defaultRustCrateVersion(crate string) string {
	switch crate {
	case "rand":
		return "0.8"
	case "serde", "serde_json":
		return "1.0"
	case "tokio":
		return "1"
	default:
		return "1"
	}
}


func cargoTomlHasDependency(content, crate string) bool {
	crate = strings.TrimSpace(crate)
	if crate == "" {
		return false
	}
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(crate) + `\s*=`)
	return re.MatchString(content)
}


func addMissingDependencyToCargoToml(existing []byte, crate string) (string, bool) {
	crate = strings.TrimSpace(crate)
	if crate == "" {
		return "", false
	}
	content := string(existing)
	if cargoTomlHasDependency(content, crate) {
		return "", false
	}
	depLine := crate + ` = "` + defaultRustCrateVersion(crate) + `"` + "\n"
	if idx := strings.Index(content, "[dependencies]"); idx >= 0 {
		insertAt := idx + len("[dependencies]")
		rest := content[insertAt:]
		switch {
		case strings.HasPrefix(rest, "\r\n"):
			insertAt += 2
		case strings.HasPrefix(rest, "\n"):
			insertAt += 1
		}
		prefix := content[:insertAt]
		suffix := content[insertAt:]
		if strings.TrimSpace(suffix) != "" && !strings.HasPrefix(suffix, "\n") && !strings.HasPrefix(suffix, "\r\n") {
			depLine = "\n" + depLine
		} else if !strings.HasSuffix(prefix, "\n") && !strings.HasSuffix(prefix, "\r\n") {
			depLine = "\n" + depLine
		}
		return prefix + depLine + suffix, true
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n[dependencies]\n" + depLine
	return content, true
}


func sanitizeRustCrateName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = invalidRustCrateNameRE.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-_")
	if name == "" {
		return "app"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "app-" + name
	}
	return name
}


func deriveCargoPackageName(wsPath string) string {
	base := filepath.Base(strings.TrimSpace(wsPath))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "app"
	}
	return sanitizeRustCrateName(base)
}


func minimalCargoTomlBody(packageName string) string {
	packageName = sanitizeRustCrateName(packageName)
	return "[package]\nname = \"" + packageName + "\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[dependencies]\n"
}


func workspaceHasRustSources(wsPath string) bool {
	wsPath = strings.TrimSpace(wsPath)
	if wsPath == "" {
		return false
	}
	for _, rel := range []string{"src/main.rs", "src/lib.rs"} {
		if _, err := os.Stat(filepath.Join(wsPath, rel)); err == nil {
			return true
		}
	}
	matches, err := filepath.Glob(filepath.Join(wsPath, "src", "*.rs"))
	if err != nil {
		return false
	}
	return len(matches) > 0
}


func workspaceMissingCargoToml(wsPath string) bool {
	if strings.TrimSpace(wsPath) == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(wsPath, "Cargo.toml"))
	return os.IsNotExist(err)
}


func messageImpliesRustGreenfield(content string, hintPaths ...string) bool {
	lower := strings.ToLower(content)
	for _, token := range []string{
		"cargo.toml", "src/main.rs", "src/lib.rs", "cargo build", "cargo run",
		"rust ", "using rust", " in rust", "with rust",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	for _, p := range hintPaths {
		p = normalizeFileChangeRelPath(p)
		if strings.HasSuffix(strings.ToLower(p), ".rs") {
			return true
		}
	}
	return false
}


func extractMissingRustDebugTypes(output string) []string {
	seen := make(map[string]bool)
	var types []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		types = append(types, name)
	}
	for _, re := range []*regexp.Regexp{rustDebugAnnotateRE, rustDebugNotImplForRE} {
		for _, m := range re.FindAllStringSubmatch(output, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}
	return types
}


func addDeriveDebugToRustSource(src, typeName string) (string, bool) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || src == "" {
		return "", false
	}
	declRE := regexp.MustCompile(`(?m)^([ \t]*)((?:pub(?:\([^)]*\))?\s+)?)(struct|enum)\s+` + regexp.QuoteMeta(typeName) + `\b`)
	loc := declRE.FindStringSubmatchIndex(src)
	if loc == nil {
		return "", false
	}
	lineStart := loc[0]
	indent := src[loc[2]:loc[3]]

	// Scan attribute lines immediately above the declaration.
	before := src[:lineStart]
	lines := strings.Split(before, "\n")
	i := len(lines) - 1
	if i >= 0 && lines[i] == "" {
		i--
	}
	for i >= 0 {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			break
		}
		if strings.HasPrefix(trimmed, "#[") || strings.HasPrefix(trimmed, "#![") {
			if strings.Contains(trimmed, "derive(") && strings.Contains(trimmed, "Debug") {
				return "", false
			}
			if strings.HasPrefix(trimmed, "#[derive(") && strings.HasSuffix(trimmed, ")]") && !strings.Contains(trimmed, "Debug") {
				inner := trimmed[len("#[derive(") : len(trimmed)-2]
				inner = strings.TrimSpace(inner)
				lead := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				if inner == "" {
					lines[i] = lead + "#[derive(Debug)]"
				} else {
					lines[i] = lead + "#[derive(" + inner + ", Debug)]"
				}
				newBefore := strings.Join(lines, "\n")
				if strings.HasSuffix(before, "\n") && !strings.HasSuffix(newBefore, "\n") {
					newBefore += "\n"
				}
				return newBefore + src[lineStart:], true
			}
			i--
			continue
		}
		break
	}
	insert := indent + "#[derive(Debug)]\n"
	return src[:lineStart] + insert + src[lineStart:], true
}


func extractMissingRustCopyCloneTypes(output string) []string {
	seen := make(map[string]bool)
	var types []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		types = append(types, name)
	}
	for _, re := range []*regexp.Regexp{rustCloneHintTypeRE, rustCopyHintTypeRE, rustMoveValueTypeRE} {
		for _, m := range re.FindAllStringSubmatch(output, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}
	return types
}


func addDeriveCopyCloneToRustSource(src, typeName string) (string, bool) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || src == "" {
		return "", false
	}
	declRE := regexp.MustCompile(`(?m)^([ \t]*)((?:pub(?:\([^)]*\))?\s+)?)(struct|enum)\s+` + regexp.QuoteMeta(typeName) + `\b`)
	loc := declRE.FindStringSubmatchIndex(src)
	if loc == nil {
		return "", false
	}
	lineStart := loc[0]
	indent := src[loc[2]:loc[3]]

	before := src[:lineStart]
	lines := strings.Split(before, "\n")
	i := len(lines) - 1
	if i >= 0 && lines[i] == "" {
		i--
	}
	for i >= 0 {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			break
		}
		if strings.HasPrefix(trimmed, "#[") || strings.HasPrefix(trimmed, "#![") {
			if strings.HasPrefix(trimmed, "#[derive(") && strings.HasSuffix(trimmed, ")]") {
				hasCopy := strings.Contains(trimmed, "Copy")
				hasClone := strings.Contains(trimmed, "Clone")
				if hasCopy && hasClone {
					return "", false
				}
				inner := trimmed[len("#[derive(") : len(trimmed)-2]
				inner = strings.TrimSpace(inner)
				parts := []string{}
				if inner != "" {
					parts = append(parts, inner)
				}
				if !hasCopy {
					parts = append(parts, "Copy")
				}
				if !hasClone {
					parts = append(parts, "Clone")
				}
				lead := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				lines[i] = lead + "#[derive(" + strings.Join(parts, ", ") + ")]"
				newBefore := strings.Join(lines, "\n")
				if strings.HasSuffix(before, "\n") && !strings.HasSuffix(newBefore, "\n") {
					newBefore += "\n"
				}
				return newBefore + src[lineStart:], true
			}
			i--
			continue
		}
		break
	}
	insert := indent + "#[derive(Copy, Clone)]\n"
	return src[:lineStart] + insert + src[lineStart:], true
}


func addDeriveTraitsToRustSource(src, typeName string, traits []string) (string, bool) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || src == "" || len(traits) == 0 {
		return "", false
	}
	declRE := regexp.MustCompile(`(?m)^([ \t]*)((?:pub(?:\([^)]*\))?\s+)?)(struct|enum)\s+` + regexp.QuoteMeta(typeName) + `\b`)
	loc := declRE.FindStringSubmatchIndex(src)
	if loc == nil {
		return "", false
	}
	lineStart := loc[0]
	indent := src[loc[2]:loc[3]]

	traitPresent := func(block string, trait string) bool {
		return regexp.MustCompile(`\b` + regexp.QuoteMeta(trait) + `\b`).MatchString(block)
	}
	allPresent := func(block string) bool {
		for _, tr := range traits {
			if !traitPresent(block, tr) {
				return false
			}
		}
		return true
	}
	missingFrom := func(block string) []string {
		var miss []string
		for _, tr := range traits {
			if !traitPresent(block, tr) {
				miss = append(miss, tr)
			}
		}
		return miss
	}

	// Scan attribute lines immediately above the declaration.
	before := src[:lineStart]
	lines := strings.Split(before, "\n")
	i := len(lines) - 1
	if i >= 0 && lines[i] == "" {
		i--
	}
	for i >= 0 {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			break
		}
		if strings.HasPrefix(trimmed, "#[") || strings.HasPrefix(trimmed, "#![") {
			if strings.HasPrefix(trimmed, "#[derive(") && strings.HasSuffix(trimmed, ")]") {
				inner := trimmed[len("#[derive(") : len(trimmed)-2]
				inner = strings.TrimSpace(inner)
				if allPresent(inner) {
					return "", false
				}
				miss := missingFrom(inner)
				if len(miss) == 0 {
					return "", false
				}
				lead := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
				joined := strings.Join(miss, ", ")
				if inner == "" {
					lines[i] = lead + "#[derive(" + joined + ")]"
				} else {
					lines[i] = lead + "#[derive(" + inner + ", " + joined + ")]"
				}
				newBefore := strings.Join(lines, "\n")
				if strings.HasSuffix(before, "\n") && !strings.HasSuffix(newBefore, "\n") {
					newBefore += "\n"
				}
				return newBefore + src[lineStart:], true
			}
			i--
			continue
		}
		break
	}
	insert := indent + "#[derive(" + strings.Join(traits, ", ") + ")]\n"
	return src[:lineStart] + insert + src[lineStart:], true
}


func extractMissingRustTraitTypes(output string, res []*regexp.Regexp) []string {
	seen := make(map[string]bool)
	var types []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		types = append(types, name)
	}
	for _, re := range res {
		for _, m := range re.FindAllStringSubmatch(output, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}
	return types
}


func extractMissingRustPartialEqTypes(output string) []string {
	types := extractMissingRustTraitTypes(output, []*regexp.Regexp{
		rustPartialEqAnnotateRE, rustPartialEqNotImplRE, rustPartialEqBoundRE, rustEqNotImplForRE,
	})
	// E0369 == without naming PartialEq still points at a type in backticks near the error.
	if len(types) == 0 && strings.Contains(strings.ToLower(output), "binary operation `==`") {
		re := regexp.MustCompile("(?i)cannot be applied to type `([A-Za-z_][A-Za-z0-9_]*)`")
		types = extractMissingRustTraitTypes(output, []*regexp.Regexp{re})
	}
	return types
}


func addDerivePartialEqToRustSource(src, typeName string) (string, bool) {
	return addDeriveTraitsToRustSource(src, typeName, []string{"PartialEq", "Eq"})
}


var (
	rustUseRandLineRE   = regexp.MustCompile(`(?m)^\s*use\s+rand(?:::[^;]+)?;\s*\n?`)
	rustShuffleCallRE   = regexp.MustCompile(`\.shuffle\s*\(\s*&mut\s+[A-Za-z_][A-Za-z0-9_]*\s*\)`)
	rustThreadRngCallRE = regexp.MustCompile(`thread_rng\s*\(\s*\)`)
)

func userPrefersStdOnlyRust(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	lower := strings.ToLower(msg.Content)
	markers := []string{
		"standard library only",
		"std only",
		"prefer the rust standard library",
		"avoid rand",
		"no rand",
		"stdlib only",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}


func rewriteRustSourceDropRand(src string) (string, bool) {
	if !strings.Contains(src, "rand") && !strings.Contains(src, ".shuffle(") {
		return "", false
	}
	out := src
	changed := false
	if rustUseRandLineRE.MatchString(out) {
		out = rustUseRandLineRE.ReplaceAllString(out, "")
		changed = true
	}
	recvShuffle := regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\.shuffle\s*\(\s*&mut\s+[A-Za-z_][A-Za-z0-9_]*\s*\)`)
	if recvShuffle.MatchString(out) {
		out = recvShuffle.ReplaceAllString(out, "nj_std_shuffle(&mut $1)")
		changed = true
	} else if rustShuffleCallRE.MatchString(out) {
		// Bare .shuffle(&mut rng) without a simple receiver — leave for model repair.
		return "", false
	}
	if rustThreadRngCallRE.MatchString(out) {
		out = regexp.MustCompile(`(?m)^\s*let\s+mut\s+\w+\s*=\s*thread_rng\s*\(\s*\)\s*;\s*\n?`).ReplaceAllString(out, "")
		out = regexp.MustCompile(`(?m)^\s*let\s+\w+\s*=\s*thread_rng\s*\(\s*\)\s*;\s*\n?`).ReplaceAllString(out, "")
		changed = true
	}
	if !changed {
		return "", false
	}
	if !strings.Contains(out, "fn nj_std_shuffle") {
		helper := "\nfn nj_std_shuffle<T>(items: &mut [T]) {\n" +
			"    let n = items.len();\n" +
			"    if n < 2 {\n" +
			"        return;\n" +
			"    }\n" +
			"    let seed = std::time::SystemTime::now()\n" +
			"        .duration_since(std::time::UNIX_EPOCH)\n" +
			"        .map(|d| d.as_nanos() as usize)\n" +
			"        .unwrap_or(1);\n" +
			"    for i in (1..n).rev() {\n" +
			"        let j = (seed + i * 2654435761) % (i + 1);\n" +
			"        items.swap(i, j);\n" +
			"    }\n" +
			"}\n"
		out = strings.TrimRight(out, "\n") + "\n" + helper
	}
	return out, true
}


func ensureRustRandShuffleImport(src string) (string, bool) {
	if !strings.Contains(src, ".shuffle(") {
		return "", false
	}
	if strings.Contains(src, "SliceRandom") {
		return "", false
	}
	insert := "use rand::seq::SliceRandom;\n"
	// Prefer after the last leading use/mod/#! line.
	lines := strings.Split(src, "\n")
	insertAt := 0
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#!") ||
			strings.HasPrefix(trim, "use ") || strings.HasPrefix(trim, "mod ") ||
			strings.HasPrefix(trim, "extern ") {
			insertAt = i + 1
			continue
		}
		break
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:insertAt]...)
	out = append(out, strings.TrimSuffix(insert, "\n"))
	out = append(out, lines[insertAt:]...)
	return strings.Join(out, "\n"), true
}


