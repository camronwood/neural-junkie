package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendGeminiCLIEnvUsesAPIKeyAndHeadlessHome(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "test-key")
	root := t.TempDir()
	home := filepath.Join(root, "gemini-headless-home")
	if err := os.MkdirAll(filepath.Join(home, ".gemini"), 0o755); err != nil {
		t.Fatal(err)
	}
	settings := `{"security":{"auth":{"selectedType":"gemini-api-key"}}}`
	if err := os.WriteFile(filepath.Join(home, ".gemini", "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEURAL_JUNKIE_GEMINI_CLI_HOME", home)

	p := &CLIAgentProvider{
		ProviderName: "gemini-cli",
		Model:        "gemini-2.5-flash",
	}
	env := appendGeminiCLIEnv(nil, p)
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "GEMINI_API_KEY=test-key") {
		t.Fatalf("expected API key in env, got %q", joined)
	}
	if !strings.Contains(joined, "GEMINI_CLI_HOME="+home) {
		t.Fatalf("expected headless home in env, got %q", joined)
	}
	if !strings.Contains(joined, "GEMINI_MODEL=gemini-2.5-flash") {
		t.Fatalf("expected model in env, got %q", joined)
	}
}

func TestAppendGeminiCLIEnvSkipsHeadlessHomeWhenUserUsesAPIKeyAuth(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "test-key")
	t.Setenv("NEURAL_JUNKIE_GEMINI_CLI_HOME", "")

	userHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(userHome, ".gemini"), 0o755); err != nil {
		t.Fatal(err)
	}
	userSettings := `{"security":{"auth":{"selectedType":"gemini-api-key"}}}`
	if err := os.WriteFile(filepath.Join(userHome, ".gemini", "settings.json"), []byte(userSettings), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", userHome)

	p := &CLIAgentProvider{ProviderName: "gemini-cli"}
	env := appendGeminiCLIEnv(nil, p)
	for _, e := range env {
		if strings.HasPrefix(e, "GEMINI_CLI_HOME=") {
			t.Fatalf("expected no headless home override, got %q", e)
		}
	}
}

func TestAppendGeminiCLIEnvWithoutAPIKey(t *testing.T) {
	t.Setenv(geminiAPIKeyEnv, "")
	p := &CLIAgentProvider{ProviderName: "gemini-cli"}
	env := appendGeminiCLIEnv(nil, p)
	for _, e := range env {
		if strings.HasPrefix(e, "GEMINI_API_KEY=") || strings.HasPrefix(e, "GEMINI_CLI_HOME=") {
			t.Fatalf("unexpected gemini auth env %q", e)
		}
	}
}
