package packremote

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/camronwood/neural-junkie/internal/mcp"
)

// RemoteMCP proxies domain tools from a pack mcp-sidecar HTTP endpoint into an in-process MCP server.
type RemoteMCP struct {
	agentType  string
	mcpServer  *server.MCPServer
	httpClient *mcpclient.Client
	baseURL    string
	mu         sync.Mutex
	started    bool
}

// NewRemoteMCP connects to the pack sidecar MCP listener for agentType.
func NewRemoteMCP(agentType string) (*RemoteMCP, error) {
	key := mcp.NormalizeAgentType(agentType)
	port := mcp.DefaultPort(key)
	if port == 0 {
		return nil, fmt.Errorf("no MCP port for agent type %q", agentType)
	}
	name := fmt.Sprintf("%s-agent-mcp", strings.ToLower(strings.ReplaceAll(key, "_", "-")))
	srv := server.NewMCPServer(name, "1.0.0")
	return &RemoteMCP{
		agentType: agentType,
		mcpServer: srv,
		baseURL:   fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
	}, nil
}

// Start initializes the HTTP MCP client and registers proxy tool handlers.
func (r *RemoteMCP) Start() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mcpclient.NewStreamableHttpClient(r.baseURL)
	if err != nil {
		return fmt.Errorf("mcp client %s: %w", r.baseURL, err)
	}
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("mcp client start %s: %w", r.baseURL, err)
	}
	initResult, err := client.Initialize(ctx, mcpgo.InitializeRequest{
		Params: mcpgo.InitializeParams{
			ProtocolVersion: mcpgo.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcpgo.Implementation{
				Name:    "neural-junkie-hub",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("mcp initialize %s: %w", r.baseURL, err)
	}
	_ = initResult
	toolsResult, err := client.ListTools(ctx, mcpgo.ListToolsRequest{})
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("mcp list tools %s: %w", r.baseURL, err)
	}
	for _, tool := range toolsResult.Tools {
		tool := tool
		handler := r.proxyHandler(client, tool.Name)
		r.mcpServer.AddTool(tool, handler)
	}
	r.httpClient = client
	r.started = true
	log.Printf("Pack remote MCP connected for %s at %s (%d tools)", r.agentType, r.baseURL, len(toolsResult.Tools))
	return nil
}

func (r *RemoteMCP) proxyHandler(client *mcpclient.Client, name string) func(context.Context, mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := req.Params.Arguments
		if args == nil {
			args = map[string]any{}
		}
		return client.CallTool(ctx, mcpgo.CallToolRequest{
			Params: mcpgo.CallToolParams{
				Name:      name,
				Arguments: args,
			},
		})
	}
}

// GetMCPServer returns the in-process MCP server with proxy handlers.
func (r *RemoteMCP) GetMCPServer() *server.MCPServer {
	if r == nil {
		return nil
	}
	return r.mcpServer
}

// SidecarHealthOK reports whether the pack MCP sidecar health endpoint responds.
func SidecarHealthOK(healthURL string) bool {
	healthURL = strings.TrimRight(strings.TrimSpace(healthURL), "/") + "/health"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// MCPAgentsJSON parses NJ_MCP_AGENTS_JSON env payload.
func MCPAgentsJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}
