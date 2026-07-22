package hub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSendMessageEchoesUIBeforeSlowSemanticResolve(t *testing.T) {
	h := NewHub()
	chName := "latency-echo"
	h.CreateChannelWithType(chName, "", "", protocol.ChannelTypePublic, "user")

	uiSub, err := h.SubscribeUI(chName)
	if err != nil {
		t.Fatal(err)
	}
	defer h.UnsubscribeUI(chName, uiSub)

	agentSub, err := h.Subscribe(chName)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Unsubscribe(chName, agentSub)

	started := make(chan struct{})
	release := make(chan struct{})
	h.SetSemanticTurnRouter(semanticRouterFunc(func(context.Context, intent.TurnFeatures) intent.TurnDecision {
		close(started)
		<-release
		return intent.TurnDecision{
			SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
			RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
			Mutation: intent.MutationNone, Confidence: 0.9, Source: intent.SourceLocalModel,
		}
	}))

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, chName, protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "hello there")

	var sendErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sendErr = h.SendMessage(msg)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("classifier never started")
	}

	select {
	case got := <-uiSub:
		if got.Content != "hello there" {
			t.Fatalf("ui echo content=%q", got.Content)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("UI did not receive echo before classifier finished")
	}

	select {
	case <-agentSub:
		t.Fatal("agents must not receive the turn before classification finishes")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	wg.Wait()
	if sendErr != nil {
		t.Fatal(sendErr)
	}

	select {
	case got := <-agentSub:
		if got.Content != "hello there" {
			t.Fatalf("agent content=%q", got.Content)
		}
		if _, ok := protocol.ExtractTurnDecision(got); !ok {
			t.Fatal("expected stamped turn decision for agents")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("agents did not receive stamped turn")
	}
}
