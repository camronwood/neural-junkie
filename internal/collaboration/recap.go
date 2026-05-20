package collaboration

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const recapContextMaxChars = 14000

// SelectRecapFacilitator returns the agent ID that should deliver a user-facing recap.
// For pre_approval: last non-system speaker in the current (planning) discussion.
// For final: last speaker in execution discussion, then planning recap agent, then first participant.
func SelectRecapFacilitator(c *Collaboration, kind RecapKind) string {
	if c == nil {
		return ""
	}
	disc := c.Discussion
	if kind == RecapKindPreApproval && c.PlanningDiscussion != nil {
		disc = c.PlanningDiscussion
	}
	if id := lastDiscussionSpeakerID(disc); id != "" {
		return id
	}
	if kind == RecapKindFinal && strings.TrimSpace(c.PlanningRecapAgentID) != "" {
		return c.PlanningRecapAgentID
	}
	if len(c.Agents) > 0 {
		return c.Agents[0].AgentID
	}
	return ""
}

// FacilitatorDisplayName resolves an agent ID to @Name for user messages.
func FacilitatorDisplayName(c *Collaboration, agentID string) string {
	if c == nil || agentID == "" {
		return "agent"
	}
	for _, a := range c.Agents {
		if a.AgentID == agentID {
			if a.AgentName != "" {
				return "@" + a.AgentName
			}
			break
		}
	}
	return "agent"
}

func lastDiscussionSpeakerID(d *DiscussionSession) string {
	if d == nil || len(d.Messages) == 0 {
		return ""
	}
	for i := len(d.Messages) - 1; i >= 0; i-- {
		m := d.Messages[i]
		if m == nil {
			continue
		}
		if m.From.ID == "" || m.From.ID == "system" {
			continue
		}
		if strings.EqualFold(m.From.Name, "System") {
			continue
		}
		if m.Type == protocol.MessageTypeSystemInfo {
			continue
		}
		return m.From.ID
	}
	return ""
}

// BuildRecapContext assembles markdown context for recap prompts.
func BuildRecapContext(c *Collaboration, kind RecapKind) string {
	if c == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Collaboration: %s\n", c.Title))
	b.WriteString(fmt.Sprintf("**Goal:** %s\n", c.Description))
	b.WriteString(fmt.Sprintf("**Phase:** %s\n", c.Phase))
	b.WriteString(fmt.Sprintf("**Recap kind:** %s\n\n", kind))

	if c.Plan != nil && strings.TrimSpace(c.Plan.Content) != "" {
		b.WriteString("## Plan artifact\n")
		b.WriteString(c.Plan.Content)
		b.WriteString("\n\n")
	}

	if len(c.Tasks) > 0 {
		b.WriteString("## Tasks\n")
		for i, t := range c.Tasks {
			line := fmt.Sprintf("%d. [%s] %s — %s", i+1, t.Status, t.Title, t.Description)
			if t.AssignedName != "" {
				line += fmt.Sprintf(" (@%s)", t.AssignedName)
			}
			b.WriteString(line)
			b.WriteString("\n")
			if kind == RecapKindFinal && strings.TrimSpace(t.Output) != "" {
				out := t.Output
				if len(out) > 1500 {
					out = out[:1500] + "…"
				}
				b.WriteString(fmt.Sprintf("   Output: %s\n", out))
			}
		}
		b.WriteString("\n")
	}

	if strings.TrimSpace(c.PlanningRecap) != "" && kind == RecapKindFinal {
		b.WriteString("## Planning session summary (already delivered)\n")
		b.WriteString(c.PlanningRecap)
		b.WriteString("\n\n")
	}

	disc := c.Discussion
	if kind == RecapKindPreApproval && c.PlanningDiscussion != nil {
		disc = c.PlanningDiscussion
	}
	if disc != nil && len(disc.Messages) > 0 {
		b.WriteString("## Discussion transcript\n")
		for _, m := range disc.Messages {
			if m == nil {
				continue
			}
			if m.From.ID == "system" || strings.EqualFold(m.From.Name, "System") {
				continue
			}
			content := strings.TrimSpace(m.Content)
			if content == "" {
				continue
			}
			if len(content) > 2000 {
				content = content[:2000] + "…"
			}
			b.WriteString(fmt.Sprintf("[%s]: %s\n\n", m.From.Name, content))
		}
	}

	if c.WorkingDirectory != "" {
		b.WriteString(fmt.Sprintf("\n**Execution workspace:** %s\n", c.WorkingDirectory))
	}

	out := b.String()
	if len(out) > recapContextMaxChars {
		out = out[:recapContextMaxChars] + "\n…(truncated)"
	}
	return out
}

// CopyDiscussionSession returns a shallow copy of a discussion including message pointers.
func CopyDiscussionSession(d *DiscussionSession) *DiscussionSession {
	if d == nil {
		return nil
	}
	cp := *d
	if len(d.Participants) > 0 {
		cp.Participants = append([]string(nil), d.Participants...)
	}
	if d.TurnsThisRound != nil {
		cp.TurnsThisRound = make(map[string]int, len(d.TurnsThisRound))
		for k, v := range d.TurnsThisRound {
			cp.TurnsThisRound[k] = v
		}
	}
	if d.Consensus != nil {
		cp.Consensus = make(map[string]ConsensusState, len(d.Consensus))
		for k, v := range d.Consensus {
			cp.Consensus[k] = v
		}
	}
	if len(d.Messages) > 0 {
		cp.Messages = append([]*protocol.Message(nil), d.Messages...)
	}
	return &cp
}
