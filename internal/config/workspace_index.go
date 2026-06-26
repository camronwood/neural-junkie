package config

// WorkspaceIndexConfig controls automatic background repo indexing for workspaces.
type WorkspaceIndexConfig struct {
	// AutoIndexOnAdd enables hidden consult-only repo agents when workspaces are added.
	// Nil/absent defaults to true.
	AutoIndexOnAdd *bool `json:"auto_index_on_add,omitempty"`
}

// DefaultWorkspaceIndexConfig returns defaults for workspace indexing.
func DefaultWorkspaceIndexConfig() WorkspaceIndexConfig {
	return WorkspaceIndexConfig{AutoIndexOnAdd: boolPtr(true)}
}

// AutoIndexOnAddEnabled reports whether workspace add should spawn hidden repo agents.
func (w WorkspaceIndexConfig) AutoIndexOnAddEnabled() bool {
	if w.AutoIndexOnAdd == nil {
		return true
	}
	return *w.AutoIndexOnAdd
}
