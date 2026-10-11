package agent

import (
	"os"
	"path/filepath"
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

func TestPreferImplementationTargetPath_atFileOverDoNotModify(t *testing.T) {
	t.Parallel()
	user := "In @file:src/App.tsx ONLY add a subtitle. Do NOT modify tailwind.config.js."
	if got := preferImplementationTargetPath("", user, ""); got != "src/App.tsx" {
		t.Fatalf("got %q, want src/App.tsx", got)
	}
}

func TestFencedContentPlausibleForPath_rejectsCargoTomlInRust(t *testing.T) {
	toml := "[package]\nname = \"blackjack\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[dependencies]\n"
	if fencedContentPlausibleForPath("src/main.rs", "rust", toml) {
		t.Fatal("Cargo.toml body must not be plausible for .rs")
	}
	ok := "fn main() {\n    println!(\"hit or stand\");\n}\n"
	if !fencedContentPlausibleForPath("src/main.rs", "rust", ok) {
		t.Fatal("expected real Rust main to be plausible")
	}
}
