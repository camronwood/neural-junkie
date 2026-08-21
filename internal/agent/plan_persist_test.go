package agent

import (
	"sync"
	"testing"

	"github.com/camronwood/neural-junkie/internal/artifacts"
	"github.com/camronwood/neural-junkie/internal/plans"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestStampPersistedPlan_pseudoYAML(t *testing.T) {
	restore := setPlanStoreForTest(plans.NewStore(t.TempDir()))
	defer restore()

	artStore, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = artStore })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	inbound := &protocol.Message{
		Metadata: map[string]interface{}{"editor_mode": "plan", "composer_mode": "plan"},
	}
	reply := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{Name: "BackendEngineer"}, "x")
	pseudo := `yaml plan: hello-world actions: - description: Add HelloWorld to docs/index.html`
	prepared := plans.PrepareMarkdown(pseudo)
	stampPersistedPlan(inbound, reply, prepared)
	if reply.Metadata[protocol.MetaPlanID] == nil {
		t.Fatalf("expected plan_id after normalize, metadata=%v", reply.Metadata)
	}
	artRef, ok := reply.Metadata["artifact_ref"].(protocol.ArtifactReference)
	if !ok || artRef.ID == "" {
		t.Fatalf("expected artifact_ref, metadata=%v", reply.Metadata)
	}
}

func TestStampPersistedPlan(t *testing.T) {
	restore := setPlanStoreForTest(plans.NewStore(t.TempDir()))
	defer restore()

	artStore, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = artStore })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

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

	// The artifact_ref must also be stamped so the desktop can open a canvas tab.
	artRef, hasArtRef := reply.Metadata["artifact_ref"].(protocol.ArtifactReference)
	if !hasArtRef {
		t.Fatalf("expected artifact_ref in metadata, got %T", reply.Metadata["artifact_ref"])
	}
	if artRef.ID == "" {
		t.Fatalf("artifact_ref.ID is empty")
	}
	if artRef.RendererID != "nj.document" {
		t.Fatalf("artifact_ref.RendererID=%q, want nj.document", artRef.RendererID)
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
