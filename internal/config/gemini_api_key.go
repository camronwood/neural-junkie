package config

import (
	"os"
	"strings"
)

const geminiAPIKeyFileName = ".gemini-api-key"

// GeminiAPIKeyFromEnvOrFile returns GEMINI_API_KEY from the environment, or the
// trimmed contents of .gemini-api-key in the current working directory.
func GeminiAPIKeyFromEnvOrFile() string {
	if v := strings.TrimSpace(os.Getenv("GEMINI_API_KEY")); v != "" {
		return v
	}
	data, err := os.ReadFile(geminiAPIKeyFileName)
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(data))
	if v != "" {
		_ = os.Setenv("GEMINI_API_KEY", v)
	}
	return v
}
