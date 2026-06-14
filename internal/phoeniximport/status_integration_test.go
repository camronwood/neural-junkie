package phoeniximport

import (
	"context"
	"strings"
	"testing"
)

// Integration test — uses local bbio credentials + auth config when PHOENIX_INTEGRATION=1.
func TestCheckStatusIntegration(t *testing.T) {
	settings := phoenixIntegrationSettings(t)
	st := requirePhoenixAuthenticated(t, settings)
	if st.Identity == "" {
		t.Fatal("expected identity")
	}
}

func TestListAnalysesIntegration(t *testing.T) {
	settings := phoenixIntegrationSettings(t)
	requirePhoenixAuthenticated(t, settings)
	items, err := ListAnalyses(context.Background(), settings, 5)
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
