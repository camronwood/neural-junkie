package hub

import (
	"context"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAbortRuntimeAgentsOnChannelCancelsActiveGens(t *testing.T) {
	h := NewHub()
	h.CreateChannel("runtime-abort-ch", "", "")
	handler, ok := h.GetCommandHandler().(*CommandHandler)
	if !ok || handler == nil {
		t.Fatal("expected *CommandHandler")
	}

	runtimeAgent := agent.NewAgent(
		protocol.AgentTypeBackend,
		"RuntimeAbort",
		nil,
		ai.NewMockProvider(),
		h,
	)
	handler.RegisterRuntimeAgent(runtimeAgent)

	genCtx, genCancel := context.WithCancel(context.Background())
	defer genCancel()
	agent.RegisterGenCancelForTest(runtimeAgent, "runtime-abort-ch", genCancel)

	done := make(chan struct{})
	go func() {
		<-genCtx.Done()
		close(done)
	}()

	handler.AbortRuntimeAgentsOnChannel("runtime-abort-ch")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected runtime agent generation to be canceled")
	}
	if agent.ActiveGenCountForTest(runtimeAgent, "runtime-abort-ch") != 0 {
		t.Fatalf("expected no active gens, got %d", agent.ActiveGenCountForTest(runtimeAgent, "runtime-abort-ch"))
	}
}

func TestAbortRuntimeAgentsOnChannelCancelsCLIAgents(t *testing.T) {
	h := NewHub()
	handler, ok := h.GetCommandHandler().(*CommandHandler)
	if !ok || handler == nil {
		t.Fatal("expected *CommandHandler")
	}

	cliAgent := agent.NewAgent(
		protocol.AgentTypeCLI,
		"Gemini",
		nil,
		ai.NewMockProvider(),
		h,
	)
	handler.cliAgents[cliAgent.Info.ID] = cliAgent

	genCtx, genCancel := context.WithCancel(context.Background())
	defer genCancel()
	agent.RegisterGenCancelForTest(cliAgent, "cli-abort-ch", genCancel)

	done := make(chan struct{})
	go func() {
		<-genCtx.Done()
		close(done)
	}()

	handler.AbortRuntimeAgentsOnChannel("cli-abort-ch")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected CLI agent generation to be canceled")
	}
	if agent.ActiveGenCountForTest(cliAgent, "cli-abort-ch") != 0 {
		t.Fatalf("expected no active CLI gens, got %d", agent.ActiveGenCountForTest(cliAgent, "cli-abort-ch"))
	}
}

func TestAbortAgentGenerationsByID(t *testing.T) {
	h := NewHub()
	handler, ok := h.GetCommandHandler().(*CommandHandler)
	if !ok || handler == nil {
		t.Fatal("expected *CommandHandler")
	}

	runtimeAgent := agent.NewAgent(
		protocol.AgentTypeBackend,
		"Pausable",
		nil,
		ai.NewMockProvider(),
		h,
	)
	handler.RegisterRuntimeAgent(runtimeAgent)

	genCtx, genCancel := context.WithCancel(context.Background())
	defer genCancel()
	agent.RegisterGenCancelForTest(runtimeAgent, "ch-a", genCancel)

	done := make(chan struct{})
	go func() {
		<-genCtx.Done()
		close(done)
	}()

	handler.AbortAgentGenerations(runtimeAgent.Info.ID)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected agent generation to be canceled on pause abort")
	}
	if agent.ActiveGenCountForTest(runtimeAgent, "ch-a") != 0 {
		t.Fatalf("expected no active gens on ch-a")
	}
}

func TestInterjectChannelSetsHoldAndAbortsRuntimeAgents(t *testing.T) {
	h := NewHub()
	h.CreateChannel("interject-runtime", "", "")
	handler, ok := h.GetCommandHandler().(*CommandHandler)
	if !ok || handler == nil {
		t.Fatal("expected *CommandHandler")
	}
	runtimeAgent := agent.NewAgent(
		protocol.AgentTypeBackend,
		"InterjectRuntime",
		nil,
		ai.NewMockProvider(),
		h,
	)
	handler.RegisterRuntimeAgent(runtimeAgent)

	genCtx, genCancel := context.WithCancel(context.Background())
	defer genCancel()
	agent.RegisterGenCancelForTest(runtimeAgent, "interject-runtime", genCancel)
	done := make(chan struct{})
	go func() {
		<-genCtx.Done()
		close(done)
	}()

	if err := h.InterjectChannel("interject-runtime", "tester"); err != nil {
		t.Fatalf("InterjectChannel: %v", err)
	}
	if !h.IsChannelHeld("interject-runtime") {
		t.Fatal("expected channel held after interject")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected interject to abort runtime agent generation")
	}
}
