package capabilities

import "strings"

// SelectResult holds a model pick outcome.
type SelectResult struct {
	Tag    string
	Reason string
}

// SelectOllamaTag picks the first installed tag from the ranked list for class.
func SelectOllamaTag(p *Profiles, class TaskClass, installed map[string]struct{}, agentDefault string) SelectResult {
	if p == nil {
		return SelectResult{Tag: strings.TrimSpace(agentDefault), Reason: "capability_profiles_missing"}
	}
	for _, tag := range p.Tags(class) {
		tag = strings.TrimSpace(tag)
		if tag != "" && tagInstalled(installed, tag) {
			return SelectResult{Tag: tag, Reason: reasonForClass(class)}
		}
	}
	def := strings.TrimSpace(agentDefault)
	if def != "" {
		return SelectResult{Tag: def, Reason: "capability_fallback_agent"}
	}
	return SelectResult{Reason: "capability_no_model"}
}

// ShouldUpgrade reports whether candidate ranks higher than current for class.
func ShouldUpgrade(p *Profiles, class TaskClass, current, candidate string) bool {
	if p == nil {
		return false
	}
	current = strings.TrimSpace(current)
	candidate = strings.TrimSpace(candidate)
	if candidate == "" || candidate == current {
		return false
	}
	curRank := p.RankIndex(class, current)
	candRank := p.RankIndex(class, candidate)
	if candRank < 0 {
		return false
	}
	if curRank < 0 {
		return true
	}
	return candRank < curRank
}

func reasonForClass(class TaskClass) string {
	switch class {
	case TaskCollabLight:
		return "capability_collab_light"
	case TaskChat:
		return "capability_chat"
	case TaskUtility:
		return "capability_utility"
	case TaskAskMode:
		return "capability_ask_mode"
	case TaskImplementHeavy:
		return "capability_implement_heavy"
	default:
		return "capability_implement"
	}
}

func tagInstalled(installed map[string]struct{}, tag string) bool {
	if len(installed) == 0 {
		return true
	}
	if _, ok := installed[tag]; ok {
		return true
	}
	base := tag
	if i := strings.Index(tag, ":"); i >= 0 {
		base = tag[:i]
	}
	for name := range installed {
		if name == tag || strings.HasPrefix(name, base+":") {
			return true
		}
	}
	return false
}
