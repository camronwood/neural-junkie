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
	matches := mentionRegex.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return []string{}
	}

	// Use map to deduplicate
	seen := make(map[string]bool)
	var mentions []string

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		end := match[1]
		// Skip IDE scoped-path syntax (@file:path, @folder:path) — not agent mentions.
		if end < len(content) && content[end] == ':' {
			continue
		}
		mention := strings.ToLower(content[match[2]:match[3]])
		if IsSlackMentionToken(mention) {
			continue
		}
		if !seen[mention] {
			mentions = append(mentions, mention)
			seen[mention] = true
		}
	}

	return mentions
}

// IsCollabTemplateMentionToken reports @placeholders in plan prose (not real agents).
// Filter these at collab routing/task-parse time — not in ParseMentions, which is the
// general protocol contract (see test/mentions_protocol_test.go).
func IsCollabTemplateMentionToken(mention string) bool {
	switch strings.ToLower(strings.TrimSpace(mention)) {
	case "agentname", "agent":
		return true
	default:
		return false
	}
}

// FilterCollabTemplateMentions removes plan-template @tokens before hub resolution.
func FilterCollabTemplateMentions(mentions []string) []string {
	if len(mentions) == 0 {
		return mentions
	}
	out := make([]string, 0, len(mentions))
	for _, m := range mentions {
		if !IsCollabTemplateMentionToken(m) {
			out = append(out, m)
		}
	}
	return out
}

// IsSlackMentionToken reports @tokens that look like Slack IDs (U0B5MLY2N2E → u0b5mly2n2e).
// Real Slack IDs (users, bots, channels, etc.) are alphanumeric and always include a digit.
// Without the digit requirement, long agent names like @SecurityExpert were misclassified.
func IsSlackMentionToken(mention string) bool {
	mention = strings.ToLower(strings.TrimSpace(mention))
	if len(mention) < 9 {
		return false
	}
	hasDigit := false
	switch mention[0] {
	case 'u', 'b', 'w', 'c', 't', 'd', 'e', 'g', 'f', 'p', 's', 'z':
		for _, r := range mention[1:] {
			if r >= '0' && r <= '9' {
				hasDigit = true
				continue
			}
			if r < 'a' || r > 'z' {
				return false
			}
		}
		return hasDigit
	default:
		return false
	}
}

// NormalizeAgentName sanitizes agent names for @mention compatibility while preserving
// caller casing (e.g. "GuitarCoach", "RustExpert").
// Spaces and punctuation become hyphens: "Day One Expert" → "Day-One-Expert".
func NormalizeAgentName(name string) string {
	// Trim whitespace
	normalized := strings.TrimSpace(name)

	// Replace spaces and special characters with hyphens
	var result strings.Builder
	for _, char := range normalized {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
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
