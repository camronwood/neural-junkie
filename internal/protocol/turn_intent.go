package protocol

import "strings"

const (
	TurnMetaComposerMode     = "composer_mode"
	TurnMetaContextTier      = "context_tier"
	TurnMetaCanProposeFiles  = "can_propose_files"
	TurnMetaCanRunImplSession = "can_run_impl_session"
	TurnMetaRequiresWorkspace = "requires_workspace"
)

// ComposerModeFromMessage returns ask, agent, or export from explicit metadata.
func ComposerModeFromMessage(m *Message) string {
	if m == nil || m.Metadata == nil {
		return ""
	}
	if v, _ := m.Metadata[TurnMetaComposerMode].(string); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(m.IdeEditorMode())
}

func (m *Message) IdeEditorModeIsAsk() bool {
	return ComposerModeFromMessage(m) == "ask"
}

func (m *Message) IdeEditorModeIsAgent() bool {
	mode := ComposerModeFromMessage(m)
	return mode == "agent" || mode == ""
}

func (m *Message) IdeEditorModeIsExport() bool {
	return ComposerModeFromMessage(m) == "export"
}

// TurnCapabilities describes what an agent may do on this message turn.
type TurnCapabilities struct {
	ComposerMode       string
	ContextTier        string
	CanProposeFiles    bool
	CanRunImplSession  bool
	RequiresWorkspace  bool
}

// ResolveTurnCapabilities derives capabilities from message metadata (UI-first).
func ResolveTurnCapabilities(m *Message) TurnCapabilities {
	cap := TurnCapabilities{
		ComposerMode: ComposerModeFromMessage(m),
		ContextTier:  ContextScopeFromMessage(m),
	}
	if cap.ComposerMode == "" {
		cap.ComposerMode = "agent"
	}
	switch cap.ComposerMode {
	case "ask":
		cap.CanProposeFiles = false
		cap.CanRunImplSession = false
	case "export":
		cap.CanProposeFiles = true
		cap.CanRunImplSession = true
		cap.RequiresWorkspace = true
	default: // agent
		cap.CanProposeFiles = true
		cap.CanRunImplSession = m != nil && m.ImplementationSession()
		cap.RequiresWorkspace = cap.CanRunImplSession
	}
	if cap.ContextTier == "" {
		cap.ContextTier = "none"
	}
	return cap
}

// ContextScopeFromMessage reads context_scope metadata.
func ContextScopeFromMessage(m *Message) string {
	if m == nil || m.Metadata == nil {
		return ""
	}
	s, _ := m.Metadata["context_scope"].(string)
	return strings.TrimSpace(s)
}
