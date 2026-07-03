package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
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

func TestTryEarlyGoMainFixtureFix_HelloWorld(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "core", "sample"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "core", "sample", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"},
		"please implement a HelloWorld function in core/sample/main.go and call it from main")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	if !ag.tryEarlyGoMainFixtureFix(context.Background(), msg, dir, state) {
		t.Fatal("expected early go main fix")
	}
	if state.ProposedCount != 1 {
		t.Fatalf("ProposedCount = %d, want 1", state.ProposedCount)
	}
	if len(state.FilesChanged) != 1 || state.FilesChanged[0] != "core/sample/main.go" {
		t.Fatalf("FilesChanged = %v", state.FilesChanged)
	}
}

func TestTryEarlyGoMainFixtureFix_alreadySatisfied(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "core", "sample"), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "package main\n\nfunc HelloWorld() {}\n\nfunc main() { HelloWorld() }\n"
	if err := os.WriteFile(filepath.Join(dir, "core", "sample", "main.go"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"},
		"please implement a HelloWorld function in core/sample/main.go and call it from main")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	if !ag.tryEarlyGoMainFixtureFix(context.Background(), msg, dir, state) {
		t.Fatal("expected satisfied early go main path")
	}
	if state.ProposedCount != 0 {
		t.Fatalf("ProposedCount = %d, want 0 when already satisfied", state.ProposedCount)
	}
}

func TestTryEarlyThemeCSSFix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"},
		"implement a simple theme.css file with light and dark variables under src/theme.css")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	if !ag.tryEarlyThemeCSSFix(context.Background(), msg, dir, state) {
		t.Fatal("expected early theme.css fix")
	}
	if state.ProposedCount != 1 {
		t.Fatalf("ProposedCount = %d, want 1", state.ProposedCount)
	}
	if len(state.FilesChanged) != 1 || state.FilesChanged[0] != "src/theme.css" {
		t.Fatalf("FilesChanged = %v", state.FilesChanged)
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
