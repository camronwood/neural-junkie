package packs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCapabilityTokensJSONLoaded(t *testing.T) {
	if len(PlatformCapabilityTokens) == 0 || len(OfficialDomainCapabilityTokens) == 0 {
		t.Fatal("expected capability tokens from capability_tokens.json")
	}
	if !IsThinPlatformCapability("customer-pack") {
		t.Fatal("customer-pack should be thin platform")
	}
	if IsThinPlatformCapability("ide-v2") {
		t.Fatal("ide-v2 is official domain, not thin platform")
	}
	if !IsKnownCapabilityToken("lora-training-sidecar") {
		t.Fatal("lora-training-sidecar should be known")
	}
	for _, tok := range PackLocalExampleCapabilityTokens() {
		if IsKnownCapabilityToken(tok) {
			t.Fatalf("pack-local example %q must not be in KnownCapabilityTokens", tok)
		}
	}
}

func TestGeneratedPackCapabilitiesTSInSync(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	gen := exec.Command("python3", filepath.Join(root, "scripts", "gen-pack-capabilities.py"))
	gen.Dir = root
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("gen-pack-capabilities: %v\n%s", err, out)
	}
	path := filepath.Join(root, "desktop", "src", "stores", "packCapabilities.generated.ts")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, tok := range KnownCapabilityTokens {
		if !strings.Contains(text, "'"+tok+"'") {
			t.Errorf("generated TS missing known token %q", tok)
		}
	}
	if i := strings.Index(text, "export const PACK_PLATFORM_CAP"); i >= 0 {
		j := strings.Index(text[i:], "} as const;")
		if j > 0 {
			platformBlock := text[i : i+j]
			if strings.Contains(platformBlock, "browser-sidecar") {
				t.Fatal("browser-sidecar must not be in PACK_PLATFORM_CAP")
			}
		}
	}
}
