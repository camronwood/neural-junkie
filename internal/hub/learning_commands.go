package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const capPersonalLearning = "personal-learning"

func (ch *CommandHandler) personalLearningAllowed() bool {
	return ch.appConfig != nil &&
		ch.appConfig.AnyPackCapability(capPersonalLearning) &&
		ch.appConfig.PersonalLearningEnabled()
}

func (ch *CommandHandler) handleLearn(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if !ch.personalLearningAllowed() {
		return ch.systemResponse(msg.Channel, "Personal learning requires the Specialist tuning pack and opt-in in Settings → AI & providers."), nil
	}
	draft := strings.TrimSpace(strings.Join(parts[1:], " "))
	target := ch.resolveLearningTarget(msg)
	if target == nil {
		return ch.systemResponse(msg.Channel, "Could not determine which expert to learn for. @mention an agent or use this in their DM."), nil
	}
	resp := ch.systemResponse(msg.Channel, fmt.Sprintf("Open the learning dialog to save a note for **%s**.", target.Name))
	if resp.Metadata == nil {
		resp.Metadata = map[string]interface{}{}
	}
	action := map[string]interface{}{
		"type":              "learning_proposal",
		"agent_id":          target.ID,
		"agent_name":        target.Name,
		"agent_type":        string(target.Type),
		"draft":             draft,
		"category":          string(learning.CategoryPreference),
		"source_message_id": msg.ID,
		"source_channel":    msg.Channel,
	}
	if collabID := ch.collaborationIDForChannel(msg.Channel); collabID != "" {
		action["collaboration_id"] = collabID
	}
	resp.Metadata["client_action"] = action
	return resp, nil
}

func (ch *CommandHandler) collaborationIDForChannel(channel string) string {
	channel = strings.TrimSpace(channel)
	if channel == "" || ch.hub == nil {
		return ""
	}
	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ""
	}
	if c := cm.GetByChannel(channel); c != nil {
		return c.ID
	}
	return ""
}

func (ch *CommandHandler) handleLearningList(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if !ch.personalLearningAllowed() {
		return ch.systemResponse(msg.Channel, "Personal learning is not enabled."), nil
	}
	agentID := ""
	if len(parts) > 1 {
		name := strings.TrimPrefix(parts[1], "@")
		for _, a := range ch.hub.ListAgents() {
			if strings.EqualFold(a.Name, name) {
				agentID = a.ID
				break
			}
		}
	}
	entries := learning.ListGlobal(agentID)
	if len(entries) == 0 {
		return ch.systemResponse(msg.Channel, "No saved learnings."), nil
	}
	var b strings.Builder
	b.WriteString("Saved learnings:\n")
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("- `%s` [%s] %s (%s)\n", e.ID[:8], e.Category, e.Content, e.AgentName))
	}
	return ch.systemResponse(msg.Channel, b.String()), nil
}

func (ch *CommandHandler) handleLearningForget(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if !ch.personalLearningAllowed() {
		return ch.systemResponse(msg.Channel, "Personal learning is not enabled."), nil
	}
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /learning-forget <learning-id-prefix>"), nil
	}
	prefix := strings.TrimSpace(parts[1])
	var forgotten int
	for _, e := range learning.ListGlobal("") {
		if strings.HasPrefix(e.ID, prefix) {
			if err := learning.ForgetGlobal(e.ID); err == nil {
				forgotten++
			}
		}
	}
	if forgotten == 0 {
		return ch.systemResponse(msg.Channel, "No matching learning found."), nil
	}
	return ch.systemResponse(msg.Channel, fmt.Sprintf("Forgot %d learning(s).", forgotten)), nil
}

func (ch *CommandHandler) resolveLearningTarget(msg *protocol.Message) *protocol.AgentInfo {
	if msg == nil {
		return nil
	}
	for _, m := range protocol.ParseMentions(msg.Content) {
		name := strings.TrimPrefix(strings.TrimSpace(m), "@")
		for _, a := range ch.hub.ListAgents() {
			if strings.EqualFold(a.Name, name) {
				cp := *a
				return &cp
			}
		}
	}
	agents, err := ch.hub.GetChannelAgents(msg.Channel)
	if err != nil {
		return nil
	}
	for _, a := range agents {
		if a.Type == protocol.AgentTypeCLI {
			continue
		}
		if protocol.IsUserLikeSender(protocol.AgentInfo{Type: a.Type, Name: a.Name}) {
			continue
		}
		cp := a
		return &cp
	}
	return nil
}
