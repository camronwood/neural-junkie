package runbooklibrary

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

// ValidateDefinition checks a definition before save or start.
func ValidateDefinition(def *RunbookDefinition, inputs map[string]string) []string {
	if def == nil {
		return []string{"definition is nil"}
	}
	var warnings []string
	if len(def.Tasks) == 0 {
		warnings = append(warnings, "no tasks defined")
	}
	if err := collaboration.ValidateDAG(def.Tasks); err != nil {
		warnings = append(warnings, err.Error())
	}
	pool := map[string]bool{}
	for _, id := range def.AgentIDs {
		pool[id] = true
	}
	for _, t := range def.Tasks {
		if t.EffectiveKind() == collaboration.TaskKindAgent && t.AssignedTo != "" && len(pool) > 0 && !pool[t.AssignedTo] {
			warnings = append(warnings, fmt.Sprintf("task %q assignee not in agent pool", t.Title))
		}
		if t.EffectiveKind() == collaboration.TaskKindAction && t.Action != nil {
			typ := strings.ToLower(strings.TrimSpace(t.Action.Type))
			switch typ {
			case "http_get", "http_post", "webhook":
				url := actionURL(t.Action.Config)
				if url == "" {
					warnings = append(warnings, fmt.Sprintf("action task %q missing url", t.Title))
				}
			}
		}
	}
	for _, in := range def.Inputs {
		if !in.Required {
			continue
		}
		val := strings.TrimSpace(inputs[in.Key])
		if val == "" {
			val = strings.TrimSpace(in.Default)
		}
		if val == "" {
			warnings = append(warnings, fmt.Sprintf("required input %q is missing", in.Key))
		}
	}
	return warnings
}

func actionURL(cfg map[string]interface{}) string {
	if cfg == nil {
		return ""
	}
	if u, ok := cfg["url"].(string); ok {
		return strings.TrimSpace(u)
	}
	return ""
}

// MergeInputDefaults fills missing inputs from definition defaults.
func MergeInputDefaults(def *RunbookDefinition, inputs map[string]string) map[string]string {
	out := map[string]string{}
	if def == nil {
		return out
	}
	for _, in := range def.Inputs {
		if v := strings.TrimSpace(inputs[in.Key]); v != "" {
			out[in.Key] = v
		} else if d := strings.TrimSpace(in.Default); d != "" {
			out[in.Key] = d
		}
	}
	for k, v := range inputs {
		if strings.TrimSpace(v) != "" {
			out[k] = v
		}
	}
	return out
}
