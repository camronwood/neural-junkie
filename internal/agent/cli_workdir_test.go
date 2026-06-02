package agent

import (
	"os"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestResolveCLIWorkDir_prefersCollabWorkspaceOverProviderDefault(t *testing.T) {
	t.Setenv("CURSOR_WORK_DIR", "")

	provider := ai.NewCursorCLIProvider("/default/hub/project", "")
	ag := &Agent{
		Info: protocol.AgentInfo{
			Type:       protocol.AgentTypeCLI,
			AIProvider: "cursor-cli",
		},
	}
	ag.SetAIProvider(provider)

	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-test", protocol.AgentInfo{ID: "system", Name: "System"}, "task")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": "/Users/test/Phoenix",
		},
	}

	got := ag.resolveCLIWorkDir(msg)
	if got != "/Users/test/Phoenix" {
		t.Fatalf("workdir = %q, want Phoenix project path", got)
	}
}

func TestResolveCLIWorkDir_envOverrideWins(t *testing.T) {
	t.Setenv("CURSOR_WORK_DIR", "/env/override")

	provider := ai.NewCursorCLIProvider("/default/hub/project", "")
	ag := &Agent{
		Info: protocol.AgentInfo{
			Type:       protocol.AgentTypeCLI,
			AIProvider: "cursor-cli",
		},
	}
	ag.SetAIProvider(provider)

	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-test", protocol.AgentInfo{ID: "system", Name: "System"}, "task")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": "/Users/test/Phoenix",
		},
	}

	got := ag.resolveCLIWorkDir(msg)
	if got != "/env/override" {
		t.Fatalf("workdir = %q, want env override", got)
	}
}

func TestPrepareCLIInvocation_appliesWorkDirAndApprovalMode(t *testing.T) {
	t.Setenv("CURSOR_WORK_DIR", "")

	provider := ai.NewCursorCLIProvider("/default/hub/project", "")
	ag := &Agent{
		Info: protocol.AgentInfo{
			Type:         protocol.AgentTypeCLI,
			AIProvider:   "cursor-cli",
			ApprovalMode: "yolo",
		},
	}
	ag.SetAIProvider(provider)

	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-test", protocol.AgentInfo{ID: "system", Name: "System"}, "task")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": "/Users/test/Phoenix",
		},
	}

	ag.prepareCLIInvocation(msg)

	p := ag.GetAIProvider().(*ai.CLIAgentProvider)
	if p.WorkDir != "/Users/test/Phoenix" {
		t.Fatalf("provider WorkDir = %q, want Phoenix", p.WorkDir)
	}
	if p.ApprovalMode != "yolo" {
		t.Fatalf("ApprovalMode = %q, want yolo", p.ApprovalMode)
	}
}

func TestResolveCLIWorkDir_fallsBackToProviderDefault(t *testing.T) {
	t.Setenv("CURSOR_WORK_DIR", "")

	provider := ai.NewCursorCLIProvider("/default/hub/project", "")
	ag := &Agent{
		Info: protocol.AgentInfo{
			Type:       protocol.AgentTypeCLI,
			AIProvider: "cursor-cli",
		},
	}
	ag.SetAIProvider(provider)

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u1", Name: "User"}, "hello")
	got := ag.resolveCLIWorkDir(msg)
	if got != "/default/hub/project" {
		t.Fatalf("workdir = %q, want provider default", got)
	}
}

func TestResolveCLIWorkDir_fallsBackToCwd(t *testing.T) {
	t.Setenv("CURSOR_WORK_DIR", "")

	provider := ai.NewCursorCLIProvider("", "")
	ag := &Agent{
		Info: protocol.AgentInfo{
			Type:       protocol.AgentTypeCLI,
			AIProvider: "cursor-cli",
		},
	}
	ag.SetAIProvider(provider)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u1", Name: "User"}, "hello")
	got := ag.resolveCLIWorkDir(msg)
	if got != cwd {
		t.Fatalf("workdir = %q, want cwd %q", got, cwd)
	}
}
