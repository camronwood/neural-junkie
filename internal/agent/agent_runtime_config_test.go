package agent

import (
	"testing"
	"time"

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

func TestImplSessionLimits_v2UsesBoundedCaps(t *testing.T) {
	msg := &protocol.Message{Metadata: map[string]interface{}{"agent_runtime_v2": true}}
	maxTool, maxRounds, maxFiles := implSessionLimits(msg)
	if maxTool > agentRuntimeMaxToolIterations {
		t.Fatalf("tool iter %d exceeds cap %d", maxTool, agentRuntimeMaxToolIterations)
	}
	if maxFiles > agentRuntimeMaxFilesPerCycle {
		t.Fatalf("max files %d exceeds cap %d", maxFiles, agentRuntimeMaxFilesPerCycle)
	}
	if maxRounds > agentRuntimeMaxRepairRounds {
		t.Fatalf("edit rounds %d exceeds cap %d", maxRounds, agentRuntimeMaxRepairRounds)
	}
}

func TestImplSessionLimits_legacyCaps(t *testing.T) {
	msg := &protocol.Message{Metadata: map[string]interface{}{"agent_runtime_v2": false}}
	maxTool, maxRounds, maxFiles := implSessionLimits(msg)
	if maxTool != implSessionMaxToolIterations || maxRounds != implSessionMaxEditRounds || maxFiles != implSessionMaxFiles {
		t.Fatalf("legacy limits: tool=%d rounds=%d files=%d", maxTool, maxRounds, maxFiles)
	}
}

func TestImplSessionLimits_implementScenariosUsesLegacyCapEvenWithV2(t *testing.T) {
	msg := &protocol.Message{
		Channel:  "implement-scenarios",
		Metadata: map[string]interface{}{"agent_runtime_v2": true},
	}
	maxTool, maxRounds, maxFiles := implSessionLimits(msg)
	if maxTool != implSessionMaxToolIterations || maxRounds != implSessionMaxEditRounds || maxFiles != implSessionMaxFiles {
		t.Fatalf("implement-scenarios limits: tool=%d rounds=%d files=%d", maxTool, maxRounds, maxFiles)
	}
}

func TestImplSessionTimeoutForMessage_implementScenariosUsesLegacyCap(t *testing.T) {
	msg := &protocol.Message{
		Channel:  "implement-scenarios",
		Metadata: map[string]interface{}{"agent_runtime_v2": true},
	}
	if got := implSessionTimeoutForMessage(msg, true); got != implScenarioSessionFrontendTimeout {
		t.Fatalf("frontend timeout = %v, want %v", got, implScenarioSessionFrontendTimeout)
	}
	if got := implSessionTimeoutForMessage(msg, false); got != implScenarioSessionTimeout {
		t.Fatalf("backend timeout = %v, want %v", got, implScenarioSessionTimeout)
	}
	if implScenarioSessionTimeout >= 420*time.Second {
		t.Fatalf("implement-scenarios backend cap %v must stay below 420s wait_reply", implScenarioSessionTimeout)
	}
}
