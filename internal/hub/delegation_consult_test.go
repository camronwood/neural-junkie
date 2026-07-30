package hub

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/delegation"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type consultTestProvider struct {
	reply string
	model string
}

func (p consultTestProvider) GenerateResponse(context.Context, string, []protocol.Message) (string, error) {
	if p.reply == "" {
		return "consult mock reply", nil
	}
	return p.reply, nil
}
func (p consultTestProvider) GenerateVisionResponse(context.Context, string, []byte, string, []protocol.Message) (string, error) {
	return "", nil
}
func (p consultTestProvider) GetModel() string {
	if p.model == "" {
		return "consult-mock"
	}
	return p.model
}

func enableDelegation(t *testing.T, h *Hub) *config.Config {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Delegation.Enabled = true
	cfg.Delegation.MaxConsultsPerTurn = 2
	cfg.Delegation.MaxDepth = 1
	cfg.Delegation.MinRelevanceScore = 2
	h.commandHandler.appConfig = cfg
	config.SetAppConfig(cfg)
	t.Cleanup(func() { config.SetAppConfig(nil) })
	return cfg
}

func registerRuntimeAgent(t *testing.T, h *Hub, typ protocol.AgentType, id, name, reply string, expertise []string) *agent.Agent {
	t.Helper()
	ag := agent.NewAgent(typ, name, expertise, consultTestProvider{reply: reply}, h)
	ag.Info.ID = id
	ag.Info.Status = "active"
	if err := h.RegisterAgent(&ag.Info); err != nil {
		t.Fatal(err)
	}
	h.commandHandler.runtimeAgents[id] = ag
	return ag
}

func TestConsultRejectsWhenDelegationDisabled(t *testing.T) {
	h := NewHub()
	cfg := config.DefaultConfig()
	cfg.Delegation.Enabled = false
	h.commandHandler.appConfig = cfg

	_, err := h.commandHandler.Consult(context.Background(), delegation.ConsultRequest{
		FromID: "a1", ToID: "a2", SubQuestion: "review this API",
	})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected delegation disabled error, got %v", err)
	}
}

func TestConsultRejectsSelfAndDepth(t *testing.T) {
	h := NewHub()
	enableDelegation(t, h)
	registerRuntimeAgent(t, h, protocol.AgentTypeBackend, "a1", "BackendEngineer", "ok", []string{"Go"})

	_, err := h.commandHandler.Consult(context.Background(), delegation.ConsultRequest{
		FromID: "a1", ToID: "a1", SubQuestion: "anything", Intent: delegation.IntentDomainTools,
	})
	if err == nil || !strings.Contains(err.Error(), "cannot consult self") {
		t.Fatalf("expected self-consult rejection, got %v", err)
	}

	_, err = h.commandHandler.Consult(context.Background(), delegation.ConsultRequest{
		FromID: "a1", ToID: "a2", SubQuestion: "anything", Depth: 1, Intent: delegation.IntentDomainTools,
	})
	if err == nil || !strings.Contains(err.Error(), "max depth") {
		t.Fatalf("expected max-depth rejection, got %v", err)
	}
}

