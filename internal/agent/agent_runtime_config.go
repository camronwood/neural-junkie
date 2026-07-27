package agent

import (
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func isImplementScenariosChannel(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	ch := strings.TrimSpace(msg.Channel)
	switch ch {
	case "implement-scenarios", "user-flow-scenarios", "parity-scenarios":
		return true
	default:
		return false
	}
}

// scenarioHarnessRequestsImplementationSession reports harness metadata that must not be
// demoted to advisory Answer by applyChatModeAdvisoryGoal (message-only; any specialist).
func scenarioHarnessRequestsImplementationSession(msg *protocol.Message) bool {
	if msg == nil || !isImplementScenariosChannel(msg) || !msg.ImplementationSession() {
		return false
	}
	return msg.IdeEditorModeIsAgent() && !msg.IdeEditorModeIsAsk() && !msg.IdeEditorModeIsPlan()
}

// scenarioHarnessForcesImplementationSession reports regression harness turns that must
// enter the bounded implementation loop regardless of classifier/advisory gates.
func scenarioHarnessForcesImplementationSession(a *Agent, msg *protocol.Message) bool {
	if !scenarioHarnessRequestsImplementationSession(msg) {
		return false
	}
	if a == nil {
		return false
	}
	if !agentTypeCanShipFileChanges(a.Info.Type) || a.Info.Type == protocol.AgentTypeCodeReview {
		return false
	}
	return !chatModeBlocksImplementationSession(a, msg)
}

const (
	agentRuntimeMaxToolIterations  = 24
	agentRuntimeMaxRepairRounds    = 3
	agentRuntimeMaxFilesPerCycle   = 8
	agentRuntimeMaxSessionTimeout  = 10 * time.Minute
	agentRuntimeMaxFrontendTimeout = 12 * time.Minute

	// Live implement-scenarios use shorter caps than generic impl sessions so agents
	// finish and post a reply before the shortest scenario wait_reply (420s).
	implScenarioSessionTimeout         = 360 * time.Second
	implScenarioSessionFrontendTimeout = 540 * time.Second
)

func agentRuntimeV2Enabled() bool {
	if cfg := mcp.AppConfig(); cfg != nil {
		return cfg.AgentRuntimeV2Enabled()
	}
	return true
}

// chatToolLoopMaxIterations is the default tool-use loop cap for normal chat/MCP turns.
func chatToolLoopMaxIterations() int {
	maxIter := agentRuntimeMaxToolIterations
	if n := performanceFromHub().AgentMaxStepsOrDefault(); n > 0 && n < maxIter {
		maxIter = n
	}
	return maxIter
}

func agentRuntimeV2ForMessage(msg *protocol.Message) bool {
	if msg != nil && msg.Metadata != nil {
		if v, ok := msg.Metadata["agent_runtime_v2"].(bool); ok && !v {
			return false
		}
	}
	return agentRuntimeV2Enabled()
}

func implSessionLimits(msg *protocol.Message) (maxToolIter, maxEditRounds, maxFiles int) {
	// Live implement-scenarios must stay within legacy caps so sessions finish before wait_reply.
	if isImplementScenariosChannel(msg) {
		return implSessionMaxToolIterations, implSessionMaxEditRounds, implSessionMaxFiles
	}
	if agentRuntimeV2ForMessage(msg) {
		perf := performanceFromHub()
		maxToolIter = perf.AgentMaxStepsOrDefault()
		if maxToolIter <= 0 || maxToolIter > agentRuntimeMaxToolIterations {
			maxToolIter = agentRuntimeMaxToolIterations
		}
		return maxToolIter, agentRuntimeMaxRepairRounds, agentRuntimeMaxFilesPerCycle
	}
	return implSessionMaxToolIterations, implSessionMaxEditRounds, implSessionMaxFiles
}

func implSessionTimeoutForMessage(msg *protocol.Message, frontend bool) time.Duration {
	// Live implement-scenarios must finish within scenario wait_reply windows (≤1200s),
	// not the open-ended agent-runtime v2 default (up to 180m).
	if isImplementScenariosChannel(msg) {
		if frontend {
			return implScenarioSessionFrontendTimeout
		}
		return implScenarioSessionTimeout
	}
	if agentRuntimeV2ForMessage(msg) {
		timeout := performanceFromHub().AgentTimeout()
		cap := agentRuntimeMaxSessionTimeout
		if frontend {
			cap = agentRuntimeMaxFrontendTimeout
		}
		if timeout > cap {
			return cap
		}
		return timeout
	}
	if frontend {
		return implSessionFrontendTimeout
	}
	return implSessionTimeout
}
