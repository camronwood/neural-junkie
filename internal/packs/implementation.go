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

// ParsePackImplementation parses implementation: pack/<slug> from pack AgentSpec.
// Slug is the agent type already declared in pack.yaml (not a free-form module path).
func ParsePackImplementation(impl string) (agentType string, ok bool) {
	impl = strings.TrimSpace(strings.ToLower(impl))
	const prefix = "pack/"
	if !strings.HasPrefix(impl, prefix) {
		return "", false
	}
	slug := strings.TrimSpace(strings.TrimPrefix(impl, prefix))
	if slug == "" || strings.Contains(slug, "/") {
		return "", false
	}
	return slug, true
}
