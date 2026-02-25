package collaboration

import (
	"fmt"
	"log"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// RecordMessage records a new agent message in the discussion, advances
// turn state, and returns nil if the message was accepted. If the
// discussion has ended (budget/timeout/convergence), it returns an error
// describing why.
func (cm *CollaborationManager) RecordMessage(collabID string, msg *protocol.Message) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Discussion == nil {
		return fmt.Errorf("no active discussion for collaboration %s", collabID)
	}

	d := c.Discussion
	if d.Status != DiscussionActive {
		return fmt.Errorf("discussion is %s, not accepting messages", d.Status)
	}

	if time.Since(d.StartedAt) > d.Timeout {
		d.Status = DiscussionTimedOut
		if c.Phase == PhasePlanning {
			c.Phase = PhaseReviewing
		}
		c.UpdatedAt = time.Now()
		log.Printf("[Discussion %s] Timed out after %v", d.ID[:8], d.Timeout)
		return fmt.Errorf("discussion timed out")
	}

	d.Messages = append(d.Messages, msg)
	d.TotalMessageCount++
	d.TurnsThisRound[msg.From.ID]++

	if d.TotalMessageCount >= d.MaxTotalMessages {
		d.Status = DiscussionBudgetExhausted
		if c.Phase == PhasePlanning {
			c.Phase = PhaseReviewing
		}
		c.UpdatedAt = time.Now()
		log.Printf("[Discussion %s] Budget exhausted (%d/%d messages)", d.ID[:8], d.TotalMessageCount, d.MaxTotalMessages)
		return nil
	}

	cm.advanceTurn(d)
	if d.Status != DiscussionActive && c.Phase == PhasePlanning {
		c.Phase = PhaseReviewing
	}
	c.UpdatedAt = time.Now()
	return nil
}

// IsAgentTurn returns true if the given agent is allowed to speak next
// in the discussion. An agent may speak if:
//   - It is their turn in the round-robin, OR
//   - They were @mentioned in the latest message (out-of-turn reply)
//
// Both paths are still subject to per-round and total message budgets.
func (cm *CollaborationManager) IsAgentTurn(collabID, agentID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok || c.Discussion == nil {
		return false
	}
	d := c.Discussion

	if d.Status != DiscussionActive {
		return false
	}

	if time.Since(d.StartedAt) > d.Timeout {
		return false
	}

	if d.TurnsThisRound[agentID] >= d.TurnBudget {
		return false
	}

	if d.CurrentTurnIndex < len(d.Participants) && d.Participants[d.CurrentTurnIndex] == agentID {
		return true
	}

	// Allow out-of-turn responses when directly @mentioned, but only if
	// the agent hasn't exhausted their per-round budget.
	if len(d.Messages) > 0 {
		last := d.Messages[len(d.Messages)-1]
		if last.IsMentioned(agentID) {
			return true
		}
	}

	return false
}

// GetDiscussionStatus returns the current status of a discussion.
func (cm *CollaborationManager) GetDiscussionStatus(collabID string) (DiscussionStatus, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return "", fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Discussion == nil {
		return "", fmt.Errorf("no discussion for collaboration %s", collabID)
	}
	return c.Discussion.Status, nil
}

// GetCurrentTurnAgent returns the agent ID of whoever should speak next.
func (cm *CollaborationManager) GetCurrentTurnAgent(collabID string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return "", fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Discussion == nil || len(c.Discussion.Participants) == 0 {
		return "", fmt.Errorf("no active discussion")
	}
	d := c.Discussion
	if d.CurrentTurnIndex >= len(d.Participants) {
		return d.Participants[0], nil
	}
	return d.Participants[d.CurrentTurnIndex], nil
}

// CheckTimeout checks whether the discussion has exceeded its wall-clock
// timeout and updates the status accordingly. Returns true if timed out.
func (cm *CollaborationManager) CheckTimeout(collabID string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok || c.Discussion == nil {
		return false
	}
	d := c.Discussion
	if d.Status != DiscussionActive {
		return d.Status == DiscussionTimedOut
	}
	if time.Since(d.StartedAt) > d.Timeout {
		d.Status = DiscussionTimedOut
		return true
	}
	return false
}

// EndDiscussion forcefully ends the active discussion with the given status.
func (cm *CollaborationManager) EndDiscussion(collabID string, status DiscussionStatus) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Discussion == nil {
		return fmt.Errorf("no discussion for collaboration %s", collabID)
	}
	c.Discussion.Status = status
	c.UpdatedAt = time.Now()
	return nil
}

// advanceTurn moves to the next participant in round-robin order.
// When all participants have used their turn budget for the current
// round, it advances to the next round (or ends the discussion).
// Must be called with cm.mu held.
func (cm *CollaborationManager) advanceTurn(d *DiscussionSession) {
	d.CurrentTurnIndex++
	if d.CurrentTurnIndex >= len(d.Participants) {
		d.CurrentTurnIndex = 0
	}

	allDone := true
	for _, pid := range d.Participants {
		if d.TurnsThisRound[pid] < d.TurnBudget {
			allDone = false
			break
		}
	}

	if allDone {
		nextRound := d.CurrentRound + 1
		if nextRound > d.MaxRounds {
			// Clamp at max so UI/status never shows impossible values (e.g. 4/3).
			d.CurrentRound = d.MaxRounds
			d.Status = DiscussionBudgetExhausted
			log.Printf("[Discussion %s] All %d rounds completed", d.ID[:8], d.MaxRounds)
			return
		}
		d.CurrentRound = nextRound
		d.TurnsThisRound = make(map[string]int)
		d.CurrentTurnIndex = 0
		log.Printf("[Discussion %s] Advanced to round %d/%d", d.ID[:8], d.CurrentRound, d.MaxRounds)
	}

	// Skip ahead if the next agent has already exhausted their budget
	// (can happen with out-of-turn @mention responses).
	for i := 0; i < len(d.Participants); i++ {
		next := d.Participants[d.CurrentTurnIndex]
		if d.TurnsThisRound[next] < d.TurnBudget {
			break
		}
		d.CurrentTurnIndex = (d.CurrentTurnIndex + 1) % len(d.Participants)
	}
}

// GetDiscussionSummary returns a textual summary of the discussion
// suitable for presenting to the user or injecting into prompts.
func (cm *CollaborationManager) GetDiscussionSummary(collabID string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return "", fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Discussion == nil {
		return "", fmt.Errorf("no discussion for collaboration %s", collabID)
	}

	d := c.Discussion
	summary := fmt.Sprintf("Discussion: %s\nStatus: %s\nRounds: %d/%d\nMessages: %d/%d\n\n",
		d.Topic, d.Status, d.CurrentRound, d.MaxRounds,
		d.TotalMessageCount, d.MaxTotalMessages)

	for _, msg := range d.Messages {
		summary += fmt.Sprintf("[%s]: %s\n\n", msg.From.Name, msg.Content)
	}

	return summary, nil
}
