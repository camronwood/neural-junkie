package learning

import (
	"strings"
	"sync"
	"time"
)

// ProposalEmitter sends a learning proposal to the desktop (wired by server).
type ProposalEmitter func(channel, agentID, agentName, agentType, draft, category, sourceMsgID, source string)

var (
	proposalEmitter      ProposalEmitter
	suggestEnabledChecker func() bool
	suggestMu            sync.Mutex
	lastSuggest     = map[string]time.Time{}
	suggestCooldown = 5 * time.Minute
)

func SetProposalEmitter(fn ProposalEmitter) {
	proposalEmitter = fn
}

func SetSuggestEnabledChecker(fn func() bool) {
	suggestEnabledChecker = fn
}

func SetSuggestCooldown(d time.Duration) {
	if d > 0 {
		suggestCooldown = d
	}
}

// MaybeSuggestAfterAgentReply emits a proposal when heuristics match (never auto-saves).
func MaybeSuggestAfterAgentReply(channel, agentID, agentName, agentType string, userMsg, agentReply string) {
	if suggestEnabledChecker == nil || !suggestEnabledChecker() {
		return
	}
	if !learningEnabled() || proposalEmitter == nil {
		return
	}
	userMsg = strings.TrimSpace(userMsg)
	agentReply = strings.TrimSpace(agentReply)
	if userMsg == "" || len(userMsg) < 12 {
		return
	}
	if len(agentReply) < 20 {
		return
	}
	draft := ExtractDraftFromMessage(userMsg)
	if draft == "" {
		// preference-like short user statements
		lower := strings.ToLower(userMsg)
		if !strings.Contains(lower, "prefer") && !strings.Contains(lower, "always") && !strings.Contains(lower, "never") {
			return
		}
		draft = userMsg
	}
	key := channel + ":" + agentID
	suggestMu.Lock()
	if t, ok := lastSuggest[key]; ok && time.Since(t) < suggestCooldown {
		suggestMu.Unlock()
		return
	}
	lastSuggest[key] = time.Now()
	suggestMu.Unlock()
	proposalEmitter(channel, agentID, agentName, agentType, draft, string(CategoryPreference), "", "agent_suggest")
}
