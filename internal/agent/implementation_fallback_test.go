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

func TestSynthesizeAppSidebarSubtitle(t *testing.T) {
	t.Parallel()
	existing := `<p className="text-sm text-slate-400">Sidebar</p>`
	got, ok := synthesizeAppSidebarSubtitle(existing)
	if !ok || !strings.Contains(got, "subtitle") {
		t.Fatalf("got ok=%v body=%q", ok, got)
	}
}

func TestPreferImplementationTargetPath_atFileOverDoNotModify(t *testing.T) {
	t.Parallel()
	user := "In @file:src/App.tsx ONLY add a subtitle. Do NOT modify tailwind.config.js."
	if got := preferImplementationTargetPath("", user, ""); got != "src/App.tsx" {
		t.Fatalf("got %q, want src/App.tsx", got)
	}
}

func TestTryEarlyScopedFileEdit_subtitle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	app := `import { useState } from "react";

export default function App() {
  return (
    <aside>
      <p className="text-sm text-slate-400">Sidebar</p>
    </aside>
  );
}
`
	writeMultifileTestFile(t, dir, "src/App.tsx", app)
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{StackManifest: DetectStackManifest(dir)}
	userContent := "@FrontendEngineer In @file:src/App.tsx ONLY add a short subtitle under the sidebar heading."
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"}, userContent)
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     "auto_apply_edits",
		"workspace_context": map[string]interface{}{
			"workspace_path":  dir,
			"unchanged_files": []interface{}{"tailwind.config.js", "package.json"},
		},
	}
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryEarlyScopedFileEdit(ctx, msg, dir, state) {
		t.Fatal("expected early scoped subtitle edit")
	}
	got, err := os.ReadFile(filepath.Join(dir, "src/App.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(string(got)), "subtitle") {
		t.Fatalf("expected subtitle on disk, got:\n%s", got)
	}
}

func TestTryEarlyThemeToggleFix_tailwindOnlyNeeded(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tailwind := `/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: { extend: {} },
  plugins: [],
};
`
	app := `import { useState } from "react";

export default function App() {
  const [theme, setTheme] = useState("dark");
  const toggleTheme = () => setTheme((t) => (t === "dark" ? "light" : "dark"));
  return (
    <aside>
      <button onClick={toggleTheme}>Toggle Theme</button>
    </aside>
  );
}
`
	writeMultifileTestFile(t, dir, "package.json", `{"dependencies":{"react":"^18"},"devDependencies":{"tailwindcss":"^3","vite":"^5"}}`)
	writeMultifileTestFile(t, dir, "tailwind.config.js", tailwind)
	writeMultifileTestFile(t, dir, "src/App.tsx", app)

	userContent := "implement light/dark theme toggle in the sidebar for this project"
	if implementationTargetSatisfied(dir, "tailwind.config.js", userContent) {
		t.Fatal("tailwind should be unsatisfied without darkMode")
	}
	if !implementationTargetSatisfied(dir, "src/App.tsx", userContent) {
		t.Fatal("App.tsx should already satisfy theme toggle")
	}

	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{StackManifest: DetectStackManifest(dir)}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"}, userContent)
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     "auto_apply_edits",
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryEarlyThemeToggleFix(ctx, msg, dir, state) {
		t.Fatal("expected early theme toggle fix for missing tailwind darkMode")
	}
	if state.ProposedCount != 1 {
		t.Fatalf("ProposedCount = %d, want 1", state.ProposedCount)
	}
	if len(state.FilesChanged) != 1 || state.FilesChanged[0] != "tailwind.config.js" {
		t.Fatalf("FilesChanged = %v", state.FilesChanged)
	}
	got, err := os.ReadFile(filepath.Join(dir, "tailwind.config.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "darkMode") {
		t.Fatalf("expected darkMode on disk after early fix, got:\n%s", got)
	}
}
