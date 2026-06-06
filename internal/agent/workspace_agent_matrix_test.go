package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var workspaceAwareAgentCases = []struct {
	name      string
	agentType protocol.AgentType
}{
	{"BackendEngineer", protocol.AgentTypeBackend},
	{"FrontendEngineer", protocol.AgentTypeFrontend},
	{"SecurityReviewer", protocol.AgentTypeSecurity},
	{"SoftwareArchitect", protocol.AgentTypeArchitecture},
	{"CodeReviewer", protocol.AgentTypeCodeReview},
	{"PlatformEngineer", protocol.AgentTypeDevOps},
	{"DatabaseSpecialist", protocol.AgentTypeDatabase},
	{"Assistant", protocol.AgentTypeAssistant},
	{"BiologyExpert", protocol.AgentTypeBiology},
	{"Cursor", protocol.AgentTypeCLI},
}

func workspaceVisibilityMessage() *protocol.Message {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-u-test",
		protocol.AgentInfo{Name: "User", Type: protocol.AgentTypeGeneral},
		"can you see my workspace I have open?",
	)
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeOutline,
		"workspace_context": map[string]interface{}{
			"workspace_name": "neural-junkie",
			"workspace_path": "/Users/me/neural-junkie",
			"file_tree":      "desktop/\ninternal/\n",
			"open_files": []interface{}{
				map[string]interface{}{
					"path":      "/Users/me/neural-junkie/desktop/src/App.tsx",
					"language":  "typescript",
					"is_active": true,
				},
			},
		},
	}
	return msg
}

func TestWorkspaceVisibilityResponse_allAgentTypes(t *testing.T) {
	msg := workspaceVisibilityMessage()
	for _, tc := range workspaceAwareAgentCases {
		t.Run(tc.name, func(t *testing.T) {
			a := &Agent{Info: protocol.AgentInfo{Name: tc.name, Type: tc.agentType}}
			out, ok := a.tryWorkspaceVisibilityResponse(msg)
			if !ok {
				t.Fatal("expected visibility shortcut")
			}
			for _, want := range []string{"Yes", "neural-junkie", "outline", "App.tsx"} {
				if !strings.Contains(out, want) {
					t.Fatalf("reply missing %q:\n%s", want, out)
				}
			}
		})
	}
}

func TestAppendWorkspaceContext_allAgentTypes(t *testing.T) {
	msg := workspaceVisibilityMessage()
	for _, tc := range workspaceAwareAgentCases {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			AppendWorkspaceContextForChannel(&b, msg, protocol.ChannelTypeDM)
			out := b.String()
			if !strings.Contains(out, "WORKSPACE CONTEXT") {
				t.Fatalf("expected workspace section for %s, got %q", tc.name, out)
			}
			if !strings.Contains(out, "file tree") {
				t.Fatal("expected file tree at outline scope")
			}
		})
	}
}

func TestClassifyWorkspaceVisibilityIntent_allAgentTypes(t *testing.T) {
	for _, tc := range workspaceAwareAgentCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := protocol.NewMessage(
				protocol.MessageTypeQuestion,
				"dm-u-"+tc.name,
				protocol.AgentInfo{Name: "User"},
				"can you see my workspace I have open?",
			)
			got := ClassifyTurnIntentPublic(msg, protocol.ChannelTypeDM, "agent-1", nil)
			if got != IntentSubstantive {
				t.Fatalf("intent: got %s want substantive", got.String())
			}
		})
	}
}

func TestFinalizeWorkspaceVisibilityReply_allAgentTypes(t *testing.T) {
	msg := workspaceVisibilityMessage()
	bad := "You should use golang.org/x/themes. go get github.com/gin-gonic/gin"
	for _, tc := range workspaceAwareAgentCases {
		t.Run(tc.name, func(t *testing.T) {
			a := &Agent{Info: protocol.AgentInfo{Name: tc.name, Type: tc.agentType}}
			out := a.finalizeWorkspaceVisibilityReply(msg, bad)
			if strings.Contains(out, "golang.org/x/themes") {
				t.Fatalf("expected deterministic visibility reply, got %q", out)
			}
			if !strings.Contains(out, "Yes") {
				t.Fatalf("expected visibility confirmation, got %q", out)
			}
		})
	}
}
