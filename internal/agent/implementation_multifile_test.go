package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

const iosTriviaPrompt = `Let's design and implement a simple iOS game where users answer basic trivia questions. ` +
	`Use .swift files under an Xcode-like folder TriviaGame/ with TriviaGameApp.swift and ContentView.swift (or GameView.swift). ` +
	`Keep everything local.`

func TestImplementationTargets_swiftTriviaPrompt(t *testing.T) {
	t.Parallel()
	got := implementationTargets(&StackManifest{}, iosTriviaPrompt)
	wantApp := "TriviaGame/TriviaGameApp.swift"
	wantUI := "TriviaGame/ContentView.swift"
	if len(got) != 2 || got[0] != wantApp || got[1] != wantUI {
		t.Fatalf("got %v want [%s %s]", got, wantApp, wantUI)
	}
	for _, p := range got {
		if strings.HasSuffix(p, "GameView.swift") {
			t.Fatalf("ContentView preferred over GameView; got %v", got)
		}
	}
}

func TestRemainingImplementationTargets_swiftPartial(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "TriviaGame/TriviaGameApp.swift", "import SwiftUI\n@main struct TriviaGameApp: App { var body: some Scene { WindowGroup { ContentView() } } }\n")
	remaining := remainingImplementationTargets(dir, &StackManifest{}, iosTriviaPrompt)
	if len(remaining) != 1 || remaining[0] != "TriviaGame/ContentView.swift" {
		t.Fatalf("remaining=%v want [TriviaGame/ContentView.swift]", remaining)
	}
}

func TestImplementationTargetSatisfied_swiftGameViewAlternative(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "TriviaGame/GameView.swift", "import SwiftUI\nstruct GameView: View { var body: some View { Text(\"hi\") } }\n")
	if !implementationTargetSatisfied(dir, "TriviaGame/ContentView.swift", iosTriviaPrompt) {
		t.Fatal("GameView should satisfy ContentView target")
	}
}

func TestRemainingImplementationTargets_swiftComplete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "TriviaGame/TriviaGameApp.swift", "import SwiftUI\n")
	writeMultifileTestFile(t, dir, "TriviaGame/ContentView.swift", "import SwiftUI\nstruct ContentView: View { var body: some View { Text(\"q\") } }\n")
	remaining := remainingImplementationTargets(dir, &StackManifest{}, iosTriviaPrompt)
	if len(remaining) != 0 {
		t.Fatalf("remaining=%v want empty", remaining)
	}
}

func TestShouldContinueImplementationSession_swiftRemainingContentView(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "TriviaGame/TriviaGameApp.swift", "import SwiftUI\n")
	a := &Agent{Info: protocol.AgentInfo{ID: "fe", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{ID: "u1", Name: "User"}, iosTriviaPrompt)
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
		FilesChanged:  []string{"TriviaGame/TriviaGameApp.swift"},
		TrustMode:     editorTrustAutoApply,
	}
	ok, note := shouldContinueImplementationSession(a, msg, state)
	if !ok {
		t.Fatalf("expected continue for ContentView, note=%q", note)
	}
	if !strings.Contains(note, "TriviaGame/ContentView.swift") {
		t.Fatalf("note missing ContentView target: %q", note)
	}
}

func TestShouldContinueImplementationSession_swiftCompleteStops(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "TriviaGame/TriviaGameApp.swift", "import SwiftUI\n")
	writeMultifileTestFile(t, dir, "TriviaGame/ContentView.swift", "import SwiftUI\n")
	a := &Agent{Info: protocol.AgentInfo{ID: "fe", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{ID: "u1", Name: "User"}, iosTriviaPrompt)
	msg.Metadata = map[string]interface{}{
		"editor_mode":        "agent",
		"editor_agent_trust": editorTrustAutoApply,
		"workspace_context": map[string]interface{}{
			"workspace_path": dir,
		},
	}
	state := &ImplementationSessionState{
		StackManifest: DetectStackManifest(dir),
		FilesChanged:  []string{"TriviaGame/TriviaGameApp.swift", "TriviaGame/ContentView.swift"},
		TrustMode:     editorTrustAutoApply,
	}
	ok, _ := shouldContinueImplementationSession(a, msg, state)
	if ok {
		t.Fatal("expected stop when both swift files exist")
	}
}

func TestQualifyBareSwiftPathFromPrompt(t *testing.T) {
	t.Parallel()
	got := qualifyBareSwiftPathFromPrompt("TriviaGameApp.swift", iosTriviaPrompt, &StackManifest{})
	if got != "TriviaGame/TriviaGameApp.swift" {
		t.Fatalf("got %q", got)
	}
	got = qualifyBareSwiftPathFromPrompt("TriviaGame/ContentView.swift", iosTriviaPrompt, &StackManifest{})
	if got != "TriviaGame/ContentView.swift" {
		t.Fatalf("already-qualified path changed: %q", got)
	}
	got = qualifyBareSwiftPathFromPrompt("main.go", iosTriviaPrompt, &StackManifest{})
	if got != "main.go" {
		t.Fatalf("non-swift path changed: %q", got)
	}
}

func TestResolveProposalPath_qualifiesBareSwift(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := &Agent{Info: protocol.AgentInfo{ID: "fe", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{ID: "u1", Name: "User"}, iosTriviaPrompt)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{"workspace_path": dir},
	}
	got := a.ResolveProposalPath(context.Background(), msg, "TriviaGameApp.swift")
	if got != "TriviaGame/TriviaGameApp.swift" {
		t.Fatalf("got %q want TriviaGame/TriviaGameApp.swift", got)
	}
}
