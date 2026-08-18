package agent

import (
	"context"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollectStreamTokens_lengthReasonMetadata(t *testing.T) {
	hub := &streamCaptureHub{imageGenTestHub: &imageGenTestHub{}}
	a := &Agent{
		Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Hub:  hub,
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-camron-assistant", protocol.AgentInfo{Name: "Camron"}, "review")
	tokens := make(chan ai.StreamToken, 4)
	tokens <- ai.StreamToken{Content: "Partial answer (no"}
	tokens <- ai.StreamToken{
		Done:                   true,
		TerminalReason:         ai.TerminalReasonLength,
		ProviderTerminalReason: "length",
	}
	close(tokens)

	var term streamTerminalCapture
	text, id, _, err := a.collectStreamTokens(context.Background(), msg, "stream-length-1", tokens, &term)
	if err != nil {
		t.Fatalf("collectStreamTokens: %v", err)
	}
	if text != "Partial answer (no" {
		t.Fatalf("text=%q", text)
	}
	if id != "stream-length-1" {
		t.Fatalf("id=%q", id)
	}
	if term.Reason != protocol.TerminalReasonLength {
		t.Fatalf("terminal capture=%q", term.Reason)
	}
	var end *protocol.Message
	for _, m := range hub.broadcasts {
		if m.Type == protocol.MessageTypeStreamEnd {
			end = m
			break
		}
	}
	if end == nil {
		t.Fatal("expected stream_end")
	}
	if end.Metadata[protocol.MetadataTerminalReason] != protocol.TerminalReasonLength {
		t.Fatalf("stream_end metadata=%v", end.Metadata)
	}
	if end.Metadata[protocol.MetadataContinuationAvailable] != true {
		t.Fatalf("expected continuation_available, got %v", end.Metadata)
	}
}

func TestCollectStreamTokens_timeoutNotLength(t *testing.T) {
	hub := &streamCaptureHub{imageGenTestHub: &imageGenTestHub{}}
	a := &Agent{
		Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Hub:  hub,
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-camron-assistant", protocol.AgentInfo{Name: "Camron"}, "review")
	tokens := make(chan ai.StreamToken, 4)
	tokens <- ai.StreamToken{Content: "hello"}
	go func() {
		time.Sleep(5 * time.Millisecond)
		tokens <- ai.StreamToken{Error: context.DeadlineExceeded, Done: true}
		close(tokens)
	}()

	var term streamTerminalCapture
	text, _, _, err := a.collectStreamTokens(context.Background(), msg, "stream-timeout-1", tokens, &term)
	if err != nil {
		t.Fatalf("unexpected hard error: %v text=%q", err, text)
	}
	if term.Reason != protocol.TerminalReasonTimeout {
		t.Fatalf("terminal=%q want timeout (text=%q)", term.Reason, text)
	}
}
