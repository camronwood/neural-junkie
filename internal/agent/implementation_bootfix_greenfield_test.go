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

func TestTryGreenfieldNodeAPIScaffold(t *testing.T) {
	dir := t.TempDir()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Let's design and implement a CRUD API for user management. Use Node.js with TypeScript. Set up src/server.ts plus package.json.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx := withImplementationSessionState(context.Background(), state)

	if !ag.tryGreenfieldNodeAPIScaffold(ctx, msg, dir, state) {
		t.Fatal("expected node greenfield scaffold")
	}
	pkg, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pkg), "typescript") || !strings.Contains(string(pkg), "express") {
		t.Fatalf("package.json = %s", pkg)
	}
	server, err := os.ReadFile(filepath.Join(dir, "src", "server.ts"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(server)
	if !strings.Contains(body, "/users") || !strings.Contains(body, "app.listen") {
		t.Fatalf("server.ts = %s", body)
	}
	if state.PlaybookUsed() != "greenfield_node_api" {
		t.Fatalf("playbook = %q", state.PlaybookUsed())
	}
}

func TestTryGreenfieldNodeAPIScaffold_askModeNoWrite(t *testing.T) {
	dir := t.TempDir()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Design a Node.js TypeScript CRUD API for users with package.json and src/server.ts")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_mode":            "ask",
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx := withImplementationSessionState(context.Background(), state)
	if ag.tryGreenfieldNodeAPIScaffold(ctx, msg, dir, state) {
		t.Fatal("ask mode must not scaffold")
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); !os.IsNotExist(err) {
		t.Fatal("package.json must not exist in ask mode")
	}
}

func TestTryGreenfieldNodeAPIScaffold_nonNodeNoWrite(t *testing.T) {
	dir := t.TempDir()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Implement Rust blackjack with Cargo.toml and src/main.rs")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx := withImplementationSessionState(context.Background(), state)
	if ag.tryGreenfieldNodeAPIScaffold(ctx, msg, dir, state) {
		t.Fatal("rust ask must not get express stub")
	}
}

func TestTryRenameNodeAPIResourceFix_notesToMemos(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "server.ts"), []byte(minimalNodeCRUDServerTS("notes")), 0o644); err != nil {
		t.Fatal(err)
	}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Correction: rename the resource from notes to memos everywhere — routes, types, and handlers. Prefer /memos.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryRenameNodeAPIResourceFix(ctx, msg, dir, state) {
		t.Fatal("expected rename fix")
	}
	body, err := os.ReadFile(filepath.Join(dir, "src", "server.ts"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "/memos") || strings.Contains(s, "/notes") {
		t.Fatalf("server after rename = %s", s)
	}
}

func TestTryGreenfieldLandingHTMLScaffoldAndBrand(t *testing.T) {
	dir := t.TempDir()
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Create a simple static landing page at index.html for Brightest Bio. Include an H1 with the brand name and a short Contact section.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryGreenfieldLandingHTMLScaffold(ctx, msg, dir, state) {
		t.Fatal("expected landing scaffold")
	}
	html, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "Brightest Bio") || !strings.Contains(string(html), "<h1") {
		t.Fatalf("index.html = %s", html)
	}

	corr := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Correction: the brand is Neural Junkie, not Brightest Bio. Update the title and H1 — remove Brightest Bio.")
	corr.Metadata = msg.Metadata
	state2 := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx2 := withImplementationSessionState(context.Background(), state2)
	if !ag.tryLandingBrandCorrectionFix(ctx2, corr, dir, state2) {
		t.Fatal("expected brand correction")
	}
	html, err = os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "Neural Junkie") || strings.Contains(string(html), "Brightest Bio") {
		t.Fatalf("after brand = %s", html)
	}

	tag := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Add a short tagline under the H1: Local multi-agent hub. Keep the Neural Junkie brand and the Contact section.")
	tag.Metadata = msg.Metadata
	state3 := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx3 := withImplementationSessionState(context.Background(), state3)
	if !ag.tryLandingTaglineFix(ctx3, tag, dir, state3) {
		t.Fatal("expected tagline fix")
	}
	html, err = os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "Local multi-agent hub") {
		t.Fatalf("after tagline = %s", html)
	}
}

