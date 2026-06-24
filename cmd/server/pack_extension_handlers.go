package main

import (
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs/sidecar"
)

var packSidecarMgr = sidecar.NewManager()

func syncPackSidecars() {
	if appConfig == nil || packSidecarMgr == nil {
		return
	}
	envs := appConfig.CollectPackSidecarEnvs()
	_ = packSidecarMgr.Sync(appConfig.ContextOrBackground(), envs)
}

func handlePackExtensionRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig == nil {
		http.Error(w, "config not loaded", http.StatusInternalServerError)
		return
	}
	routePrefix := packExtensionRoutePrefix(r.URL.Path)
	if routePrefix == "" {
		http.NotFound(w, r)
		return
	}
	capID := routeCapabilityForRoute(routePrefix)
	if capID != "" && !appConfig.HasPackCapability(capID) {
		http.Error(w, "Capability required for "+routePrefix, http.StatusForbidden)
		return
	}
	packID := appConfig.RouteOwnerPackID(routePrefix)
	if packID == "" {
		http.Error(w, "No enabled pack provides "+routePrefix, http.StatusForbidden)
		return
	}
	if packSidecarMgr == nil {
		http.Error(w, "Pack sidecar manager unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := packSidecarMgr.ProxyHTTP(w, r, packID, r.URL.Path); err != nil {
		http.Error(w, "Pack sidecar: "+err.Error(), http.StatusBadGateway)
	}
}

func packExtensionRoutePrefix(path string) string {
	path = strings.TrimSpace(path)
	for _, prefix := range []string{"/api/phoenix", "/api/scan-summary", "/api/secondary-analysis"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return prefix
		}
	}
	return ""
}

func routeCapabilityForRoute(prefix string) string {
	switch prefix {
	case "/api/phoenix":
		return "phoenix-import"
	case "/api/scan-summary":
		return "scan-summary-api"
	case "/api/secondary-analysis":
		return "secondary-analysis-api"
	default:
		return ""
	}
}

func handlePhoenixRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/phoenix") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	legacyHandlePhoenixRoute(w, r)
}

func handleScanSummaryWellImage(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/scan-summary") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	legacyHandleScanSummaryWellImage(w, r)
}

func handleSecondaryAnalysisRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/secondary-analysis") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	legacyHandleSecondaryAnalysisRoute(w, r)
}
