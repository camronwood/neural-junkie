package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/mcp/resources"
	"github.com/camronwood/neural-junkie/internal/mcp_export"
	ollamaManager "github.com/camronwood/neural-junkie/internal/ollama"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", ":18765", "HTTP service address")
	upgrader = websocket.Upgrader{
		CheckOrigin: checkWebSocketOrigin,
	}
	chatHub             *hub.Hub
	workspaceManager    *hub.WorkspaceManager
	appConfig           *config.Config
	serverStartTime     time.Time
	ollamaMgr           *ollamaManager.Manager
	globalProviderCache *ai.ProviderCache
)

// CORS middleware to allow requests from Tauri dev server
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from Tauri dev server (port 1420) and other origins
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// Note: do not send Access-Control-Allow-Credentials with wildcard Origin (invalid CORS pairing).

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
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

	// Override addr flag from config if not explicitly set via CLI
	if appConfig.Server.Port != 0 {
		defaultAddr := fmt.Sprintf(":%d", appConfig.Server.Port)
		if *addr == ":18765" {
			*addr = defaultAddr
		}
	}

	chatHub = hub.NewHub()
	chatHub.SetCollaborationAssetsRootResolver(func() string {
		return config.CollabAssetsRoot(appConfig)
	})
	globalProviderCache = ai.NewProviderCache()
	agent.SetGlobalCollabRouting(collabRoutingRuntime{})
	if err := initHFManager(); err != nil {
		log.Printf("⚠️  HF download manager init failed: %v", err)
	}
	if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
		ch.SetProviderRegistry(appConfig, globalProviderCache)
	}

	sessionPath := hub.DefaultSessionPath()
	log.Printf("💾 Session will be saved to: %s", sessionPath)
	if fi, err := os.Stat(sessionPath); err == nil {
		log.Printf("💾 Session file on disk: %.1f MiB", float64(fi.Size())/(1024*1024))
		if fi.Size() > 200*1024*1024 {
			log.Printf("⚠️  Session file is very large; consider archiving %s or set NEURAL_JUNKIE_SKIP_SESSION_RESTORE=1 once to start fresh", sessionPath)
		}
	}
	sessionRestored := false
	if os.Getenv("NEURAL_JUNKIE_SKIP_SESSION_RESTORE") == "1" {
		log.Printf("⚠️  NEURAL_JUNKIE_SKIP_SESSION_RESTORE: not loading last-session.json (hub starts with default channels only)")
	} else if err := chatHub.LoadSessionFromFile(sessionPath); err != nil {
		log.Printf("⚠️  Failed to restore previous session: %v", err)
	} else {
		sessionRestored = true
		if n := chatHub.PruneMessagesOlderThan(24 * time.Hour); n > 0 {
			log.Printf("🧹 Pruned %d message(s) older than 24h after session restore", n)
		}
	}

	// Initialize workspace manager
	workspaceManager, err = hub.NewWorkspaceManager()
	if err != nil {
		log.Fatal("Failed to initialize workspace manager:", err)
	}

	// Create some default channels (general is already created by NewHub)
	chatHub.CreateChannel("project-alpha", "Project Alpha development", "alpha")
	chatHub.CreateChannel("project-beta", "Project Beta development", "beta")

	// Initialize and start moderator agent
	initializeModeratorAgent()

	// Initialize and start assistant agent
	initializeAssistantAgent()

	// Initialize CLI agents (e.g. Cursor) if configured
	initializeCLIAgents()

	// Initialize Ollama manager
	ollamaEndpoint := ""
	if p := appConfig.GetProvider(appConfig.AI.DefaultProviderID); p != nil && p.Type == "ollama" {
		ollamaEndpoint = p.Endpoint
	}
	ollamaMgr = ollamaManager.NewManager(ollamaEndpoint)

	if appConfig.Ollama.AutoStart && len(appConfig.Ollama.ModelsToEnsure) > 0 {
		go ensureOllamaModels(context.Background())
	}

	// Initialize specialist agents from config (replaces standalone processes)
	initializeConfiguredAgents()
	if sessionRestored {
		rebindRuntimeAgentsToRestoredDMs()
		// Restored collabs keep tasks/assignees; ListCollaborationSnapshots only
		// redispatches when EnsureExecutionTasks heals data — re-prompt assignees.
		chatHub.RedispatchOpenCollaborationTasksAfterSessionRestore()
		log.Printf("♻️  Previous session restored (if available)")
	}

	// HTTP routes with CORS middleware
	http.HandleFunc("/ws", handleWebSocket) // WebSocket already handles origin
	http.HandleFunc("/api/channels", corsMiddleware(handleChannels))
	http.HandleFunc("/api/channels/create", corsMiddleware(handleCreateChannel))
	http.HandleFunc("/api/channels/create-dm-agent", corsMiddleware(handleCreateDMAgent))
	http.HandleFunc("/api/cli-agent-types", corsMiddleware(handleCLIAgentTypes))
	http.HandleFunc("/api/channels/join", corsMiddleware(handleJoinChannel))
	http.HandleFunc("/api/channels/delete", corsMiddleware(handleDeleteChannel))
	http.HandleFunc("/api/channels/agents", corsMiddleware(handleChannelAgentsManage))
	http.HandleFunc("/api/agent-channels", corsMiddleware(handleAgentChannels))
	http.HandleFunc("/api/agents", corsMiddleware(handleAgentsRoute))
	http.HandleFunc("/api/my-agents", corsMiddleware(handleMyAgents))
	http.HandleFunc("/api/cached-agents", corsMiddleware(handleCachedAgents)) // Keep for backwards compatibility
	http.HandleFunc("/api/removed-agents", corsMiddleware(handleRemovedAgents))
	http.HandleFunc("/api/messages", corsMiddleware(handleMessages))
	http.HandleFunc("/api/collaborations", corsMiddleware(handleCollaborations))
	http.HandleFunc("/api/collaboration-workspace-ack", corsMiddleware(handleCollaborationWorkspaceAck))
	http.HandleFunc("/api/hub-data/read", corsMiddleware(handleHubDataRead))
	http.HandleFunc("/api/send", corsMiddleware(handleSendMessage))
	http.HandleFunc("/api/broadcast", corsMiddleware(handleBroadcastDirect))
	http.HandleFunc("/api/threads/", corsMiddleware(handleThreads)) // Thread endpoints
	http.HandleFunc("/api/import", corsMiddleware(handleImport))
	http.HandleFunc("/api/export", corsMiddleware(handleExport))
	http.HandleFunc("/api/exports", corsMiddleware(handleExports))

	if os.Getenv("ENABLE_MCP_RESOURCES") == "true" {
		go func() {
			rs, err := resources.NewResourceServer()
			if err != nil {
				log.Printf("MCP resource server not started: %v", err)
				return
			}
			if err := rs.Start(); err != nil {
				log.Printf("MCP resource server failed: %v", err)
				return
			}
			log.Printf("MCP resource server listening (ENABLE_MCP_RESOURCES=true)")
		}()
	}

	// File system API endpoints
	http.HandleFunc("/api/workspaces", corsMiddleware(handleWorkspaces))
	http.HandleFunc("/api/files", corsMiddleware(handleFiles))
	http.HandleFunc("/api/file-content", corsMiddleware(handleFileContent))
	http.HandleFunc("/api/file-create", corsMiddleware(handleFileCreate))
	http.HandleFunc("/api/file-rename", corsMiddleware(handleFileRename))
	http.HandleFunc("/api/file-delete", corsMiddleware(handleFileDelete))
	http.HandleFunc("/api/git-status", corsMiddleware(handleGitStatus))
	http.HandleFunc("/api/git-diff", corsMiddleware(handleGitDiff))
	http.HandleFunc("/api/git-commit", corsMiddleware(handleGitCommit))
	http.HandleFunc("/api/git-push", corsMiddleware(handleGitPush))
	http.HandleFunc("/api/git-pull", corsMiddleware(handleGitPull))

	// File change API endpoints
	http.HandleFunc("/api/file-changes", corsMiddleware(handleFileChanges))
	http.HandleFunc("/api/file-changes/propose-from-message", corsMiddleware(handleProposeFileChangeFromMessage))
	http.HandleFunc("/api/file-changes/approve/", corsMiddleware(handleApproveFileChange))
	http.HandleFunc("/api/file-changes/reject/", corsMiddleware(handleRejectFileChange))
	http.HandleFunc("/api/file-changes/", corsMiddleware(handleFileChangeDiff))

	// AI Provider API endpoints
	http.HandleFunc("/api/agents/", corsMiddleware(handleAgentProvider))
	http.HandleFunc("/api/agents/switch-all-providers", corsMiddleware(handleSwitchAllProviders))
	http.HandleFunc("/api/ollama/status", corsMiddleware(handleOllamaStatus))
	http.HandleFunc("/api/ollama/models", corsMiddleware(handleOllamaModels))
	http.HandleFunc("/api/test-ollama-connection", corsMiddleware(handleTestOllamaConnection))
	http.HandleFunc("/api/lmstudio/status", corsMiddleware(handleLMStudioStatus))
	http.HandleFunc("/api/lmstudio/models", corsMiddleware(handleLMStudioModels))
	http.HandleFunc("/api/test-lmstudio-connection", corsMiddleware(handleTestLMStudioConnection))

	// Tool approval endpoints (for Gemini CLI hook integration)
	http.HandleFunc("/api/tool-approvals", corsMiddleware(handleToolApprovals))
	http.HandleFunc("/api/tool-approvals/approve/", corsMiddleware(handleApproveToolCall))
	http.HandleFunc("/api/tool-approvals/reject/", corsMiddleware(handleRejectToolCall))
	http.HandleFunc("/api/tool-approvals/pending", corsMiddleware(handlePendingToolApprovals))

	// Application config and health endpoints
	http.HandleFunc("/api/health", corsMiddleware(handleHealth))
	http.HandleFunc("/api/settings", corsMiddleware(handleSettings))
	http.HandleFunc("/api/agents/configured", corsMiddleware(handleConfiguredAgents))
	http.HandleFunc("/api/agents/restart", corsMiddleware(handleRestartAgents))
	http.HandleFunc("/api/providers", corsMiddleware(handleProviders))
	http.HandleFunc("/api/providers/", corsMiddleware(handleProviderByID))
	http.HandleFunc("/api/ollama/install-status", corsMiddleware(handleOllamaInstallStatus))
	http.HandleFunc("/api/ollama/install", corsMiddleware(handleOllamaInstall))
	http.HandleFunc("/api/ollama/start", corsMiddleware(handleOllamaStart))
	http.HandleFunc("/api/ollama/stop", corsMiddleware(handleOllamaStop))
	http.HandleFunc("/api/ollama/pull", corsMiddleware(handleOllamaPull))
	http.HandleFunc("/api/ollama/catalog", corsMiddleware(handleOllamaCatalog))
	http.HandleFunc("/api/ollama/delete", corsMiddleware(handleOllamaDelete))

	http.HandleFunc("/api/hf/status", corsMiddleware(handleHfStatus))
	http.HandleFunc("/api/hf/catalog", corsMiddleware(handleHfCatalog))
	http.HandleFunc("/api/hf/test-connection", corsMiddleware(handleHfTestConnection))
	http.HandleFunc("/api/hf/download", corsMiddleware(handleHfDownload))
	http.HandleFunc("/api/hf/local", corsMiddleware(handleHfLocal))
	http.HandleFunc("/api/hf/delete", corsMiddleware(handleHfDelete))
	http.HandleFunc("/api/hf/import-ollama", corsMiddleware(handleHfImportOllama))

	// Command palette metadata
	http.HandleFunc("/api/commands", corsMiddleware(handleCommands))
	http.HandleFunc("/api/assistant/state", corsMiddleware(handleAssistantState))
	http.HandleFunc("/api/assistant/task-done", corsMiddleware(handleAssistantTaskDone))
	http.HandleFunc("/api/assistant/reminder-dismiss", corsMiddleware(handleAssistantReminderDismiss))

	if os.Getenv("NEURAL_JUNKIE_DEBUG") == "1" {
		http.HandleFunc("/api/debug/hub-memory", corsMiddleware(handleDebugHubMemory))
		pprofAddr := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_PPROF_ADDR"))
		if pprofAddr == "" {
			pprofAddr = "127.0.0.1:6060"
		}
		go func() {
			log.Printf("🔧 NEURAL_JUNKIE_DEBUG: Go pprof on http://%s/debug/pprof/ (heap, goroutine, etc.)", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				log.Printf("NEURAL_JUNKIE_DEBUG pprof listener: %v", err)
			}
		}()
		log.Printf("🔧 NEURAL_JUNKIE_DEBUG: hub memory JSON at GET /api/debug/hub-memory")
	}

	// Home page handler (must be last to avoid catching API routes)
	http.HandleFunc("/", corsMiddleware(handleHome))

	log.Printf("Chat Hub Server starting on %s", *addr)
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", *addr)
	log.Printf("Web UI: http://localhost%s", *addr)
	log.Printf("CORS enabled for all origins")

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

	// Graceful shutdown: save session on SIGINT/SIGTERM
	server := &http.Server{Addr: *addr}
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		stopSessionSaver()
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