func TestTryGreenfieldSwiftTriviaScaffold(t *testing.T) {
	dir := t.TempDir()
	ag := NewAgent(protocol.AgentTypeArchitecture, "SoftwareArchitect", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Let's design and implement a simple iOS game where users answer basic trivia questions, get points for each correct answer and lose a life for each wrong answer. Users start with 3 lives, and each question should have a 15 second answer clock. Use .swift files under TriviaGame/ with TriviaGameApp.swift and ContentView.swift.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryGreenfieldSwiftTriviaScaffold(ctx, msg, dir, state) {
		t.Fatal("expected swift trivia scaffold")
	}
	app, err := os.ReadFile(filepath.Join(dir, "TriviaGame", "TriviaGameApp.swift"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(app), "@main") || !strings.Contains(string(app), "SwiftUI") {
		t.Fatalf("app = %s", app)
	}
	content, err := os.ReadFile(filepath.Join(dir, "TriviaGame", "ContentView.swift"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(content)
	if !strings.Contains(body, "lives") {
		t.Fatalf("content missing lives: %s", body)
	}
	if !strings.Contains(body, "secondsLeft") && !strings.Contains(body, "timer") && !strings.Contains(body, "Timer") {
		t.Fatalf("content missing timer: %s", body)
	}
	if !strings.Contains(body, "question") && !strings.Contains(body, "Question") {
		t.Fatalf("content missing question: %s", body)
	}
	if !strings.Contains(body, "score") && !strings.Contains(body, "Score") {
		t.Fatalf("content missing score: %s", body)
	}
	if !strings.Contains(body, "Text(") {
		t.Fatalf("content missing Text(: %s", body)
	}
}

func TestMessageImpliesNodeAPIGreenfield(t *testing.T) {
	if !messageImpliesNodeAPIGreenfield("Build a Node.js TypeScript CRUD API for users") {
		t.Fatal("expected node intent")
	}
	if messageImpliesNodeAPIGreenfield("Implement Rust blackjack") {
		t.Fatal("unexpected node intent for rust")
	}
}

func TestTryAddWorkspaceReadyHeadingFix(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := "export default function App() {\n  return (\n    <div className=\"p-4 text-center\">\n      <h1 className=\"text-2xl font-bold\">Fixture App</h1>\n    </div>\n  );\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "src", "App.tsx"), []byte(app), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "App.js"), []byte("diff --git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ag := NewAgent(protocol.AgentTypeArchitecture, "SoftwareArchitect", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Great. Now add an h1 that says Workspace Ready in App.tsx. Do not recreate src/App.js.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryAddWorkspaceReadyHeadingFix(ctx, msg, dir, state) {
		t.Fatal("expected workspace ready heading fix")
	}
	body, err := os.ReadFile(filepath.Join(dir, "src", "App.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Workspace Ready") {
		t.Fatalf("App.tsx = %s", body)
	}
	if _, err := os.Stat(filepath.Join(dir, "src", "App.js")); !os.IsNotExist(err) {
		t.Fatal("App.js should be removed")
	}
}

func TestTryAddNodeHealthEndpointFix(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "server.ts"), []byte(minimalNodeCRUDServerTS("users")), 0o644); err != nil {
		t.Fatal(err)
	}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		`Correction: also expose GET /health that returns {"ok": true}. Keep the user CRUD — do not replace it.`)
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryAddNodeHealthEndpointFix(ctx, msg, dir, state) {
		t.Fatal("expected health endpoint fix")
	}
	body, err := os.ReadFile(filepath.Join(dir, "src", "server.ts"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "/health") || !strings.Contains(s, "/users") {
		t.Fatalf("server after health = %s", s)
	}
}
