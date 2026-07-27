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

func TestExtractMissingRustCrates_fromE0432E0433(t *testing.T) {
	out := "$ cargo build\n" +
		"error[E0432]: unresolved import `rand`\n" +
		"error[E0433]: failed to resolve: use of undeclared crate `rand`\n" +
		"exit_code=101"
	crates := extractMissingRustCrates(out)
	if len(crates) != 1 || crates[0] != "rand" {
		t.Fatalf("crates = %v", crates)
	}
}

func TestExtractMissingRustCrates_ignoresStdPseudoCrates(t *testing.T) {
	out := "error[E0432]: unresolved import `std`"
	if crates := extractMissingRustCrates(out); len(crates) != 0 {
		t.Fatalf("crates = %v", crates)
	}
}

func TestAddMissingDependencyToCargoToml(t *testing.T) {
	existing := []byte("[package]\nname = \"blackjack\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[dependencies]\n")
	body, ok := addMissingDependencyToCargoToml(existing, "rand")
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(body, `rand = "0.8"`) {
		t.Fatalf("body = %q", body)
	}
	if !cargoTomlHasDependency(body, "rand") {
		t.Fatal("expected rand dependency")
	}
	_, ok = addMissingDependencyToCargoToml([]byte(body), "rand")
	if ok {
		t.Fatal("should not add existing dependency")
	}
}

func TestAddMissingDependencyToCargoToml_createsDependenciesSection(t *testing.T) {
	existing := []byte("[package]\nname = \"demo\"\nversion = \"0.1.0\"\nedition = \"2021\"\n")
	body, ok := addMissingDependencyToCargoToml(existing, "serde")
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(body, "[dependencies]") || !strings.Contains(body, `serde = "1.0"`) {
		t.Fatalf("body = %q", body)
	}
}

func TestTryMissingRustCrateFix(t *testing.T) {
	dir := t.TempDir()
	cargo := "[package]\nname = \"blackjack\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[dependencies]\n"
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0o644); err != nil {
		t.Fatal(err)
	}

	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"build failed")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{
		StackManifest: DetectStackManifest(dir),
		TrustMode:     editorTrustAutoApply,
	}
	state.RecordReadPath("Cargo.toml")
	state.RecordCommandRun("cargo build", 101,
		"exit_code=101\nerror[E0432]: unresolved import `rand`\nerror[E0433]: failed to resolve: use of undeclared crate `rand`")
	ctx := withImplementationSessionState(context.Background(), state)

	if !ag.tryMissingRustCrateFix(ctx, msg, dir, state, state.LastCommandOutput()) {
		t.Fatal("expected rust crate fix")
	}
	if state.ProposedCount < 1 {
		t.Fatal("expected proposal")
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !cargoTomlHasDependency(string(onDisk), "rand") {
		t.Fatalf("Cargo.toml = %q", onDisk)
	}
	if state.PlaybookUsed() != "rust_missing_crate" {
		t.Fatalf("playbook = %q", state.PlaybookUsed())
	}
}

func TestCommandOutputMatchesPlaybook_rustMissingCrate(t *testing.T) {
	out := "error[E0432]: unresolved import `rand`\nerror[E0433]: use of undeclared crate `rand`"
	if got := commandOutputMatchesPlaybook(out); got != "rust_missing_crate" {
		t.Fatalf("got %q want rust_missing_crate", got)
	}
}

func TestDeriveCargoPackageName(t *testing.T) {
	if got := deriveCargoPackageName("/tmp/user-flow-empty"); got != "user-flow-empty" {
		t.Fatalf("got %q", got)
	}
	if got := deriveCargoPackageName(""); got != "app" {
		t.Fatalf("empty got %q", got)
	}
	if got := deriveCargoPackageName("/projects/123-demo"); got != "app-123-demo" {
		t.Fatalf("digit prefix got %q", got)
	}
}

func TestMinimalCargoTomlBody(t *testing.T) {
	body := minimalCargoTomlBody("blackjack")
	for _, want := range []string{
		`name = "blackjack"`,
		`version = "0.1.0"`,
		`edition = "2021"`,
		"[dependencies]",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %q", want, body)
		}
	}
}

func TestWorkspaceHasRustSources(t *testing.T) {
	dir := t.TempDir()
	if workspaceHasRustSources(dir) {
		t.Fatal("expected false for empty dir")
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !workspaceHasRustSources(dir) {
		t.Fatal("expected true after main.rs")
	}
}

func TestMessageImpliesRustGreenfield(t *testing.T) {
	if !messageImpliesRustGreenfield("Put Cargo.toml and src/main.rs at the workspace root") {
		t.Fatal("expected rust greenfield intent")
	}
	if messageImpliesRustGreenfield("Build a Node API", "src/server.ts") {
		t.Fatal("unexpected rust intent")
	}
	if !messageImpliesRustGreenfield("", "src/main.rs") {
		t.Fatal("expected hint path to imply rust")
	}
}

func TestTryGreenfieldCargoTomlScaffold(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Implement Rust blackjack with Cargo.toml and src/main.rs")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{
		StackManifest: DetectStackManifest(dir),
		TrustMode:     editorTrustAutoApply,
	}
	ctx := withImplementationSessionState(context.Background(), state)

	if !ag.tryGreenfieldCargoTomlScaffold(ctx, msg, dir, state, "src/main.rs") {
		t.Fatal("expected greenfield cargo scaffold")
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), `edition = "2021"`) {
		t.Fatalf("Cargo.toml = %q", onDisk)
	}
	if state.PlaybookUsed() != "greenfield_cargo_toml" {
		t.Fatalf("playbook = %q", state.PlaybookUsed())
	}
}
