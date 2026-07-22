package config

import (
	"fmt"
	"strings"
)

// AddRunCommandAllowExtra appends a normalized command prefix to the user allowlist.
// Returns true when a new entry was added.
func (c *Config) AddRunCommandAllowExtra(command string) (bool, error) {
	if c == nil {
		return false, fmt.Errorf("config unavailable")
	}
	command = strings.Join(strings.Fields(strings.TrimSpace(command)), " ")
	if command == "" {
		return false, fmt.Errorf("command required")
	}
	lower := strings.ToLower(command)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, existing := range c.Security.RunCommandAllowExtra {
		if strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(existing)), " ")) == lower {
			return false, nil
		}
	}
	c.Security.RunCommandAllowExtra = append(c.Security.RunCommandAllowExtra, command)
	return true, nil
}

// RunCommandAllowExtra returns a copy of persisted user-approved run_command prefixes.
func (c *Config) RunCommandAllowExtra() []string {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.Security.RunCommandAllowExtra) == 0 {
		return nil
	}
	out := make([]string, len(c.Security.RunCommandAllowExtra))
	copy(out, c.Security.RunCommandAllowExtra)
	return out
}
