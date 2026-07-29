package config

import "strings"

// WebSearchConfig configures online lookup for shared web_search / fetch_url tools
// available to all agents when enabled in Settings → Web search.
type WebSearchConfig struct {
	Enabled    bool   `json:"enabled"`
	Provider   string `json:"provider,omitempty"` // tavily (default), brave
	APIKey     string `json:"api_key,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
	// Keyless enables Tavily keyless API when no API key is set (rate limited).
	Keyless bool `json:"keyless,omitempty"`
}

func (w WebSearchConfig) ProviderName() string {
	if p := strings.TrimSpace(strings.ToLower(w.Provider)); p != "" {
		return p
	}
	return "tavily"
}

func (w WebSearchConfig) MaxResultsOrDefault() int {
	if w.MaxResults > 0 && w.MaxResults <= 20 {
		return w.MaxResults
	}
	return 5
}

// Ready reports whether web search tools can call an external provider.
func (w WebSearchConfig) Ready() bool {
	if !w.Enabled {
		return false
	}
	if strings.TrimSpace(w.APIKey) != "" {
		return true
	}
	return w.ProviderName() == "tavily" && w.Keyless
}
