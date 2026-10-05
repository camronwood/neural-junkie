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

func TestTryMissingRustCrateFix_stdOnlyRewritesRand(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	cargo := "[package]\nname = \"blackjack\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[dependencies]\n"
	main := "use rand::seq::SliceRandom;\nuse rand::thread_rng;\nfn main() {\n    let mut deck = vec![1, 2, 3];\n    let mut rng = thread_rng();\n    deck.shuffle(&mut rng);\n    println!(\"hit stand dealer blackjack\");\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Prefer the Rust standard library only; avoid rand if you can shuffle with std.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	state.RecordReadPath("src/main.rs")
	state.RecordCommandRun("cargo build", 101,
		"exit_code=101\nerror[E0432]: unresolved import `rand`\nerror[E0433]: use of undeclared crate `rand`")
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryMissingRustCrateFix(ctx, msg, dir, state, state.LastCommandOutput()) {
		t.Fatal("expected std shuffle rewrite")
	}
	body, err := os.ReadFile(filepath.Join(dir, "src", "main.rs"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "use rand") || !strings.Contains(string(body), "nj_std_shuffle") {
		t.Fatalf("main.rs not rewritten:\n%s", body)
	}
	if cargoTomlHasDependency(string(mustRead(t, filepath.Join(dir, "Cargo.toml"))), "rand") {
		t.Fatal("Cargo.toml must not gain rand under std-only preference")
	}
	if state.PlaybookUsed() != "rust_std_shuffle_rewrite" {
		t.Fatalf("playbook = %q", state.PlaybookUsed())
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCommandOutputMatchesPlaybook_rustMissingCrate(t *testing.T) {
	out := "error[E0432]: unresolved import `rand`\nerror[E0433]: use of undeclared crate `rand`"
	if got := commandOutputMatchesPlaybook(out); got != "rust_missing_crate" {
		t.Fatalf("got %q want rust_missing_crate", got)
	}
}

func TestCommandOutputMatchesPlaybook_rustMissingBinTarget(t *testing.T) {
	out := "error: failed to parse manifest at `Cargo.toml`\n\nCaused by:\n  no targets specified in the manifest\n  either src/lib.rs, src/main.rs, a [lib] section, or [[bin]] section must be present"
	if got := commandOutputMatchesPlaybook(out); got != "rust_missing_bin_target" {
		t.Fatalf("got %q want rust_missing_bin_target", got)
	}
}

func TestTryMissingRustBinTargetFix(t *testing.T) {
	dir := t.TempDir()
	cargo := "[package]\nname = \"blackjack\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[dependencies]\n"
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0o644); err != nil {
		t.Fatal(err)
	}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"Implement local CLI blackjack with hit/stand against the dealer.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     editorTrustAutoApply,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	state.RecordCommandRun("cargo build", 101,
		"no targets specified in the manifest\neither src/lib.rs, src/main.rs, a [lib] section must be present")
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryMissingRustBinTargetFix(ctx, msg, dir, state, state.LastCommandOutput()) {
		t.Fatal("expected bin target scaffold")
	}
	body, err := os.ReadFile(filepath.Join(dir, "src", "main.rs"))
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(body))
	for _, want := range []string{"hit", "stand", "dealer", "fn main"} {
		if !strings.Contains(lower, want) {
			t.Fatalf("stub missing %q:\n%s", want, body)
		}
	}
}

func TestRewriteRustSourceDropRand(t *testing.T) {
	src := "use rand::seq::SliceRandom;\nuse rand::thread_rng;\n\nfn main() {\n    let mut deck = vec![1,2,3];\n    let mut rng = thread_rng();\n    deck.shuffle(&mut rng);\n    println!(\"hit stand dealer\");\n}\n"
	out, ok := rewriteRustSourceDropRand(src)
	if !ok {
		t.Fatal("expected rewrite")
	}
	if strings.Contains(out, "use rand") || strings.Contains(out, ".shuffle(") {
		t.Fatalf("rand remnants remain:\n%s", out)
	}
	if !strings.Contains(out, "nj_std_shuffle(&mut deck)") || !strings.Contains(out, "fn nj_std_shuffle") {
		t.Fatalf("missing std shuffle helper:\n%s", out)
	}
}

