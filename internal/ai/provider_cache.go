package ai

import (
	"fmt"
	"sync"

	"github.com/camronwood/neural-junkie/internal/config"
)

// ProviderCache holds reusable AIProvider instances keyed by config provider id.
type ProviderCache struct {
	mu    sync.Mutex
	items map[string]AIProvider
}

// NewProviderCache returns an empty cache.
func NewProviderCache() *ProviderCache {
	return &ProviderCache{items: make(map[string]AIProvider)}
}

// Get returns a cached provider for id, or builds it from cfg using ProviderFromConfig.
func (c *ProviderCache) Get(cfg *config.Config, id string) (AIProvider, error) {
	if cfg == nil || id == "" {
		return nil, fmt.Errorf("provider cache: missing config or id")
	}
	p := cfg.GetProvider(id)
	if p == nil {
		return nil, fmt.Errorf("provider %q not found in config", id)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.items[id]; ok {
		return existing, nil
	}
	prov, err := ProviderFromConfig(p)
	if err != nil {
		return nil, err
	}
	c.items[id] = prov
	return prov, nil
}

// Evict removes one id from the cache (call after provider row changes).
func (c *ProviderCache) Evict(id string) {
	if c == nil || id == "" {
		return
	}
	c.mu.Lock()
	delete(c.items, id)
	c.mu.Unlock()
}

// Clear removes all cached providers.
func (c *ProviderCache) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.items = make(map[string]AIProvider)
	c.mu.Unlock()
}
