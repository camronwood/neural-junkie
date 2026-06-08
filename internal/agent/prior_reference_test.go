package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserReferencesPriorAssistantContent(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"use the artical from a few messages back", true},
		{"store that artical in a markdown file", false},
		{"what you wrote earlier about themes", true},
		{"hello there", false},
	}
	for _, tc := range cases {
		if got := userReferencesPriorAssistantContent(tc.msg); got != tc.want {
			t.Fatalf("userReferencesPriorAssistantContent(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

func TestFindPriorAssistantContent(t *testing.T) {
	article := stringsRepeat("# LinkedIn Article\n\n", 30) + "### Hook\n\n1. First point\n\n---\n\nBody text."
	agent := protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	history := []*protocol.Message{
		protocol.NewMessage(protocol.MessageTypeChat, "dm-test", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "write article"),
		protocol.NewMessage(protocol.MessageTypeAnswer, "dm-test", agent, article),
		protocol.NewMessage(protocol.MessageTypeChat, "dm-test", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "save it"),
	}
	got := findPriorAssistantContent(history, history[2].ID, "a1", priorReferenceMinChars)
	if got != article {
		t.Fatalf("expected prior article, got len=%d", len(got))
	}
}

func TestTryPriorReferenceResponse_missingHistory(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-test", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "use the article from a few messages back")
	resp, ok := a.tryPriorReferenceResponse(msg)
	if !ok || resp != priorReferenceMissingHistoryReply {
		t.Fatalf("tryPriorReferenceResponse() = (%q, %v)", resp, ok)
	}
}

func TestResolveFileExportContent_priorAssistant(t *testing.T) {
	article := stringsRepeat("Real article paragraph. ", 40)
	agent := protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	history := []*protocol.Message{
		protocol.NewMessage(protocol.MessageTypeAnswer, "dm-test", agent, article),
	}
	a := &Agent{Info: agent}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-test", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "store that artical in docs/test.md")
	content, source := resolveFileExportContent(a, msg, history)
	if source != "prior_assistant" || strings.TrimSpace(content) != strings.TrimSpace(article) {
		t.Fatalf("resolveFileExportContent() source=%q got=%q", source, content[:minLen(32, len(content))])
	}
}

func TestLooksLikePlaceholderProposalContent(t *testing.T) {
	if !looksLikePlaceholderProposalContent("# Title\n\n[Feature Name]\n\n[Brief description of feature]") {
		t.Fatal("expected placeholder template to be rejected")
	}
	if !looksLikePlaceholderProposalContent("# App Name\n\nOverview of the app.\n\n## Features\n\n- Feature 1\n") {
		t.Fatal("expected generic template markdown to be rejected")
	}
	if looksLikePlaceholderProposalContent("# Real\n\nGrounded content from README.") {
		t.Fatal("expected real content to pass")
	}
}

func TestAppendPriorReferenceGuidance_fileExport(t *testing.T) {
	article := stringsRepeat("Real article paragraph about Neural Junkie. ", 40)
	agent := protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	history := []*protocol.Message{
		protocol.NewMessage(protocol.MessageTypeAnswer, "dm-test", agent, article),
	}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-test", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "store that artical in nj-artical-1.md")
	prompt := appendPriorReferenceGuidance("base prompt", msg, history, "a1")
	if !strings.Contains(prompt, "PRIOR ASSISTANT CONTENT") {
		t.Fatal("expected prior content block for file export without explicit back-reference")
	}
	if !strings.Contains(prompt, "Neural Junkie") {
		t.Fatal("expected injected article body")
	}
}

func TestShouldSkipAgentResponseOnFileExportApproval(t *testing.T) {
	agent := protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	history := []*protocol.Message{
		protocol.NewMessage(protocol.MessageTypeChat, "dm-test", protocol.AgentInfo{ID: "u1", Name: "camronwood", Type: protocol.AgentTypeGeneral}, "save that artical to nj-artical-1.md"),
	}
	approval := protocol.NewMessage(protocol.MessageTypeChat, "dm-test", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		"Approved and applied your create change to nj-artical-1.md. Continue with the implementation — do not ask me to approve again.")
	approval.Metadata = map[string]interface{}{
		protocol.MetaFileChangeApproved: true,
		protocol.MetaFileChangePath:     "nj-artical-1.md",
	}
	a := &Agent{
		Info: agent,
		Context: &ConversationContext{
			History: make(map[string][]*protocol.Message),
		},
	}
	a.replaceChannelHistory("dm-test", append(history, approval))
	if !shouldSkipAgentResponseOnFileExportApproval(a, approval) {
		t.Fatal("expected export approval echo to skip agent response")
	}
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
