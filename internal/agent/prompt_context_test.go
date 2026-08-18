package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAppendUserAndAgentRules_fromMetadata(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-test",
		protocol.AgentInfo{ID: "u1", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"hello")
	msg.Metadata = map[string]any{
		MetadataUserRulesMarkdown: "Always reply in bullet points.",
	}
	self := &protocol.AgentInfo{Name: "Assistant", CustomRulesMarkdown: "Be concise."}

	var b strings.Builder
	AppendUserAndAgentRules(&b, msg, self, "", 0)
	out := b.String()
	if !strings.Contains(out, "Always reply in bullet points.") {
		t.Fatalf("expected user rules in prompt: %q", out)
	}
	if !strings.Contains(out, "Be concise.") {
		t.Fatalf("expected agent rules in prompt: %q", out)
	}
}

func TestAppendUserAndAgentRules_hubFallback(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-1",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"task body")

	var b strings.Builder
	AppendUserAndAgentRules(&b, msg, &protocol.AgentInfo{Name: "Backend"}, "Use British spelling.", 0)
	out := b.String()
	if !strings.Contains(out, "British spelling") {
		t.Fatalf("expected hub fallback rules: %q", out)
	}
}

func TestAppendUserAndAgentRules_metadataWinsOverFallback(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-test",
		protocol.AgentInfo{ID: "u1", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"hello")
	msg.Metadata = map[string]any{MetadataUserRulesMarkdown: "from metadata"}

	var b strings.Builder
	AppendUserAndAgentRules(&b, msg, nil, "from fallback", 0)
	if strings.Contains(b.String(), "from fallback") {
		t.Fatal("metadata should win over hub fallback")
	}
}

func TestAppendUserAndAgentRules_compactCap(t *testing.T) {
	long := strings.Repeat("x", 5000)
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-test",
		protocol.AgentInfo{ID: "u1", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"hello")
	msg.Metadata = map[string]any{MetadataUserRulesMarkdown: long}

	var b strings.Builder
	AppendUserAndAgentRules(&b, msg, nil, "", compactUserRulesMarkdownBytes)
	out := b.String()
	if len(out) > compactUserRulesMarkdownBytes+512 {
		t.Fatalf("compact rules block too large: %d bytes", len(out))
	}
	if !strings.Contains(out, "[truncated") {
		t.Fatal("expected truncation marker for compact cap")
	}
}

func TestResolveUserRulesHubFallback(t *testing.T) {
	SetUserRulesLookup(func(username string) string {
		if username == "Camron" {
			return "Prefer markdown links."
		}
		return ""
	})
	t.Cleanup(func() { SetUserRulesLookup(nil) })

	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-test",
		protocol.AgentInfo{ID: "u1", Name: "camronwood", Type: protocol.AgentTypeGeneral},
		"hi")
	msg.Metadata = map[string]any{MetadataHubSessionUsername: "Camron"}
	if got := ResolveUserRulesHubFallback(msg); got != "Prefer markdown links." {
		t.Fatalf("session username lookup got %q", got)
	}
}

func TestResolveUserRulesHubFallback_senderName(t *testing.T) {
	SetUserRulesLookup(func(username string) string {
		if username == "Camron" {
			return "Prefer markdown links."
		}
		return ""
	})
	t.Cleanup(func() { SetUserRulesLookup(nil) })

	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-test",
		protocol.AgentInfo{ID: "u1", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"hi")
	if got := ResolveUserRulesHubFallback(msg); got != "Prefer markdown links." {
		t.Fatalf("got %q", got)
	}
}

func TestAttachUserRulesMetadataIfMissing(t *testing.T) {
	SetUserRulesLookup(func(username string) string {
		return "Always cite sources."
	})
	t.Cleanup(func() { SetUserRulesLookup(nil) })

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "u1", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"hello")
	AttachUserRulesMetadataIfMissing(msg)
	raw, ok := msg.Metadata[MetadataUserRulesMarkdown]
	if !ok {
		t.Fatal("expected user rules metadata")
	}
	if raw.(string) != "Always cite sources." {
		t.Fatalf("got %q", raw)
	}
}

func TestAppendPromptAttachments_dropsDependencyAndPlanFraming(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "u1", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"Plan how to add HelloWorld")
	msg.Metadata = map[string]any{
		"composer_mode": "plan",
		"editor_mode":   "plan",
		MetadataPromptAttachments: []any{
			map[string]any{
				"path":    ".venv-icon/lib/python3.14/site-packages/PIL/TiffImagePlugin.py",
				"content": "assert isinstance(_denominator, int)",
			},
			map[string]any{
				"path":    "internal/agent/plan_mode_prompt.go",
				"content": "func appendPlanModePrompt()",
			},
		},
	}
	var b strings.Builder
	AppendPromptAttachments(&b, msg)
	out := b.String()
	if strings.Contains(out, "TiffImagePlugin") {
		t.Fatalf("dependency chunk leaked into prompt: %q", out)
	}
	if !strings.Contains(out, "plan_mode_prompt.go") {
		t.Fatalf("expected project file kept: %q", out)
	}
	if !strings.Contains(out, "candidate search hits") {
		t.Fatalf("expected plan framing: %q", out)
	}
	if strings.Contains(out, "answer from the attached source chunks") {
		t.Fatal("plan mode must not treat search hits as the answer")
	}
}
