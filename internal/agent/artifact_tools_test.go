package agent

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/camronwood/neural-junkie/internal/artifacts"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestExecuteCreateArtifactTool(t *testing.T) {
	store, err := artifacts.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agentArtifactStoreOnce = sync.Once{}
	agentArtifactStore = nil
	agentArtifactStoreErr = nil
	agentArtifactStoreOnce.Do(func() { agentArtifactStore = store })
	t.Cleanup(func() {
		agentArtifactStoreOnce = sync.Once{}
		agentArtifactStore = nil
		agentArtifactStoreErr = nil
	})

	a := &Agent{Info: protocol.AgentInfo{ID: "a-1", Name: "Analyst"}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{}, "report")
	result, err := a.executeArtifactTool(context.Background(), msg, createArtifactToolName, []byte(`{
		"title":"Latency report",
		"renderer_id":"nj.chart",
		"media_type":"application/vnd.neural-junkie.chart+json",
		"data":{"series":[{"name":"p95","points":[[1,20]]}]},
		"fallback":"Latency p95: 20 ms"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "Created Neural Canvas artifact") {
		t.Fatalf("result=%q", result)
	}
	items, err := store.List(artifacts.Filter{ChannelID: "general"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].Renderer.ID != "nj.chart" {
		t.Fatalf("renderer=%+v", items[0].Renderer)
	}
}
