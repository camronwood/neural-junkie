package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCodeFenceForPath_tailwind(t *testing.T) {
	t.Parallel()
	resp := "Here:\n```tsx\nexport default function App() {}\n```\n```js\nexport default { darkMode: 'class', content: ['./src/**/*'] }\n```"
	got := extractCodeFenceForPath(resp, "tailwind.config.js")
	if !strings.Contains(got, "darkMode") {
		t.Fatalf("want tailwind fence, got %q", got)
	}
}

func TestSynthesizeGoMainEdit_HelloWorld(t *testing.T) {
	t.Parallel()
	existing := "package main\n\nfunc main() {}\n"
	got, ok := synthesizeGoMainEdit("please implement HelloWorld in core/sample/main.go", existing)
	if !ok || !strings.Contains(got, "func HelloWorld") || !strings.Contains(got, "HelloWorld()") {
		t.Fatalf("got ok=%v body=%q", ok, got)
	}
}

func TestSynthesizeTailwindDarkMode(t *testing.T) {
	t.Parallel()
	existing := "export default {\n  content: ['./index.html'],\n}\n"
	got, ok := synthesizeTailwindDarkMode(existing)
	if !ok || !strings.Contains(got, "darkMode") {
		t.Fatalf("got ok=%v body=%q", ok, got)
	}
}

func TestSynthesizeTailwindDarkMode_CJS(t *testing.T) {
	t.Parallel()
	existing := "module.exports = {\n  theme: { extend: {} },\n}\n"
	got, ok := synthesizeTailwindDarkMode(existing)
	if !ok || !strings.Contains(got, "darkMode") {
		t.Fatalf("got ok=%v body=%q", ok, got)
	}
}

func TestSynthesizeGoMainEdit_PrintVersion(t *testing.T) {
	t.Parallel()
	existing := "package main\n\nfunc main() {}\n"
	got, ok := synthesizeGoMainEdit("implement a PrintVersion helper in core/sample/main.go", existing)
	if !ok || !strings.Contains(got, "func PrintVersion") {
		t.Fatalf("got ok=%v body=%q", ok, got)
	}
}

func TestSynthesizeGoMathEdit_fixtureBugs(t *testing.T) {
	t.Parallel()
	addBug := "package sample\n\nfunc Add(a, b int) int {\n\treturn a + b + 1\n}\n"
	got, ok := synthesizeGoMathEdit("fix Add in core/sample/math.go so go test passes", addBug, "core/sample/math.go")
	if !ok || strings.Contains(got, "a + b + 1") {
		t.Fatalf("Add fix: ok=%v body=%q", ok, got)
	}
	mulBug := "package sample\n\nfunc Multiply(a, b int) int {\n\treturn a + b\n}\n"
	got, ok = synthesizeGoMathEdit("fix Multiply in math.go", mulBug, "core/sample/math.go")
	if !ok || !strings.Contains(got, "return a * b") {
		t.Fatalf("Multiply fix: ok=%v body=%q", ok, got)
	}
}

func TestSynthesizeTypeScriptAppCompileFix_fixtureBug(t *testing.T) {
	t.Parallel()
	bug := `export default function App() {
  const brokenCount: number = "not-a-number";
  return <p>{brokenCount}</p>;
}
`
	got, ok := synthesizeTypeScriptAppCompileFix(
		"Fix the TypeScript compile error in src/App.tsx",
		bug,
		"src/App.tsx",
	)
	if !ok || strings.Contains(got, "not-a-number") {
		t.Fatalf("App.tsx fix: ok=%v body=%q", ok, got)
	}
}

func TestCorruptAppJSEntryConflict(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("package.json", `{"dependencies":{"react":"18.0.0"}}`)
	write("src/App.js", "diff --git a/tailwind.config.js b/tailwind.config.js\n")
	write("src/App.tsx", "export default function App() { return null }\n")
	write("src/main.tsx", "import App from './App'\n")
	manifest := DetectStackManifest(dir)
	if !corruptAppJSEntryConflict(dir, manifest) {
		t.Fatal("expected corrupt App.js entry conflict")
	}
}
