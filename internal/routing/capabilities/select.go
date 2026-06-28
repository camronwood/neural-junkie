package capabilities

import "strings"

// SelectResult holds a model pick outcome.
type SelectResult struct {
	Tag    string
	Reason string
}

// SelectOllamaTag picks the first installed tag from the ranked list for class.
func SelectOllamaTag(p *Profiles, class TaskClass, installed map[string]struct{}, agentDefault string) SelectResult {
	return SelectOllamaTagWithFilter(p, class, installed, agentDefault, nil)
}

// SelectOllamaTagWithFilter picks a ranked tag; tagFilter skips candidates when it returns false.
func SelectOllamaTagWithFilter(p *Profiles, class TaskClass, installed map[string]struct{}, agentDefault string, tagFilter func(string) bool) SelectResult {
	if p == nil {
		return SelectResult{Tag: strings.TrimSpace(agentDefault), Reason: "capability_profiles_missing"}
	}
	for _, tag := range p.Tags(class) {
		tag = strings.TrimSpace(tag)
		if tag == "" || !tagInstalled(installed, tag) {
			continue
		}
		if tagFilter != nil && !tagFilter(tag) {
			continue
		}
		return SelectResult{Tag: tag, Reason: reasonForClass(class)}
	}
	def := strings.TrimSpace(agentDefault)
	if def != "" && tagFilterOk(tagFilter, def) {
		return SelectResult{Tag: def, Reason: "capability_fallback_agent"}
	}
	return SelectResult{Reason: "capability_no_model"}
}

// SelectOllamaTagRespectingAgent keeps the agent's configured model when it is installed.
// It only substitutes when the default is missing, fails tagFilter, or a strictly
// higher-ranked profile tag should upgrade it.
func SelectOllamaTagRespectingAgent(p *Profiles, class TaskClass, installed map[string]struct{}, agentDefault string, tagFilter func(string) bool) SelectResult {
	agentDefault = strings.TrimSpace(agentDefault)
	if agentDefault == "" {
		return SelectOllamaTagWithFilter(p, class, installed, agentDefault, tagFilter)
	}

	agentInstalled := tagInstalled(installed, agentDefault)
	agentPassesFilter := tagFilterOk(tagFilter, agentDefault)

	if agentInstalled && agentPassesFilter {
		return SelectResult{Tag: agentDefault, Reason: "capability_keep_agent"}
	}

	if agentInstalled && !agentPassesFilter && tagFilter != nil {
		if pick := SelectOllamaTagWithFilter(p, class, installed, "", tagFilter); pick.Tag != "" {
			return SelectResult{Tag: pick.Tag, Reason: "capability_tool_upgrade"}
		}
	}

	if !agentInstalled {
		if pick := SelectOllamaTagWithFilter(p, class, installed, agentDefault, tagFilter); pick.Tag != "" {
			return pick
		}
	}

	return SelectResult{Tag: agentDefault, Reason: "capability_fallback_agent"}
}

func tagFilterOk(tagFilter func(string) bool, tag string) bool {
	return tagFilter == nil || tagFilter(tag)
}

// RequiresToolCapableModel reports whether a task class needs native tool calling.
func RequiresToolCapableModel(class TaskClass) bool {
	switch class {
	case TaskImplement, TaskImplementHeavy:
		return true
	default:
		return false
	}
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
		// Agent-specific models not in benchmark profiles stay on their configured tag.
		return false
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
