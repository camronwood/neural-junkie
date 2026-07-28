package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/repo"
)

const stackManifestMaxWalkFiles = 200

// StackManifest summarizes detected project stack for implementation grounding.
type StackManifest struct {
	WorkspaceRoot   string
	PackageName     string // from package.json / Cargo.toml / go.mod when available
	HasReact        bool
	HasVue          bool
	HasVite         bool
	HasTauri        bool
	HasTailwind     bool
	HasGo           bool
	HasRust         bool
	HasPython       bool
	TailwindConfig  string // relative path, e.g. tailwind.config.js
	EntryPoint      string // relative path, e.g. src/App.tsx
	TsConfig        bool
	ExtTSX          int
	ExtJSX          int
	ExtVue          int
}

// HasEntryPoint reports whether a primary app entry file was detected.
func (m *StackManifest) HasEntryPoint() bool {
	return m != nil && strings.TrimSpace(m.EntryPoint) != ""
}

// DetectStackManifest builds a stack summary from workspace root files.
func DetectStackManifest(workspaceRoot string) *StackManifest {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return nil
	}
	m := &StackManifest{WorkspaceRoot: workspaceRoot}
	if _, err := os.Stat(filepath.Join(workspaceRoot, "go.mod")); err == nil {
		m.HasGo = true
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, "Cargo.toml")); err == nil {
		m.HasRust = true
	} else if workspaceHasRustSources(workspaceRoot) {
		m.HasRust = true
	}
	for _, pyMarker := range []string{"pyproject.toml", "setup.py", "requirements.txt"} {
		if _, err := os.Stat(filepath.Join(workspaceRoot, pyMarker)); err == nil {
			m.HasPython = true
			break
		}
	}
	m.readPackageJSON(workspaceRoot)
	if m.PackageName == "" {
		m.PackageName = readGoModuleName(workspaceRoot)
	}
	if m.PackageName == "" {
		m.PackageName = readCargoPackageName(workspaceRoot)
	}
	m.locateTailwindConfig(workspaceRoot)
	m.locateEntryPoint(workspaceRoot)
	if _, err := os.Stat(filepath.Join(workspaceRoot, "tsconfig.json")); err == nil {
		m.TsConfig = true
	}
	m.countExtensions(workspaceRoot)
	return m
}

func (m *StackManifest) readPackageJSON(root string) {
	b, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return
	}
	var pkg struct {
		Name            string            `json:"name"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return
	}
	if name := strings.TrimSpace(pkg.Name); name != "" {
		m.PackageName = name
	}
	all := make(map[string]string)
	for k, v := range pkg.Dependencies {
		all[k] = v
	}
	for k, v := range pkg.DevDependencies {
		all[k] = v
	}
	for name := range all {
		lower := strings.ToLower(name)
		switch {
		case lower == "react" || strings.HasPrefix(lower, "react-"):
			m.HasReact = true
		case lower == "vue" || strings.HasPrefix(lower, "vue-"):
			m.HasVue = true
		case lower == "vite" || strings.HasPrefix(lower, "@vitejs/"):
			m.HasVite = true
		case strings.HasPrefix(lower, "@tauri-apps/"):
			m.HasTauri = true
		case lower == "tailwindcss":
			m.HasTailwind = true
		}
	}
}

func readGoModuleName(root string) string {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			mod := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if mod == "" {
				return ""
			}
			if i := strings.LastIndex(mod, "/"); i >= 0 && i+1 < len(mod) {
				return mod[i+1:]
			}
			return mod
		}
	}
	return ""
}

func readCargoPackageName(root string) string {
	b, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil {
		return ""
	}
	inPackage := false
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") {
			inPackage = trim == "[package]"
			continue
		}
		if !inPackage {
			continue
		}
		if strings.HasPrefix(trim, "name") {
			parts := strings.SplitN(trim, "=", 2)
			if len(parts) != 2 {
				continue
			}
			name := strings.TrimSpace(parts[1])
			name = strings.Trim(name, `"'`)
			return strings.TrimSpace(name)
		}
	}
	return ""
}

func (m *StackManifest) locateTailwindConfig(root string) {
	candidates := []string{
		"tailwind.config.js",
		"tailwind.config.ts",
		"tailwind.config.mjs",
		"tailwind.config.cjs",
		"src-tauri/tailwind.config.js",
		"src-tauri/tailwind.config.ts",
	}
	for _, rel := range candidates {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			m.TailwindConfig = rel
			m.HasTailwind = true
			return
		}
	}
}

