package agent

import (
	"strings"
	"time"
	"unicode"

	"github.com/camronwood/neural-junkie/internal/intent"
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
	return agentComposerCanForceImplementationSession(a, msg)
}

// looksLikeExecuteImplementationAsk is a direct "do the work" ask (not a design question).
// Used for real desktop Agent-mode turns that omit implementation_session metadata.
func looksLikeExecuteImplementationAsk(text string) bool {
	if !intent.LooksLikeDirectImplementationAsk(text) && !intent.PrefersWorkspaceImplementOverOpenCanvas(text) {
		return false
	}
	// "What would you implement first?" stays advisory.
	if strings.Contains(text, "?") {
		return false
	}
	lower := strings.ToLower(text)
	// Canvas / diagram / mermaid asks are artifact delivery, not workspace implement.
	if strings.Contains(lower, "canvas") || strings.Contains(lower, "mermaid") ||
		strings.Contains(lower, "diagram") {
		return false
	}
	return true
}

// looksLikeImplementGoAhead reports short user affirmations after a plan / implement ask.
func looksLikeImplementGoAhead(text string) bool {
	c := strings.ToLower(strings.TrimSpace(text))
	if c == "" || strings.Contains(c, "?") {
		return false
	}
	fields := strings.FieldsFunc(c, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == '!' || r == '.'
	})
	if len(fields) == 0 || len(fields) > 10 {
		return false
	}
	switch c {
	case "ok", "okay", "k", "yes", "yep", "yeah", "y", "sure", "do it", "ship it",
		"go ahead", "go for it", "lgtm", "sounds good", "that sounds good",
		"ok that sounds good", "okay that sounds good", "looks good", "proceed":
		return true
	}
	if strings.HasPrefix(c, "ok ") || strings.HasPrefix(c, "okay ") {
		if strings.Contains(c, "sound") || strings.Contains(c, "good") || strings.Contains(c, "go ahead") {
			return true
		}
	}
	return false
}

func channelHasRecentExecuteImplementationAsk(history []*protocol.Message, skipMsgID string) bool {
	seen := 0
	for i := len(history) - 1; i >= 0 && seen < 12; i-- {
		m := history[i]
		if m == nil || m.ID == skipMsgID {
			continue
		}
		if !protocol.IsUserLikeSender(m.From) {
			continue
		}
		seen++
		if messageStampedImplAction(m) || looksLikeExecuteImplementationAsk(m.Content) {
			return true
		}
	}
	return false
}

func agentComposerCanForceImplementationSession(a *Agent, msg *protocol.Message) bool {
	if a == nil || msg == nil {
		return false
	}
	if msg.IdeEditorModeIsAsk() || msg.IdeEditorModeIsPlan() {
		return false
	}
	mode := msg.IdeEditorMode()
	if mode != "" && mode != "agent" && mode != "export" {
		return false
	}
	if !agentTypeCanShipFileChanges(a.Info.Type) || a.Info.Type == protocol.AgentTypeCodeReview {
		return false
	}
	return !chatModeBlocksImplementationSession(a, msg)
}

// realUIForcesImplementationSession covers desktop Agent-mode turns that omit the
// harness-only implementation_session flag (greenfield implement asks + go-aheads).
// Export mode is not forced here — save/export follows existing export-session rules
// so travel/chat asks with sticky export metadata cannot enter an impl loop.
func realUIForcesImplementationSession(a *Agent, msg *protocol.Message) bool {
	if !agentComposerCanForceImplementationSession(a, msg) {
		return false
	}
	if msg.IdeEditorModeIsExport() {
		return false
	}
	if looksLikeExecuteImplementationAsk(msg.Content) {
		return true
	}
	if looksLikeImplementGoAhead(msg.Content) &&
		channelHasRecentExecuteImplementationAsk(a.channelHistorySafe(msg.Channel), msg.ID) {
		return true
	}
	return false
}

// forcesImplementationSession reports harness or real-UI turns that must enter the
// bounded implementation loop even when the classifier stamps advisory_question.
func forcesImplementationSession(a *Agent, msg *protocol.Message) bool {
	return scenarioHarnessForcesImplementationSession(a, msg) || realUIForcesImplementationSession(a, msg)
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