func TestUserPrefersStdOnlyRust(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "c",
		protocol.AgentInfo{Name: "U", Type: "human"},
		"Prefer the Rust standard library only (no Bevy/wgpu; avoid rand if you can shuffle with std).")
	if !userPrefersStdOnlyRust(msg) {
		t.Fatal("expected std-only preference")
	}
}

func TestCommandOutputMatchesPlaybook_rustMissingDebug_beatsExplanationE0433(t *testing.T) {
	// Overnight blackjack: Debug is the real failure; rustc also lists E0433 in explanations.
	out := "error[E0277]: `Card` doesn't implement `Debug`\n" +
		"the trait `Debug` is not implemented for `Card`\n" +
		"help: consider annotating `Card` with `#[derive(Debug)]`\n" +
		"Some errors have detailed explanations: E0081, E0277, E0369, E0433, E0599.\n"
	if got := commandOutputMatchesPlaybook(out); got != "rust_missing_debug" {
		t.Fatalf("got %q want rust_missing_debug", got)
	}
}

func TestCommandOutputMatchesPlaybook_rustMissingPartialEq(t *testing.T) {
	out := "error[E0369]: binary operation `==` cannot be applied to type `Rank`\n" +
		"the trait `PartialEq` is not implemented for `Rank`\n"
	if got := commandOutputMatchesPlaybook(out); got != "rust_missing_partialeq" {
		t.Fatalf("got %q want rust_missing_partialeq", got)
	}
	// Crate noise must not win when PartialEq is the real failure.
	noisy := out + "Some errors have detailed explanations: E0369, E0433.\n"
	if got := commandOutputMatchesPlaybook(noisy); got != "rust_missing_partialeq" {
		t.Fatalf("noisy got %q want rust_missing_partialeq", got)
	}
}

func TestAddDerivePartialEqToRustSource_multiTrait(t *testing.T) {
	src := "enum Rank { Ace, Two }\nenum Suit { Hearts }\n"
	body, ok := addDerivePartialEqToRustSource(src, "Rank")
	if !ok || !strings.Contains(body, "#[derive(PartialEq, Eq)]\nenum Rank") {
		t.Fatalf("insert body=%q ok=%v", body, ok)
	}
	merged, ok := addDerivePartialEqToRustSource("#[derive(Debug, Clone)]\nenum Rank { Ace }\n", "Rank")
	if !ok || !strings.Contains(merged, "PartialEq") || !strings.Contains(merged, "Eq") {
		t.Fatalf("merge body=%q ok=%v", merged, ok)
	}
}

func TestTryMissingRustPartialEqFix(t *testing.T) {
	dir := t.TempDir()
	cargo := "[package]\nname = \"blackjack\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	mainRS := "enum Rank { Ace, Two }\nfn main() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte(mainRS), 0o644); err != nil {
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
	state.RecordReadPath("src/main.rs")
	evidence := "error[E0369]: binary operation `==` cannot be applied to type `Rank`\n" +
		"the trait `PartialEq` is not implemented for `Rank`\n"
	state.RecordCommandRun("cargo build", 101, evidence)
	ctx := withImplementationSessionState(context.Background(), state)

	if !ag.tryMissingRustPartialEqFix(ctx, msg, dir, state, evidence) {
		t.Fatal("expected rust PartialEq fix")
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, "src", "main.rs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "PartialEq") || !strings.Contains(string(onDisk), "Eq") {
		t.Fatalf("main.rs = %q", onDisk)
	}
	if state.PlaybookUsed() != "rust_missing_partialeq" {
		t.Fatalf("playbook = %q", state.PlaybookUsed())
	}
}

func TestCommandOutputMatchesPlaybook_rustMissingCopyClone_beatsE0433(t *testing.T) {
	out := "error[E0382]: borrow of moved value: `suit`\n" +
		"value moved here, in previous iteration of loop\n" +
		"note: if `Suit` implemented `Clone`, you could clone the value\n" +
		"Some errors have detailed explanations: E0382, E0433.\n"
	if got := commandOutputMatchesPlaybook(out); got != "rust_missing_copy_clone" {
		t.Fatalf("got %q want rust_missing_copy_clone", got)
	}
}

