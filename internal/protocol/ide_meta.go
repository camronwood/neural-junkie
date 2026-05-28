package protocol

import "strings"

// IDE coding metadata (desktop IDE layout).
const (
	IdeMetaRouteAgentType = "ide_route_agent_type"
	IdeMetaEditorMode     = "editor_mode"
)

// IdeRouteAgentType returns the specialist type slug routed for an IDE-scoped message (e.g. backend, frontend).
func (m *Message) IdeRouteAgentType() string {
	if m == nil || m.Metadata == nil {
		return ""
	}
	t, _ := m.Metadata[IdeMetaRouteAgentType].(string)
	return strings.TrimSpace(t)
}

// IdeEditorMode returns ask or agent when set on the message metadata.
func (m *Message) IdeEditorMode() string {
	if m == nil || m.Metadata == nil {
		return ""
	}
	t, _ := m.Metadata[IdeMetaEditorMode].(string)
	return strings.TrimSpace(t)
}
