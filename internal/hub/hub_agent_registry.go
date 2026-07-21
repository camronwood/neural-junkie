package hub

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (h *Hub) RegisterAgent(agent *protocol.AgentInfo) error {
	h.mu.Lock()

	// Remove ALL existing agents with the same name/type (handles restarts and duplicates)
	// When an agent restarts, it gets a new ID, so we need to clean up old entries
	idsToDelete := []string{}
	for id, existingAgent := range h.agents {
		if existingAgent.Name == agent.Name && existingAgent.Type == agent.Type && id != agent.ID {
			idsToDelete = append(idsToDelete, id)
		}
	}

	// Delete all old agents with this name/type
	for _, id := range idsToDelete {
		oldAgent := h.agents[id]
		// Remove from all channels
		for _, channel := range h.channels {
			newAgents := []protocol.AgentInfo{}
			for _, a := range channel.Agents {
				if a.ID != id {
					newAgents = append(newAgents, a)
				}
			}
			channel.Agents = newAgents
		}
		delete(h.agents, id)
		log.Printf("Removed duplicate agent registration: %s (old id %s)", oldAgent.Name, id[:8])
	}

	// Register a hub-owned copy so in-process agents can mutate local Info
	// and publish via SyncAgentRegistration without racing hub readers.
	stored := new(protocol.AgentInfo)
	*stored = *agent
	enrichAgentImageGeneration(stored)
	enrichAgentMusicGeneration(h, stored)
	enrichAgentArena(h, stored)
	agent.SupportsImageGeneration = stored.SupportsImageGeneration
	agent.SupportsMusicGeneration = stored.SupportsMusicGeneration
	agent.SupportsArena = stored.SupportsArena
	if h.agentRulesStore != nil {
		if md, ok := h.agentRulesStore.Get(stored.ID); ok {
			stored.CustomRulesMarkdown = md
		}
	}
	h.agents[stored.ID] = stored
	h.mu.Unlock()

	if h.collabManager != nil {
		h.collabManager.ReconcileRestoredAgentIDs()
	}
	return nil
}

// UnregisterAgent removes an agent
func (h *Hub) UnregisterAgent(agentID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.agents[agentID]; !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	delete(h.agents, agentID)

	// Remove from all channels
	for _, channel := range h.channels {
		for i, agent := range channel.Agents {
			if agent.ID == agentID {
				channel.Agents = append(channel.Agents[:i], channel.Agents[i+1:]...)
				break
			}
		}
	}

	return nil
}

// JoinChannel adds an agent to a channel. An optional greeting can be
// provided to replace the default join message content -- this avoids
// the need for a separate SendMessage call that would create a duplicate.
func (h *Hub) JoinChannel(agentID, channelName string, greeting ...string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	agent, ok := h.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	channel, ok := h.channels[channelName]
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}

	// Check if already in channel
	for _, a := range channel.Agents {
		if a.ID == agentID {
			return nil // Already in channel
		}
	}

	channel.Agents = append(channel.Agents, *agent)

	if !h.shouldSkipJoinAnnouncementLocked(channelName, agent) {
		content := fmt.Sprintf("%s (%s) has joined the channel", agent.Name, agent.Type)
		if len(greeting) > 0 && greeting[0] != "" {
			content = greeting[0]
		}

		joinMsg := protocol.NewMessage(
			protocol.MessageTypeAgentJoin,
			channelName,
			*agent,
			content,
		)

		h.appendChannelMessageLocked(channelName, joinMsg)
		h.broadcast(channelName, joinMsg)
	}

	return nil
}

// shouldSkipJoinAnnouncementLocked avoids duplicate join lines when agents rebind after
// hub restart (DM restore, specialist boot) while history already records a prior join.
// Caller must hold h.mu (write lock).
func (h *Hub) shouldSkipJoinAnnouncementLocked(channelName string, agent *protocol.AgentInfo) bool {
	if agent == nil {
		return true
	}
	if ch, ok := h.channels[channelName]; ok && ch.Type == protocol.ChannelTypeRoom {
		return true
	}
	msgs := h.messages[channelName]
	for i := len(msgs) - 1; i >= 0 && i >= len(msgs)-40; i-- {
		m := msgs[i]
		if m == nil || m.Type != protocol.MessageTypeAgentJoin {
			continue
		}
		if m.From.ID == agent.ID || m.From.Name == agent.Name {
			return true
		}
	}
	return false
}

// LeaveChannel removes an agent from a channel
func (h *Hub) LeaveChannel(agentID, channelName string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	agent, ok := h.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	channel, ok := h.channels[channelName]
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}

	// Remove from channel
	for i, a := range channel.Agents {
		if a.ID == agentID {
			channel.Agents = append(channel.Agents[:i], channel.Agents[i+1:]...)
			break
		}
	}

	// Send leave message
	leaveMsg := protocol.NewMessage(
		protocol.MessageTypeAgentLeave,
		channelName,
		*agent,
		fmt.Sprintf("%s has left the channel", agent.Name),
	)

	h.appendChannelMessageLocked(channelName, leaveMsg)
	h.broadcast(channelName, leaveMsg)

	return nil
}

// SendMessage sends a message to a channel
func (h *Hub) getAgentListString() string {
	var names []string
	for _, agent := range h.agents {
		names = append(names, "@"+agent.Name)
	}
	if len(names) == 0 {
		return "(no agents available)"
	}
	return strings.Join(names, ", ")
}

