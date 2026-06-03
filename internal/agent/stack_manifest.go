package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const stackManifestMaxWalkFiles = 200

// StackManifest summarizes detected project stack for implementation grounding.
type StackManifest struct {
	WorkspaceRoot   string
	HasReact        bool
	HasVue          bool
	HasVite         bool
	HasTauri        bool
	HasTailwind     bool
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
	m.readPackageJSON(workspaceRoot)
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
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return
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
			if base == "node_modules" || base == ".git" || base == "dist" || base == "target" {
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
	b.WriteString("Do NOT use src-tauri/tailwind.config.js when Tailwind is listed at repo root.\n")
	b.WriteString("=== END STACK MANIFEST ===\n\n")
	return b.String()
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
	return b.String()
}
