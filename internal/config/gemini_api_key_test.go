package config

import (
	"os"
	"testing"
)

func TestGeminiAPIKeyFromEnvOrFilePrefersEnv(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "from-env")
	if got := GeminiAPIKeyFromEnvOrFile(); got != "from-env" {
		t.Fatalf("got %q", got)
	}
}

func TestGeminiAPIKeyFromEnvOrFileReadsFile(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(geminiAPIKeyFileName, []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := GeminiAPIKeyFromEnvOrFile(); got != "from-file" {
		t.Fatalf("got %q", got)
	}
}
