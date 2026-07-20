package collaboration

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func isDiscussionGenerationError(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	if v, ok := msg.Metadata["generation_error"].(bool); ok && v {
		return true
	}
	_, hasCode := msg.Metadata["error_code"]
	return hasCode
}

// maxPlanningGenerationErrorsPerTurn caps retries on one planning turn before
// advancing to the next participant (prevents generation_error death spirals).
const maxPlanningGenerationErrorsPerTurn = 2

func planningGenerationErrorCount(d *DiscussionSession, agentID string) int {
	if d == nil || agentID == "" {
		return 0
	}
	n := 0
	for _, m := range d.Messages {
		if m != nil && m.From.ID == agentID && isDiscussionGenerationError(m) {
			n++
		}
	}
	return n
}

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
	var notifyReviewing string

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

	// Hub-mediated L1 consults must not consume discussion turn budget or advance rounds.
	if isCollabConsultMessage(msg) {
		cm.mu.Unlock()
		return nil
	}

	if planningDiscussionTimeoutElapsed(c, d) {
		d.Status = DiscussionTimedOut
		if c.Phase == PhasePlanning {
			if cm.enterReviewingFromPlanningLocked(c) {
				notifyReviewing = collabID
			}
		}
		c.UpdatedAt = time.Now()
		log.Printf("[Discussion %s] Timed out after %v", d.ID[:8], d.Timeout)
		cm.mu.Unlock()
		if notifyReviewing != "" && cm.onEnterReviewing != nil {
			go cm.onEnterReviewing(notifyReviewing)
		}
		return fmt.Errorf("discussion timed out")
	}

	enforced := c.DiscussionBudgetEnforced()

	if !isDiscussionGenerationError(msg) && c.Phase == PhasePlanning && enforced {
		agentID := msg.From.ID
		if d.TurnsThisRound[agentID] >= discussionTurnBudget(c, d) {
			cm.mu.Unlock()
			return fmt.Errorf("planning turn budget exhausted for agent")
		}
		if planningSpeakerCooldownBlockedLocked(c, d, agentID) {
			cm.mu.Unlock()
			return fmt.Errorf("planning cooldown: waiting for other participants")
		}
		if len(d.Participants) > 0 && d.CurrentTurnIndex < len(d.Participants) &&
			d.Participants[d.CurrentTurnIndex] != agentID {
			cm.mu.Unlock()
			return fmt.Errorf("not current planning turn")
		}
	}

	d.Messages = append(d.Messages, msg)
	if isDiscussionGenerationError(msg) {
		agentID := msg.From.ID
		errCount := planningGenerationErrorCount(d, agentID)
		// Keep the turn on the first failure so watchdog/handoff can re-prompt;
		// after repeated failures advance so other participants are not stranded.
		if errCount >= maxPlanningGenerationErrorsPerTurn &&
			c.Phase == PhasePlanning && enforced &&
			len(d.Participants) > 0 && d.CurrentTurnIndex < len(d.Participants) &&
			d.Participants[d.CurrentTurnIndex] == agentID {
			log.Printf("[Discussion %s] Advancing turn after %d generation_errors from %s",
				d.ID[:8], errCount, agentID)
			cm.advanceTurnAfterGenerationErrors(c)
		}
		c.UpdatedAt = time.Now()
		cm.mu.Unlock()
		return nil
	}
	d.TotalMessageCount++
	d.TurnsThisRound[msg.From.ID]++

	if enforced && d.TotalMessageCount >= d.MaxTotalMessages {
		d.Status = DiscussionBudgetExhausted
		if c.Phase == PhasePlanning {
			if cm.enterReviewingFromPlanningLocked(c) {
				notifyReviewing = collabID
			}
		}
		c.UpdatedAt = time.Now()
		log.Printf("[Discussion %s] Budget exhausted (%d/%d messages)", d.ID[:8], d.TotalMessageCount, d.MaxTotalMessages)
		notifyBudget = true
		cm.mu.Unlock()
		if notifyReviewing != "" && cm.onEnterReviewing != nil {
			go cm.onEnterReviewing(notifyReviewing)
		}
		if notifyBudget {
			cm.postDiscussionLimitNotice(collabID)
		}
		return nil
	}

	cm.advanceTurn(c)
	if c.Phase == PhasePlanning && d.Status == DiscussionActive && enforced &&
		participationQuorumMet(d) && cm.enterReviewingFromPlanningLocked(c) {
		notifyReviewing = collabID
		log.Printf("[Discussion %s] Participation quorum met (%d/%d messages) — reviewing",
			d.ID[:8], d.TotalMessageCount, d.MaxTotalMessages)
	}
	if d.Status != DiscussionActive && c.Phase == PhasePlanning && enforced {
		if cm.enterReviewingFromPlanningLocked(c) {
			notifyReviewing = collabID
		}
	}
	if enforced && d.Status == DiscussionBudgetExhausted {
		notifyBudget = true
	}
	c.UpdatedAt = time.Now()
	cm.mu.Unlock()

	if notifyReviewing != "" && cm.onEnterReviewing != nil {
		go cm.onEnterReviewing(notifyReviewing)
	}
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

	if planningDiscussionTimeoutElapsed(c, d) {
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

	if d.TurnsThisRound[agentID] >= discussionTurnBudget(c, d) {
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

// AdvancePlanningDiscussionIfTimedOut ends a silent planning discussion when its
// wall-clock budget expires and moves the collaboration into reviewing.
// Returns true when the collaboration newly entered reviewing.
func (cm *CollaborationManager) AdvancePlanningDiscussionIfTimedOut(collabID string) bool {
	var notifyReviewing string

	cm.mu.Lock()
	c, ok := cm.collaborations[collabID]
	if !ok || c == nil || c.Discussion == nil || c.Phase != PhasePlanning {
		cm.mu.Unlock()
		return false
	}
	d := c.Discussion
	if d.Status != DiscussionActive || !planningDiscussionTimeoutElapsed(c, d) {
		cm.mu.Unlock()
		return false
	}
	d.Status = DiscussionTimedOut
	if cm.enterReviewingFromPlanningLocked(c) {
		notifyReviewing = collabID
	}
	c.UpdatedAt = time.Now()
	log.Printf("[Discussion %s] Timed out after %v (watchdog)", d.ID[:8], d.Timeout)
	cm.mu.Unlock()

	if notifyReviewing != "" && cm.onEnterReviewing != nil {
		go cm.onEnterReviewing(notifyReviewing)
	}
	return notifyReviewing != ""
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

// SilentPlanningParticipantIDs returns participant agent IDs who have not yet posted
// a planning discussion message in the current session.
func (cm *CollaborationManager) SilentPlanningParticipantIDs(collabID string) []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok || c == nil || c.Discussion == nil || c.Phase != PhasePlanning {
		return nil
	}
	if c.Discussion.Status != DiscussionActive {
		return nil
	}
	return silentParticipantIDsLocked(c.Discussion)
}

// ParticipantAgentName resolves the display name for a collaboration participant ID.
func (cm *CollaborationManager) ParticipantAgentName(collabID, agentID string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok || c == nil {
		return ""
	}
	for _, a := range c.Agents {
		if a.AgentID == agentID {
			return a.AgentName
		}
	}
	return ""
}

// discussionTurnBudget caps each participant to one in-round reply until every
// participant has spoken at least once (prevents one agent dominating planning).
func discussionTurnBudget(c *Collaboration, d *DiscussionSession) int {
	if d == nil {
		return DefaultTurnBudget
	}
	budget := d.TurnBudget
	if budget <= 0 {
		budget = DefaultTurnBudget
	}
	if c != nil && c.Phase == PhasePlanning && !participationQuorumMet(d) && budget > 1 {
		return 1
	}
	return budget
}

// silentParticipantIDsLocked returns participant IDs with no discussion messages yet.
// Caller must hold cm.mu when used from locked paths; d is the active discussion.
func silentParticipantIDsLocked(d *DiscussionSession) []string {
	if d == nil || len(d.Participants) == 0 {
		return nil
	}
	spoken := make(map[string]bool, len(d.Participants))
	for _, m := range d.Messages {
		if m == nil {
			continue
		}
		if isDiscussionGenerationError(m) {
			continue
		}
		id := strings.TrimSpace(m.From.ID)
		if id == "" || id == "system" {
			continue
		}
		spoken[id] = true
	}
	var silent []string
	for _, pid := range d.Participants {
		if pid == "" || spoken[pid] {
			continue
		}
		silent = append(silent, pid)
	}
	return silent
}

func ollamaConcurrencyLimit() int {
	raw := strings.TrimSpace(os.Getenv("NJ_OLLAMA_MAX_CONCURRENCY"))
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// planningOllamaQueueFactor estimates how many Ollama slot turns a planning quorum
// may need when participants serialize through NJ_OLLAMA_MAX_CONCURRENCY.
func planningOllamaQueueFactor(participantCount int) int {
	if participantCount < 1 {
		participantCount = 1
	}
	conc := ollamaConcurrencyLimit()
	f := (participantCount + conc - 1) / conc
	if f < 1 {
		return 1
	}
	return f
}

// planningDiscussionTimeoutElapsed is true when the wall-clock budget expired.
// While silent participants remain and the message budget is not exhausted, a
// short grace window allows the planning watchdog to re-dispatch turn handoffs.
func planningDiscussionTimeoutElapsed(c *Collaboration, d *DiscussionSession) bool {
	if d == nil {
		return false
	}
	elapsed := time.Since(d.StartedAt)
	if elapsed <= d.Timeout {
		return false
	}
	if c == nil || c.Phase != PhasePlanning || !c.DiscussionBudgetEnforced() {
		return true
	}
	silent := silentParticipantIDsLocked(d)
	if len(silent) == 0 || d.TotalMessageCount >= d.MaxTotalMessages {
		return true
	}
	queueFactor := planningOllamaQueueFactor(len(d.Participants))
	// Keep planning open while no turn has been recorded yet — the first LLM reply
	// often exceeds the scaled wall timeout (--messages 4 → 180s), especially when
	// agents queue behind NJ_OLLAMA_MAX_CONCURRENCY.
	if d.TotalMessageCount == 0 {
		firstReplyGrace := 2 * time.Minute
		if d.MaxTotalMessages > 0 && d.MaxTotalMessages <= 4 {
			firstReplyGrace = 90 * time.Second
		}
		firstReplyGrace *= time.Duration(queueFactor)
		const maxFirstReplyGrace = 8 * time.Minute
		if firstReplyGrace > maxFirstReplyGrace {
			firstReplyGrace = maxFirstReplyGrace
		}
		if elapsed <= d.Timeout+firstReplyGrace {
			return false
		}
	}
	grace := time.Duration(len(silent)) * 45 * time.Second
	maxGrace := 3 * time.Minute * time.Duration(queueFactor)
	const hardMaxGrace = 8 * time.Minute
	if maxGrace > hardMaxGrace {
		maxGrace = hardMaxGrace
	}
	if grace > maxGrace {
		grace = maxGrace
	}
	return elapsed > d.Timeout+grace
}

// planningSpeakerCooldownBlockedLocked prevents one agent from dominating planning
// before every other participant has spoken at least once in the current round.
// Caller must hold cm.mu.
func planningSpeakerCooldownBlockedLocked(c *Collaboration, d *DiscussionSession, agentID string) bool {
	if c == nil || d == nil || c.Phase != PhasePlanning || !c.DiscussionBudgetEnforced() {
		return false
	}
	if d.Status != DiscussionActive {
		return false
	}
	if d.TurnsThisRound[agentID] == 0 {
		return false
	}
	for _, pid := range d.Participants {
		if pid == agentID {
			continue
		}
		if d.TurnsThisRound[pid] == 0 {
			return true
		}
	}
	return false
}

// participationQuorumMet is true when every discussion participant has spoken at least once.
func participationQuorumMet(d *DiscussionSession) bool {
	if d == nil || len(d.Participants) < 1 {
		return false
	}
	spoken := make(map[string]bool, len(d.Participants))
	for _, m := range d.Messages {
		if m == nil || m.From.ID == "" || isDiscussionGenerationError(m) || isCollabConsultMessage(m) {
			continue
		}
		spoken[m.From.ID] = true
	}
	for _, pid := range d.Participants {
		if !spoken[pid] {
			return false
		}
	}
	return d.TotalMessageCount >= len(d.Participants)
}

func isCollabConsultMessage(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	if event, ok := msg.Metadata["event"].(string); ok && event == "collab-consult" {
		return true
	}
	if v, ok := msg.Metadata["collab_consult"].(bool); ok && v {
		return true
	}
	return false
}

// advanceTurnAfterGenerationErrors moves to the next participant after repeated
// generation failures on the current turn. Uses simple round-robin so the turn
// does not snap back to a stuck agent when turn-budget skip logic would keep
// the failing speaker as "next".
// Must be called with cm.mu held.
func (cm *CollaborationManager) advanceTurnAfterGenerationErrors(c *Collaboration) {
	d := c.Discussion
	if d == nil || len(d.Participants) == 0 {
		return
	}
	d.CurrentTurnIndex = (d.CurrentTurnIndex + 1) % len(d.Participants)
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
		if d.TurnsThisRound[pid] < discussionTurnBudget(c, d) {
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
		if d.TurnsThisRound[next] < discussionTurnBudget(c, d) {
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
