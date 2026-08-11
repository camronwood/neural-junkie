package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/arenasidecar"
	"github.com/camronwood/neural-junkie/internal/awssidecar"
	"github.com/camronwood/neural-junkie/internal/biologysidecar"
	"github.com/camronwood/neural-junkie/internal/browser"
	"github.com/camronwood/neural-junkie/internal/cadcsidecar"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/incidentsidecar"
	"github.com/camronwood/neural-junkie/internal/maps"
	"github.com/camronwood/neural-junkie/internal/music"
	"github.com/camronwood/neural-junkie/internal/packsidecar"
)

var packSidecarMgr = packsidecar.NewManager()

func initMusicSidecarGenerator() {
	packsidecar.SetGlobalManager(packSidecarMgr)
	baseURL := func() string {
		if packSidecarMgr == nil {
			return ""
		}
		return packSidecarMgr.BaseURL(config.PackMusicCreation)
	}
	music.SidecarBaseURL = baseURL
	music.Default = music.NewSidecarGenerator(baseURL)
}

func initBrowserSidecarClient() {
	baseURL := func() string {
		if packSidecarMgr == nil {
			return ""
		}
		return packSidecarMgr.BaseURL(config.PackWebBrowser)
	}
	browser.SidecarBaseURL = baseURL
	browser.DefaultSidecarClient = browser.NewSidecarClient(baseURL)
}

func initIncidentSidecarClient() {
	baseURL := func() string {
		if packSidecarMgr == nil {
			return ""
		}
		return packSidecarMgr.BaseURL(config.PackIncidentManagement)
	}
	incidentsidecar.SidecarBaseURL = baseURL
}

func initMapsSidecarClient() {
	baseURL := func() string {
		if packSidecarMgr == nil {
			return ""
		}
		return packSidecarMgr.BaseURL(config.PackMaps)
	}
	maps.SidecarBaseURL = baseURL
	maps.DefaultSidecarClient = maps.NewSidecarClient(baseURL)
}

func initAWSSidecarClient() {
	baseURL := func() string {
		if packSidecarMgr == nil {
			return ""
		}
		return packSidecarMgr.BaseURL(config.PackAWS)
	}
	awssidecar.SidecarBaseURL = baseURL
	awssidecar.DefaultSidecarClient = awssidecar.NewSidecarClient(baseURL)
}

func initCADSidecarClient() {
	baseURL := func() string {
		if packSidecarMgr == nil {
			return ""
		}
		return packSidecarMgr.BaseURL(config.PackCAD)
	}
	cadcsidecar.SidecarBaseURL = baseURL
	cadcsidecar.DefaultSidecarClient = cadcsidecar.NewSidecarClient(baseURL)
}

func initBiologySidecarClient() {
	baseURL := func() string {
		if packSidecarMgr == nil {
			return ""
		}
		return packSidecarMgr.BaseURL(config.PackLifeSciences)
	}
	biologysidecar.SidecarBaseURL = baseURL
	biologysidecar.DefaultSidecarClient = biologysidecar.NewSidecarClient(baseURL)
}

func initArenaSidecarClient() {
	baseURL := func() string {
		if packSidecarMgr == nil {
			return ""
		}
		return packSidecarMgr.BaseURL(config.PackModelArena)
	}
	arenasidecar.SidecarBaseURL = baseURL
	arenasidecar.DefaultSidecarClient = arenasidecar.NewSidecarClient(baseURL)
}

func syncPackSidecars() {
	if appConfig == nil || packSidecarMgr == nil {
		return
	}
	packsidecar.SetGlobalManager(packSidecarMgr)
	envs := appConfig.CollectPackSidecarEnvs()
	_ = packSidecarMgr.Sync(appConfig.ContextOrBackground(), envs)
}

func restartPackSidecar(packID string) error {
	if appConfig == nil || packSidecarMgr == nil {
		return fmt.Errorf("pack sidecar manager unavailable")
	}
	packID = strings.TrimSpace(packID)
	for _, env := range appConfig.CollectPackSidecarEnvs() {
		if env.PackID == packID {
			return packSidecarMgr.RestartPack(appConfig.ContextOrBackground(), env)
		}
	}
	return fmt.Errorf("no sidecar configured for pack %q (install and enable the pack first)", packID)
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
	for _, prefix := range []string{"/api/phoenix", "/api/scan-summary", "/api/secondary-analysis", "/api/music", "/api/arena", "/api/ai-interview", "/api/browser", "/api/aws", "/api/biology", "/api/cad", "/api/maps", "/api/lora/sidecar"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return prefix
		}
	}
	return ""
}

func routeCapabilityForRoute(prefix string) string {
	if appConfig != nil {
		if capID := appConfig.CapabilityForRoutePrefix(prefix); capID != "" {
			return capID
		}
	}
	// Fallback for packs that declare routes without registry sidecars (legacy/customer).
	switch prefix {
	case "/api/phoenix":
		return "phoenix-import"
	case "/api/scan-summary":
		return "scan-summary-api"
	case "/api/secondary-analysis":
		return "secondary-analysis-api"
	case "/api/music":
		return "music-sidecar"
	case "/api/arena":
		return "model-arena-sidecar"
	case "/api/ai-interview":
		return "ai-interview-sidecar"
	case "/api/lora/sidecar":
		return "lora-training-sidecar"
	case "/api/browser":
		return "browser-sidecar"
	case "/api/aws":
		return "aws-sidecar"
	case "/api/biology":
		return "biology-sidecar"
	case "/api/cad":
		return "cad-sidecar"
	case "/api/maps":
		return "maps-sidecar"
	default:
		return ""
	}
}

func handleLoraSidecarRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/lora/sidecar") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	http.NotFound(w, r)
}

func handleBrowserRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/browser") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	http.Error(w, "Browser API requires the Web browser pack", http.StatusForbidden)
}

func handleAWSRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/aws") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	http.Error(w, "AWS API requires the AWS pack", http.StatusForbidden)
}

func handleCADRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/cad") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	path := r.URL.Path
	switch path {
	case "/api/cad/render":
		handleCADRender(w, r)
	case "/api/cad/mesh":
		handleCADMesh(w, r)
	case "/api/cad/params":
		handleCADParams(w, r)
	case "/api/cad/versions":
		handleCADVersions(w, r)
	case "/api/cad/versions/restore":
		handleCADVersionRestore(w, r)
	case "/api/cad/test-openscad":
		handleCADTestOpenSCAD(w, r)
	default:
		http.NotFound(w, r)
	}
}

func handleMapsRoute(w http.ResponseWriter, r *http.Request) {
	if isMapsLocationAPIPath(r.URL.Path) {
		handleMapsLocationAPI(w, r)
		return
	}
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/maps") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	http.Error(w, "Maps API requires the Maps pack", http.StatusForbidden)
}

func handleBiologyRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/biology") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	http.Error(w, "Biology API requires the Life sciences pack", http.StatusForbidden)
}

func handleArenaRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/api/arena/match/step" || path == "/api/arena/match/run" {
		handleArenaMatchRoute(w, r)
		return
	}
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/arena") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	http.Error(w, "Arena API requires the Model Arena pack", http.StatusForbidden)
}

func handleAIInterviewRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/ai-interview") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	http.Error(w, "AI Interview API requires the AI Interview Prep pack", http.StatusForbidden)
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
