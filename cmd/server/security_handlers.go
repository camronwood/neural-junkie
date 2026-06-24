package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
)

type systemSecuritySnapshot struct {
	HubTokenConfigured  bool `json:"hub_token_configured"`
	AuthRequired        bool `json:"auth_required"`
	RelaxedLocal        bool `json:"relaxed_local"`
	BootstrapConfigured bool `json:"bootstrap_configured"`
	ListenAll           bool `json:"listen_all"`
	LoopbackOnly        bool `json:"loopback_only"`
}

func handleSystemSecurity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap := systemSecuritySnapshot{
		HubTokenConfigured:  hub.HubTokenConfigured(),
		AuthRequired:        hub.AuthRequired(),
		RelaxedLocal:        hub.RelaxedLocal(),
		BootstrapConfigured: hub.BootstrapConfigured(),
		ListenAll:           strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_LISTEN_ALL")) == "1",
		LoopbackOnly:        !strings.EqualFold(strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_LISTEN_ALL")), "1"),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}
