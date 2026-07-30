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

func TestValidateWebAssetProposalContent(t *testing.T) {
	t.Parallel()
	cssIntoHTML := "content: /* Reset */\n* { margin: 0; }\nbody { font-family: Arial; }\n"
	if err := validateProposalContent("collabs/x/about.html", cssIntoHTML); err == nil {
		t.Fatal("expected CSS-in-HTML rejection")
	}
	good := "<!DOCTYPE html><html><body><h1>About Collaboration Station</h1></body></html>\n"
	if err := validateProposalContent("collabs/x/about.html", good); err != nil {
		t.Fatalf("valid HTML rejected: %v", err)
	}
	htmlIntoCSS := "<!DOCTYPE html><html><body>nope</body></html>\n"
	if err := validateProposalContent("collabs/x/style.css", htmlIntoCSS); err == nil {
		t.Fatal("expected HTML-in-CSS rejection")
	}
}

func TestValidateProposalContent_RejectsFileChangeDirectivePayload(t *testing.T) {
	t.Parallel()
	bad := "[FILE_CHANGE] operation: create path: collabs/x/design-system.md\n"
	if err := validateProposalContent("collabs/x/design-system.md", bad); err == nil {
		t.Fatal("expected FILE_CHANGE header rejection")
	}
	good := "# Design System\n\n## Color Palette\n\n- black\n- white\n- gray\n- blue\n- red\n"
	if err := validateProposalContent("collabs/x/design-system.md", good); err != nil {
		t.Fatalf("valid markdown rejected: %v", err)
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

func TestTryEarlyGoMathFixtureFix_directApply(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "core", "sample"), 0o755); err != nil {
		t.Fatal(err)
	}
	bug := "package sample\n\nfunc Multiply(a, b int) int {\n\treturn a + b\n}\n"
	path := filepath.Join(dir, "core", "sample", "math.go")
	if err := os.WriteFile(path, []byte(bug), 0o644); err != nil {
		t.Fatal(err)
	}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"},
		"go test ./... fails: Multiply in @file:core/sample/math.go is wrong. Fix it so tests pass.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     "auto_apply_edits",
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	if !ag.tryEarlyGoMathFixtureFix(context.Background(), msg, dir, state) {
		t.Fatal("expected early go math fix")
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "return a * b") {
		t.Fatalf("math.go not fixed on disk: %q", onDisk)
	}
}