func TestCommandOutputMatchesPlaybook_rustMissingCopyClone_liveE0507(t *testing.T) {
	out := "error[E0507]: cannot move out of `*suit` which is behind a shared reference\n" +
		"  --> src/main.rs:27:41\n" +
		"move occurs because `*suit` has type `Suit`, which does not implement the `Copy` trait\n" +
		"note: if `Suit` implemented `Clone`, you could clone the value\n" +
		"Some errors have detailed explanations: E0433, E0507.\n"
	if got := commandOutputMatchesPlaybook(out); got != "rust_missing_copy_clone" {
		t.Fatalf("got %q want rust_missing_copy_clone", got)
	}
	types := extractMissingRustCopyCloneTypes(out)
	if len(types) == 0 || types[0] != "Suit" {
		t.Fatalf("types=%v want Suit", types)
	}
}

func TestAddDeriveCopyCloneToRustSource_insertsAndMerges(t *testing.T) {
	src := "fn main() {}\n\nenum Suit {\n    Hearts,\n}\n"
	body, ok := addDeriveCopyCloneToRustSource(src, "Suit")
	if !ok || !strings.Contains(body, "#[derive(Copy, Clone)]\nenum Suit") {
		t.Fatalf("insert body=%q ok=%v", body, ok)
	}
	merged, ok := addDeriveCopyCloneToRustSource("#[derive(Debug)]\nenum Suit {}\n", "Suit")
	if !ok || !strings.Contains(merged, "Copy") || !strings.Contains(merged, "Clone") {
		t.Fatalf("merge body=%q ok=%v", merged, ok)
	}
	_, ok = addDeriveCopyCloneToRustSource("#[derive(Copy, Clone, Debug)]\nenum Suit {}\n", "Suit")
	if ok {
		t.Fatal("expected no-op when Copy+Clone already present")
	}
}

func TestEnsureRustRandShuffleImport(t *testing.T) {
	src := "fn main() {\n    let mut v = vec![1,2,3];\n    v.shuffle(&mut rng);\n}\n"
	body, ok := ensureRustRandShuffleImport(src)
	if !ok || !strings.Contains(body, "use rand::seq::SliceRandom;") {
		t.Fatalf("body=%q ok=%v", body, ok)
	}
	_, ok = ensureRustRandShuffleImport(body)
	if ok {
		t.Fatal("expected no-op when SliceRandom present")
	}
}

func TestTryMissingRustCopyCloneFix(t *testing.T) {
	dir := t.TempDir()
	cargo := "[package]\nname = \"blackjack\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	mainRS := "enum Suit { Hearts }\nfn main() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte(mainRS), 0o644); err != nil {
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
	state.RecordReadPath("src/main.rs")
	evidence := "error[E0382]: borrow of moved value: `suit`\n" +
		"value moved here, in previous iteration of loop\n" +
		"note: if `Suit` implemented `Clone`, you could clone the value\n"
	state.RecordCommandRun("cargo build", 101, evidence)
	ctx := withImplementationSessionState(context.Background(), state)

	if !ag.tryMissingRustCopyCloneFix(ctx, msg, dir, state, evidence) {
		t.Fatal("expected rust copy/clone fix")
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, "src", "main.rs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "Copy") || !strings.Contains(string(onDisk), "Clone") {
		t.Fatalf("main.rs = %q", onDisk)
	}
	if state.PlaybookUsed() != "rust_missing_copy_clone" {
		t.Fatalf("playbook = %q", state.PlaybookUsed())
	}
}

func TestAddDeriveDebugToRustSource_insertsAndMerges(t *testing.T) {
	src := "fn main() {}\n\nstruct Card {\n    rank: u8,\n}\n"
	body, ok := addDeriveDebugToRustSource(src, "Card")
	if !ok || !strings.Contains(body, "#[derive(Debug)]\nstruct Card") {
		t.Fatalf("insert body=%q ok=%v", body, ok)
	}
	merged, ok := addDeriveDebugToRustSource("#[derive(Clone)]\nstruct Card {}\n", "Card")
	if !ok || !strings.Contains(merged, "#[derive(Clone, Debug)]") {
		t.Fatalf("merge body=%q ok=%v", merged, ok)
	}
	_, ok = addDeriveDebugToRustSource("#[derive(Debug, Clone)]\nstruct Card {}\n", "Card")
	if ok {
		t.Fatal("expected no-op when Debug already present")
	}
}