func TestConsultReturnsSpecialistAnswer(t *testing.T) {
	h := NewHub()
	enableDelegation(t, h)
	registerRuntimeAgent(t, h, protocol.AgentTypeArchitecture, "arch", "SoftwareArchitect", "unused", nil)
	registerRuntimeAgent(t, h, protocol.AgentTypeBackend, "be", "BackendEngineer",
		"Use POST /v1/widgets with idempotency keys.", []string{"APIs", "Go"})

	res, err := h.commandHandler.Consult(context.Background(), delegation.ConsultRequest{
		FromID:      "arch",
		FromName:    "SoftwareArchitect",
		ToID:        "be",
		SubQuestion: "please run go tests on ./cmd/server and summarize API design notes",
		Channel:     "dm-arch",
		Depth:       0,
		Intent:      delegation.IntentDomainTools,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "POST /v1/widgets") {
		t.Fatalf("expected specialist reply, got %+v", res)
	}
	if res.AgentName != "BackendEngineer" {
		t.Fatalf("agent name = %q", res.AgentName)
	}
	if res.Intent != delegation.IntentDomainTools {
		t.Fatalf("intent = %q", res.Intent)
	}
}

func TestResolveConsultantsScoresBiologyOverBackend(t *testing.T) {
	h := NewHub()
	enableDelegation(t, h)
	from := registerRuntimeAgent(t, h, protocol.AgentTypeBackend, "be", "BackendEngineer", "x", []string{"APIs", "Go"})
	registerRuntimeAgent(t, h, protocol.AgentTypeBiology, "bio", "BiologyExpert", "y",
		[]string{"molecular biology", "protein", "sequences"})

	got := h.commandHandler.ResolveConsultants(from.Info, "what does this protein sequence imply for expression")
	if len(got) != 1 || got[0].AgentName != "BiologyExpert" {
		t.Fatalf("expected BiologyExpert, got %+v", got)
	}
}

func TestResolveConsultantsDisabledReturnsNil(t *testing.T) {
	h := NewHub()
	cfg := config.DefaultConfig()
	cfg.Delegation.Enabled = false
	h.commandHandler.appConfig = cfg
	registerRuntimeAgent(t, h, protocol.AgentTypeBiology, "bio", "BiologyExpert", "y",
		[]string{"biology", "protein"})

	got := h.commandHandler.ResolveConsultants(
		protocol.AgentInfo{ID: "be", Name: "BackendEngineer", Type: protocol.AgentTypeBackend},
		"protein sequence analysis",
	)
	if got != nil {
		t.Fatalf("expected nil when disabled, got %+v", got)
	}
}

func TestCollabVisibleConsultWorksWithDelegationDisabled(t *testing.T) {
	h := NewHub()
	cfg := config.DefaultConfig()
	cfg.Delegation.Enabled = false
	h.commandHandler.appConfig = cfg
	config.SetAppConfig(cfg)
	t.Cleanup(func() { config.SetAppConfig(nil) })

	registerRuntimeAgent(t, h, protocol.AgentTypeBackend, "be", "BackendEngineer",
		"L1 consult answer from BackendEngineer", []string{"Go"})

	res, err := h.commandHandler.CollabVisibleConsult(context.Background(), delegation.ConsultRequest{
		FromID:      "arch",
		FromName:    "SoftwareArchitect",
		ToID:        "be",
		SubQuestion: "please run go tests on ./internal/hub",
		Channel:     "collab-x",
		Depth:       0,
		Intent:      delegation.IntentDomainTools,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "L1 consult answer") {
		t.Fatalf("expected L1 answer, got %+v", res)
	}
}

func TestMaybeCollabConsultPostsVisibleAnswerWithoutJoining(t *testing.T) {
	h := NewHub()
	cfg := config.DefaultConfig()
	cfg.Delegation.Enabled = false // L1 must not require silent delegation
	h.commandHandler.appConfig = cfg
	config.SetAppConfig(cfg)
	t.Cleanup(func() { config.SetAppConfig(nil) })

	a1 := registerRuntimeAgent(t, h, protocol.AgentTypeCLI, "a1", "Gemini", "x", nil)
	_ = registerRuntimeAgent(t, h, protocol.AgentTypeArchitecture, "a2", "SoftwareArchitect", "x", nil)
	_ = registerRuntimeAgent(t, h, protocol.AgentTypeBackend, "a3", "BackendEngineer",
		"Reviewed: prefer context cancellation on the worker loop.", []string{"Go", "APIs"})

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"a1", "a2"},
		"general",
		"tester",
		collaboration.DiscussionConfig{},
		collaboration.CreateOptions{AllowAgentParticipantRequests: false},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	chName := "collab-" + collab.ID
	h.CreateChannelWithType(chName, "collab", "general", protocol.ChannelTypeCollaboration, "tester")
	if err := cm.BindCollaborationChannel(collab.ID, chName); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		chName,
		a1.Info,
		"@BackendEngineer please run go tests on ./internal/hub and note concurrency risks",
	)
	msg.SetCollaborationID(collab.ID)
	msg.SetCollaborationPhase(string(collaboration.PhasePlanning))

	h.maybeCollabConsult(msg, []string{"a3"})

	msgs, err := h.GetMessages(chName, 50)
	if err != nil {
		t.Fatal(err)
	}
	var sawStart, sawAnswer bool
	for _, m := range msgs {
		event, _ := m.Metadata["event"].(string)
		switch event {
		case "collab-consult-start":
			sawStart = true
		case "collab-consult":
			sawAnswer = true
			if m.From.ID != "a3" {
				t.Fatalf("consult reply from %s, want a3", m.From.ID)
			}
			if !strings.Contains(m.Content, "worker loop") {
				t.Fatalf("unexpected consult body: %q", m.Content)
			}
			if collabFlag, _ := m.Metadata["collab_consult"].(bool); !collabFlag {
				t.Fatal("expected collab_consult metadata")
			}
		}
	}
	if !sawStart || !sawAnswer {
		t.Fatalf("expected start+answer consult messages, start=%v answer=%v (n=%d)", sawStart, sawAnswer, len(msgs))
	}

	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Agents) != 2 {
		t.Fatalf("roster must stay at 2 participants, got %+v", snap.Agents)
	}
	if len(snap.PendingParticipantRequests) != 0 {
		t.Fatalf("L1 must not create join requests, got %+v", snap.PendingParticipantRequests)
	}
}

func TestMaybeCollabConsultPostsErrorWhenTargetNotRuntime(t *testing.T) {
	h := NewHub()
	h.commandHandler.appConfig = config.DefaultConfig()

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture, Status: "active"}
	a3 := &protocol.AgentInfo{ID: "a3", Name: "BackendEngineer", Type: protocol.AgentTypeBackend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)
	_ = h.RegisterAgent(a3)
	// a3 registered for GetAgent, but not in runtimeAgents → consult fails.

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"a1", "a2"},
		"general",
		"tester",
		collaboration.DiscussionConfig{},
		collaboration.CreateOptions{AllowAgentParticipantRequests: false},
	)
	if err != nil {
		t.Fatal(err)
	}
	chName := "collab-" + collab.ID
	h.CreateChannelWithType(chName, "collab", "general", protocol.ChannelTypeCollaboration, "tester")
	_ = cm.BindCollaborationChannel(collab.ID, chName)

	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, chName, *a1, "@BackendEngineer review this")
	msg.SetCollaborationID(collab.ID)
	msg.SetCollaborationPhase(string(collaboration.PhasePlanning))

	h.maybeCollabConsult(msg, []string{"a3"})

	msgs, _ := h.GetMessages(chName, 50)
	var sawErr bool
	for _, m := range msgs {
		if event, _ := m.Metadata["event"].(string); event == "collab-consult-error" {
			sawErr = true
		}
	}
	if !sawErr {
		t.Fatal("expected collab-consult-error when target is not a runtime agent")
	}
}
