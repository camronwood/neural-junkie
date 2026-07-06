package packs

// SidecarEnv holds overlay settings passed to a pack hub-sidecar process.
type SidecarEnv struct {
	PackID   string
	PackDir  string
	Settings map[string]string
}

// CollectSidecarEnvs builds sidecar envs for enabled manifests with hub-sidecar capabilities.
func CollectSidecarEnvs(manifests []*Manifest, packDirs map[string]string, settings map[string]map[string]string) []SidecarEnv {
	var out []SidecarEnv
	seen := make(map[string]struct{})
	for _, m := range manifests {
		if m == nil {
			continue
		}
		dir := packDirs[m.ID]
		if dir == "" {
			continue
		}
		if !PackNeedsSidecar(m) {
			continue
		}
		if _, ok := seen[m.ID]; ok {
			continue
		}
		seen[m.ID] = struct{}{}
		overlay := settings[m.ID]
		if overlay == nil {
			overlay = map[string]string{}
		}
		resolved, _ := ResolveSettingsOverlay(m, dir)
		for k, v := range resolved {
			overlay[k] = v
		}
		out = append(out, SidecarEnv{
			PackID:   m.ID,
			PackDir:  dir,
			Settings: overlay,
		})
	}
	return out
}

// PackNeedsSidecar reports whether manifest declares any hub-sidecar capability.
func PackNeedsSidecar(m *Manifest) bool {
	if m == nil {
		return false
	}
	for _, def := range m.CapabilityDefs {
		if def.Kind == "hub-sidecar" {
			return true
		}
	}
	return false
}
