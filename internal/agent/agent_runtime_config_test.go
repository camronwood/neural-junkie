package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAgentRuntimeV2ForMessage_metadataOverride(t *testing.T) {
	msg := &protocol.Message{
		Metadata: map[string]interface{}{"agent_runtime_v2": false},
	}
	if agentRuntimeV2ForMessage(msg) {
		t.Fatal("expected v2 disabled via metadata")
	}
	msg.Metadata["agent_runtime_v2"] = true
	if !agentRuntimeV2ForMessage(msg) {
		t.Fatal("expected v2 enabled via metadata")
	}
}

func TestImplSessionLimits_v2RaisesCaps(t *testing.T) {
	msg := &protocol.Message{Metadata: map[string]interface{}{"agent_runtime_v2": true}}
	maxTool, maxRounds, maxFiles := implSessionLimits(msg)
	if maxTool < agentRuntimeMaxToolIterations {
		t.Fatalf("tool iter %d want >= %d", maxTool, agentRuntimeMaxToolIterations)
	}
	if maxFiles < agentRuntimeMaxFilesPerCycle {
		t.Fatalf("max files %d want >= %d", maxFiles, agentRuntimeMaxFilesPerCycle)
	}
	if maxRounds < implSessionMaxEditRounds {
		t.Fatalf("edit rounds %d should exceed legacy %d", maxRounds, implSessionMaxEditRounds)
	}
}

func TestImplSessionLimits_legacyCaps(t *testing.T) {
	msg := &protocol.Message{Metadata: map[string]interface{}{"agent_runtime_v2": false}}
	maxTool, maxRounds, maxFiles := implSessionLimits(msg)
	if maxTool != implSessionMaxToolIterations || maxRounds != implSessionMaxEditRounds || maxFiles != implSessionMaxFiles {
		t.Fatalf("legacy limits: tool=%d rounds=%d files=%d", maxTool, maxRounds, maxFiles)
	}
}