// defaultHumanSender returns the fallback identity for UI/API messages when the
// client omits from.name (avoids generic "Human User" and malformed join lines).
func defaultHumanSender() (id string, name string, agentType protocol.AgentType) {
	if n := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_HUMAN_NAME")); n != "" {
		slug := strings.ToLower(strings.ReplaceAll(n, " ", "-"))
		return "human-" + slug, n, protocol.AgentTypeGeneral
	}
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return "human-" + strings.ToLower(u), u, protocol.AgentTypeGeneral
	}
	return "human-user", "Human User", protocol.AgentTypeGeneral
}

func handleCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defs := chatHub.GetCommandDefinitions()
	if defs == nil {
		defs = []protocol.CommandDefinition{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(defs)
}

// rebindRuntimeAgentsToRestoredDMs restores DM channel subscriptions after a
// session load. Agent IDs change on restart, so restored DMs can reference
// stale IDs and stop receiving messages until re-joined by current IDs.
func rebindRuntimeAgentsToRestoredDMs() {
	channels := chatHub.ListChannels()
	agents := chatHub.ListAgents()
	if len(channels) == 0 || len(agents) == 0 {
		return
	}

	agentsByName := make(map[string]*protocol.AgentInfo, len(agents))
	agentsBySlug := make(map[string]*protocol.AgentInfo, len(agents))
	for _, a := range agents {
		if a == nil {
			continue
		}
		agentsByName[strings.ToLower(strings.TrimSpace(a.Name))] = a
		agentsBySlug[slugifyName(a.Name)] = a
	}

	for _, ch := range channels {
		if ch == nil || ch.Type != protocol.ChannelTypeDM {
			continue
		}
		targetName := extractDMAgentName(ch)
		if targetName == "" {
			continue
		}

		target, ok := agentsByName[strings.ToLower(targetName)]
		if !ok {
			target, ok = agentsBySlug[slugifyName(targetName)]
		}
		if !ok || target == nil {
			log.Printf("ℹ️  DM rebind skipped for %s (agent not found: %s)", ch.Name, targetName)
			continue
		}

		if err := chatHub.JoinChannel(target.ID, ch.Name); err != nil {
			log.Printf("⚠️  DM rebind failed for %s -> %s: %v", target.Name, ch.Name, err)
			continue
		}
		if chHandler, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok && chHandler != nil {
			if err := chHandler.EnsureAgentSubscribedToChannel(context.Background(), target.ID, ch.Name); err != nil {
				log.Printf("⚠️  DM rebind subscribe failed for %s -> %s: %v", target.Name, ch.Name, err)
			}
		}
		log.Printf("✅ DM rebind: %s -> %s", target.Name, ch.Name)
	}
}

func extractDMAgentName(ch *protocol.Channel) string {
	if ch == nil {
		return ""
	}

	desc := strings.TrimSpace(ch.Description)
	lowerDesc := strings.ToLower(desc)
	const prefix = "direct message with "
	if strings.HasPrefix(lowerDesc, prefix) && len(desc) > len(prefix) {
		return strings.TrimSpace(desc[len(prefix):])
	}

	// Fallback: dm-<user>-<agent-slug> where <agent-slug> may contain hyphens
	// (e.g. dm-camron-cursor-buddy → "cursor-buddy", not "buddy").
	parts := strings.SplitN(ch.Name, "-", 3)
	if len(parts) == 3 && strings.EqualFold(parts[0], "dm") {
		return parts[2]
	}
	return ""
}

func slugifyName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func handleAssistantState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	storage, err := agent.NewAssistantStorage()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to initialize assistant storage: %v", err), http.StatusInternalServerError)
		return
	}

	tasks, err := storage.LoadTasks()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load tasks: %v", err), http.StatusInternalServerError)
		return
	}
	reminders, err := storage.LoadReminders()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load reminders: %v", err), http.StatusInternalServerError)
		return
	}

	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	includeDone := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_done")), "true")
	includeInactive := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_inactive")), "true")

	filteredTasks := make([]*agent.Task, 0, len(tasks))
	for _, task := range tasks {
		if channel != "" && task.Channel != channel {
			continue
		}
		if !includeDone && task.Status == "done" {
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}

	filteredReminders := make([]*agent.Reminder, 0, len(reminders))
	for _, reminder := range reminders {
		if channel != "" && reminder.Channel != channel {
			continue
		}
		if !includeInactive && !reminder.Active {
			continue
		}
		filteredReminders = append(filteredReminders, reminder)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"channel":   channel,
		"tasks":     filteredTasks,
		"reminders": filteredReminders,
	})
}

func handleAssistantTaskDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TaskID string `json:"task_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.TaskID = strings.TrimSpace(req.TaskID)
	if req.TaskID == "" {
		http.Error(w, "task_id is required", http.StatusBadRequest)
		return
	}

	storage, err := agent.NewAssistantStorage()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to initialize assistant storage: %v", err), http.StatusInternalServerError)
		return
	}
	tasks, err := storage.LoadTasks()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load tasks: %v", err), http.StatusInternalServerError)
		return
	}

	var matched *agent.Task
	for _, task := range tasks {
		if task.ID == req.TaskID {
			matched = task
			break
		}
	}
	if matched == nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	matched.Status = "done"
	if err := storage.SaveTask(matched); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update task: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"task_id": req.TaskID,
	})
}

func handleAssistantReminderDismiss(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ReminderID string `json:"reminder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.ReminderID = strings.TrimSpace(req.ReminderID)
	if req.ReminderID == "" {
		http.Error(w, "reminder_id is required", http.StatusBadRequest)
		return
	}

	storage, err := agent.NewAssistantStorage()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to initialize assistant storage: %v", err), http.StatusInternalServerError)
		return
	}
	reminders, err := storage.LoadReminders()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load reminders: %v", err), http.StatusInternalServerError)
		return
	}

	var matched *agent.Reminder
	for _, reminder := range reminders {
		if reminder.ID == req.ReminderID {
			matched = reminder
			break
		}
	}
	if matched == nil {
		http.Error(w, "Reminder not found", http.StatusNotFound)
		return
	}

	matched.Active = false
	if err := storage.SaveReminder(matched); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update reminder: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":          true,
		"reminder_id": req.ReminderID,
	})
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// This handler is registered as "/" and receives any path not matched by a more
	// specific route. Never return HTML for API paths — clients may parse bodies as JSON.
	if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Neural Junkie</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
            display: grid;
            grid-template-columns: 250px 1fr 300px;
            height: calc(100vh - 40px);
        }
        .sidebar {
            background: #2c3e50;
            color: white;
            padding: 20px;
            overflow-y: auto;
        }
        .sidebar h2 {
            margin-bottom: 15px;
            font-size: 18px;
            color: #ecf0f1;
        }
        .channel-list, .agent-list {
            margin-bottom: 30px;
        }
        .channel-item, .agent-item {
            padding: 10px;
            margin: 5px 0;
            background: rgba(255,255,255,0.1);
            border-radius: 6px;
            cursor: pointer;
            transition: all 0.2s;
        }
        .channel-item:hover, .agent-item:hover {
            background: rgba(255,255,255,0.2);
            transform: translateX(5px);
        }
        .channel-item.active {
            background: #3498db;
        }
        .agent-item {
            font-size: 13px;
        }
        .agent-type {
            display: inline-block;
            padding: 2px 6px;
            background: rgba(255,255,255,0.2);
            border-radius: 3px;
            font-size: 11px;
            margin-left: 5px;
        }
        .main-chat {
            display: flex;
            flex-direction: column;
            background: #ecf0f1;
        }
        .chat-header {
            background: white;
            padding: 20px;
            border-bottom: 1px solid #ddd;
        }
        .chat-header h1 {
            font-size: 24px;
            color: #2c3e50;
        }
        .messages {
            flex: 1;
            overflow-y: auto;
            padding: 20px;
        }
        .message {
            margin-bottom: 15px;
            padding: 12px 16px;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            animation: slideIn 0.3s ease-out;
        }
        @keyframes slideIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .message-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 8px;
        }
        .message-from {
            font-weight: bold;
            color: #2c3e50;
        }
        .message-type {
            display: inline-block;
            padding: 2px 8px;
            background: #3498db;
            color: white;
            border-radius: 12px;
            font-size: 11px;
            margin-left: 8px;
        }
        .message-time {
            color: #7f8c8d;
            font-size: 12px;
        }
        .message-content {
            color: #34495e;
            line-height: 1.5;
        }
        .message.system {
            background: #f8f9fa;
            font-style: italic;
            color: #7f8c8d;
        }
        .input-area {
            padding: 20px;
            background: white;
            border-top: 1px solid #ddd;
        }
        .input-form {
            display: flex;
            gap: 10px;
        }
        .input-form input {
            flex: 1;
            padding: 12px 16px;
            border: 2px solid #ddd;
            border-radius: 8px;
            font-size: 14px;
        }
        .input-form button {
            padding: 12px 24px;
            background: #3498db;
            color: white;
            border: none;
            border-radius: 8px;
            font-weight: bold;
            cursor: pointer;
            transition: background 0.2s;
        }
        .input-form button:hover {
            background: #2980b9;
        }
        .info-panel {
            background: #f8f9fa;
            padding: 20px;
            overflow-y: auto;
            border-left: 1px solid #ddd;
        }
        .info-panel h3 {
            margin-bottom: 15px;
            color: #2c3e50;
        }
        .stat {
            background: white;
            padding: 12px;
            margin-bottom: 10px;
            border-radius: 6px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
        }
        .stat-label {
            color: #7f8c8d;
            font-size: 12px;
            margin-bottom: 4px;
        }
        .stat-value {
            font-size: 24px;
            font-weight: bold;
            color: #2c3e50;
        }
        .status-indicator {
            display: inline-block;
            width: 8px;
            height: 8px;
            background: #2ecc71;
            border-radius: 50%;
            margin-right: 6px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="sidebar">
            <h2>📚 Channels</h2>
            <div class="channel-list" id="channels">
                <div class="channel-item active" data-channel="general"># general</div>
            </div>
            
            <h2>🤖 Active Agents</h2>
            <div class="agent-list" id="agents">
                <div style="color: #95a5a6; font-size: 13px; padding: 10px;">No agents connected</div>
            </div>
        </div>
        
        <div class="main-chat">
            <div class="chat-header">
                <h1 id="channel-name"># general</h1>
                <p style="color: #7f8c8d; margin-top: 5px;">Multi-agent collaboration chat room</p>
            </div>
            
            <div class="messages" id="messages">
                <div class="message system">
                    <div class="message-content">🎉 Welcome to the Neural Junkie! Agents will appear here as they join.</div>
                </div>
            </div>
            
            <div class="input-area">
                <form class="input-form" id="messageForm">
                    <input type="text" id="messageInput" placeholder="Type a message to the agents..." autocomplete="off">
                    <button type="submit">Send</button>
                </form>
            </div>
        </div>
        
        <div class="info-panel">
            <h3>📊 Statistics</h3>
            <div class="stat">
                <div class="stat-label">Messages</div>
                <div class="stat-value" id="message-count">0</div>
            </div>
            <div class="stat">
                <div class="stat-label">Active Agents</div>
                <div class="stat-value" id="agent-count">0</div>
            </div>
            <div class="stat">
                <div class="stat-label">Channels</div>
                <div class="stat-value" id="channel-count">0</div>
            </div>
            
            <h3 style="margin-top: 30px;">ℹ️ About</h3>
            <p style="color: #7f8c8d; font-size: 13px; line-height: 1.6;">
                This is a multi-agent collaboration system where AI agents with different specialties work together to solve problems.
            </p>
        </div>
    </div>
    
    <script>
        let ws;
        let currentChannel = 'general';
        let messageCount = 0;
        
        function connect() {
            ws = new WebSocket('ws://' + window.location.host + '/ws?channel=' + currentChannel);
            
            ws.onopen = function() {
                console.log('Connected to chat hub');
                loadChannels();
                loadAgents();
            };
            
            ws.onmessage = function(event) {
                const msg = JSON.parse(event.data);
                addMessage(msg);
            };
            
            ws.onclose = function() {
                console.log('Disconnected, reconnecting...');
                setTimeout(connect, 1000);
            };
        }
        
        function addMessage(msg) {
            const messagesDiv = document.getElementById('messages');
            const messageDiv = document.createElement('div');
            messageDiv.className = msg.type === 'agent_join' || msg.type === 'agent_leave' ? 'message system' : 'message';
            
            const time = new Date(msg.timestamp).toLocaleTimeString();
            
            messageDiv.innerHTML = ` + "`" + `
                <div class="message-header">
                    <div>
                        <span class="message-from">${msg.from.name}</span>
                        <span class="message-type">${msg.from.type}</span>
                    </div>
                    <span class="message-time">${time}</span>
                </div>
                <div class="message-content">${msg.content}</div>
            ` + "`" + `;
            
            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
            
            messageCount++;
            document.getElementById('message-count').textContent = messageCount;
        }
        
        function loadChannels() {
            fetch('/api/channels')
                .then(r => r.json())
                .then(channels => {
                    const list = document.getElementById('channels');
                    list.innerHTML = channels.map(ch => 
                        ` + "`" + `<div class="channel-item ${ch.name === currentChannel ? 'active' : ''}" 
                             data-channel="${ch.name}"># ${ch.name}</div>` + "`" + `
                    ).join('');
                    
                    document.getElementById('channel-count').textContent = channels.length;
                    
                    list.querySelectorAll('.channel-item').forEach(item => {
                        item.onclick = () => switchChannel(item.dataset.channel);
                    });
                });
        }
        
        function loadAgents() {
            fetch('/api/agents')
                .then(r => r.json())
                .then(agents => {
                    const list = document.getElementById('agents');
                    if (agents.length === 0) {
                        list.innerHTML = '<div style="color: #95a5a6; font-size: 13px; padding: 10px;">No agents connected</div>';
                    } else {
                        list.innerHTML = agents.map(agent => 
                            ` + "`" + `<div class="agent-item">
                                <span class="status-indicator"></span>
                                ${agent.name}
                                <span class="agent-type">${agent.type}</span>
                            </div>` + "`" + `
                        ).join('');
                    }
                    
                    document.getElementById('agent-count').textContent = agents.length;
                });
        }
        
        function switchChannel(channel) {
            currentChannel = channel;
            document.getElementById('channel-name').textContent = '# ' + channel;
            loadChannels();
            
            // Load channel messages
            fetch('/api/messages?channel=' + channel + '&limit=50')
                .then(r => r.json())
                .then(messages => {
                    const messagesDiv = document.getElementById('messages');
                    messagesDiv.innerHTML = '';
                    messages.forEach(addMessage);
                });
            
            // Reconnect websocket
            if (ws) ws.close();
            connect();
        }
        
        document.getElementById('messageForm').onsubmit = function(e) {
            e.preventDefault();
            const input = document.getElementById('messageInput');
            const message = input.value.trim();
            
            if (message) {
                fetch('/api/send', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        channel: currentChannel,
                        content: message,
                        type: 'question'
                    })
                });
                
                input.value = '';
            }
        };
        
        connect();
        setInterval(loadAgents, 5000); // Refresh agents every 5 seconds
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	threadID := r.URL.Query().Get("thread")

	if channel == "" {
		channel = "general"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// Subscribe to thread or channel
	var msgCh chan *protocol.Message
	if threadID != "" {
		// Thread subscription
		msgCh, err = chatHub.SubscribeToThread(threadID)
		if err != nil {
			log.Println("Thread subscribe error:", err)
			return
		}
		defer chatHub.UnsubscribeFromThread(threadID, msgCh)
	} else {
		// Channel subscription
		msgCh, err = chatHub.Subscribe(channel)
		if err != nil {
			log.Println("Subscribe error:", err)
			return
		}
		defer chatHub.Unsubscribe(channel, msgCh)
	}

	// Send messages to client
	for msg := range msgCh {
		if err := conn.WriteJSON(msg); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func handleChannels(w http.ResponseWriter, r *http.Request) {
	channels := chatHub.ListChannels()

	// Optional type filter
	typeFilter := r.URL.Query().Get("type")
	if typeFilter != "" {
		filtered := make([]*protocol.Channel, 0)
		for _, ch := range channels {
			if string(ch.Type) == typeFilter {
				filtered = append(filtered, ch)
			}
		}
		channels = filtered
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

func handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Project     string   `json:"project"`
		Type        string   `json:"type"`
		Members     []string `json:"members"`
		CreatedBy   string   `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	channelType := protocol.ChannelType(req.Type)
	if channelType == "" {
		channelType = protocol.ChannelTypePublic
	}

	// For DM channels, use the dedicated helper
	if channelType == protocol.ChannelTypeDM {
		if len(req.Members) == 0 {
			http.Error(w, "DM channels require at least one agent member", http.StatusBadRequest)
			return
		}
		ch, err := chatHub.CreateDMChannel(req.CreatedBy, req.Members[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ch)
		return
	}

	channel := chatHub.CreateChannelWithType(req.Name, req.Description, req.Project, channelType, req.CreatedBy)

	// Auto-join requested agent members
	for _, agentID := range req.Members {
		if err := chatHub.AddAgentToChannel(agentID, req.Name); err != nil {
			log.Printf("Warning: failed to add agent %s to channel %s: %v", agentID, req.Name, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channel)
}

// handleCreateDMAgent creates a new expert or CLI agent and a dedicated DM channel (agent is not joined to the caller's current channel).
func handleCreateDMAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CreatedBy   string `json:"created_by"`
		Mode        string `json:"mode"` // "expert" | "cli"
		DisplayName string `json:"display_name"`
		ExpertType  string `json:"expert_type"`
		Persona     string `json:"persona"` // optional extra instructions for custom experts
		ProviderID  string `json:"provider_id"`
		Provider    string `json:"provider"`
		Model       string `json:"model"`
		CLIType     string `json:"cli_type"`
		WorkDir     string `json:"work_dir"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.CreatedBy) == "" {
		http.Error(w, "created_by is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		http.Error(w, "display_name is required", http.StatusBadRequest)
		return
	}

	rawHandler := chatHub.GetCommandHandler()
	ch, ok := rawHandler.(*hub.CommandHandler)
	if !ok || ch == nil {
		http.Error(w, "command handler unavailable", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	mode := strings.ToLower(strings.TrimSpace(req.Mode))

	var dmCh *protocol.Channel
	var err error
	switch mode {
	case "expert":
		if strings.TrimSpace(req.ExpertType) == "" {
			http.Error(w, "expert_type is required for mode expert", http.StatusBadRequest)
			return
		}
		dmCh, err = ch.SpawnExpertAgentForDM(ctx, req.CreatedBy, req.ExpertType, req.DisplayName, req.ProviderID, req.Provider, req.Model, req.Persona)
	case "cli":
		if strings.TrimSpace(req.CLIType) == "" {
			http.Error(w, "cli_type is required for mode cli", http.StatusBadRequest)
			return
		}
		dmCh, err = ch.SpawnCLIAgentForDM(ctx, req.CreatedBy, req.CLIType, req.DisplayName, req.WorkDir)
	default:
		http.Error(w, `mode must be "expert" or "cli"`, http.StatusBadRequest)
		return
	}

	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "failed to start agent") {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dmCh)
}

func handleCLIAgentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	types := agent.ListCLIAgentTypes()
	installed := make(map[string]bool, len(types))
	for _, t := range types {
		cfg, ok := agent.GetCLIAgentConfig(t)
		if !ok {
			continue
		}
		opts := []ai.CLIAgentOption{
			ai.WithBaseArgs(cfg.BaseArgs),
			ai.WithModel(cfg.ModelName),
		}
		p := ai.NewCLIAgentProvider(cfg.Command, ".", cfg.ProviderName, opts...)
		installed[t] = p.IsCLIInstalled()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"types":     types,
		"installed": installed,
	})
}

func handleAgentsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetAgents(w, r)
	case http.MethodPost:
		handleRegisterAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetAgents(w http.ResponseWriter, r *http.Request) {
	agents := chatHub.ListAgents()

	json.NewEncoder(w).Encode(agents)
}

func handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	var agent protocol.AgentInfo
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := chatHub.RegisterAgent(&agent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "agent_id": agent.ID})
}