func (m *StackManifest) locateEntryPoint(root string) {
	candidates := []string{
		"src/App.tsx",
		"src/App.jsx",
		"src/App.vue",
		"src/main.tsx",
		"src/main.ts",
		"src/main.jsx",
		"src/main.js",
		"index.html",
	}
	for _, rel := range candidates {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			m.EntryPoint = rel
			return
		}
	}
}

func (m *StackManifest) countExtensions(root string) {
	walked := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "node_modules" || base == ".git" || base == "dist" || base == "target" || base == repo.ScenarioBaselineDir {
				return filepath.SkipDir
			}
			return nil
		}
		walked++
		if walked > stackManifestMaxWalkFiles {
			return filepath.SkipAll
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".tsx":
			m.ExtTSX++
		case ".jsx":
			m.ExtJSX++
		case ".vue":
			m.ExtVue++
		}
		return nil
	})
}

// FormatPromptBlock returns a compact manifest block for the implementation prompt.
func (m *StackManifest) FormatPromptBlock() string {
	if m == nil {
		return ""
	}
	var parts []string
	if m.HasReact {
		parts = append(parts, "React")
	}
	if m.HasVue {
		parts = append(parts, "Vue")
	}
	if m.HasVite {
		parts = append(parts, "Vite")
	}
	if m.HasTauri {
		parts = append(parts, "Tauri")
	}
	if m.HasTailwind {
		parts = append(parts, "Tailwind")
	}
	if m.HasRust {
		parts = append(parts, "Rust")
	}
	if m.HasGo {
		parts = append(parts, "Go")
	}
	stack := "unknown"
	if len(parts) > 0 {
		stack = strings.Join(parts, " + ")
	}
	var b strings.Builder
	b.WriteString("\n=== STACK MANIFEST ===\n")
	b.WriteString(fmt.Sprintf("Stack: %s\n", stack))
	if m.TailwindConfig != "" {
		b.WriteString(fmt.Sprintf("Tailwind: %s\n", m.TailwindConfig))
	}
	if m.EntryPoint != "" {
		b.WriteString(fmt.Sprintf("Entry: %s\n", m.EntryPoint))
	}
	if m.TsConfig {
		b.WriteString("TypeScript: tsconfig.json present\n")
	}
	if m.ExtTSX+m.ExtJSX+m.ExtVue > 0 {
		b.WriteString(fmt.Sprintf("Extensions: tsx=%d, jsx=%d, vue=%d\n", m.ExtTSX, m.ExtJSX, m.ExtVue))
	}
	b.WriteString("Use ONLY paths that match this stack. Do not invent files for a different framework.\n")
	if m.HasRust {
		b.WriteString("Rust: only import crates declared in Cargo.toml [dependencies]. When [dependencies] is empty, use std only — do not `use rand` unless you add it to Cargo.toml.\n")
		b.WriteString("Derive Copy+Clone on small enums (Suit, Rank) moved in loops/iterators.\n")
	}
	b.WriteString("Do NOT use src-tauri/tailwind.config.js when Tailwind is listed at repo root.\n")
	if hint := DetectEntryConflicts(m.WorkspaceRoot, m); hint != "" {
		b.WriteString(hint)
		b.WriteString("\n")
	}
	b.WriteString("=== END STACK MANIFEST ===\n\n")
	return b.String()
}

// ImplementationSeedPaths returns stack-appropriate files to preload for implementation turns.
// Used for all agent types — not limited to frontend/backend specialists.
func (m *StackManifest) ImplementationSeedPaths() []string {
	if m == nil {
		return nil
	}
	root := strings.TrimSpace(m.WorkspaceRoot)
	if root == "" {
		return nil
	}
	var out []string
	add := func(rel string) {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			return
		}
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			out = append(out, rel)
		}
	}

	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		for _, p := range []string{"go.mod", "main.go", "cmd/main.go", "cmd/server/main.go", "package.json"} {
			add(p)
		}
		return out
	}
	if _, err := os.Stat(filepath.Join(root, "Cargo.toml")); err == nil {
		for _, p := range []string{"Cargo.toml", "src/main.rs", "src/lib.rs"} {
			add(p)
		}
		return out
	}

	add("package.json")
	add("index.html")
	for _, p := range []string{"vite.config.ts", "vite.config.js", "vite.config.mjs"} {
		add(p)
	}
	add("postcss.config.js")
	if m.TailwindConfig != "" {
		add(m.TailwindConfig)
	}
	for _, p := range []string{
		"src/main.tsx", "src/main.ts", "src/main.jsx", "src/main.js",
		"src/App.js", "src/App.jsx", "src/App.tsx",
		"src/index.css",
	} {
		add(p)
	}
	if m.HasTauri {
		for _, p := range []string{
			"Makefile",
			"scripts/start-all.sh",
			"src-tauri/tauri.conf.json",
			"src-tauri/Cargo.toml",
		} {
			add(p)
		}
	}
	return out
}

