package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/hub"
	ollamaManager "github.com/camronwood/neural-junkie/internal/ollama"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", ":8080", "HTTP service address")
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for demo
		},
	}
	chatHub          *hub.Hub
	workspaceManager *hub.WorkspaceManager
	appConfig        *config.Config
	serverStartTime  time.Time
	ollamaMgr        *ollamaManager.Manager
)

// CORS middleware to allow requests from Tauri dev server
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from Tauri dev server (port 1420) and other origins
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
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
		if *addr == ":8080" {
			*addr = defaultAddr
		}
	}

	chatHub = hub.NewHub()

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

	// Initialize specialist agents from config (replaces standalone processes)
	initializeConfiguredAgents()

	// HTTP routes with CORS middleware
	http.HandleFunc("/ws", handleWebSocket) // WebSocket already handles origin
	http.HandleFunc("/api/channels", corsMiddleware(handleChannels))
	http.HandleFunc("/api/channels/create", corsMiddleware(handleCreateChannel))
	http.HandleFunc("/api/channels/join", corsMiddleware(handleJoinChannel))
	http.HandleFunc("/api/channels/delete", corsMiddleware(handleDeleteChannel))
	http.HandleFunc("/api/channels/agents", corsMiddleware(handleChannelAgentsManage))
	http.HandleFunc("/api/agent-channels", corsMiddleware(handleAgentChannels))
	http.HandleFunc("/api/agents", corsMiddleware(handleAgentsRoute))
	http.HandleFunc("/api/my-agents", corsMiddleware(handleMyAgents))
	http.HandleFunc("/api/cached-agents", corsMiddleware(handleCachedAgents)) // Keep for backwards compatibility
	http.HandleFunc("/api/removed-agents", corsMiddleware(handleRemovedAgents))
	http.HandleFunc("/api/messages", corsMiddleware(handleMessages))
	http.HandleFunc("/api/send", corsMiddleware(handleSendMessage))
	http.HandleFunc("/api/broadcast", corsMiddleware(handleBroadcastDirect))
	http.HandleFunc("/api/threads/", corsMiddleware(handleThreads)) // Thread endpoints
	http.HandleFunc("/api/import", corsMiddleware(handleImport))    // Import agent endpoint

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

	// Command palette metadata
	http.HandleFunc("/api/commands", corsMiddleware(handleCommands))

	// Home page handler (must be last to avoid catching API routes)
	http.HandleFunc("/", corsMiddleware(handleHome))

	log.Printf("Chat Hub Server starting on %s", *addr)
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", *addr)
	log.Printf("Web UI: http://localhost%s", *addr)
	log.Printf("CORS enabled for all origins")

	sessionPath := hub.DefaultSessionPath()
	log.Printf("💾 Session will be saved to: %s", sessionPath)

	// Periodic session save (every 2 minutes)
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := chatHub.SaveSessionToFile(sessionPath); err != nil {
				log.Printf("⚠️  Periodic session save failed: %v", err)
			}
		}
	}()

	// Graceful shutdown: save session on SIGINT/SIGTERM
	server := &http.Server{Addr: *addr}
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

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

