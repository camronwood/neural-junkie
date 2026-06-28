package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hfhub"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/packs"
)

func handlePacksRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/packs")
	path = strings.Trim(path, "/")
	if path == "" {
		handlePacksList(w, r)
		return
	}
	if path == "catalog" {
		handlePacksCatalog(w, r)
		return
	}
	if path == "catalog/refresh" {
		handlePacksCatalogRefresh(w, r)
		return
	}
	if path == "install-zip" {
		handlePackInstallZip(w, r)
		return
	}
	if path == "validate" {
		handlePackValidate(w, r)
		return
	}
	if path == "dev-link" {
		handlePackDevLink(w, r)
		return
	}
	if path == "dev-reload" {
		handlePackDevReload(w, r)
		return
	}
	if path == "dev-unlink" {
		handlePackDevUnlink(w, r)
		return
	}
	if path == "customer-context" {
		handleCustomerPackContext(w, r)
		return
	}
	if path == "layout-owner" {
		handlePackLayoutOwner(w, r)
		return
	}
	if path == "updates" {
		handlePackUpdates(w, r)
		return
	}
	parts := strings.Split(path, "/")
	packID := parts[0]
	if len(parts) == 2 && parts[1] == "install" {
		handlePackInstall(w, r, packID)
		return
	}
	if len(parts) == 2 && parts[1] == "install-loras" {
		handlePackInstallLoRAs(w, r, packID)
		return
	}
	if len(parts) == 2 && parts[1] == "acestep-status" {
		handleMusicACEStepStatus(w, r, packID)
		return
	}
	if len(parts) == 2 && parts[1] == "install-acestep" {
		handleMusicACEStepInstall(w, r, packID)
		return
	}
	if len(parts) == 2 && parts[1] == "restart-sidecar" {
		handleMusicSidecarRestart(w, r, packID)
		return
	}
	if len(parts) == 2 && parts[1] == "upgrade" {
		handlePackUpgrade(w, r, packID)
		return
	}
	if len(parts) == 2 && parts[1] == "asset" {
		handlePackAsset(w, r, packID)
		return
	}
	handlePackByID(w, r, packID)
}

func handlePacksList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appConfig.ListPackStatus())
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePacksCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := appConfig.ListPackCatalogStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"packs":       rows,
		"catalog_url": packs.CatalogURL(),
	})
}

func handlePacksCatalogRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	packs.InvalidateCatalogCache()
	rows, err := appConfig.ListPackCatalogStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":      "ok",
		"packs":       rows,
		"catalog_url": packs.CatalogURL(),
	})
}

