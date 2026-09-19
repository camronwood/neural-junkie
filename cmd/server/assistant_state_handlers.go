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
        .pending-section { margin-top: 30px; }
        .pending-section h3 { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
        .pending-count {
            font-size: 12px;
            font-weight: normal;
            color: #7f8c8d;
            background: #ecf0f1;
            padding: 2px 8px;
            border-radius: 10px;
        }
        .pending-empty { color: #95a5a6; font-size: 13px; padding: 8px 0; }
        .pending-error { color: #c0392b; font-size: 12px; margin-bottom: 8px; }
        .pending-batch {
            background: #eef5fb;
            border: 1px solid #c5d9ec;
            border-radius: 6px;
            padding: 8px 10px;
            margin-bottom: 10px;
        }
        .pending-batch-title {
            font-size: 12px;
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 6px;
        }
        .pending-card {
            background: white;
            border: 1px solid #e1e4e8;
            border-radius: 6px;
            padding: 10px;
            margin-bottom: 8px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04);
        }
        .pending-card.destructive { border-color: #f0c36d; background: #fffbf0; }
        .pending-path {
            font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
            font-size: 12px;
            color: #2c3e50;
            word-break: break-all;
            margin-bottom: 6px;
        }
        .pending-meta {
            display: flex;
            flex-wrap: wrap;
            gap: 6px;
            align-items: center;
            margin-bottom: 6px;
        }
        .pending-badge {
            display: inline-block;
            padding: 2px 7px;
            border-radius: 10px;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
        }
        .pending-badge.op { background: #ecf0f1; color: #566573; }
        .pending-badge.status-pending { background: #d6eaf8; color: #1a5276; }
        .pending-badge.status-other { background: #fdebd0; color: #9a7b0a; }
        .pending-reason {
            font-size: 12px;
            color: #b9770e;
            margin-bottom: 8px;
            line-height: 1.4;
        }
        .pending-actions { display: flex; flex-wrap: wrap; gap: 6px; }
        .pending-actions button {
            padding: 6px 10px;
            border: none;
            border-radius: 6px;
            font-size: 12px;
            font-weight: 600;
            cursor: pointer;
        }
        .pending-actions button:disabled { opacity: 0.5; cursor: not-allowed; }
        .btn-expand { background: #ecf0f1; color: #2c3e50; }
        .btn-approve { background: #27ae60; color: white; }
        .btn-reject { background: #c0392b; color: white; }
        .btn-batch { background: #3498db; color: white; }
        .pending-diff {
            display: none;
            margin-top: 8px;
            max-height: 220px;
            overflow: auto;
            background: #1e1e1e;
            color: #d4d4d4;
            border-radius: 4px;
            padding: 8px;
            font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
            font-size: 11px;
            white-space: pre-wrap;
            line-height: 1.35;
        }
        .pending-diff.open { display: block; }
        .pending-diff .add { color: #98c379; }
        .pending-diff .del { color: #e06c75; }
        .pending-diff .hdr { color: #61afef; }
        .pending-note { font-size: 11px; color: #95a5a6; margin-top: 12px; line-height: 1.4; }
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
            
            <div class="pending-section">
                <h3>
                    Pending changes
                    <span class="pending-count" id="pending-count">0</span>
                </h3>
                <div class="pending-error" id="pending-error" style="display:none;"></div>
                <div id="pending-changes">
                    <div class="pending-empty">No pending file changes</div>
                </div>
                <p class="pending-note">
                    Browser hub: approve/reject with diff preview. No Monaco editor, ⌘K, or tab completion — use the desktop app for IDE workflows.
                </p>
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
                if (msg.type === 'file_change' || (msg.metadata && (msg.metadata.change_proposal || msg.metadata.file_change_held_for_approval))) {
                    loadPendingChanges();
                }
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
        
        const pendingUserId = 'default';
        let pendingBusy = false;
        const expandedDiffs = {};

        function escapeHtml(s) {
            return String(s == null ? '' : s)
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;');
        }

        function displayPath(change) {
            if (change.operation === 'move') {
                return (change.old_path || change.file_path || '') + ' → ' + (change.new_path || '');
            }
            return change.file_path || '';
        }

        function holdReason(change) {
            if (change.reason) return change.reason;
            if (change.metadata && change.metadata.file_change_hold_reason) {
                return change.metadata.file_change_hold_reason;
            }
            return '';
        }

        function requestIdOf(change) {
            if (!change.metadata) return '';
            return change.metadata.request_id || '';
        }

        function formatDiffHtml(diffText) {
            if (!diffText) return '<span style="color:#888;">No unified diff for this operation.</span>';
            return escapeHtml(diffText).split('\n').map(function(line) {
                if (line.indexOf('+++') === 0 || line.indexOf('---') === 0 || line.indexOf('@@') === 0) {
                    return '<span class="hdr">' + line + '</span>';
                }
                if (line.charAt(0) === '+' && line.indexOf('+++') !== 0) {
                    return '<span class="add">' + line + '</span>';
                }
                if (line.charAt(0) === '-' && line.indexOf('---') !== 0) {
                    return '<span class="del">' + line + '</span>';
                }
                return line;
            }).join('\n');
        }

        function setPendingError(msg) {
            const el = document.getElementById('pending-error');
            if (!msg) {
                el.style.display = 'none';
                el.textContent = '';
                return;
            }
            el.style.display = 'block';
            el.textContent = msg;
        }

        function groupPending(changes) {
            const order = [];
            const groups = {};
            const singles = [];
            (changes || []).forEach(function(c) {
                const rid = requestIdOf(c);
                if (rid) {
                    if (!groups[rid]) {
                        groups[rid] = [];
                        order.push(rid);
                    }
                    groups[rid].push(c);
                } else {
                    singles.push(c);
                }
            });
            return { order: order, groups: groups, singles: singles };
        }

        function renderChangeCard(change) {
            const path = escapeHtml(displayPath(change));
            const reason = holdReason(change);
            const op = escapeHtml(change.operation || 'edit');
            const status = escapeHtml(change.status || 'pending');
            const destructive = change.operation === 'delete' || change.operation === 'move';
            const open = expandedDiffs[change.id] ? ' open' : '';
            const reasonHtml = reason
                ? '<div class="pending-reason">' + escapeHtml(reason) + '</div>'
                : '';
            return (
                '<div class="pending-card' + (destructive ? ' destructive' : '') + '" data-id="' + escapeHtml(change.id) + '">' +
                    '<div class="pending-path" title="' + path + '">' + path + '</div>' +
                    '<div class="pending-meta">' +
                        '<span class="pending-badge op">' + op + '</span>' +
                        '<span class="pending-badge status-' + (status === 'pending' ? 'pending' : 'other') + '">' + status + '</span>' +
                    '</div>' +
                    reasonHtml +
                    '<div class="pending-actions">' +
                        '<button type="button" class="btn-expand" data-action="toggle-diff" data-id="' + escapeHtml(change.id) + '">' +
                            (expandedDiffs[change.id] ? 'Hide diff' : 'Show diff') +
                        '</button>' +
                        '<button type="button" class="btn-approve" data-action="approve" data-id="' + escapeHtml(change.id) + '"' + (pendingBusy ? ' disabled' : '') + '>Approve</button>' +
                        '<button type="button" class="btn-reject" data-action="reject" data-id="' + escapeHtml(change.id) + '"' + (pendingBusy ? ' disabled' : '') + '>Reject</button>' +
                    '</div>' +
                    '<pre class="pending-diff' + open + '" id="diff-' + escapeHtml(change.id) + '"></pre>' +
                '</div>'
            );
        }

        function renderPendingChanges(changes) {
            const list = document.getElementById('pending-changes');
            const countEl = document.getElementById('pending-count');
            const items = changes || [];
            countEl.textContent = String(items.length);
            if (items.length === 0) {
                list.innerHTML = '<div class="pending-empty">No pending file changes</div>';
                return;
            }
            const grouped = groupPending(items);
            let html = '';
            grouped.order.forEach(function(rid) {
                const members = grouped.groups[rid] || [];
                html += '<div class="pending-batch">' +
                    '<div class="pending-batch-title">Batch · ' + members.length + ' file' + (members.length === 1 ? '' : 's') + '</div>' +
                    '<div class="pending-actions" style="margin-bottom:8px;">' +
                        '<button type="button" class="btn-batch" data-action="approve-request" data-request-id="' + escapeHtml(rid) + '"' + (pendingBusy ? ' disabled' : '') + '>Approve all</button>' +
                        '<button type="button" class="btn-reject" data-action="reject-request" data-request-id="' + escapeHtml(rid) + '"' + (pendingBusy ? ' disabled' : '') + '>Reject all</button>' +
                    '</div>' +
                    members.map(renderChangeCard).join('') +
                '</div>';
            });
            html += grouped.singles.map(renderChangeCard).join('');
            list.innerHTML = html;

            Object.keys(expandedDiffs).forEach(function(id) {
                if (expandedDiffs[id]) {
                    const pre = document.getElementById('diff-' + id);
                    if (pre && !pre.dataset.loaded) {
                        loadDiff(id);
                    } else if (pre && pre.dataset.loaded) {
                        pre.classList.add('open');
                    }
                }
            });
        }

        function loadPendingChanges() {
            fetch('/api/file-changes?user_id=' + encodeURIComponent(pendingUserId))
                .then(function(r) {
                    if (!r.ok) throw new Error(r.statusText || 'failed');
                    return r.json();
                })
                .then(function(changes) {
                    setPendingError('');
                    renderPendingChanges(Array.isArray(changes) ? changes : []);
                })
                .catch(function(err) {
                    setPendingError('Could not load pending changes: ' + (err.message || err));
                });
        }

        function loadDiff(changeId) {
            const pre = document.getElementById('diff-' + changeId);
            if (!pre) return;
            pre.classList.add('open');
            pre.textContent = 'Loading…';
            fetch('/api/file-changes/' + encodeURIComponent(changeId))
                .then(function(r) {
                    if (!r.ok) return r.text().then(function(t) { throw new Error(t || r.statusText); });
                    return r.json();
                })
                .then(function(data) {
                    const change = data.change || {};
                    let text = data.diff || '';
                    if (!text && change.operation === 'create') {
                        text = '+++ create ' + (change.file_path || '') + '\n' + (change.new_content || '');
                    } else if (!text && change.operation === 'delete') {
                        text = '--- delete ' + (change.file_path || '') + '\n' + (change.old_content || '');
                    } else if (!text && change.operation === 'move') {
                        text = 'move: ' + (change.old_path || '') + ' → ' + (change.new_path || '');
                    }
                    pre.innerHTML = formatDiffHtml(text);
                    pre.dataset.loaded = '1';
                })
                .catch(function(err) {
                    pre.textContent = 'Diff unavailable: ' + (err.message || err);
                });
        }

        function toggleDiff(changeId) {
            if (expandedDiffs[changeId]) {
                expandedDiffs[changeId] = false;
            } else {
                expandedDiffs[changeId] = true;
            }
            loadPendingChanges();
        }

        function runPendingAction(promise) {
            if (pendingBusy) return;
            pendingBusy = true;
            setPendingError('');
            loadPendingChanges();
            promise
                .then(function() {
                    pendingBusy = false;
                    loadPendingChanges();
                })
                .catch(function(err) {
                    pendingBusy = false;
                    setPendingError(err.message || String(err));
                    loadPendingChanges();
                });
        }

        function approveChange(changeId) {
            runPendingAction(
                fetch('/api/file-changes/approve/' + encodeURIComponent(changeId) + '?user_id=' + encodeURIComponent(pendingUserId), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: '{}'
                }).then(function(r) {
                    if (!r.ok) return r.text().then(function(t) { throw new Error(t || r.statusText); });
                })
            );
        }

        function rejectChange(changeId) {
            runPendingAction(
                fetch('/api/file-changes/reject/' + encodeURIComponent(changeId), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ user_id: pendingUserId, reason: 'Rejected from web UI' })
                }).then(function(r) {
                    if (!r.ok) return r.text().then(function(t) { throw new Error(t || r.statusText); });
                })
            );
        }

        function approveRequest(requestId) {
            runPendingAction(
                fetch('/api/file-changes/requests/' + encodeURIComponent(requestId) + '/approve?user_id=' + encodeURIComponent(pendingUserId), {
                    method: 'POST'
                }).then(function(r) {
                    if (!r.ok) return r.text().then(function(t) { throw new Error(t || r.statusText); });
                })
            );
        }

        function rejectRequest(requestId) {
            runPendingAction(
                fetch('/api/file-changes/requests/' + encodeURIComponent(requestId) + '/reject', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ user_id: pendingUserId, reason: 'Rejected from web UI' })
                }).then(function(r) {
                    if (!r.ok) return r.text().then(function(t) { throw new Error(t || r.statusText); });
                })
            );
        }

        document.getElementById('pending-changes').addEventListener('click', function(e) {
            const btn = e.target.closest('button[data-action]');
            if (!btn) return;
            const action = btn.getAttribute('data-action');
            const id = btn.getAttribute('data-id');
            const rid = btn.getAttribute('data-request-id');
            if (action === 'toggle-diff' && id) toggleDiff(id);
            else if (action === 'approve' && id) approveChange(id);
            else if (action === 'reject' && id) rejectChange(id);
            else if (action === 'approve-request' && rid) approveRequest(rid);
            else if (action === 'reject-request' && rid) rejectRequest(rid);
        });

        connect();
        loadPendingChanges();
        setInterval(loadAgents, 5000);
        setInterval(loadPendingChanges, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
