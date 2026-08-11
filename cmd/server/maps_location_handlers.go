package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	mapsloc "github.com/camronwood/neural-junkie/internal/maps"
)

func isMapsLocationAPIPath(path string) bool {
	return path == "/api/maps/device-location" ||
		path == "/api/maps/location-requests/pending" ||
		strings.HasPrefix(path, "/api/maps/location-requests/")
}

func handleMapsLocationAPI(w http.ResponseWriter, r *http.Request) {
	if appConfig == nil || appConfig.RouteOwnerPackID("/api/maps") == "" {
		http.Error(w, "Maps API requires the Maps pack", http.StatusForbidden)
		return
	}
	path := r.URL.Path
	store := mapsloc.DefaultLocationStore
	switch {
	case path == "/api/maps/device-location":
		handleDeviceLocation(w, r, store)
	case path == "/api/maps/location-requests/pending" && r.Method == http.MethodGet:
		handleListLocationRequests(w, store)
	case strings.HasPrefix(path, "/api/maps/location-requests/") && strings.HasSuffix(path, "/fulfill") && r.Method == http.MethodPost:
		id := locationRequestID(path, "/fulfill")
		handleFulfillLocationRequest(w, r, store, id)
	case strings.HasPrefix(path, "/api/maps/location-requests/") && strings.HasSuffix(path, "/reject") && r.Method == http.MethodPost:
		id := locationRequestID(path, "/reject")
		handleRejectLocationRequest(w, r, store, id)
	default:
		http.Error(w, "Invalid maps location endpoint", http.StatusNotFound)
	}
}

func locationRequestID(path, suffix string) string {
	trim := strings.TrimPrefix(path, "/api/maps/location-requests/")
	return strings.TrimSuffix(trim, suffix)
}

func handleDeviceLocation(w http.ResponseWriter, r *http.Request, store *mapsloc.LocationStore) {
	switch r.Method {
	case http.MethodGet:
		view, ok := store.Get()
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "shared": false})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "shared": view.Shared, "location": view})
	case http.MethodPost:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var req struct {
			Lat         float64 `json:"lat"`
			Lon         float64 `json:"lon"`
			AccuracyM   float64 `json:"accuracy_m"`
			DisplayName string  `json:"display_name"`
			CapturedAt  string  `json:"captured_at"`
			SessionID   string  `json:"session_id"`
			Shared      *bool   `json:"shared"`
			Source      string  `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if req.Lat < -90 || req.Lat > 90 || req.Lon < -180 || req.Lon > 180 {
			http.Error(w, "lat/lon out of range", http.StatusBadRequest)
			return
		}
		shared := true
		if req.Shared != nil {
			shared = *req.Shared
		}
		snap := mapsloc.DeviceSnapshot{
			Lat:         req.Lat,
			Lon:         req.Lon,
			AccuracyM:   req.AccuracyM,
			DisplayName: strings.TrimSpace(req.DisplayName),
			SessionID:   strings.TrimSpace(req.SessionID),
			Shared:      shared,
			Source:      strings.TrimSpace(req.Source),
		}
		if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(req.CapturedAt)); err == nil {
			snap.CapturedAt = ts.UTC()
		}
		view := store.Publish(snap)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "location": view})
	case http.MethodDelete:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		store.Clear()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleListLocationRequests(w http.ResponseWriter, store *mapsloc.LocationStore) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(store.ListPending())
}

func handleFulfillLocationRequest(w http.ResponseWriter, r *http.Request, store *mapsloc.LocationStore, id string) {
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		http.Error(w, "request id required", http.StatusBadRequest)
		return
	}
	var req struct {
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		AccuracyM   float64 `json:"accuracy_m"`
		DisplayName string  `json:"display_name"`
		CapturedAt  string  `json:"captured_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lon < -180 || req.Lon > 180 {
		http.Error(w, "lat/lon out of range", http.StatusBadRequest)
		return
	}
	snap := mapsloc.DeviceSnapshot{
		Lat:         req.Lat,
		Lon:         req.Lon,
		AccuracyM:   req.AccuracyM,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Source:      "locate",
	}
	if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(req.CapturedAt)); err == nil {
		snap.CapturedAt = ts.UTC()
	}
	out, err := store.Fulfill(id, snap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func handleRejectLocationRequest(w http.ResponseWriter, r *http.Request, store *mapsloc.LocationStore, id string) {
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		http.Error(w, "request id required", http.StatusBadRequest)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	out, err := store.Reject(id, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
