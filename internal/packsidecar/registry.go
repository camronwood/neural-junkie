package packsidecar

// globalMgr is set by the hub at startup for internal packages that need sidecar state.
var globalMgr *Manager

// SetGlobalManager registers the hub's pack sidecar manager.
func SetGlobalManager(m *Manager) {
	globalMgr = m
}

// GlobalManager returns the hub's pack sidecar manager, or nil.
func GlobalManager() *Manager {
	return globalMgr
}
