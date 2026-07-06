package config

import "testing"

func TestResolvedLoRAPolicyDefaults(t *testing.T) {
	c := &Config{Packs: DefaultPacksConfig()}
	p := c.ResolvedLoRAPolicy()
	if p.SuggestAfterTurns != 10 {
		t.Fatalf("default suggest: %d", p.SuggestAfterTurns)
	}
}
