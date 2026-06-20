package ai

import (
	"sync"

	"github.com/camronwood/neural-junkie/internal/config"
)

var (
	hubRuntimeMu sync.RWMutex
	hubRuntime   config.PerformanceConfig
	ollamaRuntimeMu sync.RWMutex
	ollamaRuntime   config.OllamaConfig
)

// SetHubRuntimeOptions wires performance and Ollama runtime settings from hub config.
func SetHubRuntimeOptions(perf config.PerformanceConfig, ollama config.OllamaConfig) {
	hubRuntimeMu.Lock()
	hubRuntime = perf
	hubRuntimeMu.Unlock()
	ollamaRuntimeMu.Lock()
	ollamaRuntime = ollama
	ollamaRuntimeMu.Unlock()
}

func performanceConfig() config.PerformanceConfig {
	hubRuntimeMu.RLock()
	defer hubRuntimeMu.RUnlock()
	return hubRuntime
}

func ollamaRuntimeConfig() config.OllamaConfig {
	ollamaRuntimeMu.RLock()
	defer ollamaRuntimeMu.RUnlock()
	return ollamaRuntime
}

// MaxHistoryMessages returns configured channel history depth for LLM calls.
func MaxHistoryMessages() int {
	return performanceConfig().MaxHistoryMessagesOrDefault()
}

// PerformanceConfig returns the wired hub performance settings snapshot.
func PerformanceConfig() config.PerformanceConfig {
	return performanceConfig()
}

// OutputShapingEnabled reports whether post-tool verbosity steering is on.
func OutputShapingEnabled() bool {
	return performanceConfig().OutputShapingEnabled
}
