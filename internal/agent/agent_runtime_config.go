package agent

import (
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	agentRuntimeMaxToolIterations = 100
	agentRuntimeMaxRepairRounds     = 5
	agentRuntimeMaxFilesPerCycle    = 50
)

func agentRuntimeV2Enabled() bool {
	if cfg := mcp.AppConfig(); cfg != nil {
		return cfg.AgentRuntimeV2Enabled()
	}
	return true
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
	if agentRuntimeV2ForMessage(msg) {
		perf := performanceFromHub()
		maxToolIter = perf.AgentMaxStepsOrDefault()
		if maxToolIter < agentRuntimeMaxToolIterations {
			maxToolIter = agentRuntimeMaxToolIterations
		}
		return maxToolIter, agentRuntimeMaxRepairRounds * 3, agentRuntimeMaxFilesPerCycle
	}
	return implSessionMaxToolIterations, implSessionMaxEditRounds, implSessionMaxFiles
}

func implSessionTimeoutForMessage(msg *protocol.Message, frontend bool) time.Duration {
	// Live implement-scenarios must finish within scenario wait_reply windows (≤1200s),
	// not the open-ended agent-runtime v2 default (up to 180m).
	if msg != nil && strings.TrimSpace(msg.Channel) == "implement-scenarios" {
		if frontend {
			return implSessionFrontendTimeout
		}
		return implSessionTimeout
	}
	if agentRuntimeV2ForMessage(msg) {
		return performanceFromHub().AgentTimeout()
	}
	if frontend {
		return implSessionFrontendTimeout
	}
	return implSessionTimeout
}
