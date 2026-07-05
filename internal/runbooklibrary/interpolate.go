package runbooklibrary

import (
	"encoding/json"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

// InterpolateString replaces runbook template tokens in s.
func InterpolateString(
	s string,
	collab *collaboration.Collaboration,
	task collaboration.CollaborationTask,
	inputs map[string]string,
) string {
	if s == "" {
		return s
	}
	if collab != nil {
		s = strings.ReplaceAll(s, "{{collab.description}}", collab.Description)
	}
	s = strings.ReplaceAll(s, "{{task.title}}", task.Title)
	s = strings.ReplaceAll(s, "{{task.description}}", task.Description)
	for k, v := range inputs {
		s = strings.ReplaceAll(s, "{{inputs."+k+"}}", v)
	}
	if collab != nil {
		for k, v := range collab.RunInputs {
			s = strings.ReplaceAll(s, "{{inputs."+k+"}}", v)
		}
		for _, t := range collab.Tasks {
			if t.Output == "" {
				continue
			}
			prefix := "{{tasks." + t.ID + "."
			if strings.Contains(s, "{{tasks."+t.ID+".output}}") {
				s = strings.ReplaceAll(s, "{{tasks."+t.ID+".output}}", t.Output)
			}
			if strings.Contains(s, prefix) {
				s = interpolateTaskJSON(s, t)
			}
		}
	}
	return s
}

func interpolateTaskJSON(s string, task collaboration.CollaborationTask) string {
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(task.Output), &envelope); err != nil {
		return s
	}
	data, _ := envelope["data"].(map[string]interface{})
	if data == nil {
		return s
	}
	tokenPrefix := "{{tasks." + task.ID + ".data."
	for k, v := range data {
		token := tokenPrefix + k + "}}"
		s = strings.ReplaceAll(s, token, fmtAny(v))
	}
	return s
}

func fmtAny(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return itoa(int(x))
		}
		return strings.TrimRight(strings.TrimRight(fmtFloat(x), "0"), ".")
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func fmtFloat(f float64) string {
	return strings.TrimSpace(strings.TrimRight(strings.TrimRight(
		strings.ReplaceAll(strings.ReplaceAll(
			jsonNumber(f), " ", ""), ",", ""), "0"), "."))
}

func jsonNumber(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

// ApplyInputsToTasks returns a copy of tasks with interpolation applied.
func ApplyInputsToTasks(
	tasks []collaboration.CollaborationTask,
	collab *collaboration.Collaboration,
	inputs map[string]string,
) []collaboration.CollaborationTask {
	out := make([]collaboration.CollaborationTask, len(tasks))
	for i, t := range tasks {
		out[i] = t
		out[i].Title = InterpolateString(t.Title, collab, t, inputs)
		out[i].Description = InterpolateString(t.Description, collab, t, inputs)
		if t.Action != nil {
			cfg := map[string]interface{}{}
			for k, v := range t.Action.Config {
				if str, ok := v.(string); ok {
					cfg[k] = InterpolateString(str, collab, t, inputs)
				} else {
					cfg[k] = v
				}
			}
			out[i].Action = &collaboration.TaskActionSpec{Type: t.Action.Type, Config: cfg}
			if t.Action.ConnectorID != "" {
				out[i].Action.ConnectorID = t.Action.ConnectorID
			}
		}
	}
	return out
}
