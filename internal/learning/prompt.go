package learning

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	globalStore    *Store
	enabledChecker func() bool
)

// SetGlobalStore wires the hub store for prompt injection.
func SetGlobalStore(s *Store) {
	globalStore = s
}

// SetEnabledChecker returns true when personal learning is pack-gated and opt-in on.
func SetEnabledChecker(fn func() bool) {
	enabledChecker = fn
}

// ForgetGlobal soft-deletes by id using the wired global store.
func ForgetGlobal(id string) error {
	if globalStore == nil {
		return fmt.Errorf("learning store unavailable")
	}
	return globalStore.Forget(id)
}

// AddGlobal saves via the wired global store.
func AddGlobal(e Entry) (Entry, error) {
	if globalStore == nil {
		return Entry{}, fmt.Errorf("learning store unavailable")
	}
	return globalStore.Add(e)
}

// ListGlobal lists active entries for optional agent_id.
func ListGlobal(agentID string) []Entry {
	if globalStore == nil {
		return nil
	}
	return globalStore.List(agentID)
}

func learningEnabled() bool {
	return enabledChecker != nil && enabledChecker()
}

// AppendForAgent writes confirmed learnings into the system prompt for an expert agent.
func AppendForAgent(system *strings.Builder, self *protocol.AgentInfo) (count int) {
	if system == nil || self == nil || !learningEnabled() || globalStore == nil {
		return 0
	}
	if !isExpertAgent(self.Type) {
		return 0
	}
	entries := globalStore.List(self.ID)
	if len(entries) == 0 {
		return 0
	}
	system.WriteString("\n=== LEARNINGS FOR THIS EXPERT (user-confirmed) ===\n")
	system.WriteString("The user explicitly approved these notes for you only. Apply when relevant.\n\n")
	budget := DefaultPromptBudget
	for _, e := range entries {
		line := fmt.Sprintf("- [%s] %s\n", e.Category, e.Content)
		if len(line) > budget {
			break
		}
		system.WriteString(line)
		budget -= len(line)
		count++
	}
	system.WriteString("\n=== END LEARNINGS FOR THIS EXPERT ===\n\n")
	return count
}

func isExpertAgent(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeCLI, protocol.AgentTypeModerator:
		return false
	default:
		return t != ""
	}
}
