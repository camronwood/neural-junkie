package ai

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestOllamaChatOptions_runtimeOverrides(t *testing.T) {
	SetHubRuntimeOptions(config.PerformanceConfig{}, config.OllamaConfig{
		NumCtx:     8192,
		NumPredict: 256,
	})
	opts := ollamaChatOptions("qwen2.5-coder:14b")
	if opts["num_ctx"] != 8192 || opts["num_predict"] != 256 {
		t.Fatalf("opts = %#v", opts)
	}
}

func TestOllamaChatOptions_defaultCapsNativeToolModels(t *testing.T) {
	SetHubRuntimeOptions(config.PerformanceConfig{}, config.OllamaConfig{})
	opts := ollamaChatOptions("qwen3.5:9b")
	if opts["num_predict"] != 512 {
		t.Fatalf("default num_predict = %#v, want 512", opts["num_predict"])
	}
}

func TestOllamaKeepAliveValue(t *testing.T) {
	SetHubRuntimeOptions(config.PerformanceConfig{}, config.OllamaConfig{KeepAlive: "5m"})
	if ollamaKeepAliveValue() != "5m" {
		t.Fatalf("keep_alive = %#v", ollamaKeepAliveValue())
	}
	SetHubRuntimeOptions(config.PerformanceConfig{}, config.OllamaConfig{KeepAlive: "-1"})
	if ollamaKeepAliveValue() != 0 {
		t.Fatalf("keep_alive unload = %#v", ollamaKeepAliveValue())
	}
}
