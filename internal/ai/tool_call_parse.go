package ai

import (
	"encoding/json"
	"regexp"
	"strings"
)

var toolCallTagRE = regexp.MustCompile(`(?is)<tool_call>\s*(\{.*?\})\s*</tool_call>`)

// ParseToolCallFromText detects a tool invocation in model output text.
// Supports <tool_call> tags, bare JSON, and code-fence patterns.
func ParseToolCallFromText(text string) (name string, input json.RawMessage, ok bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil, false
	}
	if m := toolCallTagRE.FindStringSubmatch(text); len(m) == 2 {
		if name, input, ok = tryParseToolCallJSON(m[1]); ok {
			return name, input, true
		}
	}
	for _, candidate := range toolCallCandidates(text) {
		if name, input, ok = tryParseToolCallJSON(candidate); ok {
			return name, input, true
		}
	}
	return "", nil, false
}

// StripToolCallFromText removes tool-call markup from text for user-facing replies.
func StripToolCallFromText(text string) string {
	text = toolCallTagRE.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

func toolCallCandidates(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	var candidates []string
	seen := make(map[string]bool)
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		candidates = append(candidates, s)
	}

	add(stripOuterCodeFence(text))
	for _, block := range extractInlineCodeFences(text) {
		add(block)
	}
	for _, obj := range extractJSONObjectStrings(text) {
		add(obj)
	}
	return candidates
}

func stripOuterCodeFence(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}

func extractInlineCodeFences(text string) []string {
	var blocks []string
	rest := text
	for {
		idx := strings.Index(rest, "```")
		if idx < 0 {
			break
		}
		rest = rest[idx+3:]
		if strings.HasPrefix(rest, "json") {
			rest = rest[4:]
		}
		end := strings.Index(rest, "```")
		if end < 0 {
			break
		}
		blocks = append(blocks, strings.TrimSpace(rest[:end]))
		rest = rest[end+3:]
	}
	return blocks
}

func extractJSONObjectStrings(text string) []string {
	var out []string
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		if end, ok := balancedJSONObjectEnd(text, i); ok {
			out = append(out, text[i:end+1])
			i = end
		}
	}
	return out
}

func balancedJSONObjectEnd(s string, start int) (end int, ok bool) {
	if start >= len(s) || s[start] != '{' {
		return 0, false
	}
	depth := 0
	inString := false
	escape := false
	for j := start; j < len(s); j++ {
		c := s[j]
		if escape {
			escape = false
			continue
		}
		if inString {
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return j, true
			}
		}
	}
	return 0, false
}

func tryParseToolCallJSON(text string) (name string, input json.RawMessage, ok bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil, false
	}
	var payload struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
		Input     map[string]interface{} `json:"input"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return "", nil, false
	}
	name = strings.TrimSpace(payload.Name)
	if name == "" {
		return "", nil, false
	}
	args := payload.Arguments
	if args == nil {
		args = payload.Input
	}
	if args == nil {
		args = map[string]interface{}{}
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return "", nil, false
	}
	return name, raw, true
}
