package phoeniximport

import (
	"context"
	"os"
	"strings"
	"testing"
)

const defaultPhoenixAuthConfigPath = "/Users/camronwood/development/phoenix-tim-test-suite/.phoenix-customer-cli-creds"

// phoenixIntegrationSettings returns dev TIM settings when live integration is enabled.
// Skips unless PHOENIX_INTEGRATION=1 and auth config exists.
func phoenixIntegrationSettings(t *testing.T) Settings {
	t.Helper()
	if os.Getenv("PHOENIX_INTEGRATION") != "1" {
		t.Skip("set PHOENIX_INTEGRATION=1 to run live Phoenix/TIM integration tests")
	}
	authPath := os.Getenv("PHOENIX_AUTH_CONFIG_PATH")
	if authPath == "" {
		authPath = defaultPhoenixAuthConfigPath
	}
	if _, err := os.Stat(authPath); err != nil {
		t.Skipf("auth config not available: %s", authPath)
	}
	return Settings{Environment: "dev", AuthConfigPath: authPath}
}

func requirePhoenixAuthenticated(t *testing.T, settings Settings) Status {
	t.Helper()
	st := CheckStatus(context.Background(), settings)
	if !st.Authenticated {
		msg := strings.TrimSpace(st.Hint)
		if msg == "" {
			msg = "not authenticated"
		}
		t.Skipf("phoenix credentials unavailable (%s); re-login via bbio or Phoenix device flow", msg)
	}
	return st
}
