package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/gorilla/websocket"
)

// TestMessageSending tests the message sending functionality
func TestMessageSending(t *testing.T) {
	// Start a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/send" {
			var msg map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
				t.Errorf("Failed to decode message: %v", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			// Verify message structure
			if msg["channel"] == nil {
				t.Error("Message missing channel")
			}
			if msg["content"] == nil {
				t.Error("Message missing content")
			}
			if msg["type"] == nil {
				t.Error("Message missing type")
			}

			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Test message
	testMsg := map[string]interface{}{
		"channel": "general",
		"content": "Hello from test!",
		"type":    "question",
	}

	data, err := json.Marshal(testMsg)
	if err != nil {
		t.Fatalf("Failed to marshal test message: %v", err)
	}

	// Send message
	resp, err := http.Post(server.URL+"/api/send", "application/json", strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestWebSocketMessageFlow tests the WebSocket message flow
func TestWebSocketMessageFlow(t *testing.T) {
	// Create a test WebSocket server
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// Send a test message to the client
		testMsg := protocol.Message{
			Type:    protocol.MessageTypeQuestion,
			Content: "Test message from server",
			From: protocol.AgentInfo{
				Name: "TestAgent",
				Type: protocol.AgentTypeBackend,
			},
			Channel:   "general",
			Timestamp: time.Now(),
		}

		if err := conn.WriteJSON(testMsg); err != nil {
			t.Errorf("Failed to send message: %v", err)
		}

		// Read echo from client
		var received protocol.Message
		if err := conn.ReadJSON(&received); err != nil {
			t.Logf("Client disconnected or error: %v", err)
		}
	}))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?channel=general"

	// Connect as client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Receive message
	var msg protocol.Message
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	// Verify message
	if msg.Content != "Test message from server" {
		t.Errorf("Expected 'Test message from server', got '%s'", msg.Content)
	}
	if msg.From.Name != "TestAgent" {
		t.Errorf("Expected sender 'TestAgent', got '%s'", msg.From.Name)
	}

	// Send message back
	replyMsg := protocol.Message{
		Type:    protocol.MessageTypeQuestion,
		Content: "Reply from client",
		From: protocol.AgentInfo{
			Name: "TestUser",
			Type: protocol.AgentTypeBackend,
		},
		Channel:   "general",
		Timestamp: time.Now(),
	}

	if err := conn.WriteJSON(replyMsg); err != nil {
		t.Errorf("Failed to send reply: %v", err)
	}
}

// TestEventSystemIntegration tests the event system with message sending
func TestEventSystemIntegration(t *testing.T) {
	// Simulate the event system
	eventChan := make(chan string, 10)
	stopChan := make(chan struct{})

	// Simulate event processor
	go func() {
		for {
			select {
			case event := <-eventChan:
				t.Logf("Received event: %s", event)
			case <-stopChan:
				return
			}
		}
	}()

	// Simulate sending events
	events := []string{
		"EventShowLogin",
		"EventConnectionSuccess",
		"EventShowChat",
		"EventAddMessage",
		"EventAddSystemMessage",
	}

	for _, event := range events {
		eventChan <- event
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Verify channel has processed events
	if len(eventChan) > 0 {
		t.Logf("Warning: %d events still in queue", len(eventChan))
	}

	close(stopChan)
	close(eventChan)
}

// TestMessageFormatting tests message formatting and validation
func TestMessageFormatting(t *testing.T) {
	tests := []struct {
		name    string
		content string
		channel string
		msgType string
		valid   bool
	}{
		{"Valid question", "How do I use React hooks?", "general", "question", true},
		{"Valid statement", "I think we should use TypeScript", "general", "statement", true},
		{"Empty content", "", "general", "question", false},
		{"Empty channel", "Hello", "", "question", false},
		{"Long message", strings.Repeat("A", 10000), "general", "question", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := map[string]interface{}{
				"channel": tt.channel,
				"content": tt.content,
				"type":    tt.msgType,
			}

			// Validate
			valid := true
			if tt.content == "" {
				valid = false
			}
			if tt.channel == "" {
				valid = false
			}

			if valid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.valid, valid)
			}

			if valid {
				// Should be able to marshal
				_, err := json.Marshal(msg)
				if err != nil {
					t.Errorf("Failed to marshal valid message: %v", err)
				}
			}
		})
	}
}

// TestAgentResponseSimulation tests simulated agent responses
func TestAgentResponseSimulation(t *testing.T) {
	// Simulate different agent types responding
	agents := []struct {
		name          string
		agentType     protocol.AgentType
		question      string
		shouldRespond bool
	}{
		{"React Expert", protocol.AgentTypeFrontend, "How do I use React hooks?", true},
		{"Backend Expert", protocol.AgentTypeBackend, "How do I optimize database queries?", true},
		{"DevOps Expert", protocol.AgentTypeDevOps, "How do I set up CI/CD?", true},
		{"Frontend Expert", protocol.AgentTypeFrontend, "Database optimization?", false}, // Wrong expertise
	}

	for _, agent := range agents {
		t.Run(agent.name, func(t *testing.T) {
			// Simulate relevance detection
			relevant := false
			switch agent.agentType {
			case protocol.AgentTypeFrontend:
				relevant = strings.Contains(strings.ToLower(agent.question), "react") ||
					strings.Contains(strings.ToLower(agent.question), "frontend")
			case protocol.AgentTypeBackend:
				relevant = strings.Contains(strings.ToLower(agent.question), "database") ||
					strings.Contains(strings.ToLower(agent.question), "backend")
			case protocol.AgentTypeDevOps:
				relevant = strings.Contains(strings.ToLower(agent.question), "ci/cd") ||
					strings.Contains(strings.ToLower(agent.question), "devops")
			}

			if relevant != agent.shouldRespond {
				t.Errorf("Expected shouldRespond=%v, got relevant=%v", agent.shouldRespond, relevant)
			}
		})
	}
}

// BenchmarkMessageSending benchmarks message sending performance
func BenchmarkMessageSending(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/send" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	msg := map[string]interface{}{
		"channel": "general",
		"content": "Benchmark message",
		"type":    "question",
	}

	data, _ := json.Marshal(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Post(server.URL+"/api/send", "application/json", strings.NewReader(string(data)))
		if err != nil {
			b.Fatalf("Failed to send message: %v", err)
		}
		resp.Body.Close()
	}
}

// BenchmarkEventProcessing benchmarks event processing
func BenchmarkEventProcessing(b *testing.B) {
	eventChan := make(chan string, 1000)
	stopChan := make(chan struct{})

	// Start processor
	go func() {
		for {
			select {
			case <-eventChan:
				// Process event
			case <-stopChan:
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventChan <- "EventAddMessage"
	}
	b.StopTimer()

	close(stopChan)
}