// DefaultReproCommands returns stack-appropriate non-interactive repro commands (fastest first).
func (m *StackManifest) DefaultReproCommands(userText string) []string {
	if m == nil {
		return nil
	}
	root := strings.TrimSpace(m.WorkspaceRoot)
	if root == "" {
		return nil
	}
	lower := strings.ToLower(strings.TrimSpace(userText))
	mentionTest := strings.Contains(lower, "test") || strings.Contains(lower, "pytest")

	if m.HasGo || fileExists(filepath.Join(root, "go.mod")) {
		if mentionTest {
			return []string{"go test ./..."}
		}
		return []string{"go build ./..."}
	}
	if m.HasRust || fileExists(filepath.Join(root, "Cargo.toml")) {
		if mentionTest {
			return []string{"cargo test"}
		}
		return []string{"cargo build"}
	}
	if m.HasPython {
		return []string{"python -m pytest -q"}
	}
	if fileExists(filepath.Join(root, "package.json")) {
		return nodeDefaultReproCommands(root)
	}
	if fileExists(filepath.Join(root, "Makefile")) {
		if makefileHasTarget(root, "test") && mentionTest {
			return []string{"make test"}
		}
		if makefileHasTarget(root, "build") {
			return []string{"make build"}
		}
	}
	return nil
}

func nodeDefaultReproCommands(wsPath string) []string {
	var cmds []string
	if npmScriptExists(wsPath, "build") {
		cmds = append(cmds, "npm run build")
	} else if cmd := shared.TypeScriptCheckShellCommand(wsPath); cmd != "" {
		cmds = append(cmds, cmd)
	} else if npmScriptExists(wsPath, "typecheck") {
		cmds = append(cmds, "npm run typecheck")
	}
	if npmScriptExists(wsPath, "test") {
		cmds = append(cmds, "npm test")
	}
	return cmds
}

func makefileHasTarget(wsPath, target string) bool {
	b, err := os.ReadFile(filepath.Join(wsPath, "Makefile"))
	if err != nil {
		return false
	}
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(target) + `\s*:`)
	return re.Match(b)
}

// DetectEntryConflicts reports App.js/App.tsx (or similar) resolution issues for the prompt.
func DetectEntryConflicts(workspaceRoot string, manifest *StackManifest) string {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" || manifest == nil || !manifest.HasReact {
		return ""
	}
	appJS := filepath.Join(workspaceRoot, "src", "App.js")
	appTSX := filepath.Join(workspaceRoot, "src", "App.tsx")
	if _, err := os.Stat(appJS); err != nil {
		return ""
	}
	if _, err := os.Stat(appTSX); err != nil {
		return ""
	}
	b, err := os.ReadFile(appJS)
	if err != nil {
		return ""
	}
	if !LooksLikeCorruptSourceContent(string(b)) {
		return ""
	}
	mainEntry := ""
	for _, p := range []string{"src/main.tsx", "src/main.ts", "src/main.jsx", "src/main.js"} {
		if _, err := os.Stat(filepath.Join(workspaceRoot, p)); err == nil {
			mainEntry = p
			break
		}
	}
	var hint strings.Builder
	hint.WriteString("Entry conflict: src/App.js and src/App.tsx both exist; src/App.js has invalid content. ")
	hint.WriteString("Imports like ./App resolve to App.js first. ")
	if mainEntry != "" {
		hint.WriteString(fmt.Sprintf("Delete or fix src/App.js, or change the import in %s to ./App.tsx. ", mainEntry))
	} else {
		hint.WriteString("Delete or fix src/App.js before editing App.tsx. ")
	}
	return strings.TrimSpace(hint.String())
}

// FormatRepairHints returns actionable path hints after preflight failure.
func (m *StackManifest) FormatRepairHints() string {
	if m == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nUse these paths from the stack manifest:\n")
	if m.TailwindConfig != "" {
		b.WriteString(fmt.Sprintf("- Tailwind: edit %q (repo root — NOT under src-tauri/)\n", m.TailwindConfig))
	}
	if m.EntryPoint != "" {
		b.WriteString(fmt.Sprintf("- App entry: %q\n", m.EntryPoint))
	}
	if m.HasReact && m.ExtVue == 0 {
		b.WriteString("- Use .tsx components under src/components/ (no .vue files in this repo)\n")
	}
	if m.HasTauri {
		b.WriteString("- Tauri boot: Makefile, scripts/start-all.sh, src-tauri/tauri.conf.json; vite.config at repo root\n")
		b.WriteString("- devPath in tauri.conf.json must match the Vite port in vite.config.js/ts\n")
	}
	return b.String()
}
