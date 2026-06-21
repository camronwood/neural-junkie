package aws

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	awsprofiles "github.com/camronwood/neural-junkie/internal/integrations/aws"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AWSMCP provides MCP tools for AWS CLI workflows (SSO profiles).
type AWSMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewAWSMCP creates a new AWS MCP server.
func NewAWSMCP() (*AWSMCP, error) {
	cfg := mcp.GetMCPServerConfig("AWS")
	mcpServer, httpServer, err := mcp.NewMCPServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	a := &AWSMCP{mcpServer: mcpServer, httpServer: httpServer, config: cfg}
	a.registerTools()
	return a, nil
}

func (a *AWSMCP) Start() error {
	return mcp.StartMCPServer(a.httpServer, a.config.Port)
}

func (a *AWSMCP) GetMCPServer() *server.MCPServer {
	return a.mcpServer
}

func awsSettings() config.AWSConfig {
	if cfg := mcp.AppConfig(); cfg != nil {
		return cfg.AWSSettings()
	}
	return config.AWSConfig{}
}

func (a *AWSMCP) registerTools() {
	a.mcpServer.AddTool(mcp.CreateTool(
		"aws_get_caller_identity",
		"Return AWS account/ARN for the configured SSO profile (sts get-caller-identity).",
		mcp.CreateStringInputSchema("profile", "Optional profile override (must be allowed)"),
		nil,
	), a.handleGetCallerIdentity)

	a.mcpServer.AddTool(mcp.CreateTool(
		"aws_list_profiles",
		"List named AWS profiles from ~/.aws/config.",
		mcp.CreateEmptyInputSchema(),
		nil,
	), a.handleListProfiles)

	a.mcpServer.AddTool(mcp.CreateTool(
		"aws_sso_login_hint",
		"Return the aws sso login command for the active or requested profile.",
		mcp.CreateStringInputSchema("profile", "Optional profile name"),
		nil,
	), a.handleSSOLoginHint)

	a.mcpServer.AddTool(mcp.CreateTool(
		"aws_cli_query",
		"Run a read-only AWS CLI query (ec2 describe-*, s3 ls, lambda list-*, iam list-*, cloudformation describe-*). Args are service and operation plus optional flags.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"service": map[string]interface{}{"type": "string", "description": "AWS service, e.g. ec2, s3, lambda"},
			"operation": map[string]interface{}{"type": "string", "description": "CLI operation, e.g. describe-instances"},
			"extra_args": map[string]interface{}{
				"type":        "array",
				"description": "Additional CLI args after operation",
				"items":       map[string]interface{}{"type": "string"},
			},
		}, []string{"service", "operation"}),
		nil,
	), a.handleCLIQuery)

	log.Printf("Registered %d AWS MCP tools", len(a.mcpServer.ListTools()))
}

func (a *AWSMCP) handleGetCallerIdentity(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	settings := awsSettings()
	if p := strings.TrimSpace(request.GetString("profile", "")); p != "" {
		settings.Profile = p
	}
	out, err := awsprofiles.TestCallerIdentity(settings)
	if err != nil {
		return mcp.HandleToolError(err, "aws_get_caller_identity"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (a *AWSMCP) handleListProfiles(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	profiles, err := awsprofiles.ListProfiles()
	if err != nil {
		return mcp.HandleToolError(err, "aws_list_profiles"), nil
	}
	if len(profiles) == 0 {
		return mcp.HandleToolSuccess("No profiles found in ~/.aws/config"), nil
	}
	return mcp.HandleToolSuccess(strings.Join(profiles, "\n")), nil
}

func (a *AWSMCP) handleSSOLoginHint(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	settings := awsSettings()
	profile := settings.ProfileOrDefault()
	if p := strings.TrimSpace(request.GetString("profile", "")); p != "" {
		profile = p
	}
	if profile == "" {
		return mcp.HandleToolError(fmt.Errorf("configure aws profile in Settings → Integrations"), "aws_sso_login_hint"), nil
	}
	cmd := fmt.Sprintf("aws sso login --profile %s", profile)
	msg := fmt.Sprintf("Run in your terminal:\n\n%s\n\nThen retry AWS tools.", cmd)
	if u := strings.TrimSpace(settings.SSOStartURL); u != "" {
		msg += fmt.Sprintf("\nSSO start URL (reference): %s", u)
	}
	return mcp.HandleToolSuccess(msg), nil
}

func (a *AWSMCP) handleCLIQuery(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	service := strings.TrimSpace(request.GetString("service", ""))
	op := strings.TrimSpace(request.GetString("operation", ""))
	if service == "" || op == "" {
		return mcp.HandleToolError(fmt.Errorf("service and operation required"), "aws_cli_query"), nil
	}
	args := []string{service, op}
	if raw, ok := request.GetArguments()["extra_args"]; ok {
		if arr, ok := raw.([]interface{}); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					args = append(args, s)
				}
			}
		}
	}
	out, err := awsprofiles.RunAWS(ctx, awsSettings(), args...)
	if err != nil {
		return mcp.HandleToolError(err, "aws_cli_query"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}
