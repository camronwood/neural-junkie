package routing

import "testing"

func TestSelectComposedTag_security(t *testing.T) {
	tags := map[string]struct{}{"nj-security:14b": {}}
	tag, reason := SelectComposedTag(LoRAInput{
		TaskText:      "Review OWASP JWT auth handling.",
		InstalledTags: tags,
	})
	if tag != "nj-security:14b" || reason != "security_lora_tag" {
		t.Fatalf("got tag=%q reason=%q", tag, reason)
	}
}

func TestSelectComposedTag_biology(t *testing.T) {
	tags := map[string]struct{}{"nj-biology:8b": {}}
	tag, reason := SelectComposedTag(LoRAInput{
		TaskText:      "Analyze this protein sequence for binding sites.",
		InstalledTags: tags,
	})
	if tag != "nj-biology:8b" || reason != "biology_lora_tag" {
		t.Fatalf("got tag=%q reason=%q", tag, reason)
	}
}

func TestSelectComposedTag_agentConfigured(t *testing.T) {
	tags := map[string]struct{}{"nj-backend:14b": {}}
	tag, reason := SelectComposedTag(LoRAInput{
		TaskText:      "Implement the payment service.",
		AgentModel:    "nj-backend:14b",
		InstalledTags: tags,
	})
	if tag != "nj-backend:14b" || reason != "agent_configured_lora" {
		t.Fatalf("got tag=%q reason=%q", tag, reason)
	}
}

func TestSelectComposedTag_missingTag(t *testing.T) {
	tag, reason := SelectComposedTag(LoRAInput{
		TaskText:      "Review OWASP JWT auth handling.",
		InstalledTags: map[string]struct{}{},
	})
	if tag != "" || reason != "no_lora_override" {
		t.Fatalf("got tag=%q reason=%q", tag, reason)
	}
}
