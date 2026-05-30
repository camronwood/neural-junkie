package agent

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// PromptPersonaTier selects how the system prompt frames the agent's role.
type PromptPersonaTier int

const (
	PersonaDirect PromptPersonaTier = iota
	PersonaChannel
	PersonaCollaboration
)

func (t PromptPersonaTier) String() string {
	switch t {
	case PersonaDirect:
		return "direct"
	case PersonaChannel:
		return "channel"
	case PersonaCollaboration:
		return "collaboration"
	default:
		return "unknown"
	}
}

func (a *Agent) promptPersonaTier(msg *protocol.Message) PromptPersonaTier {
	if msg == nil {
		return PersonaChannel
	}
	if collab := a.getCollaborationContext(msg); collab.ID != "" {
		return PersonaCollaboration
	}
	if a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeCollaboration {
		return PersonaCollaboration
	}
	if a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeDM {
		return PersonaDirect
	}
	if strings.HasPrefix(strings.ToLower(msg.Channel), "dm-") {
		return PersonaDirect
	}
	if a.Hub != nil {
		if agents, err := a.Hub.GetChannelAgents(msg.Channel); err == nil {
			active := 0
			for _, ag := range agents {
				if ag.Status == "" || ag.Status == "active" {
					active++
				}
			}
			if active <= 1 {
				return PersonaDirect
			}
		}
	}
	return PersonaChannel
}

func (a *Agent) shouldIncludeToolingInPrompt(msg *protocol.Message, intent TurnIntent) bool {
	if intent == IntentClosure || intent == IntentLowSignal || intent == IntentMeta {
		return false
	}
	mode := EffectiveConversationMode(msg, a.effectiveChannelType(msg.Channel))
	if mode == ConversationModeChat && intent != IntentTask {
		return false
	}
	if a.promptPersonaTier(msg) == PersonaDirect && mode == ConversationModeChat {
		return false
	}
	return intent == IntentTask || intent == IntentSubstantive
}

func (a *Agent) writePersonaOpening(system *strings.Builder, msg *protocol.Message, tier PromptPersonaTier) {
	specialty := string(a.Info.Type)
	if a.Info.Type == protocol.AgentTypeExpert && len(a.Info.Expertise) > 0 {
		specialty = a.Info.Expertise[0]
	}
	switch tier {
	case PersonaDirect:
		fmt.Fprintf(system, "You are %s, a %s specialist speaking directly with the user in a private conversation.\n\n", a.Info.Name, specialty)
		system.WriteString("Respond naturally and conversationally. Keep replies concise unless the user asks for depth.\n\n")
	default:
		fmt.Fprintf(system, "You are %s, a %s specialist agent in a multi-agent collaboration chat room.\n\n", a.Info.Name, specialty)
	}
	fmt.Fprintf(system, "Your expertise: %s\n\n", strings.Join(a.Info.Expertise, ", "))
}
