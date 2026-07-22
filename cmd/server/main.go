package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	slackint "github.com/camronwood/neural-junkie/internal/integrations/slack"
	lspserver "github.com/camronwood/neural-junkie/internal/lsp/server"
	ollamaManager "github.com/camronwood/neural-junkie/internal/ollama"
	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", ":18765", "HTTP service address")
	upgrader = websocket.Upgrader{
		CheckOrigin: checkWebSocketOrigin,
	}
	chatHub                  *hub.Hub
	workspaceManager         *hub.WorkspaceManager
	projectSetManager        *hub.ProjectSetManager
	workspaceBackendResolver *workspacebackend.Resolver
	lspManager               *lspserver.Manager
	appConfig                *config.Config
	serverStartTime          time.Time
	ollamaMgr                *ollamaManager.Manager
	globalProviderCache      *ai.ProviderCache
	apiRateLimiter           = hub.NewRateLimiter()
	slackBridgeCtx           context.Context
	stopSlackBridgeCtx       context.CancelFunc
)

// CORS middleware to allow requests from Tauri dev server
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w, r)

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		hub.RateLimitMiddleware(apiRateLimiter, next)(w, r)
	}
}

// checkWebSocketOrigin restricts browser WebSocket hijacking (CSWSH). Non-browser clients often omit Origin.
// Override with NEURAL_JUNKIE_WS_ORIGINS (comma-separated full Origin URLs) for extra dev hosts.
func checkWebSocketOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	u, err := url.Parse(o)
	if err != nil || u.Hostname() == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	if strings.HasSuffix(host, ".localhost") {
		return true
	}
	if rh := r.Host; rh != "" {
		reqHost, _, splitErr := net.SplitHostPort(rh)
		if splitErr != nil {
			reqHost = rh
		}
		if strings.EqualFold(host, strings.ToLower(reqHost)) {
			return true
		}
	}
	for _, extra := range strings.Split(os.Getenv("NEURAL_JUNKIE_WS_ORIGINS"), ",") {
		if strings.TrimSpace(extra) == o {
			return true
		}
	}
	log.Printf("websocket: rejected Origin %q for Host %q (set NEURAL_JUNKIE_WS_ORIGINS to allow)", o, r.Host)
	return false
}

