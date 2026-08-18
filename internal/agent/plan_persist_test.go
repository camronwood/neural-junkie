package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/plans"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestStampPersistedPlan(t *testing.T) {
	restore := setPlanStoreForTest(plans.NewStore(t.TempDir()))
	defer restore()

	inbound := &protocol.Message{
		Metadata: map[string]interface{}{"editor_mode": "plan", "composer_mode": "plan"},
	}
	reply := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{Name: "BackendEngineer"}, "x")
	markdown := `---
name: HelloWorld plan
overview: Add a HelloWorld helper.
todos:
  - id: add-fn
    content: Add HelloWorld in core/sample/main.go
    status: pending
isProject: false
---

# HelloWorld

## Out of scope

- Tests
`
	stampPersistedPlan(inbound, reply, markdown)
	id, _ := reply.Metadata[protocol.MetaPlanID].(string)
	if id == "" {
		t.Fatalf("expected plan_id, metadata=%v", reply.Metadata)
	}

	askIn := &protocol.Message{Metadata: map[string]interface{}{"editor_mode": "ask"}}
	askOut := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{}, "x")
	stampPersistedPlan(askIn, askOut, markdown)
	if askOut.Metadata[protocol.MetaPlanID] != nil {
		t.Fatal("ask mode must not persist a plan")
	}

	plain := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{}, "x")
	stampPersistedPlan(inbound, plain, "just numbered steps\n1. do it")
	if plain.Metadata[protocol.MetaPlanID] != nil {
		t.Fatal("non-plan markdown must not persist")
	}
}