func handleJoinChannel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID  string `json:"agent_id"`
		Channel  string `json:"channel"`
		Greeting string `json:"greeting,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := chatHub.JoinChannel(req.AgentID, req.Channel, req.Greeting); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := chatHub.DeleteChannel(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleChannelAgentsManage(w http.ResponseWriter, r *http.Request) {
	channelName := r.URL.Query().Get("channel")
	if channelName == "" {
		http.Error(w, "channel query parameter required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req struct {
			AgentIDs []string `json:"agent_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, id := range req.AgentIDs {
			if err := chatHub.AddAgentToChannel(id, channelName); err != nil {
				log.Printf("Warning: failed to add agent %s to %s: %v", id, channelName, err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case http.MethodDelete:
		agentID := r.URL.Query().Get("agent_id")
		if agentID == "" {
			http.Error(w, "agent_id query parameter required", http.StatusBadRequest)
			return
		}
		if err := chatHub.RemoveAgentFromChannel(agentID, channelName); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgentChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "agent_id query parameter required", http.StatusBadRequest)
		return
	}

	channels := chatHub.GetAgentChannels(agentID)
	if channels == nil {
		channels = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"channels": channels})
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "general"
	}

	messages, err := chatHub.GetMessages(channel, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	secret := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_FULL_METADATA_SECRET"))
	allowFull := secret != "" && strings.TrimSpace(r.Header.Get("X-NJ-Full-Metadata")) == secret

	out := make([]*protocol.Message, 0, len(messages))
	for _, m := range messages {
		cp, cerr := protocol.CloneMessage(m)
		if cerr != nil || cp == nil {
			continue
		}
		if !allowFull {
			protocol.RedactImageBinaryMetadata(cp)
		}
		out = append(out, cp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func handleCollaborations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	includeTerminal := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_terminal")), "true")
	collaborations := chatHub.ListCollaborationSnapshots(channel, includeTerminal)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(collaborations)
}

func handleCollaborationWorkspaceAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		CollaborationID string `json:"collaboration_id"`
		SourceRepoPath  string `json:"source_repo_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(req.CollaborationID)
	if id == "" {
		http.Error(w, "collaboration_id required", http.StatusBadRequest)
		return
	}
	if err := chatHub.AcknowledgeCollaborationWorkspace(id, strings.TrimSpace(req.SourceRepoPath)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleBroadcastDirect accepts a message and broadcasts it to channel
// subscribers WITHOUT storing it or running it through the SendMessage
// pipeline (mentions, commands, path detection). This is used by external
// agents to deliver stream_delta / stream_end tokens with minimal overhead.
func handleBroadcastDirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chatHub.BroadcastDirect(msg.Channel, &msg)
	w.WriteHeader(http.StatusNoContent)
}

func writeSendMessageOKResponse(w http.ResponseWriter) {
	resp := map[string]string{"status": "ok"}
	if h := chatHub.GetCommandHandler(); h != nil {
		if ch, ok := h.(*hub.CommandHandler); ok {
			if collabCh, collabID, ok2 := ch.TakeCollaborateRedirect(); ok2 {
				resp["collaboration_channel"] = collabCh
				resp["collaboration_id"] = collabID
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	// Try to decode as full message first (for agents)
	var fullMsg protocol.Message
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Try to parse as full message (agents send this)
	if err := json.Unmarshal(body, &fullMsg); err == nil && fullMsg.ID != "" {
		// This is a full message from an agent, use it directly
		if err := chatHub.SendMessage(&fullMsg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeSendMessageOKResponse(w)
		return
	}

	// Otherwise, parse as simplified request (for UI/human users)
	var req struct {
		Channel       string                 `json:"channel"`
		Content       string                 `json:"content"`
		Type          string                 `json:"type"`
		ThreadID      string                 `json:"thread_id,omitempty"`
		IsThreadReply bool                   `json:"is_thread_reply,omitempty"`
		ReplyTo       string                 `json:"reply_to,omitempty"`
		Metadata      map[string]interface{} `json:"metadata,omitempty"`
		From          *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"from"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msgType := protocol.MessageType(req.Type)
	if msgType == "" {
		msgType = protocol.MessageTypeChat
	}

	senderID, senderName, senderType := defaultHumanSender()

	if req.From != nil {
		if req.From.ID != "" {
			senderID = req.From.ID
		}
		if req.From.Name != "" {
			senderName = req.From.Name
		}
		if req.From.Type != "" {
			senderType = protocol.AgentType(req.From.Type)
		}
	}

	msg := protocol.NewMessage(
		msgType,
		req.Channel,
		protocol.AgentInfo{
			ID:   senderID,
			Name: senderName,
			Type: senderType,
		},
		req.Content,
	)

	// Preserve thread context if provided
	if req.ThreadID != "" {
		msg.ThreadID = req.ThreadID
		msg.IsThreadReply = req.IsThreadReply
	}

	// Preserve reply-to if provided
	if req.ReplyTo != "" {
		msg.ReplyTo = req.ReplyTo
	}

	// Copy metadata from the request (workspace_context, credentials, etc.)
	if req.Metadata != nil {
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Metadata {
			msg.Metadata[k] = v
		}

		// Size guard: truncate workspace_context if it's too large
		if wsCtx, ok := msg.Metadata["workspace_context"]; ok {
			raw, _ := json.Marshal(wsCtx)
			if len(raw) > 500*1024 { // 500KB limit
				log.Printf("Warning: workspace_context too large (%d bytes), removing open_files to reduce size", len(raw))
				if ctxMap, ok := wsCtx.(map[string]interface{}); ok {
					// Remove open_files to drastically reduce size; keep the file tree
					delete(ctxMap, "open_files")
					msg.Metadata["workspace_context"] = ctxMap
				}
			}
		}
		agent.SanitizeInboundMessageMetadata(msg)
	}

	if err := chatHub.SendMessage(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeSendMessageOKResponse(w)
}

func handleThreads(w http.ResponseWriter, r *http.Request) {
	// Parse URL path: /api/threads/{threadID}/messages or /api/threads/{threadID}/reply or /api/threads/{threadID}/metadata
	path := r.URL.Path

	// Remove /api/threads/ prefix
	if len(path) <= len("/api/threads/") {
		http.Error(w, "Invalid thread URL", http.StatusBadRequest)
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(path, "/api/threads/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid thread URL", http.StatusBadRequest)
		return
	}

	threadID := pathParts[0]
	action := pathParts[1]

	switch action {
	case "messages":
		handleThreadMessages(w, r, threadID)
	case "reply":
		handleThreadReply(w, r, threadID)
	case "metadata":
		handleThreadMetadata(w, r, threadID)
	case "parent-author":
		handleThreadParentAuthor(w, r, threadID)
	default:
		http.Error(w, "Unknown thread action", http.StatusBadRequest)
	}
}

func handleThreadMessages(w http.ResponseWriter, r *http.Request, threadID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := chatHub.GetThreadMessages(threadID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}

func handleThreadReply(w http.ResponseWriter, r *http.Request, threadID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Channel  string                 `json:"channel"`
		Content  string                 `json:"content"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
		From     *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"from"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	senderID, senderName, senderType := defaultHumanSender()

	if req.From != nil {
		if req.From.ID != "" {
			senderID = req.From.ID
		}
		if req.From.Name != "" {
			senderName = req.From.Name
		}
		if req.From.Type != "" {
			senderType = protocol.AgentType(req.From.Type)
		}
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		req.Channel,
		protocol.AgentInfo{
			ID:   senderID,
			Name: senderName,
			Type: senderType,
		},
		req.Content,
	)

	// Mark as thread reply
	msg.ThreadID = threadID
	msg.IsThreadReply = true

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			msg.Metadata[k] = v
		}
		agent.SanitizeInboundMessageMetadata(msg)
	}

	if err := chatHub.SendMessage(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleThreadMetadata(w http.ResponseWriter, r *http.Request, threadID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metadata, err := chatHub.GetThreadMetadata(threadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(metadata)
}

func handleThreadParentAuthor(w http.ResponseWriter, r *http.Request, threadID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authorID := chatHub.GetThreadParentAuthor(threadID)

	// Return the author ID as JSON
	response := map[string]string{"author_id": authorID}
	json.NewEncoder(w).Encode(response)
}

// initializeModeratorAgent creates and starts the system moderator agent
func initializeModeratorAgent() {
	log.Println("🤖 Initializing moderator agent...")

	// Create AI provider for moderator
	var aiProvider ai.AIProvider
	ollamaProvider, err := ai.NewOllamaProvider()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to initialize Ollama provider for moderator: %v", err)
		log.Println("⚠️  Using mock AI provider for moderator. Make sure Ollama is running on localhost:11434")
		aiProvider = ai.NewMockProvider()
	} else {
		aiProvider = ollamaProvider
		log.Printf("✅ Ollama provider initialized for moderator (model: %s)", ollamaProvider.GetModel())
	}

	// Create moderator agent
	moderator := agent.NewModeratorAgent("ChatModerator", aiProvider, chatHub)
	moderator.SetCollabClient(chatHub.NewCollaborationClientAdapter())

	// Register moderator with hub
	if err := chatHub.RegisterAgent(&moderator.Info); err != nil {
		log.Printf("❌ Failed to register moderator agent: %v", err)
		return
	}
	if commandHandler := chatHub.GetCommandHandler(); commandHandler != nil {
		if ch, ok := commandHandler.(*hub.CommandHandler); ok {
			ch.RegisterRuntimeAgent(moderator.Agent)
		}
	}

	// Join general channel with greeting
	if err := chatHub.JoinChannel(moderator.Info.ID, "general",
		"👋 ChatModerator online! I'm here to help with chat features and commands. Type @ChatModerator to ask me anything about using this chat system!"); err != nil {
		log.Printf("❌ Failed to join moderator to general channel: %v", err)
		return
	}

	// Start moderator in general channel
	ctx := context.Background()
	go func() {
		if err := moderator.Start(ctx, "general"); err != nil {
			log.Printf("❌ Failed to start moderator agent: %v", err)
			return
		}
	}()

	log.Println("✅ Moderator agent started successfully")
}

// initializeAssistantAgent creates and starts the system assistant agent
func initializeAssistantAgent() {
	log.Println("🤖 Initializing assistant agent...")

	// Create AI provider for assistant - use Ollama since Claude API key is invalid
	var aiProvider ai.AIProvider

	// Use Ollama for assistant since Claude API key is invalid
	ollamaProvider, err := ai.NewOllamaProvider()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to initialize Ollama provider for assistant: %v", err)
		log.Println("⚠️  Using mock AI provider for assistant.")
		aiProvider = ai.NewMockProvider()
	} else {
		aiProvider = ollamaProvider
		log.Printf("✅ Ollama provider initialized for assistant (model: %s, endpoint: %s)", ollamaProvider.GetModel(), ollamaProvider.GetEndpoint())
	}

	// Create assistant agent
	assistant := agent.NewAssistantAgent("Assistant", aiProvider, chatHub)
	assistant.SetCollabClient(chatHub.NewCollaborationClientAdapter())

	// Register assistant with hub
	if err := chatHub.RegisterAgent(&assistant.Info); err != nil {
		log.Printf("❌ Failed to register assistant agent: %v", err)
		return
	}

	// Register assistant with command handler for meeting notes functionality
	if commandHandler := chatHub.GetCommandHandler(); commandHandler != nil {
		if ch, ok := commandHandler.(*hub.CommandHandler); ok {
			ch.SetAssistantAgent(assistant)
			ch.RegisterRuntimeAgent(assistant.Agent)
		}
	}

	// Join general channel with greeting
	if err := chatHub.JoinChannel(assistant.Info.ID, "general",
		"👋 Personal Assistant online! I can help with reminders, tasks, notes, and more. Ask me '/help-assistant' to learn what I can do!"); err != nil {
		log.Printf("❌ Failed to join assistant to general channel: %v", err)
		return
	}

	// Rebind assistant to restored DM channels after restart/session restore.
	for _, ch := range chatHub.ListChannels() {
		if ch == nil || ch.Type != protocol.ChannelTypeDM {
			continue
		}
		nameLower := strings.ToLower(ch.Name)
		descLower := strings.ToLower(ch.Description)
		if strings.Contains(nameLower, "assistant") || strings.Contains(descLower, "assistant") {
			if err := chatHub.JoinChannel(assistant.Info.ID, ch.Name); err != nil {
				log.Printf("⚠️  Failed to rejoin assistant to DM channel %s: %v", ch.Name, err)
			} else {
				log.Printf("✅ Assistant rejoined restored DM channel: %s", ch.Name)
			}
		}
	}

	// Start assistant in general channel
	ctx := context.Background()
	go func() {
		if err := assistant.Start(ctx, "general"); err != nil {
			log.Printf("❌ Failed to start assistant agent: %v", err)
			return
		}
	}()

	log.Println("✅ Assistant agent started successfully")
}