func main() {
	flag.Parse()
	serverStartTime = time.Now()

	// Load application config (falls back to defaults if no config.json exists)
	var err error
	appConfig, err = config.Load()
	if err != nil {
		log.Printf("⚠️  Failed to load config, using defaults: %v", err)
		appConfig = config.DefaultConfig()
	}
	syncMCPFromConfig()
	initPersonalLearningStore()
	initCodeIndexEmbed()

	// Resolve bind address (loopback by default; see docs/SECURITY.md)
	*addr = resolveListenAddr(*addr, appConfig)
	config.SetAppConfig(appConfig)
	sec := appConfig.ResolvedSecurity()
	// Rate limiter is constructed at package init (before config load) with defaults.
	// Reconfigure immediately so disk settings (e.g. rate_limit_enabled=false) take effect.
	if apiRateLimiter != nil {
		apiRateLimiter.Reconfigure(sec.RateLimitEnabledOrDefault(), sec.RateReadPerMinute, sec.RateMutatePerMinute)
	}
	if sec.ListenAll && !hub.HubTokenConfigured() {
		if appConfig.ResolvedDebug().Enabled || hub.RelaxedLocal() {
			log.Printf("⚠️  NEURAL_JUNKIE_LISTEN_ALL=1 without NEURAL_JUNKIE_HUB_TOKEN — hub is exposed on the LAN without network auth")
		} else {
			log.Fatal("NEURAL_JUNKIE_LISTEN_ALL=1 requires NEURAL_JUNKIE_HUB_TOKEN (set a shared secret before binding 0.0.0.0)")
		}
	}
	hubListenHost := hubPublicHost(*addr)
	slackint.SetHubPublicBaseURL("http://" + hubListenHost)
	if slackint.SeedBundledAppToken(&appConfig.Slack) {
		if err := appConfig.Save(); err != nil {
			log.Printf("[slack] save bundled app token: %v", err)
		} else {
			log.Println("[slack] seeded Socket Mode app token from bundled credentials")
		}
	}
	chatHub = hub.NewHub()
	chatHub.SetSemanticTurnRouter(semanticTurnRouter(appConfig))
	defer func() {
		if err := chatHub.CloseOrchestrationStore(); err != nil {
			log.Printf("⚠️  Failed to close orchestration store: %v", err)
		}
	}()
	applyCollabActionConfig()
	initMessageStore()
	initAuthStore()
	hub.EnsureBootstrapToken()
	initConversationMemory()
	initInferenceStats()
	if appConfig != nil {
		for _, ch := range appConfig.Server.DurableChannels {
			if strings.TrimSpace(ch) != "" {
				chatHub.MarkChannelDurable(strings.TrimSpace(ch))
			}
		}
	}
	chatHub.SetCollaborationAssetsRootResolver(func() string {
		return config.CollabAssetsRoot(appConfig)
	})
	globalProviderCache = ai.NewProviderCache()
	initChannelSummaryGenerator(appConfig, chatHub)
	agent.SetGlobalCollabRouting(collabRoutingRuntime{})
	agent.SetGlobalImplementationRouting(implementationRoutingRuntime{})
	agent.SetGlobalChatRouting(chatRoutingRuntime{})
	if _, err := capabilities.Load(); err != nil {
		log.Printf("⚠️  Capability profiles unavailable: %v", err)
	}
	if err := initHFManager(); err != nil {
		log.Printf("⚠️  HF download manager init failed: %v", err)
	}
	if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
		ch.SetProviderRegistry(appConfig, globalProviderCache)
	}

	sessionPath := hub.DefaultSessionPath()
	log.Printf("💾 Session will be saved to: %s", sessionPath)
	sessCfg := appConfig.ResolvedSession()
	restoreLastSession := sessCfg.RestoreOnStartup
	if !restoreLastSession {
		if fi, err := os.Stat(sessionPath); err == nil {
			log.Printf("🧹 Resetting previous session on startup: removing %s (%.1f MiB)", sessionPath, float64(fi.Size())/(1024*1024))
			if err := os.Remove(sessionPath); err != nil {
				log.Printf("⚠️  Failed to remove previous session file %s: %v", sessionPath, err)
			}
		} else if !os.IsNotExist(err) {
			log.Printf("⚠️  Failed to stat previous session file %s: %v", sessionPath, err)
		}
		log.Printf("💾 Previous session restore is disabled by default; starting with a fresh session (enable in Settings → Server & network)")
	} else if fi, err := os.Stat(sessionPath); err == nil {
		log.Printf("💾 Session file on disk: %.1f MiB", float64(fi.Size())/(1024*1024))
		if fi.Size() > 200*1024*1024 {
			log.Printf("⚠️  Session file is very large; consider archiving %s or unset NEURAL_JUNKIE_RESTORE_LAST_SESSION to start fresh", sessionPath)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("⚠️  Failed to stat session file %s: %v", sessionPath, err)
	}
	sessionRestored := false
	if restoreLastSession {
		if sessCfg.SkipRestoreOnce {
			log.Printf("⚠️  skip_restore_once: not loading last-session.json (hub starts with default channels only)")
		} else if err := chatHub.LoadSessionFromFile(sessionPath); err != nil {
			log.Printf("⚠️  Failed to restore previous session: %v", err)
		} else {
			sessionRestored = true
			if n := chatHub.PruneMessagesOlderThan(24 * time.Hour); n > 0 {
				log.Printf("🧹 Pruned %d message(s) older than 24h after session restore", n)
			}
		}
	}
	// Initialize workspace manager
	workspaceManager, err = hub.NewWorkspaceManager()
	if err != nil {
		log.Fatal("Failed to initialize workspace manager:", err)
	}
	projectSetManager, err = hub.NewProjectSetManager()
	if err != nil {
		log.Printf("Warning: project set manager unavailable: %v", err)
	}
	workspaceBackendResolver = workspacebackend.NewResolver(hubWorkspaceSource{m: workspaceManager})
	lspManager = lspserver.NewManager()
	registerRemoteWorkspacesOnStartup()
	chatHub.SetFileChangeBackendFn(backendForWorkspaceRoot)
	chatHub.SetCollabWorktreeBackendResolver(resolveWorktreeBackend)
	chatHub.RestoreDurableOrchestrationInputs()

	// Drop legacy demo channels (project-alpha / project-beta) from restored sessions.
	if n := chatHub.RemoveLegacySeedChannels(); n > 0 {
		log.Printf("🧹 Removed %d legacy seed channel(s) (project-alpha, project-beta)", n)
	}

	// Initialize and start assistant agent
	initializeAssistantAgent()

	// Initialize CLI agents (e.g. Cursor) if configured
	syncCLIProviderModelsFromConfig()
	initializeCLIAgents()

	// Initialize Ollama manager
	ollamaEndpoint := ""
	if p := appConfig.GetProvider(appConfig.AI.DefaultProviderID); p != nil && p.Type == "ollama" {
		ollamaEndpoint = p.Endpoint
	}
	ollamaMgr = ollamaManager.NewManager(ollamaEndpoint)

	go func() {
		ctx := context.Background()
		if appConfig.Ollama.AutoStart && len(appConfig.Ollama.ModelsToEnsure) > 0 {
			ensureOllamaModels(ctx)
		}
		ensurePackLoRAs(ctx, "")
	}()

	// Initialize specialist agents from config (replaces standalone processes)
	initializeConfiguredAgents()
	reconcileHiddenRepoAgentsOnStartup()

	restorePersistedDMAgents()

	slackBridgeCtx, stopSlackBridgeCtx = context.WithCancel(context.Background())
	defer stopSlackBridgeCtx()
	streamBridgeCtx = slackBridgeCtx

	if sessionRestored {
		rebindRuntimeAgentsToRestoredDMs()
		// Restored collabs keep tasks/assignees; ListCollaborationSnapshots only
		// redispatches when EnsureExecutionTasks heals data — re-prompt assignees.
		chatHub.RedispatchOpenCollaborationTasksAfterSessionRestore()
		log.Printf("♻️  Previous session restored (if available)")
	}

	registerRoutes()
	initMusicSidecarGenerator()
	initBrowserSidecarClient()
	initIncidentSidecarClient()
	initAWSSidecarClient()
	initCADSidecarClient()
	initBiologySidecarClient()
	initArenaSidecarClient()
	syncPackSidecars()

	log.Printf("Chat Hub Server starting on %s", *addr)
	log.Printf("WebSocket endpoint: ws://%s/ws", hubPublicHost(*addr))
	log.Printf("Web UI: http://%s", hubPublicHost(*addr))
	if os.Getenv("NEURAL_JUNKIE_CORS_ANY") == "1" {
		log.Printf("CORS: wildcard mode (NEURAL_JUNKIE_CORS_ANY=1)")
	} else {
		log.Printf("CORS: restricted to local dev origins (set NEURAL_JUNKIE_CORS_ANY=1 to allow all)")
	}

	// Periodic session save (every 2 minutes), cancellable for clean shutdown.
	sessionSaverCtx, stopSessionSaver := context.WithCancel(context.Background())
	var sessionSaverWG sync.WaitGroup
	sessionSaverWG.Add(1)
	go func() {
		defer sessionSaverWG.Done()
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-sessionSaverCtx.Done():
				return
			case <-ticker.C:
				if err := chatHub.SaveSessionToFile(sessionPath); err != nil {
					log.Printf("⚠️  Periodic session save failed: %v", err)
				}
			}
		}
	}()

	// Drop channel/thread messages older than 24h periodically (WebSocket resync to clients/agents).
	sessionSaverWG.Add(1)
	go func() {
		defer sessionSaverWG.Done()
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-sessionSaverCtx.Done():
				return
			case <-ticker.C:
				if n := chatHub.PruneMessagesOlderThan(24 * time.Hour); n > 0 {
					log.Printf("🧹 Periodic prune: removed %d message(s) older than 24h", n)
				}
			}
		}
	}()

	// Collaboration idle watchdog (post-approve stall healing).
	sessionSaverWG.Add(1)
	go func() {
		defer sessionSaverWG.Done()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-sessionSaverCtx.Done():
				return
			case <-ticker.C:
				chatHub.TickCollabScheduler(time.Now())
			}
		}
	}()

	// Graceful shutdown: save session on SIGINT/SIGTERM
	server := &http.Server{Addr: *addr}
	go func() {
		// Outbound bridge uses WebSocket to localhost; start after the hub is listening.
		time.Sleep(300 * time.Millisecond)
		startSlackBridge(slackBridgeCtx)
		startStreamManager(streamBridgeCtx)
	}()
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		stopSessionSaver()
		stopSlackBridgeCtx()
		stopSlackBridge()
		stopStreamManager()
		sessionSaverWG.Wait()
		log.Println("🛑 Shutdown signal received, saving session...")
		if err := chatHub.SaveSessionToFile(sessionPath); err != nil {
			log.Printf("⚠️  Failed to save session on shutdown: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("ListenAndServe: ", err)
	}

	log.Println("👋 Server stopped.")
}
