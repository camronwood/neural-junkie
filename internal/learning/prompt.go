package learning

import (
	"fmt"
	"os"
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

func learningEnabled() bool {
	return enabledChecker != nil && enabledChecker()
}

// ForgetGlobal soft-deletes by id using the wired global store.
func ForgetGlobal(id string) error {
	if globalStore == nil {
		return fmt.Errorf("learning store unavailable")
	}
	if globalEmbedStore != nil {
		globalEmbedStore.Delete(id)
	}
	return globalStore.Forget(id)
}

// AddGlobal saves via the wired global store.
func AddGlobal(e Entry) (Entry, error) {
	if globalStore == nil {
		return Entry{}, fmt.Errorf("learning store unavailable")
	}
	out, err := globalStore.Add(e)
	if err != nil {
		return Entry{}, err
	}
	ScheduleEmbed(out)
	return out, nil
}

// ListGlobal lists active entries for optional agent_id.
func ListGlobal(agentID string) []Entry {
	if globalStore == nil {
		return nil
	}
	return globalStore.List(agentID)
}

// AppendForAgent writes retrieved learnings into the system prompt for an expert agent.
func AppendForAgent(system *strings.Builder, self *protocol.AgentInfo, pctx PromptContext) PromptResult {
	var res PromptResult
	if system == nil || self == nil || !learningEnabled() || globalStore == nil {
		return res
	}
	if !isExpertAgent(self.Type) {
		return res
	}
	if pctx.CollaborationID == "" && pctx.Channel != "" {
		pctx.CollaborationID = ResolveCollabID(pctx.Channel)
	}
	globalEntries, agentEntries, collabEntries, workspaceEntries, ids := SelectForPrompt(nil, pctx, self.ID)
	res.IDs = ids

	writeSection := func(title, endTitle, hint string, entries []Entry, budget int) {
		if len(entries) == 0 {
			return
		}
		system.WriteString("\n=== " + title + " ===\n")
		system.WriteString(hint + "\n\n")
		for _, e := range entries {
			line := fmt.Sprintf("- [%s] %s\n", e.Category, e.Content)
			if len(line) > budget {
				break
			}
			system.WriteString(line)
			budget -= len(line)
			res.Count++
		}
		system.WriteString("\n=== END " + endTitle + " ===\n\n")
	}

	writeSection(
		"LEARNINGS FOR ALL EXPERTS (user-confirmed)",
		"LEARNINGS FOR ALL EXPERTS",
		"The user approved these notes for every expert. Apply when relevant.",
		globalEntries,
		DefaultGlobalBudget,
	)
	writeSection(
		"LEARNINGS FOR THIS EXPERT (user-confirmed)",
		"LEARNINGS FOR THIS EXPERT",
		"The user explicitly approved these notes for you only. Apply when relevant.",
		agentEntries,
		DefaultPromptBudget,
	)
	writeSection(
		"LEARNINGS FOR THIS COLLABORATION (user-confirmed)",
		"LEARNINGS FOR THIS COLLABORATION",
		"The user approved these notes for this collaboration context. Apply when relevant.",
		collabEntries,
		DefaultCollabBudget,
	)
	if len(workspaceEntries) > 0 {
		writeSection(
			"LEARNINGS FOR THIS WORKSPACE (user-confirmed)",
			"LEARNINGS FOR THIS WORKSPACE",
			"The user approved these notes for this project workspace. Apply when relevant.",
			workspaceEntries,
			DefaultCollabBudget,
		)
	}

	if os.Getenv("NEURAL_JUNKIE_DEBUG") == "1" && len(res.IDs) > 0 {
		// caller attaches to metadata
	}
	return res
}

func isExpertAgent(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeCLI:
		return false
	default:
		return t != ""
	}
}