// initializeCLIAgents creates and starts any CLI-backed agents based on environment configuration.
// Each CLI agent is independent; if one binary is missing, the others still start.
func initializeCLIAgents() {
	defaultWorkDir, err := os.Getwd()
	if err != nil {
		log.Printf("⚠️  Failed to get working directory for CLI agents: %v", err)
		return
	}

	for _, cliType := range agent.ListCLIAgentTypes() {
		cfg, _ := agent.GetCLIAgentConfig(cliType)
		initCLIAgentFromConfig(cfg, defaultWorkDir)
	}
}

func initCLIAgentFromConfig(cfg agent.CLIAgentConfig, defaultWorkDir string) {
	log.Printf("🤖 Checking for %s CLI agent (%s)...", cfg.DefaultName, cfg.Command)

	workDir := defaultWorkDir
	if cfg.WorkDirEnv != "" {
		if envDir := os.Getenv(cfg.WorkDirEnv); envDir != "" {
			workDir = envDir
		}
	}

	opts := []ai.CLIAgentOption{
		ai.WithBaseArgs(cfg.BaseArgs),
		ai.WithModel(cfg.ModelName),
	}
	if cfg.Type == "gemini" {
		// Default Gemini to a faster profile unless explicitly overridden.
		model := ""
		if appConfig != nil {
			if p := appConfig.GetProvider(cfg.ProviderName); p != nil {
				model = strings.TrimSpace(p.Model)
			}
		}
		if model == "" {
			model = strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
		}
		if model == "" {
			model = "gemini-2.5-flash"
		}
		_ = os.Setenv("GEMINI_MODEL", model)
		opts = append(opts, ai.WithEnv("GEMINI_MODEL", model))
	}
	provider := ai.NewCLIAgentProvider(cfg.Command, workDir, cfg.ProviderName, opts...)

	// Forward configured env vars
	for _, envKey := range cfg.EnvVars {
		if val := os.Getenv(envKey); val != "" {
			provider.Env[envKey] = val
		}
	}

	if !provider.IsCLIInstalled() {
		log.Printf("ℹ️  %s CLI ('%s') not found on PATH — skipping. %s", cfg.DefaultName, cfg.Command, cfg.InstallHint)
		return
	}

	log.Printf("✅ %s CLI binary found, initializing agent...", cfg.DefaultName)

	// Auto-register as a provider in the config so it appears in Settings > AI Providers
	if existing := appConfig.GetProvider(cfg.ProviderName); existing == nil {
		autoProvider := config.ProviderConfig{
			ID:      cfg.ProviderName,
			Type:    cfg.ProviderName,
			Name:    cfg.DefaultName + " (Auto-detected)",
			WorkDir: workDir,
		}
		if err := appConfig.AddProvider(autoProvider); err == nil {
			log.Printf("📝 Auto-registered provider %q for %s CLI", cfg.ProviderName, cfg.DefaultName)
			_ = appConfig.Save()
		}
	}

	// Gemini-specific: configure tool approval hook
	if cfg.Type == "gemini" {
		configureGeminiApprovalHook()
	}

	cliAgent := agent.NewCLIAgentFromConfig(cfg, cfg.DefaultName, provider, chatHub)
	cliAgent.SetCollabClient(chatHub.NewCollaborationClientAdapter())

	if cfg.ApprovalMode != "" {
		cliAgent.Info.ApprovalMode = cfg.ApprovalMode
	}

	if err := chatHub.RegisterAgent(&cliAgent.Info); err != nil {
		log.Printf("❌ Failed to register %s CLI agent: %v", cfg.DefaultName, err)
		return
	}
	if commandHandler := chatHub.GetCommandHandler(); commandHandler != nil {
		if ch, ok := commandHandler.(*hub.CommandHandler); ok {
			ch.RegisterRuntimeAgent(cliAgent)
		}
	}

	if err := chatHub.JoinChannel(cliAgent.Info.ID, "general", cfg.JoinMessage); err != nil {
		log.Printf("❌ Failed to join %s agent to general channel: %v", cfg.DefaultName, err)
		return
	}

	ctx := context.Background()
	go func() {
		if err := cliAgent.Start(ctx, "general"); err != nil {
			log.Printf("❌ Failed to start %s CLI agent: %v", cfg.DefaultName, err)
			return
		}
	}()

	log.Printf("✅ %s CLI agent started (workDir: %s)", cfg.DefaultName, workDir)
}

// configureGeminiApprovalHook installs the Neural Junkie BeforeTool hook into
// Gemini CLI's settings.json so tool calls are routed through the approval UI.
func configureGeminiApprovalHook() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("⚠️  Cannot determine home directory for Gemini hook config: %v", err)
		return
	}

	settingsDir := filepath.Join(homeDir, ".gemini")
	settingsPath := filepath.Join(settingsDir, "settings.json")

	// Find the hook binary
	hookBin, err := exec.LookPath("tool-approval-hook")
	if err != nil {
		// Try relative to server binary
		exePath, _ := os.Executable()
		hookBin = filepath.Join(filepath.Dir(exePath), "tool-approval-hook")
		if _, err := os.Stat(hookBin); err != nil {
			// Try building it from source
			hookBin = filepath.Join("cmd", "tool-approval-hook", "tool-approval-hook")
			if _, err := os.Stat(hookBin); err != nil {
				log.Printf("ℹ️  tool-approval-hook binary not found — Gemini will use default approval mode. Build it with: go build -o tool-approval-hook ./cmd/tool-approval-hook")
				return
			}
		}
	}

	hookBinAbs, errAbs := filepath.Abs(hookBin)
	if errAbs != nil {
		log.Printf("⚠️  Could not resolve absolute path for tool-approval-hook: %v", errAbs)
		return
	}
	serverURL := fmt.Sprintf("http://localhost%s", *addr)
	hookCommand := fmt.Sprintf("%s --server %s --agent Gemini --agent-id gemini-cli --mode interactive", hookBinAbs, serverURL)

	// Read existing settings or start fresh
	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if jerr := json.Unmarshal(data, &settings); jerr != nil {
			log.Printf("⚠️  Could not parse Gemini settings.json: %v", jerr)
			settings = nil
		}
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	// Check if our hook is already configured
	if hooks, ok := settings["hooks"].(map[string]interface{}); ok {
		if beforeTool, ok := hooks["BeforeTool"].([]interface{}); ok {
			for _, group := range beforeTool {
				if g, ok := group.(map[string]interface{}); ok {
					if hookList, ok := g["hooks"].([]interface{}); ok {
						for _, h := range hookList {
							if hm, ok := h.(map[string]interface{}); ok {
								if name, _ := hm["name"].(string); name == "neural-junkie-approval" {
									// Update the command in case path changed
									hm["command"] = hookCommand
									writeGeminiSettings(settingsPath, settings)
									log.Println("✅ Gemini BeforeTool hook already configured (updated)")
									return
								}
							}
						}
					}
				}
			}
		}
	}

	// Install the hook
	hookEntry := map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":        "command",
				"command":     hookCommand,
				"name":        "neural-junkie-approval",
				"timeout":     180000,
				"description": "Routes tool approval through Neural Junkie chat UI",
			},
		},
	}

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	beforeTool, _ := hooks["BeforeTool"].([]interface{})
	beforeTool = append(beforeTool, hookEntry)
	hooks["BeforeTool"] = beforeTool
	settings["hooks"] = hooks

	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		log.Printf("⚠️  Failed to create Gemini settings dir: %v", err)
		return
	}

	writeGeminiSettings(settingsPath, settings)
	log.Printf("✅ Installed Neural Junkie BeforeTool hook in %s", settingsPath)
}

func writeGeminiSettings(path string, settings map[string]interface{}) {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Printf("⚠️  Failed to marshal Gemini settings: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("⚠️  Failed to write Gemini settings to %s: %v", path, err)
	}
}

// ── Ollama management endpoints ─────────────────────────────────────

func handleOllamaInstallStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	status := ollamaMgr.DetectInstallation()
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	running := ollamaMgr.IsServerRunning(ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"installed": status.Installed,
		"version":   status.Version,
		"path":      status.Path,
		"running":   running,
	})
}

func handleOllamaInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	err := ollamaMgr.InstallOllama(r.Context(), func(msg string) {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		flusher.Flush()
	})
	if err != nil {
		fmt.Fprintf(w, "data: ERROR: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	fmt.Fprintf(w, "data: DONE\n\n")
	flusher.Flush()
}

func handleOllamaStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := ollamaMgr.StartServer(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func handleOllamaStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := ollamaMgr.StopServer(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func handleOllamaPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	err := ollamaMgr.PullModel(r.Context(), req.Model, func(p ollamaManager.PullProgress) {
		data, _ := json.Marshal(p)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		flusher.Flush()
	})
	if err != nil {
		line, mErr := json.Marshal(map[string]string{"status": "error", "error": err.Error()})
		if mErr != nil {
			line = []byte(`{"status":"error","error":"pull failed"}`)
		}
		fmt.Fprintf(w, "data: %s\n\n", string(line))
		flusher.Flush()
		return
	}
	fmt.Fprintf(w, "data: {\"status\":\"success\"}\n\n")
	flusher.Flush()
}

func handleOllamaCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	models, err := ollamaManager.Library()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models); err != nil {
		log.Printf("ollama catalog encode: %v", err)
	}
}

func handleOllamaDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Model) == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	if err := ollamaMgr.DeleteModel(ctx, req.Model); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "model": req.Model})
}