// Subscribe creates a subscription to a channel for real-time updates
func (h *Hub) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channel, ok := h.channels[channelName]
	if !ok {
		return nil, fmt.Errorf("channel %s not found", channelName)
	}

	return channel.Agents, nil
}

// GetAgent returns agent info by ID
func (h *Hub) GetAgent(agentID string) (*protocol.AgentInfo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agent, ok := h.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return agent, nil
}

// FindLiveAgentByDisplayName returns a copy of a registered agent matching
// name (case-insensitive). When typ is non-empty, the agent type must match.
func (h *Hub) FindLiveAgentByDisplayName(name string, typ protocol.AgentType) *protocol.AgentInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return nil
	}
	for _, a := range h.agents {
		if a == nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(a.Name)) != want {
			continue
		}
		if typ != "" && a.Type != typ {
			continue
		}
		cp := *a
		return &cp
	}
	return nil
}

// SetAgentCustomRulesMarkdown updates persisted per-agent instructions (markdown).
func (h *Hub) syncAgentInfoCopiesInChannelsLocked(agentID string, ag *protocol.AgentInfo) {
	for _, ch := range h.channels {
		for i := range ch.Agents {
			if ch.Agents[i].ID == agentID {
				ch.Agents[i] = *ag
			}
		}
	}
}

// SyncAgentRegistration copies the latest agent fields into the hub registry.
// Repo and Confluence agents mutate Info in-process; call this after those updates
// so ListAgents/GetAgent readers do not race with background indexing.
func (h *Hub) SyncAgentRegistration(agentID string, info protocol.AgentInfo) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ag, ok := h.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}
	*ag = info
	enrichAgentImageGeneration(ag)
	enrichAgentMusicGeneration(h, ag)
	enrichAgentArena(h, ag)
	h.syncAgentInfoCopiesInChannelsLocked(agentID, ag)
	return nil
}

// ListAgents returns all registered agents
func (h *Hub) ListAgents() []*protocol.AgentInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]*protocol.AgentInfo, 0, len(h.agents))
	for _, agent := range h.agents {
		if agent == nil {
			continue
		}
		cloned := *agent
		agents = append(agents, &cloned)
	}

	return agents
}

// GetThreadMessages returns messages from a thread
func (h *Hub) GetRemovedAgents() []*protocol.AgentInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]*protocol.AgentInfo, 0, len(h.removedAgents))
	for _, agent := range h.removedAgents {
		agents = append(agents, agent)
	}
	return agents
}

// IsAgentRemoved checks if an agent is in the removed state
func (h *Hub) IsAgentRemoved(agentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, exists := h.removedAgents[agentID]
	return exists
}

// AddRemovedAgent adds an agent to the removed agents list
func (h *Hub) AddRemovedAgent(agent *protocol.AgentInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.removedAgents[agent.ID] = agent
}

// RemoveFromRemovedAgents removes an agent from the removed agents list
func (h *Hub) RemoveFromRemovedAgents(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.removedAgents, agentID)
}

// IsAgentInAnyChannel checks if an agent is currently in any channel
func (h *Hub) IsAgentInAnyChannel(agentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, channel := range h.channels {
		if channel == nil || channel.Archived {
			continue
		}
		for _, agent := range channel.Agents {
			if agent.ID == agentID {
				return true
			}
		}
	}
	return false
}

// EnsureAgentSubscribedToChannel starts the in-process agent listener on channelName
// immediately after JoinChannel (do not rely on discoverChannels delay).
func (h *Hub) EnsureAgentSubscribedToChannel(agentID, channelName string) {
	h.ensureAgentSubscribed(agentID, channelName)
}

// CreateDMChannel creates (or returns an existing) DM channel between a user and an agent.
// The agent is automatically joined to the channel.
func (h *Hub) ensureAgentSubscribed(agentID, channelName string) {
	if h.commandHandler == nil {
		return
	}
	if err := h.commandHandler.EnsureAgentSubscribedToChannel(context.Background(), agentID, channelName); err != nil {
		log.Printf("Warning: failed to subscribe agent %s to %s: %v", agentID, channelName, err)
	}
}

// GetAgentChannels returns the names of all channels an agent is currently in
func (h *Hub) GetAgentChannels(agentID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var channels []string
	for name, channel := range h.channels {
		if channel == nil || channel.Archived {
			continue
		}
		for _, a := range channel.Agents {
			if a.ID == agentID {
				channels = append(channels, name)
				break
			}
		}
	}
	return channels
}

// AddAgentToChannel joins an agent to a channel and records it as an explicit member
func (h *Hub) AddAgentToChannel(agentID, channelName string) error {
	if err := h.JoinChannel(agentID, channelName); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.channels[channelName]
	if !ok {
		return nil
	}

	// Track as explicit member (deduplicated)
	for _, m := range ch.Members {
		if m == agentID {
			return nil
		}
	}
	ch.Members = append(ch.Members, agentID)
	return nil
}

// RemoveAgentFromChannel removes an agent from a channel and its member list
func (h *Hub) RemoveAgentFromChannel(agentID, channelName string) error {
	if err := h.LeaveChannel(agentID, channelName); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.channels[channelName]
	if !ok {
		return nil
	}

	// Remove from explicit members
	for i, m := range ch.Members {
		if m == agentID {
			ch.Members = append(ch.Members[:i], ch.Members[i+1:]...)
			break
		}
	}
	return nil
}

// DeleteChannel removes a channel entirely. The general channel, public rooms, and DM
// channels cannot be deleted; custom and collaboration channels may be removed.