func handleHome(w http.ResponseWriter, r *http.Request) {
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
		AgentID string `json:"agent_id"`
		Channel string `json:"channel"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := chatHub.JoinChannel(req.AgentID, req.Channel); err != nil {
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

	json.NewEncoder(w).Encode(messages)
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
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

	// Use provided 'from' info or default to "Human User"
	senderID := "human-user"
	senderName := "Human User"
	senderType := protocol.AgentTypeGeneral

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
	}

	if err := chatHub.SendMessage(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
		Channel string `json:"channel"`
		Content string `json:"content"`
		From    *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"from"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use provided 'from' info or default to "Human User"
	senderID := "human-user"
	senderName := "Human User"
	senderType := protocol.AgentTypeGeneral

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

	// Register moderator with hub
	if err := chatHub.RegisterAgent(&moderator.Info); err != nil {
		log.Printf("❌ Failed to register moderator agent: %v", err)
		return
	}

	// Join general channel
	if err := chatHub.JoinChannel(moderator.Info.ID, "general"); err != nil {
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

	// Send join message to announce moderator
	joinMsg := protocol.NewMessage(
		protocol.MessageTypeAgentJoin,
		"general",
		moderator.Info,
		"👋 ChatModerator online! I'm here to help with chat features and commands. Type @ChatModerator to ask me anything about using this chat system!",
	)
	if err := chatHub.SendMessage(joinMsg); err != nil {
		log.Printf("⚠️  Failed to send moderator join message: %v", err)
	}

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

	// Register assistant with hub
	if err := chatHub.RegisterAgent(&assistant.Info); err != nil {
		log.Printf("❌ Failed to register assistant agent: %v", err)
		return
	}

	// Register assistant with command handler for meeting notes functionality
	if commandHandler := chatHub.GetCommandHandler(); commandHandler != nil {
		if ch, ok := commandHandler.(*hub.CommandHandler); ok {
			ch.SetAssistantAgent(assistant)
		}
	}

	// Join general channel
	if err := chatHub.JoinChannel(assistant.Info.ID, "general"); err != nil {
		log.Printf("❌ Failed to join assistant to general channel: %v", err)
		return
	}

	// Start assistant in general channel
	ctx := context.Background()
	go func() {
		if err := assistant.Start(ctx, "general"); err != nil {
			log.Printf("❌ Failed to start assistant agent: %v", err)
			return
		}
	}()

	// Send join message to announce assistant
	joinMsg := protocol.NewMessage(
		protocol.MessageTypeAgentJoin,
		"general",
		assistant.Info,
		"👋 Personal Assistant online! I can help with reminders, tasks, notes, and more. Ask me '/help-assistant' to learn what I can do!",
	)
	if err := chatHub.SendMessage(joinMsg); err != nil {
		log.Printf("⚠️  Failed to send assistant join message: %v", err)
	}

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

	// Gemini-specific: configure tool approval hook
	if cfg.Type == "gemini" {
		configureGeminiApprovalHook()
	}

	cliAgent := agent.NewCLIAgentFromConfig(cfg, cfg.DefaultName, provider, chatHub)

	if cfg.ApprovalMode != "" {
		cliAgent.Info.ApprovalMode = cfg.ApprovalMode
	}

	if err := chatHub.RegisterAgent(&cliAgent.Info); err != nil {
		log.Printf("❌ Failed to register %s CLI agent: %v", cfg.DefaultName, err)
		return
	}

	if err := chatHub.JoinChannel(cliAgent.Info.ID, "general"); err != nil {
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

	joinMsg := protocol.NewMessage(
		protocol.MessageTypeAgentJoin,
		"general",
		cliAgent.Info,
		cfg.JoinMessage,
	)
	if err := chatHub.SendMessage(joinMsg); err != nil {
		log.Printf("⚠️  Failed to send %s agent join message: %v", cfg.DefaultName, err)
	}

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

	hookBinAbs, _ := filepath.Abs(hookBin)
	serverURL := fmt.Sprintf("http://localhost%s", *addr)
	hookCommand := fmt.Sprintf("%s --server %s --agent Gemini --agent-id gemini-cli --mode interactive", hookBinAbs, serverURL)

	// Read existing settings or start fresh
	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		json.Unmarshal(data, &settings)
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
		fmt.Fprintf(w, "data: {\"status\":\"error\",\"error\":\"%s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}
	fmt.Fprintf(w, "data: {\"status\":\"success\"}\n\n")
	flusher.Flush()
}

// buildProviderFromConfig creates an ai.AIProvider from a config.ProviderConfig.
func buildProviderFromConfig(pcfg *config.ProviderConfig) (ai.AIProvider, error) {
	switch pcfg.Type {
	case "ollama":
		endpoint := pcfg.Endpoint
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		}
		model := pcfg.Model
		if model == "" {
			model = "llama3.1"
		}
		return ai.NewOllamaProviderWithConfig(endpoint, model), nil

	case "anthropic":
		if pcfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic provider %q has no API key", pcfg.ID)
		}
		return ai.NewClaudeProviderWithConfig(pcfg.APIKey, false, "", pcfg.Model), nil

	case "openai-compatible":
		endpoint := pcfg.Endpoint
		if endpoint == "" {
			return nil, fmt.Errorf("openai-compatible provider %q has no endpoint", pcfg.ID)
		}
		model := pcfg.Model
		return ai.NewOpenAICompatProvider(endpoint, pcfg.APIKey, model, pcfg.Headers), nil

	case "cursor-cli":
		workDir := pcfg.WorkDir
		if workDir == "" {
			workDir, _ = os.Getwd()
		}
		return ai.NewCursorCLIProvider(workDir, pcfg.APIKey), nil

	case "gemini-cli":
		workDir := pcfg.WorkDir
		if workDir == "" {
			workDir, _ = os.Getwd()
		}
		return ai.NewGeminiCLIProvider(workDir), nil

	default:
		return nil, fmt.Errorf("unknown provider type %q", pcfg.Type)
	}
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

		aiProvider, err := buildProviderFromConfig(pcfg)
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

		if err := chatHub.RegisterAgent(&agentObj.Info); err != nil {
			log.Printf("❌ Failed to register agent %s: %v", acfg.Name, err)
			continue
		}

		if err := chatHub.JoinChannel(agentObj.Info.ID, "general"); err != nil {
			log.Printf("❌ Failed to join agent %s to general channel: %v", acfg.Name, err)
			continue
		}

		ctx := context.Background()
		go func(name string) {
			if err := agentObj.Start(ctx, "general"); err != nil {
				log.Printf("❌ Failed to start agent %s: %v", name, err)
			}
		}(acfg.Name)

		joinMsg := protocol.NewMessage(
			protocol.MessageTypeAgentJoin,
			"general",
			agentObj.Info,
			fmt.Sprintf("👋 %s online! Ready to help with %s questions.", acfg.Name, acfg.Type),
		)
		if err := chatHub.SendMessage(joinMsg); err != nil {
			log.Printf("⚠️  Failed to send join message for %s: %v", acfg.Name, err)
		}

		log.Printf("✅ Agent %s started (type=%s, provider=%s, model=%s)",
			acfg.Name, acfg.Type, pcfg.Name, aiProvider.GetModel())
	}
}

// ── Health, Settings, Provider, Agent Config endpoints ───────────────

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

		appConfig.Server = incoming.Server
		appConfig.AI = incoming.AI
		appConfig.Agents = incoming.Agents
		appConfig.Ollama = incoming.Ollama
		appConfig.Updates = incoming.Updates

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
		provider, err := buildProviderFromConfig(pcfg)
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
		if err := appConfig.UpdateProvider(p); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		appConfig.Save()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	case http.MethodDelete:
		if err := appConfig.RemoveProvider(id); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		appConfig.Save()
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get cached agents from all storage types
	myAgents, err := getAllCachedAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"my_agents": myAgents,
	}

	json.NewEncoder(w).Encode(response)
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

	// Get cached helper agents
	helperStorage, err := agent.NewHelperAgentStorage()
	if err == nil {
		helperAgents, err := helperStorage.ListConfigsWithMetadata()
		if err == nil {
			allAgents = append(allAgents, helperAgents...)
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

// File system API handlers

func handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
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

	fullPath := filepath.Join(workspace.Path, path)

	// Security check - ensure path is within workspace
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	workspaceAbsPath, err := filepath.Abs(workspace.Path)
	if err != nil {
		http.Error(w, "Invalid workspace path", http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(absPath, workspaceAbsPath) {
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

		// Security check
		absPath, err := filepath.Abs(fullPath)
		if err != nil {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		workspaceAbsPath, err := filepath.Abs(workspace.Path)
		if err != nil {
			http.Error(w, "Invalid workspace path", http.StatusInternalServerError)
			return
		}
		if !strings.HasPrefix(absPath, workspaceAbsPath) {
			http.Error(w, "Path outside workspace", http.StatusForbidden)
			return
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

		// Security check
		absPath, err := filepath.Abs(fullPath)
		if err != nil {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		workspaceAbsPath, err := filepath.Abs(workspace.Path)
		if err != nil {
			http.Error(w, "Invalid workspace path", http.StatusInternalServerError)
			return
		}
		if !strings.HasPrefix(absPath, workspaceAbsPath) {
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

	// Security check
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	workspaceAbsPath, err := filepath.Abs(workspace.Path)
	if err != nil {
		http.Error(w, "Invalid workspace path", http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(absPath, workspaceAbsPath) {
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

	// Security checks
	oldAbsPath, err := filepath.Abs(oldFullPath)
	if err != nil {
		http.Error(w, "Invalid old path", http.StatusBadRequest)
		return
	}
	newAbsPath, err := filepath.Abs(newFullPath)
	if err != nil {
		http.Error(w, "Invalid new path", http.StatusBadRequest)
		return
	}
	workspaceAbsPath, err := filepath.Abs(workspace.Path)
	if err != nil {
		http.Error(w, "Invalid workspace path", http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(oldAbsPath, workspaceAbsPath) || !strings.HasPrefix(newAbsPath, workspaceAbsPath) {
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

	// Security check
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	workspaceAbsPath, err := filepath.Abs(workspace.Path)
	if err != nil {
		http.Error(w, "Invalid workspace path", http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(absPath, workspaceAbsPath) {
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
	if request.Provider != "claude" && request.Provider != "ollama" && request.Provider != "lmstudio" {
		http.Error(w, "Invalid provider. Use 'claude', 'ollama', or 'lmstudio'", http.StatusBadRequest)
		return
	}

	// Find the agent
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

	// Set default model if not provided
	if request.Model == "" {
		if request.Provider == "ollama" {
			request.Model = "llama3.1"
		} else if request.Provider == "lmstudio" {
			request.Model = "" // Will be determined from available models
		} else {
			request.Model = "claude-sonnet"
		}
	}

	// Update agent info
	targetAgent.AIProvider = request.Provider
	targetAgent.AIModel = request.Model
	targetAgent.Model = request.Model

	// Broadcast the change
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		"general",
		*targetAgent,
		fmt.Sprintf("🔄 %s switched to %s (%s)", targetAgent.Name, request.Provider, request.Model),
	)
	statusMsg.Metadata = map[string]interface{}{
		"ai_provider": request.Provider,
		"ai_model":    request.Model,
		"model":       request.Model,
	}

	chatHub.SendMessage(statusMsg)

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent %s switched to %s (%s)", targetAgent.Name, request.Provider, request.Model),
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
	if request.Provider != "claude" && request.Provider != "ollama" && request.Provider != "lmstudio" {
		http.Error(w, "Invalid provider. Use 'claude', 'ollama', or 'lmstudio'", http.StatusBadRequest)
		return
	}

	// Set default model if not provided
	if request.Model == "" {
		if request.Provider == "ollama" {
			request.Model = "llama3.1"
		} else if request.Provider == "lmstudio" {
			request.Model = "" // Will be determined from available models
		} else {
			request.Model = "claude-sonnet"
		}
	}

	// Switch all agents
	agents := chatHub.ListAgents()
	switchedCount := 0

	for _, agent := range agents {
		// Update agent info
		agent.AIProvider = request.Provider
		agent.AIModel = request.Model
		agent.Model = request.Model

		// Broadcast the change
		statusMsg := protocol.NewMessage(
			protocol.MessageTypeAgentStatus,
			"general",
			*agent,
			fmt.Sprintf("🔄 %s switched to %s (%s)", agent.Name, request.Provider, request.Model),
		)
		statusMsg.Metadata = map[string]interface{}{
			"ai_provider": request.Provider,
			"ai_model":    request.Model,
			"model":       request.Model,
		}

		chatHub.SendMessage(statusMsg)
		switchedCount++
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

	// Create Ollama provider to get models
	ollamaProvider := ai.NewOllamaProviderWithConfig("http://localhost:11434", "llama3.1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := ollamaProvider.GetAvailableModels(ctx)

	response := map[string]interface{}{
		"models":   models,
		"endpoint": "http://localhost:11434",
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
