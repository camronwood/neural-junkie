package packs

import (
	"path/filepath"
	"testing"
)

func TestValidateCapabilityDefs_requiresDefsForPackLocal(t *testing.T) {
	m := &Manifest{
		ID:    "test-lab",
		Title: "Test",
		Capabilities: []string{
			"customer-pack",
			"phoenix-import",
		},
	}
	_, errs := m.ValidateCapabilityDefs("")
	if len(errs) == 0 {
		t.Fatal("expected error for missing capability_defs")
	}
}

func TestValidateCapabilityDefs_validFixture(t *testing.T) {
	dir := "testdata/customer-lab-pack"
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	warns, errs := m.ValidateCapabilityDefs(dir)
	if len(errs) > 0 {
		t.Fatalf("errors=%v", errs)
	}
	_ = warns
}

func TestMergeResolvedCapabilities_shortCollision(t *testing.T) {
	m1 := &Manifest{
		ID:           "pack-a",
		Title:        "A",
		Capabilities: []string{"customer-pack", "phoenix-import"},
		CapabilityDefs: map[string]CapabilityDef{
			"phoenix-import": {Kind: "hub-sidecar", Routes: []string{"/api/phoenix"}},
		},
	}
	m2 := &Manifest{
		ID:           "pack-b",
		Title:        "B",
		Capabilities: []string{"customer-pack", "phoenix-import"},
		CapabilityDefs: map[string]CapabilityDef{
			"phoenix-import": {Kind: "hub-sidecar", Routes: []string{"/api/phoenix"}},
		},
	}
	_, tokens, collisions := MergeResolvedCapabilities([]*Manifest{m1, m2})
	if len(collisions) != 1 || collisions[0] != "phoenix-import" {
		t.Fatalf("collisions=%v", collisions)
	}
	if len(tokens) == 0 {
		t.Fatal("expected capability tokens")
	}
}

func TestResolveCapabilityQuery_qualified(t *testing.T) {
	m := &Manifest{
		ID:           "brightest-bio-lab",
		Title:        "BBio",
		Capabilities: []string{"phoenix-import"},
		CapabilityDefs: map[string]CapabilityDef{
			"phoenix-import": {Kind: "hub-sidecar"},
		},
	}
	resolved := BuildResolvedCapabilities(m)
	rc, ok := ResolveCapabilityQuery(resolved, "brightest-bio-lab/phoenix-import")
	if !ok || rc.PackID != "brightest-bio-lab" {
		t.Fatalf("resolve: ok=%v rc=%+v", ok, rc)
	}
}

func TestIsPlatformCapability(t *testing.T) {
	if !IsPlatformCapability("customer-pack") {
		t.Fatal("customer-pack should be platform")
	}
	if IsPlatformCapability("phoenix-import") {
		t.Fatal("phoenix-import should not be platform")
	}
	if IsPlatformCapability("room-chat") {
		t.Fatal("room-chat should be pack-local (toolbar-chip), not platform")
	}
}

func TestBuildResolvedCapabilities_officialSidecarRoutes(t *testing.T) {
	dir := "testdata/official/model-arena"
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	warns, errs := m.ValidateCapabilityDefs(dir)
	if len(errs) > 0 {
		t.Fatalf("errors=%v warns=%v", errs, warns)
	}
	_ = warns
	resolved := BuildResolvedCapabilities(m)
	var sidecar *ResolvedCapability
	for i := range resolved {
		if resolved[i].ID == "model-arena-sidecar" {
			sidecar = &resolved[i]
			break
		}
	}
	if sidecar == nil {
		t.Fatal("missing model-arena-sidecar")
	}
	if sidecar.Kind != SidecarKindHub {
		t.Fatalf("kind=%q", sidecar.Kind)
	}
	if len(sidecar.Routes) != 1 || sidecar.Routes[0] != "/api/arena" {
		t.Fatalf("routes=%v", sidecar.Routes)
	}
	if !sidecar.Platform {
		t.Fatal("expected platform token flag on official sidecar capability")
	}
}