// initializeConfiguredAgents starts specialist agents defined in the config
// file. Each enabled agent runs in-process using the hub's push-based
// message delivery (same as moderator/assistant).
func initializeConfiguredAgents() {
	if appConfig == nil {
		return
	}

	enabled := appConfig.EnabledAgents()
	if len(enabled) == 0 {
		log.Println("ℹ️  No specialist agents configured")
		return
	}

	log.Printf("🤖 Starting %d configured specialist agent(s)...", len(enabled))

	for _, acfg := range enabled {
		pcfg := appConfig.ProviderForAgent(acfg)
		if pcfg == nil {
			log.Printf("⚠️  No provider found for agent %s (provider_id=%q, default=%q) — skipping",
				acfg.Name, acfg.ProviderID, appConfig.AI.DefaultProviderID)
			continue
		}

		aiProvider, err := globalProviderCache.Get(appConfig, pcfg.ID)
		if err != nil {
			log.Printf("⚠️  Failed to build provider for agent %s: %v — skipping", acfg.Name, err)
			continue
		}

		agentType := protocol.AgentType(acfg.Type)
		agentObj, err := agent.AgentFactory(agentType, acfg.Name, aiProvider, chatHub)
		if err != nil {
			log.Printf("❌ Failed to create agent %s (type=%s): %v", acfg.Name, acfg.Type, err)
			continue
		}
		agentObj.SetCollabClient(chatHub.NewCollaborationClientAdapter())

		if err := chatHub.RegisterAgent(&agentObj.Info); err != nil {
			log.Printf("❌ Failed to register agent %s: %v", acfg.Name, err)
			continue
		}
		if commandHandler := chatHub.GetCommandHandler(); commandHandler != nil {
			if ch, ok := commandHandler.(*hub.CommandHandler); ok {
				ch.RegisterRuntimeAgent(agentObj)
			}
		}

		greeting := fmt.Sprintf("👋 %s online! Ready to help with %s questions.", acfg.Name, acfg.Type)
		if err := chatHub.JoinChannel(agentObj.Info.ID, "general", greeting); err != nil {
			log.Printf("❌ Failed to join agent %s to general channel: %v", acfg.Name, err)
			continue
		}

		ctx := context.Background()
		go func(name string) {
			if err := agentObj.Start(ctx, "general"); err != nil {
				log.Printf("❌ Failed to start agent %s: %v", name, err)
			}
		}(acfg.Name)

		log.Printf("✅ Agent %s started (type=%s, provider=%s, model=%s)",
			acfg.Name, acfg.Type, pcfg.Name, aiProvider.GetModel())
	}
}

// ── Health, Settings, Provider, Agent Config endpoints ───────────────

// handleDebugHubMemory returns hub message counts and Go runtime memory stats.
// Enabled only when NEURAL_JUNKIE_DEBUG=1 (localhost tooling; do not expose publicly).
func handleDebugHubMemory(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("NEURAL_JUNKIE_DEBUG") != "1" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(chatHub.HubMemoryReport()); err != nil {
		log.Printf("handleDebugHubMemory: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents := chatHub.ListAgents()
	health := map[string]interface{}{
		"status":      "ok",
		"uptime_secs": int(time.Since(serverStartTime).Seconds()),
		"agent_count": len(agents),
		"version":     "1.0.0",
		"snapshot":    chatHub.GetSessionSaveHealth(),
		"features":    []string{"hub_data_read"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appConfig.Redacted())

	case http.MethodPut:
		var incoming config.Config
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		// Preserve API keys that are redacted in the incoming payload
		for i := range incoming.AI.Providers {
			ip := &incoming.AI.Providers[i]
			if strings.Contains(ip.APIKey, "...") || ip.APIKey == "***" {
				if existing := appConfig.GetProvider(ip.ID); existing != nil {
					ip.APIKey = existing.APIKey
				}
			}
		}
		if strings.Contains(incoming.HF.Token, "...") || incoming.HF.Token == "***" {
			incoming.HF.Token = appConfig.HF.Token
		}

		appConfig.Server = incoming.Server
		appConfig.AI = incoming.AI
		appConfig.Agents = incoming.Agents
		appConfig.Ollama = incoming.Ollama
		appConfig.HF = incoming.HF
		appConfig.Updates = incoming.Updates
		appConfig.Collaboration = incoming.Collaboration

		globalProviderCache.Clear()
		if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
			ch.SetProviderRegistry(appConfig, globalProviderCache)
		}

		if err := appConfig.Save(); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleConfiguredAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appConfig.Agents)
}

func handleRestartAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Re-run the configured agents initializer; existing agents keep running
	// (hub silently skips re-registration of duplicate IDs).
	initializeConfiguredAgents()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "restarted"})
}

func handleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		redacted := appConfig.Redacted()
		json.NewEncoder(w).Encode(redacted.AI.Providers)

	case http.MethodPost:
		var p config.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if p.ID == "" || p.Type == "" {
			http.Error(w, "id and type are required", http.StatusBadRequest)
			return
		}
		if p.Type == "gemini-cli" {
			model := strings.TrimSpace(p.Model)
			if model == "" {
				model = "gemini-2.5-flash"
				p.Model = model
			}
			_ = os.Setenv("GEMINI_MODEL", model)
		}
		if err := appConfig.AddProvider(p); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		appConfig.Save()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleProviderByID(w http.ResponseWriter, r *http.Request) {
	// Path: /api/providers/{id} or /api/providers/{id}/test
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/providers/"), "/")
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if action == "test" && r.Method == http.MethodPost {
		pcfg := appConfig.GetProvider(id)
		if pcfg == nil {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		provider, err := ai.ProviderFromConfig(pcfg)
		if err != nil {
			http.Error(w, "Failed to build provider: "+err.Error(), http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		testResult := map[string]interface{}{"provider_id": id, "success": true}
		_, err = provider.GenerateResponse(ctx, "Say hello in one word.", nil)
		if err != nil {
			testResult["success"] = false
			testResult["error"] = err.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResult)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var p config.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		p.ID = id
		if p.Type == "gemini-cli" || id == "gemini-cli" {
			model := strings.TrimSpace(p.Model)
			if model == "" {
				model = "gemini-2.5-flash"
				p.Model = model
			}
			_ = os.Setenv("GEMINI_MODEL", model)
		}
		if err := appConfig.UpdateProvider(p); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		appConfig.Save()
		globalProviderCache.Evict(id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	case http.MethodDelete:
		if err := appConfig.RemoveProvider(id); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		appConfig.Save()
		globalProviderCache.Evict(id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCachedAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get cached agents from all storage types
	cachedAgents, err := getAllCachedAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"cached_agents": cachedAgents,
	}

	json.NewEncoder(w).Encode(response)
}

func handleMyAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		myAgents, err := getAllCachedAgents()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"my_agents": myAgents})
	case http.MethodDelete:
		var req struct {
			Type string `json:"type"`
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		deleted, err := hub.DeleteCachedAgent(req.Type, req.Name, req.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !deleted {
			http.Error(w, "cached agent not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAllCachedAgents aggregates cached agents from all storage types
func getAllCachedAgents() ([]map[string]interface{}, error) {
	var allAgents []map[string]interface{}

	// Get cached repository agents
	repoStorage, err := repo.NewStorage()
	if err == nil {
		repoAgents, err := repoStorage.GetAllCachedRepos()
		if err == nil {
			allAgents = append(allAgents, repoAgents...)
		}
	}

	// TODO: Add confluence agents when storage is implemented
	// confluenceAgents, err := confluenceStorage.GetAllCachedSpaces()
	// if err == nil {
	//     allAgents = append(allAgents, confluenceAgents...)
	// }

	// Get cached CLI agents
	cliStorage, err := agent.NewCLIAgentStorage()
	if err == nil {
		cliAgents, err := cliStorage.ListWithMetadata()
		if err == nil {
			allAgents = append(allAgents, cliAgents...)
		}
	}

	return allAgents, nil
}

func handleRemovedAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get removed agents from hub
	removedAgents := chatHub.GetRemovedAgents()

	response := map[string]interface{}{
		"removed_agents": removedAgents,
	}

	json.NewEncoder(w).Encode(response)
}

// handleImport handles agent import requests from CLI
func handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var request struct {
		FilePath string `json:"file_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.FilePath == "" {
		http.Error(w, "file_path is required", http.StatusBadRequest)
		return
	}

	// Create a command handler to process the import
	commandHandler, err := hub.NewCommandHandler(chatHub)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create command handler: %v", err), http.StatusInternalServerError)
		return
	}

	// Create a mock message for the import command
	msg := &protocol.Message{
		ID:        "import-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:      protocol.MessageTypeSystemInfo,
		Channel:   "general",
		From:      protocol.AgentInfo{ID: "cli", Name: "CLI", Type: "system"},
		Content:   fmt.Sprintf("/import-agent-mcp %s", request.FilePath),
		Timestamp: time.Now(),
	}

	// Process the import command
	response, err := commandHandler.ProcessCommand(context.Background(), msg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Import failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	responseData := map[string]interface{}{
		"success": true,
		"message": response.Content,
	}

	// Try to extract agent info from the response
	if strings.Contains(response.Content, "Imported") {
		// Parse agent name from response
		parts := strings.Fields(response.Content)
		for i, part := range parts {
			if part == "agent" && i+2 < len(parts) {
				responseData["name"] = strings.Trim(parts[i+1], "'")
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

// handleExports lists exported agents for the CLI.
func handleExports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	storage, err := mcp_export.NewExportStorage()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open export storage: %v", err), http.StatusInternalServerError)
		return
	}

	exports, err := storage.ListExports()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list exports: %v", err), http.StatusInternalServerError)
		return
	}

	type exportJSON struct {
		Name           string  `json:"name"`
		Type           string  `json:"type"`
		ResourceCount  int     `json:"resourceCount"`
		PromptCount    int     `json:"promptCount"`
		FileSize       int64   `json:"fileSize"`
		Description    string  `json:"description,omitempty"`
		ExportPath     string  `json:"exportPath,omitempty"`
	}

	out := make([]exportJSON, 0, len(exports))
	for _, e := range exports {
		out = append(out, exportJSON{
			Name:          e.Name,
			Type:          e.Type,
			ResourceCount: e.ResourceCount,
			PromptCount:   e.PromptCount,
			FileSize:      e.FileSize,
			Description:   e.Description,
			ExportPath:    e.ExportPath,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// handleExport exports an agent via the hub command handler.
func handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		AgentType  string `json:"agent_type"`
		AgentName  string `json:"agent_name"`
		OutputPath string `json:"output_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if request.AgentName == "" {
		http.Error(w, "agent_name is required", http.StatusBadRequest)
		return
	}

	commandHandler, err := hub.NewCommandHandler(chatHub)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create command handler: %v", err), http.StatusInternalServerError)
		return
	}

	msg := &protocol.Message{
		ID:        "export-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:      protocol.MessageTypeSystemInfo,
		Channel:   "general",
		From:      protocol.AgentInfo{ID: "cli", Name: "CLI", Type: "system"},
		Content:   fmt.Sprintf("/export-agent-mcp %s", request.AgentName),
		Timestamp: time.Now(),
	}

	response, err := commandHandler.ProcessCommand(context.Background(), msg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
		return
	}

	responseData := map[string]interface{}{
		"success": strings.Contains(response.Content, "✅"),
		"message": response.Content,
	}

	storage, err := mcp_export.NewExportStorage()
	if err == nil {
		if exports, err := storage.ListExports(); err == nil {
			for _, e := range exports {
				if strings.EqualFold(e.Name, request.AgentName) {
					responseData["resources"] = float64(e.ResourceCount)
					responseData["prompts"] = float64(e.PromptCount)
					responseData["size"] = float64(e.FileSize)
					responseData["name"] = e.Name
					responseData["type"] = e.Type
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

// File system API handlers

func handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if removed, err := workspaceManager.PruneUnavailableTestWorkspaces(); err != nil {
			log.Printf("Warning: failed to prune unavailable test workspaces: %v", err)
		} else if removed > 0 {
			log.Printf("Pruned %d unavailable test workspace(s) from registry", removed)
		}
		workspaces := workspaceManager.ListWorkspaces()
		json.NewEncoder(w).Encode(workspaces)
	case "POST":
		var req struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		workspace, err := workspaceManager.AddWorkspace(req.Name, req.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(workspace)
	case "DELETE":
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id parameter required", http.StatusBadRequest)
			return
		}
		if err := workspaceManager.RemoveWorkspace(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.URL.Query().Get("workspace")
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	workspace, exists := workspaceManager.GetWorkspace(workspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	if info, err := os.Stat(workspace.Path); err != nil || !info.IsDir() {
		http.Error(w, "Workspace path is unavailable", http.StatusNotFound)
		return
	}

	fullPath := filepath.Join(workspace.Path, path)

	absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Calculate the relative path from workspace root
		entryPath := filepath.Join(path, entry.Name())
		// Clean up the path to use forward slashes
		entryPath = strings.TrimPrefix(entryPath, "/")
		if entryPath == "" {
			entryPath = entry.Name()
		}

		files = append(files, map[string]interface{}{
			"name":     entry.Name(),
			"path":     entryPath,
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"mod_time": info.ModTime(),
		})
	}

	json.NewEncoder(w).Encode(files)
}

func isWorkspaceImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".ico", ".svg":
		return true
	default:
		return false
	}
}

func handleFileContent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		workspaceID := r.URL.Query().Get("workspace")
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "path parameter required", http.StatusBadRequest)
			return
		}

		workspace, exists := workspaceManager.GetWorkspace(workspaceID)
		if !exists {
			http.Error(w, "Workspace not found", http.StatusNotFound)
			return
		}

		fullPath := filepath.Join(workspace.Path, path)

		absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
		if err != nil {
			http.Error(w, "Path outside workspace", http.StatusForbidden)
			return
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if r.URL.Query().Get("binary") == "1" || isWorkspaceImageFile(path) {
			mimeType := mime.TypeByExtension(filepath.Ext(path))
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"mime":            mimeType,
				"content_base64": base64.StdEncoding.EncodeToString(content),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": string(content),
		})
	case "POST":
		var req struct {
			WorkspaceID string `json:"workspace_id"`
			Path        string `json:"path"`
			Content     string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		workspace, exists := workspaceManager.GetWorkspace(req.WorkspaceID)
		if !exists {
			http.Error(w, "Workspace not found", http.StatusNotFound)
			return
		}

		fullPath := filepath.Join(workspace.Path, req.Path)

		absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
		if err != nil {
			http.Error(w, "Path outside workspace", http.StatusForbidden)
			return
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleFileCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkspaceID string `json:"workspace_id"`
		Path        string `json:"path"`
		Content     string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	workspace, exists := workspaceManager.GetWorkspace(req.WorkspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	fullPath := filepath.Join(workspace.Path, req.Path)

	absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}

	// Check if file already exists
	if _, err := os.Stat(absPath); err == nil {
		http.Error(w, "File already exists", http.StatusConflict)
		return
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleFileRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkspaceID string `json:"workspace_id"`
		OldPath     string `json:"old_path"`
		NewPath     string `json:"new_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	workspace, exists := workspaceManager.GetWorkspace(req.WorkspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	oldFullPath := filepath.Join(workspace.Path, req.OldPath)
	newFullPath := filepath.Join(workspace.Path, req.NewPath)

	oldAbsPath, err := pathutil.WithinRoot(workspace.Path, oldFullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}
	newAbsPath, err := pathutil.WithinRoot(workspace.Path, newFullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}

	if err := os.Rename(oldAbsPath, newAbsPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleFileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.URL.Query().Get("workspace")
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	workspace, exists := workspaceManager.GetWorkspace(workspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	fullPath := filepath.Join(workspace.Path, path)

	absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}

	if err := os.RemoveAll(absPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Git operations handlers (stubs for now)
func handleGitStatus(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Git operations not yet implemented", http.StatusNotImplemented)
}

func handleGitDiff(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Git operations not yet implemented", http.StatusNotImplemented)
}

func handleGitCommit(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Git operations not yet implemented", http.StatusNotImplemented)
}

func handleGitPush(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Git operations not yet implemented", http.StatusNotImplemented)
}

func handleGitPull(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Git operations not yet implemented", http.StatusNotImplemented)
}

// File change API handlers

func handleFileChanges(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from query parameter (for now, using a simple approach)
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default" // Default user for demo
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	pendingChanges := fileChangeManager.ListPendingFileChanges(userID)

	// Ensure we always return an array, never null
	if pendingChanges == nil {
		pendingChanges = []*filechange.FileChange{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pendingChanges)
}

func handleProposeFileChangeFromMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Channel     string `json:"channel"`
		MessageID   string `json:"message_id"`
		WorkspaceID string `json:"workspace_id"`
		TargetPath  string `json:"target_path"`
		UserID      string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Channel) == "" || strings.TrimSpace(req.MessageID) == "" {
		http.Error(w, "channel and message_id are required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.WorkspaceID) == "" {
		http.Error(w, "workspace_id is required", http.StatusBadRequest)
		return
	}

	workspace, ok := workspaceManager.GetWorkspace(req.WorkspaceID)
	if !ok || workspace == nil || strings.TrimSpace(workspace.Path) == "" {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	msgs, err := chatHub.GetMessages(req.Channel, 1000)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load channel messages: %v", err), http.StatusBadRequest)
		return
	}

	var source *protocol.Message
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i] != nil && msgs[i].ID == req.MessageID {
			source = msgs[i]
			break
		}
	}
	if source == nil {
		http.Error(w, "Source message not found", http.StatusNotFound)
		return
	}
	if source.From.Type == "human" || source.Type != protocol.MessageTypeChat {
		http.Error(w, "Only agent chat messages can be proposed from", http.StatusBadRequest)
		return
	}

	newContent := extractLongestCodeFence(source.Content)
	newContent = stripEditorLineNumberPrefixes(newContent)
	if strings.TrimSpace(newContent) == "" {
		http.Error(w, "No editable content block found in message", http.StatusBadRequest)
		return
	}

	targetPath := strings.TrimSpace(req.TargetPath)
	if targetPath == "" {
		targetPath = inferTargetPathFromWorkspaceContext(source)
	}
	if targetPath == "" {
		http.Error(w, "No target file path available", http.StatusBadRequest)
		return
	}

	candidate := filepath.Clean(targetPath)
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(workspace.Path, candidate)
	}
	absTarget, err := pathutil.WithinRoot(workspace.Path, candidate)
	if err != nil {
		http.Error(w, "Target path is outside workspace", http.StatusBadRequest)
		return
	}
	targetPath = absTarget

	info, statErr := os.Stat(targetPath)
	if statErr != nil || info.IsDir() {
		http.Error(w, "Target file does not exist or is not a file", http.StatusBadRequest)
		return
	}

	oldBytes, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		http.Error(w, fmt.Sprintf("Failed to read target file: %v", readErr), http.StatusBadRequest)
		return
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	fileChangeManager.GetExecutor().SetWorkspaceRoot(workspace.Path)
	change, err := fileChangeManager.ProposeFileChange(
		filechange.FileOperationEdit,
		targetPath,
		"",
		"",
		string(oldBytes),
		newContent,
		source.From,
		req.Channel,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create file change proposal: %v", err), http.StatusBadRequest)
		return
	}

	systemFrom := protocol.AgentInfo{
		ID:     "system",
		Name:   "System",
		Type:   protocol.AgentTypeGeneral,
		Status: "active",
	}
	infoMsg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		req.Channel,
		systemFrom,
		fmt.Sprintf("Created file change proposal `%s` for `%s` from message `%s`.", change.ID, change.FilePath, source.ID),
	)
	if sendErr := chatHub.SendMessage(infoMsg); sendErr != nil {
		log.Printf("Failed to send propose-from-message system message: %v", sendErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(change)
}

func extractLongestCodeFence(content string) string {
	re := regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\\s*\\n(.*?)\\n```")
	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return ""
	}
	longest := ""
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		if len(m[1]) > len(longest) {
			longest = m[1]
		}
	}
	return longest
}

func stripEditorLineNumberPrefixes(content string) string {
	if content == "" {
		return content
	}
	re := regexp.MustCompile(`(?m)^\s*\d+\s*\|\s?`)
	return re.ReplaceAllString(content, "")
}

func inferTargetPathFromWorkspaceContext(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	wsCtxRaw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return ""
	}
	wsCtx, ok := wsCtxRaw.(map[string]interface{})
	if !ok {
		return ""
	}
	openFilesRaw, ok := wsCtx["open_files"]
	if !ok {
		return ""
	}
	openFiles, ok := openFilesRaw.([]interface{})
	if !ok {
		return ""
	}
	for _, f := range openFiles {
		m, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		isActive, _ := m["is_active"].(bool)
		path, _ := m["path"].(string)
		if isActive && strings.TrimSpace(path) != "" {
			return path
		}
	}
	if len(openFiles) > 0 {
		if first, ok := openFiles[0].(map[string]interface{}); ok {
			if path, ok := first["path"].(string); ok {
				return path
			}
		}
	}
	return ""
}

func handleApproveFileChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract change ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/file-changes/approve/")
	if path == "" {
		http.Error(w, "Change ID required", http.StatusBadRequest)
		return
	}

	// Get user ID from request body or query
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default" // Default user for demo
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	change, err := fileChangeManager.ApproveFileChange(path, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Emit a user-visible confirmation in the change channel.
	channel := change.Channel
	if strings.TrimSpace(channel) == "" {
		channel = "general"
	}
	systemFrom := protocol.AgentInfo{
		ID:     "system",
		Name:   "System",
		Type:   protocol.AgentTypeGeneral,
		Status: "active",
	}
	confirm := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		systemFrom,
		fmt.Sprintf("Applied change `%s` to `%s`.", change.ID, change.FilePath),
	)
	if sendErr := chatHub.SendMessage(confirm); sendErr != nil {
		log.Printf("Failed to send file-change confirmation message: %v", sendErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(change)
}

func handleRejectFileChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract change ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/file-changes/reject/")
	if path == "" {
		http.Error(w, "Change ID required", http.StatusBadRequest)
		return
	}

	// Get user ID and reason from request body
	var req struct {
		UserID string `json:"user_id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.UserID = "default"
		req.Reason = "No reason provided"
	}

	if req.UserID == "" {
		req.UserID = "default"
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	change, err := fileChangeManager.RejectFileChange(path, req.UserID, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(change)
}

func handleFileChangeDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract change ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/file-changes/")
	if path == "" {
		http.Error(w, "Change ID required", http.StatusBadRequest)
		return
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	change, err := fileChangeManager.GetFileChange(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Generate diff for edit operations
	var diff string
	if change.Operation == "edit" {
		// Simple diff implementation - in production, use a proper diff library
		diff = "--- Old content\n+++ New content\n"
		oldLines := strings.Split(change.OldContent, "\n")
		newLines := strings.Split(change.NewContent, "\n")

		maxLines := len(oldLines)
		if len(newLines) > maxLines {
			maxLines = len(newLines)
		}

		for i := 0; i < maxLines; i++ {
			oldLine := ""
			newLine := ""

			if i < len(oldLines) {
				oldLine = oldLines[i]
			}
			if i < len(newLines) {
				newLine = newLines[i]
			}

			if oldLine != newLine {
				diff += fmt.Sprintf("@@ -%d +%d @@\n", i+1, i+1)
				if oldLine != "" {
					diff += fmt.Sprintf("-%s\n", oldLine)
				}
				if newLine != "" {
					diff += fmt.Sprintf("+%s\n", newLine)
				}
			}
		}
	}

	response := map[string]interface{}{
		"change": change,
		"diff":   diff,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleToolApprovals creates a new tool approval request (called by the hook binary).
// The request blocks until the user approves/rejects or a timeout occurs.
func handleToolApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID   string                 `json:"agent_id"`
		AgentName string                 `json:"agent_name"`
		SessionID string                 `json:"session_id"`
		ToolName  string                 `json:"tool_name"`
		ToolInput map[string]interface{} `json:"tool_input"`
		Channel   string                 `json:"channel"`
		Mode      string                 `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ToolName == "" {
		http.Error(w, "tool_name is required", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		req.Channel = "general"
	}

	// If mode is yolo, auto-approve without user interaction
	if req.Mode == "yolo" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "approved",
			"decision": "allow",
		})
		return
	}

	// If mode is auto_edit, auto-approve read/edit tools but prompt for shell commands
	if req.Mode == "auto_edit" {
		autoApproveTools := map[string]bool{
			"read_file": true, "write_file": true, "edit_file": true,
			"list_directory": true, "search_files": true, "read_many_files": true,
		}
		if autoApproveTools[req.ToolName] {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":   "approved",
				"decision": "allow",
			})
			return
		}
	}

	tam := chatHub.GetToolApprovalManager()
	approval := tam.CreateApproval(req.AgentID, req.AgentName, req.SessionID, req.ToolName, req.Channel, req.ToolInput)

	log.Printf("[ToolApproval] Created approval %s for %s.%s", approval.ID, req.AgentName, req.ToolName)

	// Block until user decides (up to 3 minutes)
	status, reason := tam.WaitForDecision(approval.ID, hub.ToolApprovalTTL)

	decision := "deny"
	if status == hub.ToolApprovalApproved {
		decision = "allow"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   string(status),
		"decision": decision,
		"reason":   reason,
	})
}

