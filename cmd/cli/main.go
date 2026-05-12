package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	serverAddr = flag.String("server", "http://localhost:18765", "Chat hub server address")
	channel    = flag.String("channel", "general", "Channel name")
	message    = flag.String("message", "", "Message to send")
	list       = flag.String("list", "", "List items (channels, agents, messages)")
	watch      = flag.Bool("watch", false, "Watch for new messages")
	create     = flag.String("create", "", "Create a new channel")

	// Export commands
	export     = flag.String("export", "", "Export agent (repo-agent)")
	exportName = flag.String("name", "", "Agent name to export")
	output     = flag.String("output", "", "Output file path")
	importPath = flag.String("import", "", "Import agent from file path")
	serveMCP   = flag.Bool("serve-mcp", false, "Start MCP resource server")
	mcpPort    = flag.Int("port", 8086, "MCP resource server port")
)

func main() {
	flag.Parse()

	client := &http.Client{Timeout: 10 * time.Second}

	// Handle different commands
	switch {
	case *list == "channels":
		listChannels(client)
	case *list == "agents":
		listAgents(client)
	case *list == "messages":
		listMessages(client, *channel)
	case *list == "exports":
		listExports(client)
	case *create != "":
		createChannel(client, *create)
	case *watch:
		watchMessages(client, *channel)
	case *message != "":
		sendMessage(client, *channel, *message)
	case *export != "":
		exportAgent(client, *export, *exportName, *output)
	case *importPath != "":
		importAgent(client, *importPath)
	case *serveMCP:
		serveMCPResources(*mcpPort)
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Neural Junkie CLI")
	fmt.Println("\nUsage:")
	fmt.Println("  Send a message:")
	fmt.Println("    go run cmd/cli/main.go --channel general --message \"How do we optimize the API?\"")
	fmt.Println("\n  List channels:")
	fmt.Println("    go run cmd/cli/main.go --list channels")
	fmt.Println("\n  List agents:")
	fmt.Println("    go run cmd/cli/main.go --list agents")
	fmt.Println("\n  List messages:")
	fmt.Println("    go run cmd/cli/main.go --list messages --channel general")
	fmt.Println("\n  Watch for new messages:")
	fmt.Println("    go run cmd/cli/main.go --watch --channel general")
	fmt.Println("\n  Create a channel:")
	fmt.Println("    go run cmd/cli/main.go --create \"new-project\"")
	fmt.Println("\n  Export agents:")
	fmt.Println("    go run cmd/cli/main.go --export repo-agent --name \"MyProject Expert\" --output export.json")
	fmt.Println("\n  List exports:")
	fmt.Println("    go run cmd/cli/main.go --list exports")
	fmt.Println("\n  Import agent:")
	fmt.Println("    go run cmd/cli/main.go --import /path/to/export.json")
	fmt.Println("\n  Start MCP resource server:")
	fmt.Println("    go run cmd/cli/main.go --serve-mcp --port 8086")
	fmt.Println()
}

func listChannels(client *http.Client) {
	resp, err := client.Get(*serverAddr + "/api/channels")
	if err != nil {
		log.Fatalf("Failed to fetch channels: %v", err)
	}
	defer resp.Body.Close()

	var channels []protocol.Channel
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}

	fmt.Println("📚 Available Channels:")
	fmt.Println()
	for _, ch := range channels {
		fmt.Printf("  # %s\n", ch.Name)
		fmt.Printf("    Description: %s\n", ch.Description)
		fmt.Printf("    Agents: %d\n", len(ch.Agents))
		fmt.Printf("    Created: %s\n", ch.Created.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}

func listAgents(client *http.Client) {
	resp, err := client.Get(*serverAddr + "/api/agents")
	if err != nil {
		log.Fatalf("Failed to fetch agents: %v", err)
	}
	defer resp.Body.Close()

	var agents []protocol.AgentInfo
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}

	if len(agents) == 0 {
		fmt.Println("No agents currently active.")
		return
	}

	fmt.Println("🤖 Active Agents:")
	fmt.Println()
	for _, agent := range agents {
		fmt.Printf("  %s (%s)\n", agent.Name, agent.Type)
		fmt.Printf("    ID: %s\n", agent.ID)
		fmt.Printf("    Model: %s\n", agent.Model)
		fmt.Printf("    Status: %s\n", agent.Status)
		if len(agent.Expertise) > 0 {
			fmt.Printf("    Expertise: %v\n", agent.Expertise[:min(5, len(agent.Expertise))])
		}
		fmt.Println()
	}
}

func listMessages(client *http.Client, channel string) {
	url := fmt.Sprintf("%s/api/messages?channel=%s&limit=20", *serverAddr, channel)
	resp, err := client.Get(url)
	if err != nil {
		log.Fatalf("Failed to fetch messages: %v", err)
	}
	defer resp.Body.Close()

	var messages []protocol.Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}

	if len(messages) == 0 {
		fmt.Printf("No messages in channel '%s'.\n", channel)
		return
	}

	fmt.Printf("💬 Messages in #%s:\n\n", channel)
	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		fmt.Printf("[%s] %s (%s):\n", timestamp, msg.From.Name, msg.From.Type)
		fmt.Printf("  %s\n\n", msg.Content)
	}
}

