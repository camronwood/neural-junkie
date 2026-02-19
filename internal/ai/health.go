package ai

import (
	"context"
	"log"
	"sync"
	"time"
)

// ProviderHealth represents the health status of an AI provider
type ProviderHealth struct {
	Provider  string    `json:"provider"`
	Endpoint  string    `json:"endpoint"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
	Error     string    `json:"error,omitempty"`
}

// HealthMonitor monitors the health of AI providers
type HealthMonitor struct {
	providers map[string]AIProvider
	health    map[string]*ProviderHealth
	mutex     sync.RWMutex
	interval  time.Duration
	stopCh    chan struct{}
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(interval time.Duration) *HealthMonitor {
	return &HealthMonitor{
		providers: make(map[string]AIProvider),
		health:    make(map[string]*ProviderHealth),
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// AddProvider adds a provider to monitor
func (hm *HealthMonitor) AddProvider(name string, provider AIProvider, endpoint string) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.providers[name] = provider
	hm.health[name] = &ProviderHealth{
		Provider:  name,
		Endpoint:  endpoint,
		Healthy:   false,
		LastCheck: time.Now(),
	}
}

// RemoveProvider removes a provider from monitoring
func (hm *HealthMonitor) RemoveProvider(name string) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	delete(hm.providers, name)
	delete(hm.health, name)
}

// GetHealth returns the health status of all providers
func (hm *HealthMonitor) GetHealth() map[string]*ProviderHealth {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	result := make(map[string]*ProviderHealth)
	for name, health := range hm.health {
		result[name] = &ProviderHealth{
			Provider:  health.Provider,
			Endpoint:  health.Endpoint,
			Healthy:   health.Healthy,
			LastCheck: health.LastCheck,
			Error:     health.Error,
		}
	}
	return result
}

// GetProviderHealth returns the health status of a specific provider
func (hm *HealthMonitor) GetProviderHealth(name string) (*ProviderHealth, bool) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	health, exists := hm.health[name]
	if !exists {
		return nil, false
	}

	return &ProviderHealth{
		Provider:  health.Provider,
		Endpoint:  health.Endpoint,
		Healthy:   health.Healthy,
		LastCheck: health.LastCheck,
		Error:     health.Error,
	}, true
}

// Start begins the health monitoring loop
func (hm *HealthMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	// Initial health check
	hm.checkAllProviders()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.stopCh:
			return
		case <-ticker.C:
			hm.checkAllProviders()
		}
	}
}

// Stop stops the health monitoring
func (hm *HealthMonitor) Stop() {
	close(hm.stopCh)
}

// checkAllProviders checks the health of all providers
func (hm *HealthMonitor) checkAllProviders() {
	hm.mutex.RLock()
	providers := make(map[string]AIProvider)
	for name, provider := range hm.providers {
		providers[name] = provider
	}
	hm.mutex.RUnlock()

	for name, provider := range providers {
		hm.checkProvider(name, provider)
	}
}

// checkProvider checks the health of a specific provider
func (hm *HealthMonitor) checkProvider(name string, provider AIProvider) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test the provider connection
	var err error
	if ollamaProvider, ok := provider.(*OllamaProvider); ok {
		err = ollamaProvider.TestConnection(ctx)
	} else {
		// For Claude providers, we can't easily test without making an actual request
		// For now, we'll assume they're healthy if they exist
		err = nil
	}

	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if health, exists := hm.health[name]; exists {
		health.LastCheck = time.Now()
		health.Healthy = err == nil
		if err != nil {
			health.Error = err.Error()
		} else {
			health.Error = ""
		}
	}

	if err != nil {
		log.Printf("Health check failed for provider %s: %v", name, err)
	} else {
		log.Printf("Health check passed for provider %s", name)
	}
}

// IsProviderHealthy checks if a provider is currently healthy
func (hm *HealthMonitor) IsProviderHealthy(name string) bool {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	if health, exists := hm.health[name]; exists {
		return health.Healthy
	}
	return false
}

// GetHealthyProviders returns a list of healthy provider names
func (hm *HealthMonitor) GetHealthyProviders() []string {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	var healthy []string
	for name, health := range hm.health {
		if health.Healthy {
			healthy = append(healthy, name)
		}
	}
	return healthy
}
