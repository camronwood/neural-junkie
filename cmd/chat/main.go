package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/gorilla/websocket"
)

var (
	serverAddr = flag.String("server", "localhost:8080", "Chat hub server address")
	channel    = flag.String("channel", "general", "Channel to join")
	username   = flag.String("name", "", "Your display name (optional)")
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

type ChatClient struct {
	conn     *websocket.Conn
	channel  string
	username string
	stopCh   chan struct{}
}

func main() {
	flag.Parse()

	// Get username
	name := *username
	if name == "" {
		fmt.Print("Enter your name: ")
		reader := bufio.NewReader(os.Stdin)
		name, _ = reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name == "" {
			name = "Anonymous"
		}
	}

	client := &ChatClient{
		channel:  *channel,
		username: name,
		stopCh:   make(chan struct{}),
	}

	// Connect to server
	if err := client.connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.conn.Close()

	// Print welcome message
	client.printWelcome()

	// Start message receiver
	go client.receiveMessages()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n" + colorYellow + "Goodbye! 👋" + colorReset)
		client.stopCh <- struct{}{}
	}()

	// Start input loop
	client.inputLoop()
}

func (c *ChatClient) connect() error {
	u := url.URL{Scheme: "ws", Host: *serverAddr, Path: "/ws"}
	q := u.Query()
	q.Set("channel", c.channel)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial error: %w", err)
	}

	c.conn = conn

	// Send join message
	go func() {
		time.Sleep(500 * time.Millisecond)
		c.sendJoinMessage()
	}()

	return nil
}

func (c *ChatClient) sendJoinMessage() {
	apiURL := fmt.Sprintf("http://%s/api/send", *serverAddr)

	msg := map[string]interface{}{
		"channel": c.channel,
		"content": fmt.Sprintf("👋 %s has joined the chat", c.username),
		"type":    "system_info",
		"from": map[string]interface{}{
			"id":   "human-" + c.username,
			"name": c.username,
			"type": "general",
		},
	}

	data, _ := json.Marshal(msg)
	resp, err := http.Post(apiURL, "application/json", strings.NewReader(string(data)))
	if err == nil {
		resp.Body.Close()
	}
}

func (c *ChatClient) printWelcome() {
	fmt.Println(colorCyan + "╔════════════════════════════════════════════════╗" + colorReset)
	fmt.Println(colorCyan + "║" + colorBold + "       Neural Junkie - Interactive Mode         " + colorReset + colorCyan + "║" + colorReset)
	fmt.Println(colorCyan + "╚════════════════════════════════════════════════╝" + colorReset)
	fmt.Println()
	fmt.Printf(colorGreen+"Welcome, %s!"+colorReset+"\n", c.username)
	fmt.Printf("Channel: "+colorBlue+"#%s"+colorReset+"\n", c.channel)
	fmt.Println()
	fmt.Println(colorGray + "Commands:" + colorReset)
	fmt.Println(colorGray + "  /help     - Show this help" + colorReset)
	fmt.Println(colorGray + "  /agents   - List active agents" + colorReset)
	fmt.Println(colorGray + "  /channels - List available channels" + colorReset)
	fmt.Println(colorGray + "  /clear    - Clear screen" + colorReset)
	fmt.Println(colorGray + "  /quit     - Exit chat" + colorReset)
	fmt.Println()
	fmt.Println(colorYellow + "Type your message and press Enter to send." + colorReset)
	fmt.Println(colorGray + "════════════════════════════════════════════════" + colorReset)
	fmt.Println()
}

func (c *ChatClient) receiveMessages() {
	for {
		select {
		case <-c.stopCh:
			return
		default:
			var msg protocol.Message
			err := c.conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			// Don't show our own messages
			if msg.From.Name == c.username {
				continue
			}

			c.displayMessage(&msg)
		}
	}
}

func (c *ChatClient) displayMessage(msg *protocol.Message) {
	timestamp := msg.Timestamp.Format("15:04:05")

	// Color based on agent type
	nameColor := colorGreen
	switch msg.From.Type {
	case protocol.AgentTypeFrontend:
		nameColor = colorCyan
	case protocol.AgentTypeBackend:
		nameColor = colorBlue
	case protocol.AgentTypeDevOps:
		nameColor = colorPurple
	case protocol.AgentTypeDatabase:
		nameColor = colorYellow
	case protocol.AgentTypeSecurity:
		nameColor = colorRed
	}

	// Format based on message type
	switch msg.Type {
	case protocol.MessageTypeAgentJoin, protocol.MessageTypeAgentLeave:
		fmt.Printf("%s[%s]%s %s%s%s\n",
			colorGray, timestamp, colorReset,
			colorGray, msg.Content, colorReset)
	case protocol.MessageTypeSystemInfo:
		fmt.Printf("%s[%s]%s %s%s%s\n",
			colorGray, timestamp, colorReset,
			colorYellow, msg.Content, colorReset)
	default:
		// Regular message
		fmt.Printf("%s[%s]%s %s%s%s %s(%s)%s\n",
			colorGray, timestamp, colorReset,
			nameColor+colorBold, msg.From.Name, colorReset,
			colorGray, msg.From.Type, colorReset)

		// Wrap long messages
		content := c.wrapText(msg.Content, 70)
		for _, line := range strings.Split(content, "\n") {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
	}
}

func (c *ChatClient) wrapText(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				lines = append(lines, word)
			}
		} else {
			if currentLine != "" {
				currentLine += " " + word
			} else {
				currentLine = word
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n  ")
}

func (c *ChatClient) inputLoop() {
	reader := bufio.NewReader(os.Stdin)

	for {
		select {
		case <-c.stopCh:
			return
		default:
			fmt.Print(colorGreen + "You: " + colorReset)
			input, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}

			// Handle commands
			if strings.HasPrefix(input, "/") {
				c.handleCommand(input)
				continue
			}

			// Send message
			c.sendMessage(input)
		}
	}
}

