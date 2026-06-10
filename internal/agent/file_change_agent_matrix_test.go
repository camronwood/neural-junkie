package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var fileEditAgentCases = []struct {
	name      string
	agentType protocol.AgentType
	userMsg   string
	wantPath  string
}{
	{
		name:      "BackendEngineer",
		agentType: protocol.AgentTypeBackend,
		userMsg:   "Please implement HelloWorld in core/sample/main.go",
		wantPath:  "core/sample/main.go",
	},
	{
		name:      "FrontendEngineer",
		agentType: protocol.AgentTypeFrontend,
		userMsg:   "Emit [FILE_CHANGE] for tailwind.config.js",
		wantPath:  "tailwind.config.js",
	},
	{
		name:      "SoftwareArchitect",
		agentType: protocol.AgentTypeArchitecture,
		userMsg:   "Document the API in docs/api-design.md",
		wantPath:  "docs/api-design.md",
	},
	{
		name:      "CodeReviewer",
		agentType: protocol.AgentTypeCodeReview,
		userMsg:   "Apply fixes in internal/agent/agent.go",
		wantPath:  "internal/agent/agent.go",
	},
	{
		name:      "PlatformEngineer",
		agentType: protocol.AgentTypeDevOps,
		userMsg:   "Update core/sample/main.go for deployment hooks",
		wantPath:  "core/sample/main.go",
	},
}

func TestPreferImplementationTargetPath_agentMatrix(t *testing.T) {
	for _, tc := range fileEditAgentCases {
		t.Run(tc.name, func(t *testing.T) {
			got := preferImplementationTargetPath("", tc.userMsg, "File:")
			if got != tc.wantPath {
				t.Fatalf("got %q want %q", got, tc.wantPath)
			}
		})
	}
}

func TestParseLooseFileChange_agentMatrix(t *testing.T) {
	resp := "[FILE_CHANGE] File:\npath: src/App.tsx\n```tsx\nexport {}\n```"
	for _, tc := range fileEditAgentCases[:2] {
		t.Run(tc.name, func(t *testing.T) {
			_ = tc.agentType
			if got := resolveLooseFileChangePath(resp); got != "src/App.tsx" {
				t.Fatalf("got %q", got)
			}
		})
	}
}

func TestAttachWorkspaceContextToProposalMessage_devAgents(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "be-1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{},
		},
	}
	ch := "dm-u-be"
	ws := map[string]interface{}{
		"workspace_name": "proj",
		"workspace_path": "/proj",
		"file_tree":      "src/\n",
	}
	userMsg := protocol.NewMessage(protocol.MessageTypeQuestion, ch, protocol.AgentInfo{Name: "User"}, "edit main.go")
	userMsg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeOutline,
		"workspace_context":  ws,
	}
	a.Context.History[ch] = []*protocol.Message{userMsg}

	proposalMsg := protocol.NewMessage(protocol.MessageTypeFileChange, ch, a.Info, "proposal")
	proposalMsg.Metadata = map[string]interface{}{}
	proposal := &protocol.FileChangeProposal{
		FilePath:   "src/main.go",
		Operation:  "edit",
		NewContent: "package main\n",
		Metadata:   map[string]interface{}{},
	}
	a.attachWorkspaceContextToProposalMessage(ch, proposalMsg, proposal)

	if proposalMsg.Metadata["workspace_context"] == nil {
		t.Fatal("expected workspace_context on proposal message")
	}
	if proposal.Metadata["workspace_context"] == nil {
		t.Fatal("expected workspace_context on proposal payload")
	}
}
