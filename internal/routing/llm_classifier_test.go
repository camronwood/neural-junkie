package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type plainRoutingProvider struct {
	response string
	err      error
	calls    int
}

func (p *plainRoutingProvider) GenerateResponse(context.Context, string, []protocol.Message) (string, error) {
	p.calls++
	return p.response, p.err
}

func (p *plainRoutingProvider) GenerateVisionResponse(context.Context, string, []byte, string, []protocol.Message) (string, error) {
	return "", errors.New("not implemented")
}

func (p *plainRoutingProvider) GetModel() string { return "routing-test" }

type structuredRoutingProvider struct {
	plainRoutingProvider
	structuredResponse string
	structuredErr      error
	structuredCalls    int
	request            ai.StructuredOutputRequest
}

func (p *structuredRoutingProvider) GenerateStructuredResponse(_ context.Context, request ai.StructuredOutputRequest) (ai.StructuredOutputResult, error) {
	p.structuredCalls++
	p.request = request
	return ai.StructuredOutputResult{Content: p.structuredResponse}, p.structuredErr
}

func TestLLMClassifierUsesStructuredProvider(t *testing.T) {
	provider := &structuredRoutingProvider{
		plainRoutingProvider: plainRoutingProvider{response: "plain response must not be used"},
		structuredResponse:   `{"domain":"security","tool_need":true,"cost_tier":"premium","confidence":0.9,"reason":"security_review","lora_tag":""}`,
	}
	decision, err := NewLLMClassifierFromProvider(provider).Classify(context.Background(), Input{Text: "review auth"})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Domain != DomainSecurity || decision.Confidence != 0.9 {
		t.Fatalf("decision = %#v", decision)
	}
	if provider.structuredCalls != 1 || provider.calls != 0 {
		t.Fatalf("structured calls = %d, plain calls = %d", provider.structuredCalls, provider.calls)
	}
	if provider.request.SchemaName != "routing_decision" || len(provider.request.JSONSchema) == 0 {
		t.Fatalf("structured request missing schema: %#v", provider.request)
	}
}

func TestLLMClassifierFallsBackToGenerateResponse(t *testing.T) {
	provider := &plainRoutingProvider{
		response: `{"domain":"backend","tool_need":false,"cost_tier":"standard","confidence":0.8,"reason":"api_task","lora_tag":""}`,
	}
	decision, err := NewLLMClassifierFromProvider(provider).Classify(context.Background(), Input{Text: "design an API"})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Domain != DomainBackend || provider.calls != 1 {
		t.Fatalf("decision = %#v, calls = %d", decision, provider.calls)
	}
}

func TestLLMClassifierRejectsMalformedAndInvalidOutput(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{name: "malformed", response: `{"domain":`},
		{name: "invalid domain", response: `{"domain":"finance","tool_need":false,"cost_tier":"standard","confidence":0.8,"reason":"task","lora_tag":""}`},
		{name: "invalid cost tier", response: `{"domain":"general","tool_need":false,"cost_tier":"free","confidence":0.8,"reason":"task","lora_tag":""}`},
		{name: "confidence above one", response: `{"domain":"general","tool_need":false,"cost_tier":"standard","confidence":1.1,"reason":"task","lora_tag":""}`},
		{name: "missing confidence", response: `{"domain":"general","tool_need":false,"cost_tier":"standard","reason":"task","lora_tag":""}`},
		{name: "unknown field", response: `{"domain":"general","tool_need":false,"cost_tier":"standard","confidence":0.8,"reason":"task","lora_tag":"","extra":true}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := &plainRoutingProvider{response: tc.response}
			_, err := NewLLMClassifierFromProvider(provider).Classify(context.Background(), Input{Text: "task"})
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestLLMClassifierPreservesZeroConfidence(t *testing.T) {
	provider := &plainRoutingProvider{
		response: `{"domain":"general","tool_need":false,"cost_tier":"standard","confidence":0,"reason":"uncertain","lora_tag":""}`,
	}
	decision, err := NewLLMClassifierFromProvider(provider).Classify(context.Background(), Input{Text: "ambiguous"})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Confidence != 0 {
		t.Fatalf("confidence = %v, want 0", decision.Confidence)
	}
}

func TestClassifyRulesFallbackBehavior(t *testing.T) {
	in := Input{Text: "security review JWT auth"}
	malformed := NewLLMClassifierFromProvider(&plainRoutingProvider{response: `not json`})

	withFallback := Classify(context.Background(), in, Options{
		Classifier:    "llm",
		RulesFallback: true,
		LLMClassifier: malformed,
	})
	if withFallback.Source != SourceRules || withFallback.Domain != DomainSecurity {
		t.Fatalf("with fallback = %#v", withFallback)
	}

	withoutFallback := Classify(context.Background(), in, Options{
		Classifier:    "llm",
		RulesFallback: false,
		LLMClassifier: malformed,
	})
	if withoutFallback.Source != SourceLLM || withoutFallback.Domain != DomainGeneral ||
		withoutFallback.Confidence != 0 || withoutFallback.Reason != "llm_classifier_failed" {
		t.Fatalf("without fallback = %#v", withoutFallback)
	}

	unavailable := Classify(context.Background(), in, Options{
		Classifier:    "llm",
		RulesFallback: false,
	})
	if unavailable.Source != SourceLLM || unavailable.Reason != "llm_classifier_unavailable" {
		t.Fatalf("unavailable = %#v", unavailable)
	}
}

func TestClassifyZeroConfidenceFallbackBehavior(t *testing.T) {
	classifier := NewLLMClassifierFromProvider(&plainRoutingProvider{
		response: `{"domain":"backend","tool_need":false,"cost_tier":"standard","confidence":0,"reason":"uncertain","lora_tag":""}`,
	})
	in := Input{Text: "security review JWT auth"}

	withFallback := Classify(context.Background(), in, Options{
		Classifier:    "llm",
		RulesFallback: true,
		MinConfidence: 0.6,
		LLMClassifier: classifier,
	})
	if withFallback.Source != SourceRules {
		t.Fatalf("with fallback = %#v", withFallback)
	}

	withoutFallback := Classify(context.Background(), in, Options{
		Classifier:    "llm",
		RulesFallback: false,
		MinConfidence: 0.6,
		LLMClassifier: classifier,
	})
	if withoutFallback.Source != SourceLLM || withoutFallback.Domain != DomainBackend || withoutFallback.Confidence != 0 {
		t.Fatalf("without fallback = %#v", withoutFallback)
	}
}
