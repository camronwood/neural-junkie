package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectStackManifest_reactTailwindLayout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.2.0","@tauri-apps/api":"^1.5.3"},"devDependencies":{"vite":"^5.0.8","tailwindcss":"^3.4.0"}}`)
	writeFile(t, dir, "tailwind.config.js", "export default { content: ['./src/**/*.{js,ts,jsx,tsx}'] }\n")
	writeFile(t, dir, "tsconfig.json", `{}`)
	writeFile(t, dir, "src/App.tsx", "export default function App() { return null; }\n")
	writeFile(t, dir, "src/main.tsx", "import App from './App';\n")

	m := DetectStackManifest(dir)
	if m == nil {
		t.Fatal("expected manifest")
	}
	if !m.HasReact || !m.HasTailwind || !m.HasTauri || !m.HasVite {
		t.Fatalf("unexpected flags: react=%v tailwind=%v tauri=%v vite=%v", m.HasReact, m.HasTailwind, m.HasTauri, m.HasVite)
	}
	if m.TailwindConfig != "tailwind.config.js" {
		t.Fatalf("tailwind config: got %q", m.TailwindConfig)
	}
	if m.EntryPoint != "src/App.tsx" {
		t.Fatalf("entry: got %q", m.EntryPoint)
	}
	if m.ExtTSX != 2 {
		t.Fatalf("tsx count: got %d", m.ExtTSX)
	}
	if m.ExtVue != 0 {
		t.Fatalf("vue count: got %d", m.ExtVue)
	}
	block := m.FormatPromptBlock()
	if !strings.Contains(block, "src/App.tsx") {
		t.Fatalf("prompt block missing entry: %q", block)
	}
}

func TestStackManifest_ImplementationSeedPaths_includesAppJS(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"},"devDependencies":{"vite":"^5"}}`)
	writeFile(t, dir, "src/App.tsx", "export default function App() {}\n")
	writeFile(t, dir, "src/App.js", "diff --git a/foo b/foo\n")
	writeFile(t, dir, "src/main.tsx", "import App from './App'\n")

	m := DetectStackManifest(dir)
	paths := m.ImplementationSeedPaths()
	foundJS := false
	for _, p := range paths {
		if p == "src/App.js" {
			foundJS = true
		}
	}
	if !foundJS {
		t.Fatalf("expected src/App.js in implementation seeds, got %v", paths)
	}
}

func TestDetectEntryConflicts_appJSAndTSX(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18"}}`)
	writeFile(t, dir, "src/App.tsx", "export default function App() {}\n")
	writeFile(t, dir, "src/App.js", "diff --git a/tailwind.config.js b/tailwind.config.js\n")
	writeFile(t, dir, "src/main.tsx", "import App from './App'\n")

	m := DetectStackManifest(dir)
	hint := DetectEntryConflicts(dir, m)
	if hint == "" {
		t.Fatal("expected entry conflict hint")
	}
	if !strings.Contains(hint, "src/App.js") {
		t.Fatalf("hint missing App.js: %q", hint)
	}
	block := m.FormatPromptBlock()
	if !strings.Contains(block, "Entry conflict") {
		t.Fatalf("prompt block missing conflict: %q", block)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
