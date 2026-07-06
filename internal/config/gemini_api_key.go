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
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		_ = os.Setenv("GEMINI_API_KEY", line)
		return line
	}
	return ""
}