func TestBuildResolvedCapabilities_roomChatToolbar(t *testing.T) {
	dir := "testdata/official/room-chat"
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	warns, errs := m.ValidateCapabilityDefs(dir)
	if len(errs) > 0 {
		t.Fatalf("errors=%v warns=%v", errs, warns)
	}
	resolved := BuildResolvedCapabilities(m)
	if len(resolved) != 1 {
		t.Fatalf("resolved=%+v", resolved)
	}
	rc := resolved[0]
	if rc.Platform {
		t.Fatal("room-chat must not resolve as platform-only")
	}
	if rc.Kind != "toolbar-chip" {
		t.Fatalf("kind=%q", rc.Kind)
	}
	if rc.UI == nil || rc.UI.Toolbar == nil || rc.UI.Toolbar.ID != "room" {
		t.Fatalf("toolbar ui=%+v", rc.UI)
	}
	if rc.UI.Modal != "room-chat" {
		t.Fatalf("modal=%q", rc.UI.Modal)
	}
}

func TestBuildResolvedCapabilities_officialAgentCapabilitiesAreClassified(t *testing.T) {
	m := &Manifest{
		ID:           "web-browser",
		Title:        "Browser",
		Description:  "Browser tools",
		Capabilities: []string{"web-browser"},
		MCPAgents:    []string{"browser"},
	}
	resolved := BuildResolvedCapabilities(m)
	if len(resolved) != 1 {
		t.Fatalf("resolved=%+v", resolved)
	}
	if resolved[0].Exposure != CapabilityExposureSafe {
		t.Fatalf("exposure=%q", resolved[0].Exposure)
	}
	if len(resolved[0].MCPAgents) != 1 || resolved[0].MCPAgents[0] != "browser" {
		t.Fatalf("mcp_agents=%v", resolved[0].MCPAgents)
	}
}

func TestBuildResolvedCapabilities_sensitiveOfficialPack(t *testing.T) {
	m := &Manifest{
		ID:           "aws",
		Title:        "AWS",
		Capabilities: []string{"aws-api"},
		MCPAgents:    []string{"aws"},
	}
	resolved := BuildResolvedCapabilities(m)
	if len(resolved) != 1 || resolved[0].Exposure != CapabilityExposureSensitive {
		t.Fatalf("resolved=%+v", resolved)
	}
}

func TestArtifactRendererCapabilityUsesTrustedHostRenderer(t *testing.T) {
	m := &Manifest{
		ID:           "lab",
		Capabilities: []string{"assay-report"},
		CapabilityDefs: map[string]CapabilityDef{
			"assay-report": {
				Kind:        "artifact-renderer",
				Renderer:    "nj.chart",
				MediaTypes:  []string{"application/vnd.neural-junkie.chart+json"},
				RendererAPI: 1,
				SchemaMin:   1,
				SchemaMax:   2,
				Fallback:    "nj.table",
			},
		},
	}
	warnings, errs := m.ValidateCapabilityDefs("")
	if len(errs) != 0 {
		t.Fatalf("errors=%v warnings=%v", errs, warnings)
	}
	resolved := BuildResolvedCapabilities(m)
	if len(resolved) != 1 || resolved[0].Renderer != "nj.chart" || resolved[0].RendererAPI != 1 {
		t.Fatalf("resolved=%+v", resolved)
	}
}

func TestArtifactRendererCapabilityRejectsPackCode(t *testing.T) {
	m := &Manifest{
		ID:           "unsafe-pack",
		Capabilities: []string{"custom-ui"},
		CapabilityDefs: map[string]CapabilityDef{
			"custom-ui": {
				Kind:       "artifact-renderer",
				Renderer:   "unsafe-pack/react-component",
				MediaTypes: []string{"application/json"},
			},
		},
	}
	_, errs := m.ValidateCapabilityDefs("")
	if len(errs) == 0 {
		t.Fatal("expected unknown renderer validation error")
	}
}

func TestOfficialCapabilityFixturesValidate(t *testing.T) {
	files, err := filepath.Glob("testdata/official/*/pack.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		dir := filepath.Dir(file)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			manifest, err := LoadManifest(dir)
			if err != nil {
				t.Fatal(err)
			}
			warnings, validationErrors := manifest.ValidateCapabilityDefs(dir)
			if len(validationErrors) > 0 {
				t.Fatalf("errors=%v warnings=%v", validationErrors, warnings)
			}
		})
	}
}
