package hub

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// CreateChannel creates a new channel
func (h *Hub) CreateChannel(name, description, project string) *protocol.Channel {
	return h.CreateChannelWithType(name, description, project, protocol.ChannelTypePublic, "")
}

// CreateChannelWithType creates a new channel with an explicit type and creator
func (h *Hub) CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Return existing channel if one with this name already exists
	if existing, ok := h.channels[name]; ok {
		return existing
	}

	channel := &protocol.Channel{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Project:     project,
		Type:        channelType,
		CreatedBy:   createdBy,
		Created:     time.Now(),
		Agents:      []protocol.AgentInfo{},
		Members:     []string{},
		Tags:        []string{},
	}

	h.channels[name] = channel
	h.messages[name] = []*protocol.Message{}
	h.subscribers[name] = []chan *protocol.Message{}

	return channel
}

// GetChannel returns a channel by name
func (h *Hub) GetChannel(name string) (*protocol.Channel, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channel, ok := h.channels[name]
	if !ok {
		return nil, fmt.Errorf("channel %s not found", name)
	}

	return channel, nil
}

// SetChannelDescription updates the sidebar-visible description for a channel.
func (h *Hub) SetChannelDescription(name, description string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.channels[name]
	if !ok {
		return fmt.Errorf("channel %s not found", name)
	}
	ch.Description = strings.TrimSpace(description)
	return nil
}

// SetChannelDisplay updates display_name and description (e.g. Slack mirror labels).
func (h *Hub) SetChannelDisplay(name, displayName, description string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.channels[name]
	if !ok {
		return fmt.Errorf("channel %s not found", name)
	}
	if dn := strings.TrimSpace(displayName); dn != "" {
		ch.DisplayName = dn
	}
	if desc := strings.TrimSpace(description); desc != "" {
		ch.Description = desc
	}
	return nil
}

// inferChannelTypeForName fixes DM classification when legacy snapshots omitted
// "type" or stored the wrong value — the UI sidebar keys off type === dm.
func inferChannelTypeForName(name string, t protocol.ChannelType) protocol.ChannelType {
	n := strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(n, "dm-") {
		return protocol.ChannelTypeDM
	}
	if strings.HasPrefix(n, "collab-") {
		return protocol.ChannelTypeCollaboration
	}
	if t == "" {
		return protocol.ChannelTypePublic
	}
	return t
}

func (h *Hub) repairChannelTypesLocked() {
	for _, ch := range h.channels {
		if ch == nil {
			continue
		}
		if want := inferChannelTypeForName(ch.Name, ch.Type); want != ch.Type {
			ch.Type = want
		}
	}
}

// channelListedInSidebarLocked reports whether a channel belongs in /api/channels and the
// desktop sidebar. DM rows whose agent was unregistered (e.g. domain pack off) are omitted.
// Caller must hold h.mu.
func (h *Hub) channelListedInSidebarLocked(ch *protocol.Channel) bool {
	if ch == nil {
		return false
	}
	if inferChannelTypeForName(ch.Name, ch.Type) != protocol.ChannelTypeDM {
		return true
	}
	for _, a := range ch.Agents {
		if _, ok := h.agents[a.ID]; ok {
			return true
		}
	}
	for _, memberID := range ch.Members {
		if _, ok := h.agents[memberID]; ok {
			return true
		}
	}
	return false
}

// ListChannels returns all available channels in a stable order:
// public first, then custom, then collaboration, then DM, alphabetical within each group.
func (h *Hub) ListChannels() []*protocol.Channel {
	h.mu.Lock()
	h.repairChannelTypesLocked()
	channels := make([]*protocol.Channel, 0, len(h.channels))
	for _, ch := range h.channels {
		if h.channelListedInSidebarLocked(ch) {
			channels = append(channels, ch)
		}
	}
	h.mu.Unlock()

	typeOrder := map[protocol.ChannelType]int{
		protocol.ChannelTypePublic:        0,
		protocol.ChannelTypeCustom:        1,
		protocol.ChannelTypeCollaboration: 2,
		protocol.ChannelTypeDM:            3,
	}
	sort.Slice(channels, func(i, j int) bool {
		oi, oj := typeOrder[channels[i].Type], typeOrder[channels[j].Type]
		if oi != oj {
			return oi < oj
		}
		return channels[i].Name < channels[j].Name
	})

	return channels
}

