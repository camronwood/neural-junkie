package intent

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOntologyRegistryCoreDefaults(t *testing.T) {
	SetOntologyRegistry(nil)
	t.Cleanup(func() { SetOntologyRegistry(nil) })

	reg := CurrentOntology()
	if !reg.ValidDomain("frontend") || !reg.ValidDomain("cad") {
		t.Fatal("core domains must validate")
	}
	if !reg.ValidRecipient("assistant") || !reg.ValidRecipient("code-review") {
		t.Fatal("core recipients must validate")
	}
	if reg.ValidDomain("genomics") {
		t.Fatal("unknown pack domain must reject with core-only registry")
	}
	if !reg.ValidDomain("") || !reg.ValidRecipient("") {
		t.Fatal("empty domain/recipient must remain valid")
	}
}

func TestOntologyFromAgentTypesExtends(t *testing.T) {
	reg := OntologyFromAgentTypes([]string{"genomics", "sre", "aws", "code-review"})
	SetOntologyRegistry(reg)
	t.Cleanup(func() { SetOntologyRegistry(nil) })

	if !CurrentOntology().ValidDomain("genomics") {
		t.Fatal("expected genomics domain")
	}
	if !CurrentOntology().ValidRecipient("genomics") {
		t.Fatal("expected genomics recipient")
	}
	if !CurrentOntology().ValidDomain("sre") || !CurrentOntology().ValidRecipient("sre") {
		t.Fatal("expected sre tokens")
	}
	if !CurrentOntology().ValidDomain("code_review") {
		t.Fatal("code-review agent type should also register code_review domain")
	}
	if CurrentOntology().ValidDomain("totally_unknown_xyz") {
		t.Fatal("unknown must still reject")
	}

	schema := SemanticIntentSchemaJSON()
	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatal(err)
	}
	props := parsed["properties"].(map[string]any)
	domain := props["domain"].(map[string]any)
	enums := domain["enum"].([]any)
	joined := ""
	for _, e := range enums {
		joined += e.(string) + "|"
	}
	if !strings.Contains(joined, "genomics") {
		t.Fatalf("schema enum missing genomics: %v", enums)
	}
	prompt := semanticClassifierPrompt()
	if !strings.Contains(prompt, "genomics") {
		t.Fatal("prompt missing genomics enum")
	}
}

func TestNormalizeDomainRecipientTokens(t *testing.T) {
	if got := normalizeDomainToken("Code-Review"); got != "code_review" {
		t.Fatalf("domain got %q", got)
	}
	if got := normalizeRecipientToken("code_review"); got != "code-review" {
		t.Fatalf("recipient got %q", got)
	}
}
