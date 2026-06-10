package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

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
