package protocol

import (
	"regexp"
	"strings"
)

// mentionRegex matches @mentions in message content
// Supports: @alice, @backend, @agent-name-123, @ArchitectureExpert
// Agent names must be one word or kebab-case (no spaces allowed)
var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*)`)

// ParseMentions extracts @mentions from message content
// Returns a list of mentioned names/types (without the @ symbol, lowercase)
// Example: "@Frontend can you help with @Backend integration?"
// Returns: ["frontend", "backend"]
func ParseMentions(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return []string{}
	}

	// Use map to deduplicate
	seen := make(map[string]bool)
	var mentions []string

	for _, match := range matches {
		if len(match) > 1 {
			mention := strings.ToLower(match[1])
			if !seen[mention] {
				mentions = append(mentions, mention)
				seen[mention] = true
			}
		}
	}

	return mentions
}

// NormalizeAgentName converts agent names to kebab-case format for @mention compatibility
// Converts spaces to hyphens, removes special characters, and ensures valid format
// Example: "Day One Expert" → "day-one-expert", "neural-junkie Expert" → "neural-junkie-expert"
func NormalizeAgentName(name string) string {
	// Trim whitespace
	normalized := strings.TrimSpace(name)

	// Convert to lowercase for consistency
	normalized = strings.ToLower(normalized)

	// Replace spaces and special characters with hyphens
	var result strings.Builder
	for _, char := range normalized {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			result.WriteRune(char)
		} else if char == ' ' || char == '-' || char == '_' || char == '@' || char == '#' || char == '+' || char == '=' || char == '!' || char == '?' || char == '.' || char == ',' || char == ':' || char == ';' || char == '(' || char == ')' || char == '[' || char == ']' || char == '{' || char == '}' || char == '<' || char == '>' || char == '|' || char == '\\' || char == '/' || char == '*' || char == '&' || char == '%' || char == '$' || char == '^' || char == '~' || char == '`' {
			// Only add hyphen if previous character wasn't a hyphen
			if result.Len() > 0 && result.String()[result.Len()-1] != '-' {
				result.WriteRune('-')
			}
		}
	}
	normalized = result.String()

	// Collapse multiple consecutive hyphens to one
	for strings.Contains(normalized, "--") {
		normalized = strings.ReplaceAll(normalized, "--", "-")
	}

	// Remove leading/trailing hyphens
	normalized = strings.Trim(normalized, "-")

	// Ensure we have at least one character
	if normalized == "" {
		normalized = "agent"
	}

	return normalized
}