func TestTryEarlyGoMathFixtureFix_affirmationUsesPriorPlanCue(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "core", "sample"), 0o755); err != nil {
		t.Fatal(err)
	}
	bug := "package sample\n\nfunc Add(a, b int) int {\n\treturn a + b + 1\n}\n"
	path := filepath.Join(dir, "core", "sample", "math.go")
	if err := os.WriteFile(path, []byte(bug), 0o644); err != nil {
		t.Fatal(err)
	}
	prior := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"},
		"Inspect the failing Add test and plan the smallest correction in core/sample/math.go.")
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	ag.Context = &ConversationContext{History: map[string][]*protocol.Message{
		"implement-scenarios": {prior},
	}}
	state := &ImplementationSessionState{}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"},
		"Approve that plan and implement it now.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     "auto_apply_edits",
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	if !ag.tryEarlyGoMathFixtureFix(context.Background(), msg, dir, state) {
		t.Fatal("expected early go math fix from prior plan cue")
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(onDisk), "a + b + 1") {
		t.Fatalf("math.go still buggy: %q", onDisk)
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

func TestUserRequestsThemeCSSDeliverable(t *testing.T) {
	if !userRequestsThemeCSSDeliverable("please create src/theme.css with light and dark variables") {
		t.Fatal("expected affirmative theme request")
	}
	if userRequestsThemeCSSDeliverable("Do not reference src/theme.css or other paths. Write findings.md summarizing README.md") {
		t.Fatal("negated theme.css mention must not count as a request")
	}
	if userRequestsThemeCSSDeliverable("Use ONLY README.md and core/sample/main.go") {
		t.Fatal("findings-only goal should not request theme.css")
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

func TestTryEarlyThemeCSSFixSkipsNegatedMention(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{}
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-scenarios",
		protocol.AgentInfo{ID: "sys", Name: "System", Type: "system"},
		"Do not reference src/theme.css. Write collabs/x/findings.md summarizing README.md and core/sample/main.go only.")
	msg.SetCollaborationID("cid")
	msg.SetCollaborationPhase("executing")
	msg.SetTaskID("task-1")
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata[MetadataContextScope] = ContextScopeFocus
	msg.Metadata["task_context_paths"] = []string{"README.md", "core/sample/main.go"}
	msg.Metadata["workspace_context"] = map[string]interface{}{"workspace_path": dir}
	if ag.tryEarlyThemeCSSFix(context.Background(), msg, dir, state) {
		t.Fatal("negated theme.css in focus-scoped findings task must not early-fix theme.css")
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

// TestShouldRepairCorruptAppJSEntry_implSessionWithoutPhraseHeuristics ensures the
// early App.js delete still fires after messageHasBootOrBuildError was stubbed.
func TestShouldRepairCorruptAppJSEntry_implSessionWithoutPhraseHeuristics(t *testing.T) {
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
	ag := NewAgent(protocol.AgentTypeArchitecture, "SoftwareArchitect", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"},
		"the app is not booting")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     "auto_apply_edits",
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	if !ag.shouldRepairCorruptAppJSEntry(msg, dir, msg.Content, manifest) {
		t.Fatal("impl session + entry conflict must authorize repair without phrase heuristics")
	}
	state := &ImplementationSessionState{
		BootFixIntent:  true,
		StackManifest:  manifest,
		TrustMode:      editorTrustAutoApply,
	}
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryEarlyCorruptAppJSBootFix(ctx, msg, dir, state) {
		t.Fatal("expected early corrupt App.js delete")
	}
	if _, err := os.Stat(filepath.Join(dir, "src", "App.js")); !os.IsNotExist(err) {
		t.Fatalf("App.js should be deleted, err=%v", err)
	}
	if len(state.FilesChanged) == 0 || state.FilesChanged[0] != "src/App.js" {
		t.Fatalf("FilesChanged=%v", state.FilesChanged)
	}
}

func TestBootFixIntent_entryConflictImplSession(t *testing.T) {
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
	write("src/App.js", "diff --git a/x b/x\n")
	write("src/App.tsx", "export default function App() { return null }\n")
	write("src/main.tsx", "import App from './App'\n")
	manifest := DetectStackManifest(dir)
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"},
		"please fix the boot failure")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	// Simulate the BootFixIntent assignment block from runImplementationSession.
	state := &ImplementationSessionState{StackManifest: manifest}
	history := []*protocol.Message{}
	semanticDecision, hasSemanticDecision := protocol.ExtractTurnDecision(msg)
	if hasSemanticDecision {
		t.Fatal("expected no stamped decision")
	}
	_ = semanticDecision
	if !state.BootFixIntent && DetectEntryConflicts(dir, state.StackManifest) != "" &&
		(msg.ImplementationSession() ||
			messageImpliesBootFix(msg.Content, history) ||
			messageHasBootOrBuildError(msg.Content)) {
		state.BootFixIntent = true
		state.FixLikeIntent = true
	}
	if !state.BootFixIntent || !state.FixLikeIntent {
		t.Fatalf("expected BootFixIntent from entry conflict + impl session, got boot=%v fix=%v",
			state.BootFixIntent, state.FixLikeIntent)
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

func TestTryEarlySidebarFooterExtract(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	app := `import "./index.css";
import { useState } from "react";

function renderSidebarFooter() {
  return (
    <footer className="text-xs text-slate-600 border-t border-slate-800 pt-3">
      <p>Neural Junkie fixture</p>
      <p>Selection extract target block</p>
      <p>Keep this helper scoped to sidebar only</p>
      <p>Do not move theme toggle logic here</p>
    </footer>
  );
}

export default function App() {
  return (
    <aside>
      {renderSidebarFooter()}
    </aside>
  );
}
`
	writeMultifileTestFile(t, dir, "src/App.tsx", app)
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{StackManifest: DetectStackManifest(dir)}
	userContent := "@FrontendEngineer Extract the selected sidebar footer block from src/App.tsx into src/components/SidebarFooter.tsx and import it in App."
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"}, userContent)
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     "auto_apply_edits",
		"workspace_context": map[string]interface{}{
			"workspace_path": dir,
		},
	}
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryEarlySidebarFooterExtract(ctx, msg, dir, state) {
		t.Fatal("expected early sidebar footer extract")
	}
	gotApp, err := os.ReadFile(filepath.Join(dir, "src/App.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(gotApp), "function renderSidebarFooter") {
		t.Fatalf("App.tsx still has helper:\n%s", gotApp)
	}
	if !strings.Contains(string(gotApp), "SidebarFooter") {
		t.Fatalf("App.tsx missing SidebarFooter import/use:\n%s", gotApp)
	}
	gotFooter, err := os.ReadFile(filepath.Join(dir, "src/components/SidebarFooter.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gotFooter), "Neural Junkie fixture") {
		t.Fatalf("footer missing content:\n%s", gotFooter)
	}
}

func TestTryEarlyGoMainFixtureFix_skipsCollabCodingDeliverable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "core", "sample"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "core", "sample", "main.go"), []byte("package sample\n"), 0o644)
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{}
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-x",
		protocol.AgentInfo{ID: "human", Name: "User", Type: "human"},
		"implement HelloWorld in core/sample/main.go")
	msg.Metadata = map[string]interface{}{
		"deliverable_kind":       "file",
		"implementation_session": true,
		"task_title":             "Impl",
		"task_description":       "Create core/sample/main.go with HelloWorld",
	}
	if skipCollabCodingFixtureSynths(msg) != true {
		t.Fatal("expected skip for collab coding")
	}
	if ag.tryEarlyGoMainFixtureFix(context.Background(), msg, dir, state) {
		t.Fatal("fixture synth must not run for collab coding deliverable_kind=file")
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