func TestTryMissingRustDebugFix(t *testing.T) {
	dir := t.TempDir()
	cargo := "[package]\nname = \"blackjack\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	mainRS := "struct Card { rank: u8 }\nfn main() { println!(\"{:?}\", Card{rank:1}); }\n"
	if err := os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte(mainRS), 0o644); err != nil {
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
	state.RecordReadPath("src/main.rs")
	evidence := "error[E0277]: `Card` doesn't implement `Debug`\n" +
		"the trait `Debug` is not implemented for `Card`\n" +
		"help: consider annotating `Card` with `#[derive(Debug)]`\n"
	state.RecordCommandRun("cargo build", 101, evidence)
	ctx := withImplementationSessionState(context.Background(), state)

	if !ag.tryMissingRustDebugFix(ctx, msg, dir, state, evidence) {
		t.Fatal("expected rust debug fix")
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, "src", "main.rs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "#[derive(Debug)]") {
		t.Fatalf("main.rs = %q", onDisk)
	}
	if state.PlaybookUsed() != "rust_missing_debug" {
		t.Fatalf("playbook = %q", state.PlaybookUsed())
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

func TestExtractMissingRustCrates_unlinkedRand(t *testing.T) {
	out := "error[E0433]: failed to resolve: use of unresolved module or unlinked crate `rand`\n" +
		"  --> src/main.rs:34:23\n" +
		"error[E0599]: no method named `shuffle` found for struct `Vec<Card>`\n"
	if got := commandOutputMatchesPlaybook(out); got != "rust_missing_crate" {
		t.Fatalf("playbook=%q", got)
	}
	crates := extractMissingRustCrates(out)
	if len(crates) == 0 || crates[0] != "rand" {
		t.Fatalf("crates=%v want [rand]", crates)
	}
}

func TestTryMissingRustCrateFix_randAddsShuffleImport(t *testing.T) {
	dir := t.TempDir()
	cargo := "[package]\nname = \"blackjack\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	mainRS := "fn main() {\n    let mut cards = vec![1, 2, 3];\n    cards.shuffle(&mut rand::thread_rng());\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte(mainRS), 0o644); err != nil {
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
	state.RecordReadPath("src/main.rs")
	state.RecordReadPath("Cargo.toml")
	evidence := "error[E0433]: failed to resolve: use of unresolved module or unlinked crate `rand`\n" +
		"error[E0599]: no method named `shuffle` found for struct `Vec<i32>`\n"
	state.RecordCommandRun("cargo build", 101, evidence)
	ctx := withImplementationSessionState(context.Background(), state)

	if !ag.tryMissingRustCrateFix(ctx, msg, dir, state, evidence) {
		t.Fatal("expected rust crate fix for rand")
	}
	tomlBody, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tomlBody), "rand") {
		t.Fatalf("Cargo.toml missing rand: %s", tomlBody)
	}
	mainBody, err := os.ReadFile(filepath.Join(dir, "src", "main.rs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainBody), "use rand::seq::SliceRandom;") {
		t.Fatalf("main.rs missing SliceRandom import: %s", mainBody)
	}
}

func TestTryMisplacedRustCargoTomlFix(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte("[package]\nname = \"x\"\n"), 0o644); err != nil {
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
	state := &ImplementationSessionState{TrustMode: editorTrustAutoApply}
	evidence := "error: expected item, found `[`\n --> src/main.rs:1:1\n"
	ctx := withImplementationSessionState(context.Background(), state)
	if !ag.tryMisplacedRustCargoTomlFix(ctx, msg, dir, state, evidence) {
		t.Fatal("expected misplaced cargo toml fix")
	}
	mainBody, _ := os.ReadFile(filepath.Join(dir, "src", "main.rs"))
	if strings.Contains(string(mainBody), "[package]") || !strings.Contains(string(mainBody), "fn main") {
		t.Fatalf("main.rs=%q", mainBody)
	}
	tomlBody, _ := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if !strings.Contains(string(tomlBody), "[package]") {
		t.Fatalf("Cargo.toml=%q", tomlBody)
	}
}

