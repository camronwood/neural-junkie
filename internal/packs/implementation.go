package packs

import "strings"

// ParseBuiltinImplementation parses implementation: builtin/<slug> from pack AgentSpec.
// Returns the agent type slug (e.g. "music") when the form is valid.
func ParseBuiltinImplementation(impl string) (agentType string, ok bool) {
	impl = strings.TrimSpace(strings.ToLower(impl))
	const prefix = "builtin/"
	if !strings.HasPrefix(impl, prefix) {
		return "", false
	}
	slug := strings.TrimSpace(strings.TrimPrefix(impl, prefix))
	if slug == "" || strings.Contains(slug, "/") {
		return "", false
	}
	return slug, true
}