func (c *ChatClient) handleCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "/help":
		c.showHelp()
	case "/agents":
		c.listAgents()
	case "/channels":
		c.listChannels()
	case "/clear":
		fmt.Print("\033[H\033[2J")
		c.printWelcome()
	case "/quit", "/exit":
		c.stopCh <- struct{}{}
	default:
		fmt.Printf(colorRed+"Unknown command: %s"+colorReset+"\n", command)
		fmt.Println(colorGray + "Type /help for available commands" + colorReset)
	}
}

func (c *ChatClient) showHelp() {
	fmt.Println()
	fmt.Println(colorCyan + "Available Commands:" + colorReset)
	fmt.Println(colorGray + "  /help     - Show this help message" + colorReset)
	fmt.Println(colorGray + "  /agents   - List all active AI agents" + colorReset)
	fmt.Println(colorGray + "  /channels - List all available channels" + colorReset)
	fmt.Println(colorGray + "  /clear    - Clear the screen" + colorReset)
	fmt.Println(colorGray + "  /quit     - Exit the chat room" + colorReset)
	fmt.Println()
}

func (c *ChatClient) listAgents() {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/agents", *serverAddr))
	if err != nil {
		fmt.Printf(colorRed+"Error fetching agents: %v"+colorReset+"\n", err)
		return
	}
	defer resp.Body.Close()

	var agents []protocol.AgentInfo
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		fmt.Printf(colorRed+"Error decoding agents: %v"+colorReset+"\n", err)
		return
	}

	fmt.Println()
	fmt.Println(colorCyan + "🤖 Active AI Agents:" + colorReset)
	if len(agents) == 0 {
		fmt.Println(colorGray + "  No agents currently active" + colorReset)
	} else {
		for _, agent := range agents {
			emoji := "🤖"
			switch agent.Type {
			case protocol.AgentTypeFrontend:
				emoji = "🎨"
			case protocol.AgentTypeBackend:
				emoji = "⚙️"
			case protocol.AgentTypeDevOps:
				emoji = "🔧"
			case protocol.AgentTypeDatabase:
				emoji = "💾"
			case protocol.AgentTypeSecurity:
				emoji = "🔒"
			}
			fmt.Printf("  %s %s%s%s (%s%s%s)\n",
				emoji,
				colorBold, agent.Name, colorReset,
				colorGray, agent.Type, colorReset)
		}
	}
	fmt.Println()
}

func (c *ChatClient) listChannels() {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/channels", *serverAddr))
	if err != nil {
		fmt.Printf(colorRed+"Error fetching channels: %v"+colorReset+"\n", err)
		return
	}
	defer resp.Body.Close()

	var channels []protocol.Channel
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		fmt.Printf(colorRed+"Error decoding channels: %v"+colorReset+"\n", err)
		return
	}

	fmt.Println()
	fmt.Println(colorCyan + "📚 Available Channels:" + colorReset)
	for _, ch := range channels {
		current := ""
		if ch.Name == c.channel {
			current = colorGreen + " (current)" + colorReset
		}
		fmt.Printf("  %s#%s%s - %s%s\n",
			colorBlue, ch.Name, colorReset,
			ch.Description, current)
	}
	fmt.Println()
}

func (c *ChatClient) sendMessage(content string) {
	apiURL := fmt.Sprintf("http://%s/api/send", *serverAddr)

	msg := map[string]interface{}{
		"channel": c.channel,
		"content": content,
		"type":    "question",
		"from": map[string]interface{}{
			"id":   "human-" + c.username,
			"name": c.username,
			"type": "general",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf(colorRed+"Error encoding message: %v"+colorReset+"\n", err)
		return
	}

	resp, err := http.Post(apiURL, "application/json", strings.NewReader(string(data)))
	if err != nil {
		fmt.Printf(colorRed+"Error sending message: %v"+colorReset+"\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf(colorRed+"Server error: %d"+colorReset+"\n", resp.StatusCode)
	}
}
