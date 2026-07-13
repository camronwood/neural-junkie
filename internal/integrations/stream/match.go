package stream

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MatchEvent returns true if the event payload matches the subscription filter.
func MatchEvent(match *MatchSpec, payload string) bool {
	if match == nil || strings.TrimSpace(match.JSONPath) == "" {
		return true
	}
	val, ok := lookupJSONPath(payload, match.JSONPath)
	if !ok {
		return false
	}
	want := match.Value
	op := match.Op
	if op == "" {
		op = MatchEquals
	}
	switch op {
	case MatchContains:
		return strings.Contains(val, want)
	default:
		return val == want
	}
}

// BuildRunbookInputs always sets topic and payload, then maps InputMap JSON paths.
func BuildRunbookInputs(ev Event, inputMap map[string]string) map[string]string {
	out := map[string]string{
		"topic":   ev.Topic,
		"payload": ev.Payload,
	}
	if ev.Key != "" {
		out["key"] = ev.Key
	}
	for path, key := range inputMap {
		if key == "" {
			continue
		}
		if v, ok := lookupJSONPath(ev.Payload, path); ok {
			out[key] = v
		}
	}
	return out
}

// RenderTemplate replaces {{payload}}, {{topic}}, and {{key}}.
func RenderTemplate(tmpl, payload, topic, key string) string {
	if tmpl == "" {
		return payload
	}
	r := strings.NewReplacer(
		"{{payload}}", payload,
		"{{topic}}", topic,
		"{{key}}", key,
	)
	return r.Replace(tmpl)
}

func lookupJSONPath(payload, path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, ".")
	var raw interface{}
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return "", false
	}
	cur := raw
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return "", false
		}
		next, ok := m[part]
		if !ok {
			return "", false
		}
		cur = next
	}
	switch v := cur.(type) {
	case string:
		return v, true
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v)), true
		}
		return fmt.Sprintf("%v", v), true
	case bool:
		return fmt.Sprintf("%t", v), true
	case nil:
		return "", true
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), true
		}
		return string(b), true
	}
}
