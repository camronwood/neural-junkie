package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/connectors"
)

func handleConnectorsRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/connectors")
	path = strings.Trim(path, "/")
	if path == "" {
		switch r.Method {
		case http.MethodGet:
			list, err := connectors.List()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeCollabJSON(w, list)
		case http.MethodPost:
			handleConnectorCreate(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	id := path
	switch r.Method {
	case http.MethodPut:
		handleConnectorUpdate(w, r, id)
	case http.MethodDelete:
		handleConnectorDelete(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleConnectorCreate(w http.ResponseWriter, r *http.Request) {
	var p connectors.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	saved, err := connectors.Create(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeCollabJSON(w, saved)
}

func handleConnectorUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var p connectors.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	saved, err := connectors.Update(id, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeCollabJSON(w, saved)
}

func handleConnectorDelete(w http.ResponseWriter, r *http.Request, id string) {
	if err := connectors.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeCollabJSON(w, map[string]string{"status": "deleted"})
}
