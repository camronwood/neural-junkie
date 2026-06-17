package protocol

import "strings"

const (
	MetaFileChangeApproved     = "file_change_approved"
	MetaFileChangeAutoApproved = "file_change_auto_approved"
	MetaFileChangeID           = "file_change_id"
	MetaFileChangePath         = "file_change_path"
	MetaFileChangeAgentID      = "file_change_agent_id"
)

// FileChangeApproved reports whether the message records a user-approved file change.
func (m *Message) FileChangeApproved() bool {
	if m == nil || m.Metadata == nil {
		return false
	}
	v, ok := m.Metadata[MetaFileChangeApproved].(bool)
	return ok && v
}

// FileChangeAutoApproved reports whether the hub applied this proposal without user review.
func (m *Message) FileChangeAutoApproved() bool {
	if m == nil || m.Metadata == nil {
		return false
	}
	v, ok := m.Metadata[MetaFileChangeAutoApproved].(bool)
	return ok && v
}

// FileChangeApprovalAgentID returns the agent that proposed the approved change, if set.
func (m *Message) FileChangeApprovalAgentID() string {
	if m == nil || m.Metadata == nil {
		return ""
	}
	id, _ := m.Metadata[MetaFileChangeAgentID].(string)
	return strings.TrimSpace(id)
}
