package phoeniximport

import (
	"os"
	"strings"
)

// AuthAppConfig holds Auth0 native-app settings (e.g. phoenix-customer-cli creds file).
type AuthAppConfig struct {
	Domain       string
	ClientID     string
	ClientSecret string
}

// ParseAuthConfigFile reads a human-readable creds file (same layout as phoenix-tim-test-suite).
func ParseAuthConfigFile(path string) (AuthAppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AuthAppConfig{}, err
	}
	return parseAuthConfigText(string(data)), nil
}

func parseAuthConfigText(raw string) AuthAppConfig {
	var cfg AuthAppConfig
	lines := strings.Split(raw, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		lower := strings.ToLower(strings.TrimSuffix(line, ":"))
		val := ""
		if i+1 < len(lines) {
			val = strings.TrimSpace(lines[i+1])
		}
		switch {
		case strings.Contains(lower, "domain"):
			if val != "" && !strings.Contains(strings.ToLower(val), "client") {
				cfg.Domain = val
				i++
			}
		case strings.Contains(lower, "client id"):
			if val != "" {
				cfg.ClientID = val
				i++
			}
		case strings.Contains(lower, "client secret"):
			if val != "" {
				cfg.ClientSecret = val
				i++
			}
		}
	}
	return cfg
}
