package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestChannelAllowsImplementationSession_regressionChat(t *testing.T) {
	msg := &protocol.Message{Metadata: map[string]interface{}{}}
	if channelAllowsImplementationSession("chat-scenarios", msg) {
		t.Fatal("chat-scenarios without implementation_session should block impl loop")
	}
	msg.Metadata["implementation_session"] = true
	if !channelAllowsImplementationSession("chat-scenarios", msg) {
		t.Fatal("explicit implementation_session should allow on chat-scenarios")
	}
	if !channelAllowsImplementationSession("implement-scenarios", nil) {
		t.Fatal("implement-scenarios should always allow")
	}
}