// handleApproveToolCall approves a pending tool call
func handleApproveToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	approvalID := strings.TrimPrefix(r.URL.Path, "/api/tool-approvals/approve/")
	if approvalID == "" {
		http.Error(w, "Approval ID required", http.StatusBadRequest)
		return
	}

	tam := chatHub.GetToolApprovalManager()
	if err := tam.Approve(approvalID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

// handleRejectToolCall rejects a pending tool call
func handleRejectToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	approvalID := strings.TrimPrefix(r.URL.Path, "/api/tool-approvals/reject/")
	if approvalID == "" {
		http.Error(w, "Approval ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Reason == "" {
		req.Reason = "User rejected"
	}

	tam := chatHub.GetToolApprovalManager()
	if err := tam.Reject(approvalID, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

// handlePendingToolApprovals lists all currently pending tool approvals
func handlePendingToolApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tam := chatHub.GetToolApprovalManager()
	pending := tam.ListPending()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pending)
}

// handleSetApprovalMode updates the approval mode for a CLI agent
func handleSetApprovalMode(w http.ResponseWriter, r *http.Request, agentID string) {
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	validModes := map[string]bool{"interactive": true, "auto_edit": true, "yolo": true}
	if !validModes[req.Mode] {
		http.Error(w, "Invalid mode. Use 'interactive', 'auto_edit', or 'yolo'", http.StatusBadRequest)
		return
	}

	agents := chatHub.ListAgents()
	var targetAgent *protocol.AgentInfo
	for _, agent := range agents {
		if agent.ID == agentID {
			targetAgent = agent
			break
		}
	}

	if targetAgent == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	if targetAgent.Type != protocol.AgentTypeCLI {
		http.Error(w, "Approval mode only applies to CLI agents", http.StatusBadRequest)
		return
	}

	targetAgent.ApprovalMode = req.Mode
	log.Printf("[ApprovalMode] Set %s (%s) to %s", targetAgent.Name, agentID, req.Mode)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "mode": req.Mode})
}

// handleSetAgentCustomRules updates persisted markdown instructions for any registered agent.
func handleSetAgentCustomRules(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := chatHub.SetAgentCustomRulesMarkdown(agentID, req.Markdown); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAgentProvider handles switching individual agent providers and approval mode
func handleAgentProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID and action from URL path: /api/agents/{id}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
		return
	}

	action := parts[1]

	// Route to approval-mode handler
	if action == "approval-mode" {
		handleSetApprovalMode(w, r, parts[0])
		return
	}

	if action == "rules" {
		handleSetAgentCustomRules(w, r, parts[0])
		return
	}

	if action != "provider" {
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
		return
	}

	agentID := parts[0]
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	var request struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate provider
	if !isAllowedRuntimeProvider(request.Provider) {
		http.Error(w, "Invalid provider. Use 'claude', 'ollama', 'lmstudio', or 'huggingface'", http.StatusBadRequest)
		return
	}

	commandHandler := chatHub.GetCommandHandler()
	if commandHandler == nil {
		http.Error(w, "Command handler not initialized", http.StatusServiceUnavailable)
		return
	}
	ch, ok := commandHandler.(*hub.CommandHandler)
	if !ok {
		http.Error(w, "Unsupported command handler type", http.StatusInternalServerError)
		return
	}

	targetAgent, err := ch.SwitchAgentProvider(agentID, request.Provider, request.Model, "general", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent %s switched to %s (%s)", targetAgent.Name, targetAgent.AIProvider, targetAgent.AIModel),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSwitchAllProviders handles switching all agents to the same provider
func handleSwitchAllProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate provider
	if !isAllowedRuntimeProvider(request.Provider) {
		http.Error(w, "Invalid provider. Use 'claude', 'ollama', 'lmstudio', or 'huggingface'", http.StatusBadRequest)
		return
	}

	commandHandler := chatHub.GetCommandHandler()
	if commandHandler == nil {
		http.Error(w, "Command handler not initialized", http.StatusServiceUnavailable)
		return
	}
	ch, ok := commandHandler.(*hub.CommandHandler)
	if !ok {
		http.Error(w, "Unsupported command handler type", http.StatusInternalServerError)
		return
	}

	switchedCount, err := ch.SwitchAllProviders(request.Provider, request.Model, "general", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success":        true,
		"message":        fmt.Sprintf("Switched %d agents to %s (%s)", switchedCount, request.Provider, request.Model),
		"switched_count": switchedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleOllamaStatus checks if Ollama is running
func handleOllamaStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create Ollama provider to test connection
	ollamaProvider := ai.NewOllamaProviderWithConfig("http://localhost:11434", "llama3.1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ollamaProvider.TestConnection(ctx)

	response := map[string]interface{}{
		"running":  err == nil,
		"endpoint": "http://localhost:11434",
	}

	if err != nil {
		response["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleOllamaModels returns available Ollama models
func handleOllamaModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	endpoint := strings.TrimSpace(r.URL.Query().Get("endpoint"))
	if endpoint == "" && appConfig != nil {
		endpoint = appConfig.FirstOllamaEndpoint()
	}
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	mgr := ollamaManager.NewManager(endpoint)
	models, err := mgr.ListModels(ctx)

	response := map[string]interface{}{
		"models":   models,
		"endpoint": endpoint,
	}

	if err != nil {
		response["error"] = err.Error()
		response["models"] = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTestOllamaConnection tests Ollama connection
func handleTestOllamaConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Endpoint string `json:"endpoint"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if request.Endpoint == "" {
		request.Endpoint = "http://localhost:11434"
	}
	if request.Model == "" {
		request.Model = "llama3.1"
	}

	// Create Ollama provider and test connection
	ollamaProvider := ai.NewOllamaProviderWithConfig(request.Endpoint, request.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := ollamaProvider.TestConnection(ctx)

	response := map[string]interface{}{
		"success":  err == nil,
		"endpoint": request.Endpoint,
		"model":    request.Model,
	}

	if err != nil {
		response["error"] = err.Error()
		response["message"] = "Failed to connect to Ollama"
	} else {
		response["message"] = "Successfully connected to Ollama"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLMStudioStatus checks if LM Studio is running
func handleLMStudioStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create LM Studio provider to test connection
	lmStudioProvider := ai.NewLMStudioProviderWithConfig("http://localhost:1234/v1", "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := lmStudioProvider.TestConnection(ctx)

	response := map[string]interface{}{
		"running":  err == nil,
		"endpoint": "http://localhost:1234/v1",
	}

	if err != nil {
		response["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLMStudioModels returns available LM Studio models
func handleLMStudioModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get endpoint from query parameter or use default
	endpoint := r.URL.Query().Get("endpoint")
	if endpoint == "" {
		endpoint = "http://localhost:1234/v1"
	}

	// Create LM Studio provider to get models
	lmStudioProvider := ai.NewLMStudioProviderWithConfig(endpoint, "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := lmStudioProvider.GetAvailableModels(ctx)

	response := map[string]interface{}{
		"models":   models,
		"endpoint": endpoint,
	}

	if err != nil {
		response["error"] = err.Error()
		response["models"] = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTestLMStudioConnection tests LM Studio connection
func handleTestLMStudioConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Endpoint string `json:"endpoint"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if request.Endpoint == "" {
		request.Endpoint = "http://localhost:1234/v1"
	}

	// Create LM Studio provider and test connection
	lmStudioProvider := ai.NewLMStudioProviderWithConfig(request.Endpoint, request.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := lmStudioProvider.TestConnection(ctx)

	response := map[string]interface{}{
		"success":  err == nil,
		"endpoint": request.Endpoint,
		"model":    request.Model,
	}

	if err != nil {
		response["error"] = err.Error()
		response["message"] = "Failed to connect to LM Studio"
	} else {
		response["message"] = "Successfully connected to LM Studio"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHubDataRead returns bounded text from ~/.neural-junkie after the user grants access in the desktop app.
func handleHubDataRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Targets []hub.HubDataReadTarget `json:"targets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.Targets) == 0 {
		http.Error(w, "at least one target is required", http.StatusBadRequest)
		return
	}
	result, err := hub.ReadHubDataForAgent(req.Targets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("hub-data/read encode error: %v", err)
	}
}
