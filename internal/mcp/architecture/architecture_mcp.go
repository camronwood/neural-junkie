package architecture

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ArchitectureMCP provides read-only diagnostic MCP tools for architecture review.
type ArchitectureMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewArchitectureMCP creates a new Architecture MCP server.
func NewArchitectureMCP() (*ArchitectureMCP, error) {
	config := mcp.GetMCPServerConfig("architecture")
	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	a := &ArchitectureMCP{mcpServer: mcpServer, httpServer: httpServer, config: config}
	a.registerTools()
	return a, nil
}

func (a *ArchitectureMCP) Start() error {
	if a.httpServer == nil {
		return fmt.Errorf("MCP server not configured")
	}
	return mcp.StartMCPServer(a.httpServer, a.config.Port)
}

func (a *ArchitectureMCP) GetMCPServer() *server.MCPServer {
	return a.mcpServer
}

func (a *ArchitectureMCP) registerTools() {
	a.mcpServer.AddTool(mcp.CreateTool(
		"validate_yaml",
		"Validate Kubernetes or Helm YAML for architecture review (read-only)",
		mcp.CreateStringInputSchema("yaml_file", "Path to YAML file"),
		nil,
	), a.handleValidateYaml)

	a.mcpServer.AddTool(mcp.CreateTool(
		"validate_schema",
		"Review SQL schema DDL file structure (read-only, no DB connection)",
		mcp.CreateStringInputSchema("schema_file", "Path to SQL schema file"),
		nil,
	), a.handleValidateSchema)

	a.mcpServer.AddTool(mcp.CreateTool(
		"explain_query",
		"Review SQL query structure (EXPLAIN requires DATABASE_URL on DatabaseSpecialist)",
		mcp.CreateStringInputSchema("sql_query", "SQL query to review"),
		nil,
	), a.handleExplainQuery)

	a.mcpServer.AddTool(mcp.CreateTool(
		"check_dependencies",
		"Review Go module dependencies for architecture impact",
		mcp.CreateStringInputSchema("module_path", "Path to Go module"),
		nil,
	), a.handleCheckDependencies)

	log.Printf("Registered %d Architecture MCP tools", len(a.mcpServer.ListTools()))
}

func (a *ArchitectureMCP) handleValidateYaml(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	yamlFile := request.GetString("yaml_file", "")
	if yamlFile == "" || !shared.PathExists(yamlFile) {
		return mcp.HandleToolError(fmt.Errorf("yaml_file not found"), "validate_yaml"), nil
	}
	out, err := shared.RunCommand(ctx, "", "kubectl", "apply", "--dry-run=client", "-f", yamlFile)
	return mcp.HandleToolSuccess(shared.FormatCommandResult("kubectl dry-run:", out, err)), nil
}

func (a *ArchitectureMCP) handleValidateSchema(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	schemaFile := request.GetString("schema_file", "")
	data, err := os.ReadFile(schemaFile)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("schema_file not readable: %w", err), "validate_schema"), nil
	}
	content := strings.ToLower(string(data))
	var notes []string
	for _, kw := range []string{"create table", "alter table", "foreign key", "index", "primary key"} {
		if strings.Contains(content, kw) {
			notes = append(notes, "Found: "+kw)
		}
	}
	if len(notes) == 0 {
		return mcp.HandleToolSuccess("No common DDL keywords found; verify this is a schema file."), nil
	}
	return mcp.HandleToolSuccess("Schema review:\n- " + strings.Join(notes, "\n- ")), nil
}

func (a *ArchitectureMCP) handleExplainQuery(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	query := strings.TrimSpace(request.GetString("sql_query", ""))
	if query == "" {
		return mcp.HandleToolError(fmt.Errorf("sql_query required"), "explain_query"), nil
	}
	q := strings.ToUpper(query)
	var notes []string
	if strings.Contains(q, "SELECT *") {
		notes = append(notes, "Uses SELECT * — consider explicit columns")
	}
	if !strings.Contains(q, "WHERE") && strings.HasPrefix(q, "SELECT") {
		notes = append(notes, "SELECT without WHERE may scan full table")
	}
	if len(notes) == 0 {
		notes = append(notes, "No obvious structural issues; use DatabaseSpecialist explain_query for EXPLAIN ANALYZE")
	}
	return mcp.HandleToolSuccess("Query architecture review:\n- " + strings.Join(notes, "\n- ")), nil
}

func (a *ArchitectureMCP) handleCheckDependencies(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root := shared.FindProjectRoot(request.GetString("module_path", "."), "go.mod")
	out, err := shared.RunCommand(ctx, root, "go", "list", "-m", "all")
	return mcp.HandleToolSuccess(shared.FormatCommandResult("go list -m all:", out, err)), nil
}
