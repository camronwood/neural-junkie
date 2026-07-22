package intent

import (
	"context"
	"testing"
)

type responseProvider struct {
	responses []string
	err       error
	calls     int
}

func (p *responseProvider) Generate(context.Context, string, string) (string, error) {
	p.calls++
	if p.err != nil {
		return "", p.err
	}
	index := p.calls - 1
	if index >= len(p.responses) {
		index = len(p.responses) - 1
	}
	return p.responses[index], nil
}

func (p *responseProvider) Model() string { return "utility-local" }

const validDebugIntent = `{
  "schema_version":1,
  "interaction":"task",
  "requested_action":"debug",
  "domain":"frontend",
  "recipient_type":"frontend",
  "retrieval":["codebase"],
  "mutation_requested":"workspace",
  "complexity":"standard",
  "confidence":0.94,
  "reason_codes":["runtime_failure"]
}`

func TestLLMClassifierParsesValidatedIntent(t *testing.T) {
	provider := &responseProvider{responses: []string{validDebugIntent}}
	classifier := NewLLMClassifier(provider)
	got, err := classifier.Classify(context.Background(), TurnFeatures{Text: "The process exits during startup."})
	if err != nil {
		t.Fatal(err)
	}
	if got.RequestedAction != ActionDebug || got.MutationRequested != MutationWorkspace ||
		len(got.Retrieval) != 1 || got.Retrieval[0] != RetrievalCodebase {
		t.Fatalf("intent=%+v", got)
	}
}

func TestLLMClassifierRepairsMalformedOutputOnce(t *testing.T) {
	provider := &responseProvider{responses: []string{"not json", validDebugIntent}}
	classifier := NewLLMClassifier(provider)
	got, err := classifier.Classify(context.Background(), TurnFeatures{Text: "Diagnose the crash."})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 || got.RequestedAction != ActionDebug {
		t.Fatalf("calls=%d intent=%+v", provider.calls, got)
	}
}

func TestLLMClassifierRejectsInvalidEnumsAfterRepair(t *testing.T) {
	invalid := `{"schema_version":1,"interaction":"task","requested_action":"destroy","mutation_requested":"workspace","confidence":1}`
	provider := &responseProvider{responses: []string{invalid, invalid}}
	_, err := NewLLMClassifier(provider).Classify(context.Background(), TurnFeatures{Text: "anything"})
	if err == nil {
		t.Fatal("invalid semantic output accepted")
	}
}

func TestParseSemanticIntentMapsWorkspaceRetrievalAlias(t *testing.T) {
	raw := `{
  "schema_version":1,
  "interaction":"task",
  "requested_action":"debug",
  "domain":"frontend",
  "recipient_type":"frontend",
  "retrieval":["workspace","codebase","bogus"],
  "mutation_requested":"workspace",
  "confidence":0.91,
  "reason_codes":["startup_failure"]
}`
	got, err := parseSemanticIntent(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Retrieval) != 1 || got.Retrieval[0] != RetrievalCodebase {
		t.Fatalf("retrieval=%v, want [codebase]", got.Retrieval)
	}
}
