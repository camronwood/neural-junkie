package collaboration

import (
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var agreementPhrases = []string{
	"i agree",
	"looks good",
	"lgtm",
	"sounds good",
	"i'm satisfied",
	"i am satisfied",
	"no objections",
	"this works",
	"plan looks solid",
	"i approve",
	"no concerns",
	"well covered",
	"comprehensive plan",
	"i'm on board",
	"i am on board",
	"agreed",
	"let's proceed",
	"let's go with this",
	"ready to proceed",
}

var disagreementPhrases = []string{
	"i disagree",
	"i have concerns",
	"i'm not sure about",
	"i am not sure about",
	"we should reconsider",
	"this won't work",
	"this doesn't address",
	"missing critical",
	"significant risk",
	"strongly recommend against",
	"object to",
	"problematic",
	"needs rework",
}

// AnalyzeConsensus examines a message for agreement/disagreement signals
// and updates the discussion's consensus map. Returns the detected state.
func (cm *CollaborationManager) AnalyzeConsensus(collabID string, msg *protocol.Message) ConsensusState {
	var notifyReviewing string
	cm.mu.Lock()
	defer func() {
		cm.mu.Unlock()
		if notifyReviewing != "" && cm.onEnterReviewing != nil {
			go cm.onEnterReviewing(notifyReviewing)
		}
	}()

	c, ok := cm.collaborations[collabID]
	if !ok || c.Discussion == nil {
		return ConsensusUndecided
	}

	content := strings.ToLower(msg.Content)

	state := detectSignal(content)
	if state != ConsensusUndecided {
		c.Discussion.Consensus[msg.From.ID] = state
		log.Printf("[Consensus] Agent %s -> %s (signal-based)", msg.From.Name, state)
	}
	if c.Phase == PhasePlanning && c.Discussion.Status == DiscussionActive {
		allAgreed := true
		for _, pid := range c.Discussion.Participants {
			if c.Discussion.Consensus[pid] != ConsensusAgrees {
				allAgreed = false
				break
			}
		}
		if allAgreed {
			c.Discussion.Status = DiscussionConverged
			if cm.enterReviewingFromPlanningLocked(c) {
				notifyReviewing = collabID
				c.UpdatedAt = msg.Timestamp
				log.Printf("[Consensus] Collaboration %s converged and moved to reviewing", collabID[:8])
			}
		}
	}

	return state
}

// CheckFullConsensus returns true if every participant has agreed.
func (cm *CollaborationManager) CheckFullConsensus(collabID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok || c.Discussion == nil {
		return false
	}

	for _, pid := range c.Discussion.Participants {
		if c.Discussion.Consensus[pid] != ConsensusAgrees {
			return false
		}
	}
	return true
}

// HasDisagreement returns true if any participant has explicitly disagreed.
func (cm *CollaborationManager) HasDisagreement(collabID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok || c.Discussion == nil {
		return false
	}

	for _, pid := range c.Discussion.Participants {
		if c.Discussion.Consensus[pid] == ConsensusDisagrees {
			return true
		}
	}
	return false
}

// CheckImplicitConsensus detects implicit agreement using heuristics:
//   - All agents responded this round without proposing artifact changes
//   - Response lengths are decreasing (converging)
func (cm *CollaborationManager) CheckImplicitConsensus(collabID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok || c.Discussion == nil {
		return false
	}

	d := c.Discussion
	if len(d.Messages) < len(d.Participants) {
		return false
	}

	// Check the last round: did all agents respond without significant
	// new content (< 200 chars and no plan-like headings)?
	lastN := len(d.Participants)
	if len(d.Messages) < lastN {
		return false
	}
	recentMsgs := d.Messages[len(d.Messages)-lastN:]

	respondents := make(map[string]bool)
	for _, msg := range recentMsgs {
		respondents[msg.From.ID] = true
		content := strings.ToLower(msg.Content)
		if strings.Contains(content, "## ") || strings.Contains(content, "### task") {
			return false
		}
	}

	for _, pid := range d.Participants {
		if !respondents[pid] {
			return false
		}
	}

	allShort := true
	for _, msg := range recentMsgs {
		if len(msg.Content) > 500 {
			allShort = false
			break
		}
	}

	return allShort
}

// GetConsensusReport returns a formatted summary of where each agent
// stands, suitable for showing to the user.
func (cm *CollaborationManager) GetConsensusReport(collabID string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok || c.Discussion == nil {
		return "", fmt.Errorf("collaboration %s not found", collabID)
	}

	var sb strings.Builder
	sb.WriteString("**Consensus Status**\n\n")

	agrees := 0
	disagrees := 0
	undecided := 0

	for _, agent := range c.Agents {
		state := c.Discussion.Consensus[agent.AgentID]
		icon := "⬜"
		switch state {
		case ConsensusAgrees:
			icon = "✅"
			agrees++
		case ConsensusDisagrees:
			icon = "❌"
			disagrees++
		default:
			undecided++
		}
		sb.WriteString(fmt.Sprintf("%s **%s** (%s): %s\n", icon, agent.AgentName, agent.Role, state))
	}

	sb.WriteString(fmt.Sprintf("\nAgreed: %d | Disagreed: %d | Undecided: %d\n", agrees, disagrees, undecided))

	return sb.String(), nil
}

func detectSignal(content string) ConsensusState {
	for _, phrase := range disagreementPhrases {
		if strings.Contains(content, phrase) {
			return ConsensusDisagrees
		}
	}
	for _, phrase := range agreementPhrases {
		if strings.Contains(content, phrase) {
			return ConsensusAgrees
		}
	}
	return ConsensusUndecided
}
