package ai

import "testing"

func TestImageGenConfigFromEnvOllamaDefaults(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_IMAGE_PROVIDER", "")
	t.Setenv("OLLAMA_ENDPOINT", "")
	t.Setenv("OLLAMA_IMAGE_MODEL", "")
	t.Setenv("NEURAL_JUNKIE_IMAGE_MODEL", "")

	cfg := ImageGenConfigFromEnv()
	if cfg.Provider != "ollama" {
		t.Fatalf("provider = %q", cfg.Provider)
	}
	if cfg.Model != DefaultOllamaImageModel {
		t.Fatalf("model = %q", cfg.Model)
	}
	if cfg.Endpoint != "http://localhost:11434" {
		t.Fatalf("endpoint = %q", cfg.Endpoint)
	}
}

func TestBuildImageGenStatusOllamaReady(t *testing.T) {
	cfg := ImageGenConfig{Provider: "ollama", Model: "x/flux2-klein:4b", Endpoint: "http://localhost:11434"}
	st := BuildImageGenStatus(cfg, true, true)
	if !st.Ready || !st.ModelPulled || st.PullCommand == "" {
		t.Fatalf("status = %+v", st)
	}
}

func TestBuildImageGenStatusOpenAI(t *testing.T) {
	cfg := ImageGenConfig{Provider: "openai", Model: "dall-e-3", OpenAIKeySet: true}
	st := BuildImageGenStatus(cfg, false, false)
	if !st.Ready {
		t.Fatal("expected ready with openai key")
	}
}

func TestBuildImageGenStatusDisabled(t *testing.T) {
	cfg := ImageGenConfig{Provider: "none"}
	st := BuildImageGenStatus(cfg, true, true)
	if !st.Disabled || st.Ready {
		t.Fatalf("status = %+v", st)
	}
}
