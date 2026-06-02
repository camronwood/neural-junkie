package agent

import (
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/testutil"
)

func TestUserRulesStorageResolveDefaultFallback(t *testing.T) {
	home := testutil.IsolateNeuralJunkieHome(t)
	path := filepath.Join(home, ".neural-junkie", "user-rules.json")
	store, err := NewUserRulesStorage()
	if err != nil {
		t.Fatal(err)
	}
	store.path = path

	if err := store.Set("", "default rules"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("Camron", "camron rules"); err != nil {
		t.Fatal(err)
	}

	if got := store.Resolve("Camron"); got != "camron rules" {
		t.Fatalf("Resolve(Camron) = %q", got)
	}
	if got := store.Resolve("Other"); got != "default rules" {
		t.Fatalf("Resolve(Other) = %q, want default fallback", got)
	}
}
