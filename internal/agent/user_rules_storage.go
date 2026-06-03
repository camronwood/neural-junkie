package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const UserRulesDefaultKey = "default"

// UserRulesFile persists global user instructions keyed by username slug or "default".
type UserRulesFile struct {
	Rules map[string]string `json:"rules"`
}

// UserRulesStorage persists ~/.neural-junkie/user-rules.json.
type UserRulesStorage struct {
	path string
	mu   sync.Mutex
}

// NewUserRulesStorage returns storage at ~/.neural-junkie/user-rules.json.
func NewUserRulesStorage() (*UserRulesStorage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".neural-junkie")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &UserRulesStorage{path: filepath.Join(dir, "user-rules.json")}, nil
}

func userRulesKey(username string) string {
	key := learning.SlugUserID(username)
	if key == "" {
		return UserRulesDefaultKey
	}
	return key
}

func (s *UserRulesStorage) load() (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var f UserRulesFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Rules == nil {
		f.Rules = map[string]string{}
	}
	return f.Rules, nil
}

func (s *UserRulesStorage) save(rules map[string]string) error {
	f := UserRulesFile{Rules: rules}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Resolve returns persisted markdown for username, falling back to the "default" key.
func (s *UserRulesStorage) Resolve(username string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules, err := s.load()
	if err != nil {
		return ""
	}
	key := userRulesKey(username)
	if key != UserRulesDefaultKey {
		if v, ok := rules[key]; ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	if v, ok := rules[UserRulesDefaultKey]; ok {
		return strings.TrimSpace(v)
	}
	return soleUserRulesFallback(rules)
}

// soleUserRulesFallback returns rules when exactly one non-default key exists.
// Helps local single-user installs where the login name changed after rules were saved.
func soleUserRulesFallback(rules map[string]string) string {
	var found string
	count := 0
	for k, v := range rules {
		if k == UserRulesDefaultKey || strings.TrimSpace(v) == "" {
			continue
		}
		found = strings.TrimSpace(v)
		count++
		if count > 1 {
			return ""
		}
	}
	if count == 1 {
		return found
	}
	return ""
}

// Get returns markdown stored for an exact key (username slug or "default").
func (s *UserRulesStorage) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules, err := s.load()
	if err != nil {
		return "", false
	}
	v, ok := rules[key]
	return v, ok && strings.TrimSpace(v) != ""
}

// Set stores markdown for username (slugged) or "default" when username is empty.
func (s *UserRulesStorage) Set(username, markdown string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules, err := s.load()
	if err != nil {
		return err
	}
	key := userRulesKey(username)
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		delete(rules, key)
	} else {
		if len(markdown) > maxUserRulesMarkdownBytes {
			markdown = truncateStringBytes(markdown, maxUserRulesMarkdownBytes)
		}
		rules[key] = markdown
	}
	return s.save(rules)
}

var userRulesLookup func(username string) string

// SetUserRulesLookup registers hub-backed lookup for prompt assembly when message metadata is empty.
func SetUserRulesLookup(fn func(username string) string) {
	userRulesLookup = fn
}

// UsernamesForRulesLookup returns candidate usernames for hub rules lookup, in priority order.
func UsernamesForRulesLookup(msg *protocol.Message) []string {
	if msg == nil {
		return nil
	}
	seen := map[string]bool{}
	var names []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	if msg.Metadata != nil {
		if raw, ok := msg.Metadata[MetadataHubSessionUsername]; ok {
			if s, ok := raw.(string); ok {
				add(s)
			}
		}
	}
	if protocol.IsUserLikeSender(msg.From) {
		add(msg.From.Name)
	}
	return names
}

// ResolveUserRulesHubFallback loads persisted global rules for a message sender (or default).
func ResolveUserRulesHubFallback(msg *protocol.Message) string {
	if userRulesLookup == nil || msg == nil {
		return ""
	}
	for _, name := range UsernamesForRulesLookup(msg) {
		if rules := strings.TrimSpace(userRulesLookup(name)); rules != "" {
			return rules
		}
	}
	return ""
}

// AttachUserRulesMetadataIfMissing writes hub-resolved rules onto msg.Metadata when absent.
func AttachUserRulesMetadataIfMissing(msg *protocol.Message) {
	if msg == nil || !protocol.IsUserLikeSender(msg.From) {
		return
	}
	if msg.Metadata != nil {
		if raw, ok := msg.Metadata[MetadataUserRulesMarkdown]; ok {
			if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
				return
			}
		}
	}
	rules := ResolveUserRulesHubFallback(msg)
	if rules == "" {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata[MetadataUserRulesMarkdown] = rules
}
