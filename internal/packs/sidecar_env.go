package packs

// SidecarKind identifies how the hub starts a pack sidecar process.
const (
	SidecarKindHub = "hub-sidecar"
	SidecarKindMCP = "mcp-sidecar"
)

// SidecarEnv holds overlay settings passed to a pack sidecar process.
type SidecarEnv struct {
	PackID    string
	PackDir   string
	Settings  map[string]string
	Kind      string   // hub-sidecar | mcp-sidecar
	BinaryRel string   // pack-relative binary for mcp-sidecar
	MCPAgents []string // agent types served by mcp-sidecar
}

// CollectSidecarEnvs builds sidecar env for enabled manifests with hub-sidecar or mcp-sidecar capabilities.
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
		kind, binaryRel, mcpAgents := sidecarSpecForManifest(m)
		if kind == "" {
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
			PackID:    m.ID,
			PackDir:   dir,
			Settings:  overlay,
			Kind:      kind,
			BinaryRel: binaryRel,
			MCPAgents: append([]string(nil), mcpAgents...),
		})
	}
	return out
}

func sidecarSpecForManifest(m *Manifest) (kind, binaryRel string, mcpAgents []string) {
	if m == nil {
		return "", "", nil
	}
	for _, def := range m.CapabilityDefs {
		switch def.Kind {
		case SidecarKindMCP:
			if def.Sidecar != nil && binaryRel == "" {
				binaryRel = def.Sidecar.Binary
			}
			if len(def.MCPAgents) > 0 {
				mcpAgents = append(mcpAgents, def.MCPAgents...)
			}
			if kind == "" {
				kind = SidecarKindMCP
			}
		case SidecarKindHub:
			if kind == "" {
				kind = SidecarKindHub
			}
		}
	}
	if kind == SidecarKindMCP && len(mcpAgents) == 0 && len(m.MCPAgents) > 0 {
		mcpAgents = append(mcpAgents, m.MCPAgents...)
	}
	return kind, binaryRel, mcpAgents
}

// PackNeedsSidecar reports whether manifest declares any hub-sidecar or mcp-sidecar capability.
func PackNeedsSidecar(m *Manifest) bool {
	kind, _, _ := sidecarSpecForManifest(m)
	return kind != ""
}

// PackHasMCPSidecar reports whether manifest declares an mcp-sidecar capability.
func PackHasMCPSidecar(m *Manifest) bool {
	if m == nil {
		return false
	}
	for _, def := range m.CapabilityDefs {
		if def.Kind == SidecarKindMCP {
			return true
		}
	}
	return false
}