func watchMessages(client *http.Client, channel string) {
	fmt.Printf("👀 Watching channel #%s for new messages (Ctrl+C to stop)...\n\n", channel)

	lastCheck := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		url := fmt.Sprintf("%s/api/messages?channel=%s&limit=50", *serverAddr, channel)
		resp, err := client.Get(url)
		if err != nil {
			continue
		}

		var messages []protocol.Message
		if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Print only new messages
		for _, msg := range messages {
			if msg.Timestamp.After(lastCheck) {
				timestamp := msg.Timestamp.Format("15:04:05")
				fmt.Printf("[%s] %s (%s):\n", timestamp, msg.From.Name, msg.From.Type)
				fmt.Printf("  %s\n\n", msg.Content)
			}
		}

		lastCheck = time.Now()
	}
}

func sendMessage(client *http.Client, channel, content string) {
	data := map[string]interface{}{
		"channel": channel,
		"content": content,
		"type":    "question",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Failed to marshal data: %v", err)
	}

	resp, err := client.Post(*serverAddr+"/api/send", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server returned error: %d", resp.StatusCode)
	}

	fmt.Printf("✅ Message sent to #%s\n", channel)
	fmt.Println("\nWaiting for agent responses...")

	// Watch for responses for a few seconds
	time.Sleep(1 * time.Second)
	listMessages(client, channel)
}

func createChannel(client *http.Client, name string) {
	data := map[string]interface{}{
		"name":        name,
		"description": fmt.Sprintf("Channel for %s", name),
		"project":     name,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Failed to marshal data: %v", err)
	}

	resp, err := client.Post(*serverAddr+"/api/channels/create", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server returned error: %d", resp.StatusCode)
	}

	fmt.Printf("✅ Channel #%s created successfully\n", name)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// listExports lists all exported agents
func listExports(client *http.Client) {
	resp, err := client.Get(*serverAddr + "/api/exports")
	if err != nil {
		log.Fatalf("Failed to fetch exports: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server returned error: %d", resp.StatusCode)
	}

	var exports []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&exports); err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}

	if len(exports) == 0 {
		fmt.Println("📦 No exported agents found")
		return
	}

	fmt.Println("📦 Exported Agents:")
	for _, export := range exports {
		fmt.Printf("• %s (%s)\n", export["name"], export["type"])
		fmt.Printf("  - Resources: %v\n", export["resourceCount"])
		fmt.Printf("  - Prompts: %v\n", export["promptCount"])
		fmt.Printf("  - Size: %v bytes\n", export["fileSize"])
		if desc, ok := export["description"].(string); ok && desc != "" {
			fmt.Printf("  - Description: %s\n", desc)
		}
		fmt.Println()
	}
}

// exportAgent exports an agent to MCP format
func exportAgent(client *http.Client, agentType, agentName, outputPath string) {
	if agentName == "" {
		log.Fatal("Agent name is required (use --name)")
	}

	data := map[string]interface{}{
		"agent_type":  agentType,
		"agent_name":  agentName,
		"output_path": outputPath,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Failed to marshal data: %v", err)
	}

	resp, err := client.Post(*serverAddr+"/api/export", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to export agent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server returned error: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}

	fmt.Printf("✅ Exported %s agent '%s'\n", agentType, agentName)
	if outputPath != "" {
		fmt.Printf("📁 Saved to: %s\n", outputPath)
	}
	if resources, ok := result["resources"].(float64); ok {
		fmt.Printf("📄 Resources: %.0f\n", resources)
	}
	if prompts, ok := result["prompts"].(float64); ok {
		fmt.Printf("💬 Prompts: %.0f\n", prompts)
	}
	if size, ok := result["size"].(float64); ok {
		fmt.Printf("💾 Size: %.0f bytes\n", size)
	}
}

// importAgent imports an agent from MCP export file
func importAgent(client *http.Client, filePath string) {
	data := map[string]interface{}{
		"file_path": filePath,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Failed to marshal data: %v", err)
	}

	resp, err := client.Post(*serverAddr+"/api/import", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to import agent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server returned error: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}

	fmt.Printf("✅ Imported agent from %s\n", filePath)
	if name, ok := result["name"].(string); ok {
		fmt.Printf("📝 Agent: %s\n", name)
	}
	if agentType, ok := result["type"].(string); ok {
		fmt.Printf("🏷️  Type: %s\n", agentType)
	}
	if resources, ok := result["resources"].(float64); ok {
		fmt.Printf("📄 Resources: %.0f\n", resources)
	}
	if prompts, ok := result["prompts"].(float64); ok {
		fmt.Printf("💬 Prompts: %.0f\n", prompts)
	}
}

// serveMCPResources starts the MCP resource server
func serveMCPResources(port int) {
	fmt.Printf("🚀 Starting MCP Resource Server on port %d...\n", port)
	fmt.Println("📡 Server will expose exported agents via MCP protocol")
	fmt.Println("🔗 Connect Claude Desktop or other MCP clients to this server")
	fmt.Println("⏹️  Press Ctrl+C to stop")

	// This would start the MCP resource server
	// For now, just show the message since the actual server
	// would need to be implemented as a separate service
	fmt.Println("\n⚠️  Note: MCP resource server implementation requires additional setup")
	fmt.Println("   Use the chat commands in the main application instead:")
	fmt.Println("   /export-agent-mcp <name>")
	fmt.Println("   /list-exports")
}
