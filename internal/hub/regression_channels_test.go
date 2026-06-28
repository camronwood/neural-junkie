package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHubChannelAllowsIDEAutoApprove(t *testing.T) {
	msg := &protocol.Message{Metadata: map[string]interface{}{"implementation_session": true}}
	if hubChannelAllowsIDEAutoApprove("chat-scenarios", msg) {
		t.Fatal("chat-scenarios must not auto-approve")
	}
	if !hubChannelAllowsIDEAutoApprove("implement-scenarios", msg) {
		t.Fatal("implement-scenarios should auto-approve")
	}
	if !hubChannelAllowsIDEAutoApprove("general", msg) {
		t.Fatal("normal channels should auto-approve when trust allows")
	}
}
