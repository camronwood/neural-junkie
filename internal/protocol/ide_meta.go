package protocol

import "strings"

// IDE coding metadata (desktop IDE layout).
const (
	IdeMetaRouteAgentType          = "ide_route_agent_type"
	IdeMetaEditorMode              = "editor_mode"
	IdeMetaImplementationSession   = "implementation_session"
	IdeMetaImplementationComplete  = "implementation_session_complete"
	IdeMetaImplementationFiles     = "implementation_files_changed"
	IdeMetaImplementationOutcome   = "implementation_session_outcome"
	IdeMetaCADFilesWritten         = "cad_files_written"
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

// ImplementationSession reports whether the message requests a multi-step implementation session.
func (m *Message) ImplementationSession() bool {
	if m == nil || m.Metadata == nil {
		return false
	}
	v, ok := m.Metadata[IdeMetaImplementationSession].(bool)
	return ok && v
}

// EditorAgentTrust returns interactive, auto_apply_edits, or yolo from metadata.
func (m *Message) EditorAgentTrust() string {
	if m == nil || m.Metadata == nil {
		return ""
	}
	t, _ := m.Metadata["editor_agent_trust"].(string)
	return strings.TrimSpace(t)
}
