package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// ForwardRuleType selects how channel messages are forwarded to the personal inbox.
type ForwardRuleType string

const (
	ForwardRuleMentionOfMe ForwardRuleType = "mention_of_me"
	ForwardRulePrefix      ForwardRuleType = "prefix"
	ForwardRuleReaction    ForwardRuleType = "reaction"
)

// ForwardRule forwards matching Slack channel messages into the personal inbox.
type ForwardRule struct {
	ID              string          `json:"id,omitempty"`
	Type            ForwardRuleType `json:"type"`
	Enabled         bool            `json:"enabled"`
	SlackChannelIDs []string        `json:"slack_channel_ids,omitempty"`
	Prefix          string          `json:"prefix,omitempty"`
	Emoji           string          `json:"emoji,omitempty"`
}

// InboxConfig is the personal Slack DM inbox for the NJ owner.
type InboxConfig struct {
	Enabled            bool          `json:"enabled"`
	OwnerSlackUserID   string        `json:"owner_slack_user_id,omitempty"`
	OwnerSlackUserName string        `json:"owner_slack_user_name,omitempty"`
	AgentID            string        `json:"agent_id,omitempty"`
	AgentName          string        `json:"agent_name,omitempty"`
	NJChannel          string        `json:"nj_channel,omitempty"`
	SlackDMChannelID   string        `json:"slack_dm_channel_id,omitempty"`
	ReplyInThread      bool              `json:"reply_in_thread"`
	ForwardEnabled     bool              `json:"forward_enabled"`
	ForwardRules       []ForwardRule     `json:"forward_rules,omitempty"`
	HumanDMAway        HumanDMAwayConfig `json:"human_dm_away,omitempty"`
}

// NJInboxChannelName returns the hub channel for a Slack user inbox.
func NJInboxChannelName(ownerSlackUserID string) string {
	return "slack:inbox:" + ownerSlackUserID
}

// NJInboxPeerChannelName returns the hub channel for one human Slack DM peer.
func NJInboxPeerChannelName(ownerSlackUserID, peerSlackUserID string) string {
	ownerSlackUserID = strings.TrimSpace(ownerSlackUserID)
	peerSlackUserID = strings.TrimSpace(peerSlackUserID)
	if ownerSlackUserID == "" || peerSlackUserID == "" {
		return ""
	}
	return NJInboxChannelName(ownerSlackUserID) + ":" + peerSlackUserID
}

// NJInboxPeerChannelPrefix returns the prefix for all peer inbox channels for an owner.
func NJInboxPeerChannelPrefix(ownerSlackUserID string) string {
	ownerSlackUserID = strings.TrimSpace(ownerSlackUserID)
	if ownerSlackUserID == "" {
		return ""
	}
	return NJInboxChannelName(ownerSlackUserID) + ":"
}

// IsInboxPeerHubChannel reports whether channel is a peer inbox for ownerSlackUserID.
func IsInboxPeerHubChannel(channel, ownerSlackUserID string) bool {
	prefix := NJInboxPeerChannelPrefix(ownerSlackUserID)
	return prefix != "" && strings.HasPrefix(channel, prefix)
}

// InboxStore persists inbox configuration.
type InboxStore struct {
	mu       sync.RWMutex
	filePath string
	cfg      InboxConfig
}

func inboxPath() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return dir + "/inbox.json", nil
}

// NewInboxStore loads inbox config from disk.
func NewInboxStore() (*InboxStore, error) {
	p, err := inboxPath()
	if err != nil {
		return nil, err
	}
	s := &InboxStore{filePath: p}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Reload reads inbox config from disk.
func (s *InboxStore) Reload() error {
	return s.load()
}

func (s *InboxStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var cfg InboxConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	return nil
}

func (s *InboxStore) saveLocked() error {
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

// Get returns a copy of the inbox config.
func (s *InboxStore) Get() InboxConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Save replaces the inbox config.
func (s *InboxStore) Save(cfg InboxConfig) (InboxConfig, error) {
	if cfg.OwnerSlackUserID != "" && cfg.NJChannel == "" {
		cfg.NJChannel = NJInboxChannelName(cfg.OwnerSlackUserID)
	}
	cfg.ReplyInThread = true
	normalizeHumanDMAway(&cfg.HumanDMAway)
	cfg.HumanDMAway.UserTokenSet = false
	cfg.HumanDMAway.MonitoringStatus = ""
	s.mu.Lock()
	s.cfg = cfg
	err := s.saveLocked()
	out := s.cfg
	s.mu.Unlock()
	if err != nil {
		return InboxConfig{}, err
	}
	return out, nil
}

// UpdateDMChannelID caches the Slack DM channel id for the owner.
func (s *InboxStore) UpdateDMChannelID(dmChannelID string) error {
	dmChannelID = strings.TrimSpace(dmChannelID)
	if dmChannelID == "" {
		return fmt.Errorf("dm channel id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg.SlackDMChannelID == dmChannelID {
		return nil
	}
	s.cfg.SlackDMChannelID = dmChannelID
	return s.saveLocked()
}

// SeedInboxFromInstall creates a default inbox config when OAuth completes.
func SeedInboxFromInstall(ownerUserID, ownerUserName string) error {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil
	}
	store, err := NewInboxStore()
	if err != nil {
		return err
	}
	cfg := store.Get()
	if cfg.OwnerSlackUserID != "" {
		if cfg.OwnerSlackUserName == "" && ownerUserName != "" {
			cfg.OwnerSlackUserName = ownerUserName
			_, err = store.Save(cfg)
		}
		return err
	}
	cfg.OwnerSlackUserID = ownerUserID
	cfg.OwnerSlackUserName = ownerUserName
	cfg.NJChannel = NJInboxChannelName(ownerUserID)
	cfg.ReplyInThread = true
	cfg.ForwardRules = defaultForwardRules()
	_, err = store.Save(cfg)
	return err
}

// DefaultForwardRules returns the default selective forwarding rules for a new inbox.
func DefaultForwardRules() []ForwardRule {
	return defaultForwardRules()
}

func defaultForwardRules() []ForwardRule {
	return []ForwardRule{
		{ID: "mentions", Type: ForwardRuleMentionOfMe, Enabled: false, SlackChannelIDs: []string{}},
		{ID: "nj-prefix", Type: ForwardRulePrefix, Enabled: false, Prefix: "nj:", SlackChannelIDs: []string{"*"}},
		{ID: "robot-react", Type: ForwardRuleReaction, Enabled: false, Emoji: "robot_face", SlackChannelIDs: []string{}},
	}
}