func handlePackInstallZip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var body struct {
		PackZipBase64 string `json:"pack_zip_base64"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	raw := strings.TrimSpace(body.PackZipBase64)
	if raw == "" {
		http.Error(w, "pack_zip_base64 required", http.StatusBadRequest)
		return
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		http.Error(w, "invalid base64 zip payload", http.StatusBadRequest)
		return
	}
	m, err := appConfig.InstallPackFromZip(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := appConfig.Save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writePacksMutationResponse(w, m.ID, map[string]any{
		"pack_id":   m.ID,
		"title":     m.Title,
		"version":   m.Version,
		"custom":    m.IsCustomerPack(),
		"publisher": m.Publisher,
		"installed": true,
		"enabled":   false,
	})
}

func handlePackValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var body struct {
		PackZipBase64 string `json:"pack_zip_base64"`
		PackDir       string `json:"pack_dir"`
		PackYAML      string `json:"pack_yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	var report *packs.ValidationReport
	var err error
	switch {
	case strings.TrimSpace(body.PackZipBase64) != "":
		data, decErr := base64.StdEncoding.DecodeString(strings.TrimSpace(body.PackZipBase64))
		if decErr != nil {
			http.Error(w, "invalid base64 zip payload", http.StatusBadRequest)
			return
		}
		report, err = appConfig.ValidatePackZip(data)
	case strings.TrimSpace(body.PackYAML) != "":
		// Prefer in-memory YAML (editor live validate); pack_dir is the asset root.
		report, err = appConfig.ValidatePackYAML(body.PackYAML, strings.TrimSpace(body.PackDir))
	case strings.TrimSpace(body.PackDir) != "":
		report, err = appConfig.ValidatePackDir(strings.TrimSpace(body.PackDir))
	default:
		http.Error(w, "pack_zip_base64, pack_dir, or pack_yaml required", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func handlePackDevLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var body struct {
		PackDir string `json:"pack_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	m, err := appConfig.DevLinkPack(strings.TrimSpace(body.PackDir))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := appConfig.Save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writePacksMutationResponse(w, m.ID, map[string]any{
		"pack_id":         m.ID,
		"title":           m.Title,
		"version":         m.Version,
		"custom":          m.IsCustomerPack(),
		"dev_linked":      true,
		"dev_source_path": appConfig.DevSourcePath(m.ID),
		"installed":       true,
		"enabled":         false,
	})
}

func handlePackDevReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var body struct {
		PackID string `json:"pack_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	packID := strings.TrimSpace(body.PackID)
	if packID == "" {
		http.Error(w, "pack_id required", http.StatusBadRequest)
		return
	}
	m, err := appConfig.DevReloadPack(packID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if appConfig.IsPackEnabled(packID) {
		syncMCPFromConfig()
		globalProviderCache.Clear()
		if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
			ch.SetProviderRegistry(appConfig, globalProviderCache)
		}
		reconcileConfiguredSpecialists()
		initializeConfiguredAgents()
	}
	if err := appConfig.Save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writePacksMutationResponse(w, m.ID, map[string]any{
		"pack_id":         m.ID,
		"dev_linked":      true,
		"dev_source_path": appConfig.DevSourcePath(m.ID),
	})
}

func handlePackDevUnlink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var body struct {
		PackID string `json:"pack_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	packID := strings.TrimSpace(body.PackID)
	if packID == "" {
		http.Error(w, "pack_id required", http.StatusBadRequest)
		return
	}
	if err := appConfig.DevUnlinkPack(packID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := appConfig.Save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writePacksMutationResponse(w, packID, map[string]any{
		"pack_id":    packID,
		"dev_linked": false,
	})
}

func handleCustomerPackContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctxs, err := appConfig.EnabledCustomerPackContexts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"packs": ctxs})
}

func handlePackInstall(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	if err := appConfig.InstallPack(packID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := appConfig.Save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writePacksMutationResponse(w, packID, nil)
}

func handlePackInstallLoRAs(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	if !requireLoRACapability(w, capLoRAAdapters) {
		return
	}
	if packID != config.PackSpecialistTuning {
		http.Error(w, "LoRA install is only available for the Specialist tuning pack", http.StatusBadRequest)
		return
	}
	if hfMgr == nil {
		http.Error(w, "HF manager not initialized", http.StatusInternalServerError)
		return
	}
	m, err := appConfig.InstalledPackManifestByID(packID)
	if err != nil || m == nil {
		http.Error(w, "pack not found or not installed", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	token := hfhub.TokenFromConfig(appConfig)
	results, err := hfhub.InstallPackLoRAs(ctx, hfMgr, m, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"pack_id": packID,
		"results": results,
	})
}

func handlePackByID(w http.ResponseWriter, r *http.Request, packID string) {
	switch r.Method {
	case http.MethodPut, http.MethodDelete:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
	}
	switch r.Method {
	case http.MethodPut:
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := appConfig.SetPackEnabled(packID, body.Enabled); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		syncMCPFromConfig()
		globalProviderCache.Clear()
		if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
			ch.SetProviderRegistry(appConfig, globalProviderCache)
		}
		if err := appConfig.Save(); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		reconcileConfiguredSpecialists()
		initializeConfiguredAgents()
		if body.Enabled {
			triggerEnsurePackLoRAs(packID)
		}
		writePacksMutationResponse(w, packID, map[string]any{"enabled": body.Enabled})
	case http.MethodDelete:
		if err := appConfig.UninstallPack(packID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		syncMCPFromConfig()
		globalProviderCache.Clear()
		if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
			ch.SetProviderRegistry(appConfig, globalProviderCache)
		}
		if err := appConfig.Save(); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		reconcileConfiguredSpecialists()
		initializeConfiguredAgents()
		writePacksMutationResponse(w, packID, nil)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePackLayoutOwner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var body struct {
		LayoutOwner string `json:"layout_owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := appConfig.SetLayoutOwner(body.LayoutOwner); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := appConfig.Save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writePacksMutationResponse(w, body.LayoutOwner, map[string]any{"layout_owner": body.LayoutOwner})
}

func handlePackAsset(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !appConfig.IsPackInstalled(packID) {
		http.Error(w, "Pack not installed", http.StatusNotFound)
		return
	}
	rel := strings.TrimSpace(r.URL.Query().Get("path"))
	if rel == "" {
		http.Error(w, "path query parameter required", http.StatusBadRequest)
		return
	}
	dir, err := packs.InstalledPackDir(packID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	abs, err := packs.ResolvePackRelativePath(dir, rel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}
	if ct := mime.TypeByExtension(filepath.Ext(abs)); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeFile(w, r, abs)
}

func writePacksMutationResponse(w http.ResponseWriter, packID string, extra map[string]any) {
	syncPackSidecars()
	st := appConfig.ListPackStatus()
	out := map[string]any{
		"status":               "ok",
		"pack_id":              packID,
		"packs":                st.Packs,
		"layout_owner":         st.LayoutOwner,
		"layout_profile":       st.LayoutProfile,
		"capabilities":         st.Capabilities,
		"capability_registry":  st.CapabilityRegistry,
		"short_id_collisions":  st.ShortIDCollisions,
	}
	for k, v := range extra {
		out[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func handleExpertPresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appConfig.AvailableExpertPresets())
}
