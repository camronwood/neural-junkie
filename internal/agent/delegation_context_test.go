package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/delegation"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type stubDelegationClient struct {
	enabled    bool
	candidates []delegation.Candidate
	results    map[string]delegation.ConsultResult
	errs       map[string]error
	consults   []delegation.ConsultRequest
}

func (s *stubDelegationClient) DelegationEnabled() bool { return s.enabled }

func (s *stubDelegationClient) ResolveConsultants(protocol.AgentInfo, string) []delegation.Candidate {
	return append([]delegation.Candidate(nil), s.candidates...)
}

func (s *stubDelegationClient) Consult(_ context.Context, req delegation.ConsultRequest) (delegation.ConsultResult, error) {
	s.consults = append(s.consults, req)
	if err, ok := s.errs[req.ToID]; ok {
		return delegation.ConsultResult{}, err
	}
	if res, ok := s.results[req.ToID]; ok {
		return res, nil
	}
	return delegation.ConsultResult{Text: "default", AgentName: req.ToID}, nil
}

func (s *stubDelegationClient) RequestCapabilityHelp(context.Context, delegation.CapabilityHelpRequest) (delegation.CapabilityHelpResult, error) {
	return delegation.CapabilityHelpResult{}, nil
}

func (s *stubDelegationClient) CapabilityDirectory(string) []delegation.CapabilityProvider { return nil }

type delegationTestHub struct {
	shouldRespondTestHub
	client DelegationClient
}

func (h delegationTestHub) GetDelegation() DelegationClient { return h.client }

func humanTaskMsg(content string) *protocol.Message {
	return protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-backend",
		protocol.AgentInfo{ID: "human-camron", Name: "camron", Type: "human"},
		content,
	)
}

func TestShouldSkipDelegation(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{ID: "a1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend}}

	if !a.shouldSkipDelegation(nil) {
		t.Fatal("nil message should skip")
	}

	collab := humanTaskMsg("review the auth middleware race")
	collab.SetCollaborationID("c1")
	if !a.shouldSkipDelegation(collab) {
		t.Fatal("collaboration messages should skip silent delegation")
	}

	slash := humanTaskMsg("/status")
	if !a.shouldSkipDelegation(slash) {
		t.Fatal("slash commands should skip")
	}

	peer := protocol.NewMessage(
		protocol.MessageTypeChat,
		"general",
		protocol.AgentInfo{ID: "a2", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
		"what about the API?",
	)
	if !a.shouldSkipDelegation(peer) {
		t.Fatal("non-human messages should skip")
	}

	ok := humanTaskMsg("please explain how the session refresh race can happen in auth.go")
	if a.shouldSkipDelegation(ok) {
		t.Fatal("human DM task should not skip")
	}
}

func TestAppendDelegationContextInjectsDelegateResults(t *testing.T) {
	stub := &stubDelegationClient{
		enabled: true,
		candidates: []delegation.Candidate{{
			AgentID:   "bio",
			AgentName: "BiologyExpert",
			AgentType: protocol.AgentTypeBiology,
			Score:     5,
			Intent:    delegation.IntentDomainReasoning,
		}},
		results: map[string]delegation.ConsultResult{
			"bio": {
				Text:      "This peptide motif suggests nuclear localization.",
				AgentName: "BiologyExpert",
				Intent:    delegation.IntentDomainReasoning,
			},
		},
	}
	a := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"Go"}, ai.NewMockProvider(),
		delegationTestHub{client: stub})

	msg := humanTaskMsg("what does this protein sequence imply for nuclear localization?")
	out := a.appendDelegationContext(context.Background(), msg, "BASE_PROMPT")

	if !strings.Contains(out, "=== DELEGATE_RESULTS ===") {
		t.Fatalf("expected DELEGATE_RESULTS block, got %q", out)
	}
	if !strings.Contains(out, "BiologyExpert") || !strings.Contains(out, "nuclear localization") {
		t.Fatalf("expected specialist answer merged, got %q", out)
	}
	if !strings.Contains(out, "=== END DELEGATE_RESULTS ===") {
		t.Fatal("expected end marker")
	}
	if len(stub.consults) != 1 || stub.consults[0].ToID != "bio" {
		t.Fatalf("expected one consult to bio, got %+v", stub.consults)
	}
	consulted := a.TakeDelegationConsulted()
	if len(consulted) != 1 || consulted[0] != "BiologyExpert" {
		t.Fatalf("TakeDelegationConsulted = %v", consulted)
	}
	if again := a.TakeDelegationConsulted(); len(again) != 0 {
		t.Fatalf("buffer should clear, got %v", again)
	}
}

func TestAppendDelegationContextSkipsWhenDisabledOrEmpty(t *testing.T) {
	disabled := &stubDelegationClient{enabled: false, candidates: []delegation.Candidate{{
		AgentID: "bio", AgentName: "BiologyExpert",
	}}}
	a := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(),
		delegationTestHub{client: disabled})
	msg := humanTaskMsg("analyze this protein sequence for expression effects")
	if got := a.appendDelegationContext(context.Background(), msg, "BASE"); got != "BASE" {
		t.Fatalf("disabled delegation must leave prompt unchanged, got %q", got)
	}

	empty := &stubDelegationClient{enabled: true}
	a.Hub = delegationTestHub{client: empty}
	if got := a.appendDelegationContext(context.Background(), msg, "BASE"); got != "BASE" {
		t.Fatalf("no candidates must leave prompt unchanged, got %q", got)
	}
}

func TestAppendDelegationContextSkipsFailedOrEmptyConsults(t *testing.T) {
	stub := &stubDelegationClient{
		enabled: true,
		candidates: []delegation.Candidate{
			{AgentID: "bio", AgentName: "BiologyExpert", Intent: delegation.IntentDomainReasoning},
			{AgentID: "sec", AgentName: "SecurityReviewer", Intent: delegation.IntentDomainReasoning},
		},
		results: map[string]delegation.ConsultResult{
			"bio": {Text: "   ", AgentName: "BiologyExpert"},
		},
		errs: map[string]error{
			"sec": context.Canceled,
		},
	}

	a := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(),
		delegationTestHub{client: stub})
	msg := humanTaskMsg("review auth.go for a race around session refresh and protein notes")
	out := a.appendDelegationContext(context.Background(), msg, "BASE")
	// empty bio text + failed sec → no usable results → original prompt
	if out != "BASE" {
		t.Fatalf("all failed/empty consults should leave prompt unchanged, got %q", out)
	}
}

func TestGenerateConsultResponseUsesProvider(t *testing.T) {
	a := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	text, err := a.GenerateConsultResponse(context.Background(), "summarize the REST handler layout",
		delegation.IntentDomainReasoning, "delegation-internal")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "mock response") {
		t.Fatalf("expected mock provider reply, got %q", text)
	}
}
