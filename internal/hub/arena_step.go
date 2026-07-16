package hub

import "encoding/json"

func arenaWithStep(session map[string]any, step map[string]any) map[string]any {
	raw, err := json.Marshal(session)
	if err != nil {
		out := map[string]any{}
		for k, v := range session {
			out[k] = v
		}
		out["_arena_step"] = step
		return out
	}
	var out map[string]any
	if json.Unmarshal(raw, &out) != nil || out == nil {
		out = map[string]any{}
		for k, v := range session {
			out[k] = v
		}
	}
	out["_arena_step"] = step
	return out
}

func arenaSkippedStep(session map[string]any, seat, reason string) map[string]any {
	return arenaWithStep(session, map[string]any{
		"skipped": true,
		"reason":  reason,
		"seat":    seat,
	})
}
