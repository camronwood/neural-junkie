package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/camronwood/neural-junkie/internal/awssidecar"
	awsprofiles "github.com/camronwood/neural-junkie/internal/integrations/aws"
	"github.com/camronwood/neural-junkie/internal/integrations/ticketing"
	jiraclient "github.com/camronwood/neural-junkie/internal/integrations/jira"
)

func handleIntegrationsAWSProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	profiles, err := awsprofiles.ListProfiles()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": profiles,
	})
}

func handleIntegrationsAWSTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if appConfig == nil {
		http.Error(w, "config unavailable", http.StatusInternalServerError)
		return
	}
	out, err := awsprofiles.TestCallerIdentity(appConfig.AWSSettings())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error(), "output": out})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "output": out})
}

func handleIntegrationsAWSOrgAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	client := awssidecar.DefaultSidecarClient
	if client == nil {
		http.Error(w, "AWS sidecar not running", http.StatusServiceUnavailable)
		return
	}
	out, err := client.Post(context.Background(), "/api/aws/list-organization-accounts", map[string]any{})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func handleIntegrationsJiraTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if appConfig == nil {
		http.Error(w, "config unavailable", http.StatusInternalServerError)
		return
	}
	client, err := jiraclient.NewClient(appConfig.JiraSettings())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	out, err := client.GetMyself(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "output": out})
}

func handleIntegrationsGitHubTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if appConfig == nil {
		http.Error(w, "config unavailable", http.StatusInternalServerError)
		return
	}
	p := ticketing.NewGitHubProvider(appConfig.GitHubIssuesSettings())
	out, err := p.Search(r.Context(), "is:open", 1)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "output": out})
}

func handleIntegrationsLinearTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if appConfig == nil {
		http.Error(w, "config unavailable", http.StatusInternalServerError)
		return
	}
	p := ticketing.NewLinearProvider(appConfig.LinearSettings())
	out, err := p.Search(r.Context(), "", 1)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "output": out})
}
