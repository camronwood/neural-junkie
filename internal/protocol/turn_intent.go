package protocol

import (
	"encoding/json"
	"strings"
)

const (
	TurnMetaComposerMode      = "composer_mode"
	TurnMetaContextTier       = "context_tier"
	TurnMetaCanProposeFiles   = "can_propose_files"
	TurnMetaCanRunImplSession = "can_run_impl_session"
	TurnMetaRequiresWorkspace = "requires_workspace"
	TurnMetaGovernance        = "turn_governance"
)

const TurnGovernanceVersion = 1

type TurnGovernance struct {
	Version           int    `json:"version"`
	ComposerMode      string `json:"composer_mode"`
	ContextTier       string `json:"context_tier"`
	TrustPreference   string `json:"trust_preference,omitempty"`
	CanProposeFiles   bool   `json:"can_propose_files"`
	CanRunImplSession bool   `json:"can_run_impl_session"`
	RequiresWorkspace bool   `json:"requires_workspace"`
	Provenance        string `json:"provenance"`
}

func StampTurnGovernance(m *Message, governance TurnGovernance) {
	if m == nil {
		return
	}
	governance.Version = TurnGovernanceVersion
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	encoded, _ := json.Marshal(governance)
	var value map[string]interface{}
	_ = json.Unmarshal(encoded, &value)
	m.Metadata[TurnMetaGovernance] = value
}

func ExtractTurnGovernance(m *Message) (TurnGovernance, bool) {
	if m == nil || m.Metadata == nil {
		return TurnGovernance{}, false
	}
	raw, ok := m.Metadata[TurnMetaGovernance]
	if !ok {
		return TurnGovernance{}, false
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return TurnGovernance{}, false
	}
	var governance TurnGovernance
	if json.Unmarshal(encoded, &governance) != nil || governance.Version != TurnGovernanceVersion {
		return TurnGovernance{}, false
	}
	switch governance.ComposerMode {
	case "ask", "agent", "plan", "export":
	default:
		return TurnGovernance{}, false
	}
	return governance, true
}

// ComposerModeFromMessage returns ask, agent, plan, or export from explicit metadata.
func ComposerModeFromMessage(m *Message) string {
	if governance, ok := ExtractTurnGovernance(m); ok {
		return governance.ComposerMode
	}
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

func (m *Message) IdeEditorModeIsPlan() bool {
	return ComposerModeFromMessage(m) == "plan"
}

func (m *Message) IdeEditorModeIsExport() bool {
	return ComposerModeFromMessage(m) == "export"
}

// TurnCapabilities describes what an agent may do on this message turn.
type TurnCapabilities struct {
	ComposerMode      string
	ContextTier       string
	CanProposeFiles   bool
	CanRunImplSession bool
	RequiresWorkspace bool
}

// ResolveTurnCapabilities derives capabilities from message metadata (UI-first).
func ResolveTurnCapabilities(m *Message) TurnCapabilities {
	if governance, ok := ExtractTurnGovernance(m); ok {
		cap := TurnCapabilities{
			ComposerMode: governance.ComposerMode, ContextTier: governance.ContextTier,
			CanProposeFiles:   governance.CanProposeFiles,
			CanRunImplSession: governance.CanRunImplSession,
			RequiresWorkspace: governance.RequiresWorkspace,
		}
		if cap.ComposerMode == "ask" || cap.ComposerMode == "plan" {
			cap.CanProposeFiles = false
			cap.CanRunImplSession = false
			cap.RequiresWorkspace = false
		}
		if cap.ContextTier == "" {
			cap.ContextTier = "none"
		}
		return cap
	}
	cap := TurnCapabilities{
		ComposerMode: ComposerModeFromMessage(m),
		ContextTier:  ContextScopeFromMessage(m),
	}
	if cap.ComposerMode == "" {
		cap.ComposerMode = "agent"
	}
	switch cap.ComposerMode {
	case "ask", "plan":
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
	if governance, ok := ExtractTurnGovernance(m); ok {
		return governance.ContextTier
	}
	if m == nil || m.Metadata == nil {
		return ""
	}
	s, _ := m.Metadata["context_scope"].(string)
	return strings.TrimSpace(s)
}
