package phoeniximport

import (
	"context"
	"os"
	"strings"
	"testing"
)

// Integration test — uses local bbio credentials + auth config when present.
func TestCheckStatusIntegration(t *testing.T) {
	authPath := os.Getenv("PHOENIX_AUTH_CONFIG_PATH")
	if authPath == "" {
		authPath = "/Users/camronwood/development/phoenix-tim-test-suite/.phoenix-customer-cli-creds"
	}
	if _, err := os.Stat(authPath); err != nil {
		t.Skip("auth config not available")
	}
	st := CheckStatus(context.Background(), Settings{
		Environment:    "dev",
		AuthConfigPath: authPath,
	})
	if !st.Authenticated {
		t.Fatalf("expected authenticated: %+v", st)
	}
	if st.Identity == "" {
		t.Fatal("expected identity")
	}
}

func TestListAnalysesIntegration(t *testing.T) {
	authPath := os.Getenv("PHOENIX_AUTH_CONFIG_PATH")
	if authPath == "" {
		authPath = "/Users/camronwood/development/phoenix-tim-test-suite/.phoenix-customer-cli-creds"
	}
	if _, err := os.Stat(authPath); err != nil {
		t.Skip("auth config not available")
	}
	items, err := ListAnalyses(context.Background(), Settings{
		Environment:    "dev",
		AuthConfigPath: authPath,
	}, 5)
	if err != nil {
		if strings.Contains(err.Error(), "HTTP 500") {
			t.Skip("TIM list analyses returned 500 (server-side; auth OK)")
		}
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one analysis")
	}
}
