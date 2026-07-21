package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSendMessage_autoApproveTailwindConfigImplementScenario(t *testing.T) {
	h := newTestHub(t)
	chName := "implement-scenarios"
	_ = h.CreateChannel(chName, "test", "tester")

	agent := &protocol.AgentInfo{ID: "fe1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(agent)

	repoRoot := t.TempDir()
	tailRel := "tailwind.config.js"
	existing := `/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: { extend: {} },
  plugins: [],
};
`
	newBody := `/** @type {import('tailwindcss').Config} */
export default {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: { extend: {} },
  plugins: [],
};
`
	if err := os.WriteFile(filepath.Join(repoRoot, tailRel), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *agent, "Proposing tailwind edit")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": repoRoot,
		},
		"editor_agent_trust":                  "auto_apply_edits",
		"editor_mode":                         "agent",
		protocol.IdeMetaImplementationSession: true,
		"file_change_proposal": map[string]interface{}{
			"operation":   "edit",
			"file_path":   tailRel,
			"old_content": existing,
			"new_content": newBody,
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if !msg.FileChangeAutoApproved() {
		t.Fatal("expected tailwind.config.js auto-approved on implement-scenarios channel")
	}

	got, err := os.ReadFile(filepath.Join(repoRoot, tailRel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "darkMode") {
		t.Fatalf("expected darkMode on disk, got:\n%s", got)
	}
}

func TestSendMessage_holdsDestructiveLocalModelRewriteForApproval(t *testing.T) {
	h := newTestHub(t)
	chName := "implementation-safety"
	_ = h.CreateChannel(chName, "test", "tester")

	agentInfo := &protocol.AgentInfo{
		ID: "fe-local", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend,
		Status: "active", AIProvider: "ollama", AIModel: "qwen2.5-coder:14b",
	}
	_ = h.RegisterAgent(agentInfo)
	repoRoot := t.TempDir()
	rel := "src/App.tsx"
	if err := os.MkdirAll(filepath.Join(repoRoot, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := strings.Repeat("const existingArchitecture = true;\n", 60)
	replacement := "export default function App() { return <main>Welcome</main> }\n"
	if err := os.WriteFile(filepath.Join(repoRoot, rel), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *agentInfo, "Proposing app rewrite")
	msg.Metadata = map[string]interface{}{
		"workspace_context":                   map[string]interface{}{"workspace_path": repoRoot},
		"editor_agent_trust":                  "auto_apply_edits",
		"editor_mode":                         "agent",
		protocol.IdeMetaImplementationSession: true,
		"file_change_proposal": map[string]interface{}{
			"operation": "edit", "file_path": rel,
			"old_content": existing, "new_content": replacement,
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg.FileChangeAutoApproved() {
		t.Fatal("destructive local-model rewrite must require explicit approval")
	}
	got, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Fatal("destructive rewrite reached disk before approval")
	}
}

func TestSendMessage_routedReliableProviderMayApproveDestructiveRewrite(t *testing.T) {
	h := newTestHub(t)
	chName := "implementation-reliable-review"
	_ = h.CreateChannel(chName, "test", "tester")
	agentInfo := &protocol.AgentInfo{
		ID: "fe-routed", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend,
		Status: "active", AIProvider: "ollama", AIModel: "qwen2.5-coder:14b",
	}
	_ = h.RegisterAgent(agentInfo)
	repoRoot := t.TempDir()
	rel := "src/App.tsx"
	if err := os.MkdirAll(filepath.Join(repoRoot, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := strings.Repeat("const existingArchitecture = true;\n", 60)
	replacement := "export default function App() { return <main>Reviewed</main> }\n"
	if err := os.WriteFile(filepath.Join(repoRoot, rel), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *agentInfo, "Reviewed app rewrite")
	msg.Metadata = map[string]interface{}{
		"workspace_context":                   map[string]interface{}{"workspace_path": repoRoot},
		"editor_agent_trust":                  "auto_apply_edits",
		"editor_mode":                         "agent",
		protocol.IdeMetaImplementationSession: true,
		protocol.MetadataRoutingProviderID:    "claude",
		"file_change_proposal": map[string]interface{}{
			"operation": "edit", "file_path": rel,
			"old_content": existing, "new_content": replacement,
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if !msg.FileChangeAutoApproved() {
		t.Fatal("reliable routed provider should be eligible to approve the rewrite")
	}
	got, _ := os.ReadFile(filepath.Join(repoRoot, rel))
	if strings.TrimSpace(string(got)) != strings.TrimSpace(replacement) {
		t.Fatal("approved reliable rewrite was not applied")
	}
}

func TestSendMessage_holdsRewriteDestructiveAgainstGitBaseline(t *testing.T) {
	h := newTestHub(t)
	chName := "implementation-git-baseline"
	_ = h.CreateChannel(chName, "test", "tester")
	agentInfo := &protocol.AgentInfo{
		ID: "fe-git", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend,
		Status: "active", AIProvider: "ollama", AIModel: "qwen2.5-coder:14b",
	}
	_ = h.RegisterAgent(agentInfo)
	root := t.TempDir()
	rel := "src/App.tsx"
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	current := strings.Repeat("const damaged = true;\n", 12)
	proposal := strings.Repeat("const placeholder = true;\n", 11)
	if err := os.WriteFile(filepath.Join(root, rel), []byte(current), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, chName, *agentInfo, "Proposing another small rewrite")
	msg.Metadata = map[string]interface{}{
		"workspace_context":          map[string]interface{}{"workspace_path": root},
		"editor_agent_trust":         "auto_apply_edits",
		"editor_mode":                "agent",
		"git_baseline_destructive":   true,
		"git_baseline_rewrite_ratio": 0.98,
		"file_change_proposal": map[string]interface{}{
			"operation": "edit", "file_path": rel,
			"old_content": current, "new_content": proposal,
		},
	}
	if err := h.SendMessage(msg); err != nil {
		t.Fatal(err)
	}
	if msg.FileChangeAutoApproved() {
		t.Fatal("rewrite destructive against Git baseline must require approval")
	}
	got, _ := os.ReadFile(filepath.Join(root, rel))
	if string(got) != current {
		t.Fatal("Git-baseline guard allowed rewrite onto disk")
	}
}
