package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestImplementationTargets_themeTask(t *testing.T) {
	t.Parallel()
	m := &StackManifest{
		HasReact:       true,
		HasTailwind:    true,
		TailwindConfig: "tailwind.config.js",
		EntryPoint:     "src/App.tsx",
	}
	got := implementationTargets(m, "implement light/dark theme toggle in the sidebar")
	if len(got) != 2 || got[0] != "tailwind.config.js" || got[1] != "src/App.tsx" {
		t.Fatalf("got %v", got)
	}
}

func TestImplementationTargetSatisfied_tailwindDarkMode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "tailwind.config.js", `module.exports = { darkMode: "class" };`)
	if !implementationTargetSatisfied(dir, "tailwind.config.js", "add dark theme") {
		t.Fatal("expected satisfied when darkMode present")
	}
}

func TestRemainingImplementationTargets_partial(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "tailwind.config.js", `module.exports = { darkMode: "class" };`)
	writeMultifileTestFile(t, dir, "src/App.tsx", `export default function App() { return <div />; }`)
	m := &StackManifest{
		HasReact:       true,
		HasTailwind:    true,
		TailwindConfig: "tailwind.config.js",
		EntryPoint:     "src/App.tsx",
	}
	remaining := remainingImplementationTargets(dir, m, "implement theme toggle")
	if len(remaining) != 1 || remaining[0] != "src/App.tsx" {
		t.Fatalf("got %v", remaining)
	}
}

func TestRemainingImplementationTargets_appSatisfiedTailMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "tailwind.config.js", `export default { content: ["./src/**/*"] };`)
	writeMultifileTestFile(t, dir, "src/App.tsx", `export default function App() {
  const [theme, setTheme] = useState("dark");
  const toggleTheme = () => setTheme(t => t === "dark" ? "light" : "dark");
  return <button onClick={toggleTheme}>Toggle Theme</button>;
}`)
	m := &StackManifest{
		HasReact:       true,
		HasTailwind:    true,
		TailwindConfig: "tailwind.config.js",
		EntryPoint:     "src/App.tsx",
	}
	remaining := remainingImplementationTargets(dir, m, "implement light/dark theme toggle in the sidebar")
	if len(remaining) != 1 || remaining[0] != "tailwind.config.js" {
		t.Fatalf("got %v, want [tailwind.config.js]", remaining)
	}
}

func TestShouldContinueImplementationSession_remainingApp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "tailwind.config.js", `module.exports = { darkMode: "class" };`)
	writeMultifileTestFile(t, dir, "src/App.tsx", `export default function App() { return <div />; }`)
	a := &Agent{Info: protocol.AgentInfo{ID: "fe", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{ID: "u1", Name: "User"}, "implement light/dark theme in sidebar")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"editor_agent_trust":     editorTrustAutoApply,
		"implementation_session": true,
		"workspace_context": map[string]interface{}{
			"workspace_path": dir,
			"workspace_name": "fixture",
		},
	}
	state := &ImplementationSessionState{
		StackManifest: DetectStackManifest(dir),
		FilesChanged:  []string{"tailwind.config.js"},
		TrustMode:     editorTrustAutoApply,
	}
	ok, note := shouldContinueImplementationSession(a, msg, state)
	if !ok || note == "" {
		t.Fatalf("expected continue, ok=%v note=%q", ok, note)
	}
}

func writeMultifileTestFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