func (h *Hub) CreateDMChannel(username, agentID string) (*protocol.Channel, error) {
	agent, err := h.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	dmName := fmt.Sprintf("dm-%s-%s", strings.ToLower(username), strings.ToLower(agent.Name))

	// Check if it already exists
	h.mu.RLock()
	if existing, ok := h.channels[dmName]; ok {
		h.mu.RUnlock()
		// Channel was restored or left over from a prior session — still join this
		// agent so channel.Agents, GetAgentChannels, and DM rebind logic stay correct.
		if err := h.JoinChannel(agentID, dmName); err != nil {
			return nil, fmt.Errorf("failed to join agent to existing DM %s: %w", dmName, err)
		}
		h.ensureAgentSubscribed(agentID, dmName)
		if existing.DisplayName == "" {
			_ = h.SetChannelDisplay(dmName, agent.Name, existing.Description)
		}
		return existing, nil
	}
	h.mu.RUnlock()

	ch := h.CreateChannelWithType(
		dmName,
		fmt.Sprintf("Direct message with %s", agent.Name),
		"",
		protocol.ChannelTypeDM,
		username,
	)
	ch.DisplayName = agent.Name

	// Auto-join the agent to the DM channel
	if err := h.JoinChannel(agentID, dmName); err != nil {
		return nil, fmt.Errorf("failed to join agent to DM channel: %w", err)
	}

	h.ensureAgentSubscribed(agentID, dmName)
	return ch, nil
}

// ensureAgentSubscribed starts the agent hub listener on channelName immediately.
// JoinChannel alone only updates membership; runtime agents need an active subscription
// before the caller can send (avoids losing the first DM message to discoverChannels delay).
func (h *Hub) DeleteChannel(channelName string) error {
	if channelName == "general" {
		return fmt.Errorf("cannot delete the general channel")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.channels[channelName]
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}

	effectiveType := inferChannelTypeForName(ch.Name, ch.Type)
	switch effectiveType {
	case protocol.ChannelTypePublic, protocol.ChannelTypeDM:
		return fmt.Errorf("cannot delete channel %q (type %s); only custom and collaboration channels can be deleted", channelName, effectiveType)
	}

	// Close all subscriber channels
	for _, sub := range h.subscribers[channelName] {
		close(sub)
	}

	delete(h.channels, channelName)
	delete(h.messages, channelName)
	delete(h.subscribers, channelName)
	store := h.persistentStore

	if store != nil {
		if err := store.ClearChannelMessages(channelName); err != nil {
			log.Printf("[hub] clear persistent messages for deleted channel %q: %v", channelName, err)
		}
	}

	return nil
}

// legacySeedChannels were demo public rooms created on every hub start in early betas.
var legacySeedChannels = []string{"project-alpha", "project-beta"}

// RemoveLegacySeedChannels drops built-in demo channels from fresh or restored sessions.
func (h *Hub) RemoveLegacySeedChannels() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	removed := 0
	for _, name := range legacySeedChannels {
		if h.removeChannelUnlocked(name) {
			removed++
		}
	}
	return removed
}

func (h *Hub) removeChannelUnlocked(channelName string) bool {
	if _, ok := h.channels[channelName]; !ok {
		return false
	}
	for _, sub := range h.subscribers[channelName] {
		close(sub)
	}
	delete(h.channels, channelName)
	delete(h.messages, channelName)
	delete(h.subscribers, channelName)
	h.clearChannelContextLocked(channelName)
	return true
}

// GetChannelType returns the type of the named channel
func (h *Hub) GetChannelType(channelName string) protocol.ChannelType {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if ch, ok := h.channels[channelName]; ok {
		return ch.Type
	}
	return protocol.ChannelTypePublic
}
