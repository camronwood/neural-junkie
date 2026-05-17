package collaboration

import (
	"fmt"
	"log"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (cm *CollaborationManager) postDiscussionLimitNotice(collabID string) {
	go func() {
		cm.mu.RLock()
		c, ok := cm.collaborations[collabID]
		if !ok || c == nil || c.Channel == "" {
			cm.mu.RUnlock()
			return
		}
		phase := c.Phase
		ch := c.Channel
		prefix := collabID
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		cm.mu.RUnlock()

		if phase != PhasePlanning && phase != PhaseReviewing {
			return
		}

		txt := fmt.Sprintf(
			"📊 **Collaboration discussion limits reached** (`%s`)\n\n"+
				"Raise caps: `/collab-extend %s --rounds <n> --messages <m>` (each flag **adds** to the current max; hard caps %d rounds / %d messages per session).\n"+
				"Stop the session: `/cancel-plan %s`",
			prefix, prefix, HardMaxRounds, HardMaxTotalMessages, prefix,
		)
		msg := protocol.NewMessage(
			protocol.MessageTypeSystemInfo,
			ch,
			protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
			txt,
		)
		msg.SetCollaborationID(collabID)
		msg.SetCollaborationPhase(string(phase))
		if err := cm.hub.SendMessage(msg); err != nil {
			log.Printf("[Collaboration] limit notice send failed: %v", err)
		}
	}()
}

// RecordMessage records a new agent message in the discussion, advances
// turn state, and returns nil if the message was accepted. If the
// discussion has ended (budget/timeout/convergence), it returns an error
// describing why.
func (cm *CollaborationManager) RecordMessage(collabID string, msg *protocol.Message) error {
	var notifyBudget bool

	cm.mu.Lock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		cm.mu.Unlock()
		return fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Discussion == nil {
		cm.mu.Unlock()
		return fmt.Errorf("no active discussion for collaboration %s", collabID)
	}

	d := c.Discussion
	if d.Status != DiscussionActive {
		cm.mu.Unlock()
		return fmt.Errorf("discussion is %s, not accepting messages", d.Status)
	}

	if time.Since(d.StartedAt) > d.Timeout {
		d.Status = DiscussionTimedOut
		if c.Phase == PhasePlanning {
			c.Phase = PhaseReviewing
		}
		c.UpdatedAt = time.Now()
		log.Printf("[Discussion %s] Timed out after %v", d.ID[:8], d.Timeout)
		cm.mu.Unlock()
		return fmt.Errorf("discussion timed out")
	}

	enforced := c.DiscussionBudgetEnforced()

	d.Messages = append(d.Messages, msg)
	d.TotalMessageCount++
	d.TurnsThisRound[msg.From.ID]++

	if enforced && d.TotalMessageCount >= d.MaxTotalMessages {
		d.Status = DiscussionBudgetExhausted
		if c.Phase == PhasePlanning {
			c.Phase = PhaseReviewing
		}
		cm.synthesizePlanFromDiscussionLocked(c)
		c.UpdatedAt = time.Now()
		log.Printf("[Discussion %s] Budget exhausted (%d/%d messages)", d.ID[:8], d.TotalMessageCount, d.MaxTotalMessages)
		notifyBudget = true
		cm.mu.Unlock()
		if notifyBudget {
			cm.postDiscussionLimitNotice(collabID)
		}
		return nil
	}

	cm.advanceTurn(c)
	if d.Status != DiscussionActive && c.Phase == PhasePlanning && enforced {
		c.Phase = PhaseReviewing
	}
	if enforced && d.Status == DiscussionBudgetExhausted {
		notifyBudget = true
	}
	c.UpdatedAt = time.Now()
	cm.mu.Unlock()

	if notifyBudget {
		cm.postDiscussionLimitNotice(collabID)
	}
	return nil
}

// IsAgentTurn returns true if the given agent is allowed to speak next
// in the discussion. An agent may speak if:
//   - It is their turn in the round-robin, OR
//   - They were @mentioned in the latest message (out-of-turn reply)
//
// Planning/review phases apply per-round and total message budgets.
// Execution (after plan approval) does not cap discussion.
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

	if !c.DiscussionBudgetEnforced() {
		if len(d.Participants) == 0 {
			return false
		}
		if d.CurrentTurnIndex < len(d.Participants) && d.Participants[d.CurrentTurnIndex] == agentID {
			return true
		}
		if len(d.Messages) > 0 {
			last := d.Messages[len(d.Messages)-1]
			if last.IsMentioned(agentID) {
				return true
			}
		}
		return false
	}

	if d.TurnsThisRound[agentID] >= d.TurnBudget {
		return false
	}

	if d.CurrentTurnIndex < len(d.Participants) && d.Participants[d.CurrentTurnIndex] == agentID {
		return true
	}

	if len(d.Messages) > 0 {
		last := d.Messages[len(d.Messages)-1]
		if last.IsMentioned(agentID) {
			return true
		}
	}

	return false
}

// GetDiscussionStatus returns the status of a discussion.
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
func (cm *CollaborationManager) advanceTurn(c *Collaboration) {
	d := c.Discussion
	if d == nil || len(d.Participants) == 0 {
		return
	}

	if !c.DiscussionBudgetEnforced() {
		d.CurrentTurnIndex = (d.CurrentTurnIndex + 1) % len(d.Participants)
		return
	}

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
