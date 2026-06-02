package routing

import "testing"

func TestLooksLightweightCollabTask_identify(t *testing.T) {
	if !LooksLightweightCollabTask("Identify relevant API documents and schema files in resource-api/") {
		t.Fatal("expected lightweight")
	}
}

func TestLooksLightweightCollabTask_compileNotLight(t *testing.T) {
	if LooksLightweightCollabTask("Compile findings from the above tasks into schema_registration.md") {
		t.Fatal("synthesis tasks should not use light model")
	}
}

func TestSelectLightOllamaTag(t *testing.T) {
	installed := map[string]struct{}{"qwen2.5:3b": {}, "qwen2.5:7b": {}}
	tag, reason := SelectLightOllamaTag(installed)
	if tag != "qwen2.5:3b" || reason != "light_local_model" {
		t.Fatalf("got tag=%q reason=%q", tag, reason)
	}
}
