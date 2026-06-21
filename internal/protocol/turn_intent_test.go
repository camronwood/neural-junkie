package protocol

import "testing"

func TestResolveTurnCapabilities_export(t *testing.T) {
	msg := NewMessage(MessageTypeChat, "dm-test", AgentInfo{ID: "u1", Name: "User", Type: "human"}, "save it")
	msg.Metadata = map[string]interface{}{
		IdeMetaEditorMode:            "export",
		IdeMetaImplementationSession: true,
		"context_scope":              "outline",
	}
	cap := ResolveTurnCapabilities(msg)
	if !cap.CanProposeFiles || !cap.CanRunImplSession || !cap.RequiresWorkspace {
		t.Fatalf("export capabilities: %+v", cap)
	}
	if cap.ComposerMode != "export" {
		t.Fatalf("mode=%q", cap.ComposerMode)
	}
}

func TestResolveTurnCapabilities_ask(t *testing.T) {
	msg := NewMessage(MessageTypeChat, "dm-test", AgentInfo{ID: "u1", Name: "User", Type: "human"}, "hi")
	msg.Metadata = map[string]interface{}{IdeMetaEditorMode: "ask"}
	cap := ResolveTurnCapabilities(msg)
	if cap.CanProposeFiles || cap.CanRunImplSession {
		t.Fatalf("ask should be read-only: %+v", cap)
	}
}

func TestResolveTurnCapabilities_plan(t *testing.T) {
	msg := NewMessage(MessageTypeChat, "dm-test", AgentInfo{ID: "u1", Name: "User", Type: "human"}, "plan a refactor")
	msg.Metadata = map[string]interface{}{IdeMetaEditorMode: "plan"}
	cap := ResolveTurnCapabilities(msg)
	if cap.CanProposeFiles || cap.CanRunImplSession {
		t.Fatalf("plan should be read-only: %+v", cap)
	}
	if cap.ComposerMode != "plan" {
		t.Fatalf("mode=%q", cap.ComposerMode)
	}
}
