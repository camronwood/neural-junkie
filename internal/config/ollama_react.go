package config

import "strings"

const defaultReactToolModel = "gemma3:12b"

// ModelUsesReactTools reports whether tag should use the ReAct tool wrapper.
func (o OllamaConfig) ModelUsesReactTools(tag string) bool {
	if !o.ReactToolsEnabled {
		return false
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return false
	}
	for _, m := range o.ReactToolModelsOrDefault() {
		if strings.EqualFold(strings.TrimSpace(m), tag) {
			return true
		}
	}
	return false
}

// ReactToolModelsOrDefault returns configured ReAct allowlist or gemma3:12b.
func (o OllamaConfig) ReactToolModelsOrDefault() []string {
	if len(o.ReactToolModels) > 0 {
		return o.ReactToolModels
	}
	return []string{defaultReactToolModel}
}
