package cli

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
)

func TestManager_StatusForFeaturedCLIs(t *testing.T) {
	m := NewManager()
	for _, cliType := range agent.FeaturedCLITypes {
		st, err := m.StatusFor(cliType, nil)
		if err != nil {
			t.Fatalf("StatusFor(%q): %v", cliType, err)
		}
		if !st.Featured {
			t.Errorf("%q should be featured", cliType)
		}
		if !st.CanInstall {
			t.Errorf("%q should support install", cliType)
		}
		if st.Auth == nil {
			t.Errorf("%q should have auth spec", cliType)
		}
	}
}

func TestManager_AuthLoginInfo(t *testing.T) {
	m := NewManager()
	info, err := m.AuthLoginInfo("cursor")
	if err != nil {
		t.Fatal(err)
	}
	if info["command"] != "agent login" {
		t.Fatalf("got command %q", info["command"])
	}
}

func TestManager_InstallUnknownType(t *testing.T) {
	m := NewManager()
	err := m.Install(t.Context(), "not-a-cli", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
