package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/awssidecar"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type sidecarTools struct{}

func attachSidecarTools(mcpServer *server.MCPServer) {
	if mcpServer == nil {
		return
	}
	t := &sidecarTools{}
	t.register(mcpServer)
}

func (t *sidecarTools) client() *awssidecar.Client {
	return awssidecar.DefaultSidecarClient
}

func (t *sidecarTools) register(mcpServer *server.MCPServer) {
	pageSchema := func(extra map[string]interface{}, required []string) mcpgo.ToolInputSchema {
		props := map[string]interface{}{
			"page_size": map[string]interface{}{"type": "number", "description": "Page size (default 20, max 100)"},
			"next_token": map[string]interface{}{"type": "string", "description": "Pagination token from prior response"},
		}
		for k, v := range extra {
			props[k] = v
		}
		return mcp.CreateObjectInputSchema(props, required)
	}

	mcpServer.AddTool(mcp.CreateTool(
		"describe_ec2_instances",
		"Describe EC2 instances (typed boto3, paginated). Prefer over aws_cli_query.",
		pageSchema(map[string]interface{}{
			"instance_ids": map[string]interface{}{
				"type": "array", "items": map[string]interface{}{"type": "string"},
				"description": "Optional instance IDs",
			},
		}, nil),
		nil,
	), t.handleDescribeEC2)

	mcpServer.AddTool(mcp.CreateTool(
		"list_s3_buckets",
		"List S3 buckets (typed boto3, paginated).",
		pageSchema(nil, nil),
		nil,
	), t.handleListS3)

	mcpServer.AddTool(mcp.CreateTool(
		"get_lambda_config",
		"Get Lambda function configuration and code location.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"function_name": map[string]interface{}{"type": "string", "description": "Lambda function name or ARN"},
			"qualifier":     map[string]interface{}{"type": "string", "description": "Optional version or alias"},
		}, []string{"function_name"}),
		nil,
	), t.handleGetLambda)

	mcpServer.AddTool(mcp.CreateTool(
		"list_lambda_functions",
		"List Lambda functions (paginated).",
		pageSchema(nil, nil),
		nil,
	), t.handleListLambda)

	mcpServer.AddTool(mcp.CreateTool(
		"describe_iam_role",
		"Get IAM role details and trust policy.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"role_name": map[string]interface{}{"type": "string", "description": "IAM role name"},
		}, []string{"role_name"}),
		nil,
	), t.handleDescribeIAMRole)

	mcpServer.AddTool(mcp.CreateTool(
		"describe_cloudformation_stack",
		"Describe a CloudFormation stack (status, outputs, parameters).",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"stack_name": map[string]interface{}{"type": "string", "description": "Stack name"},
		}, []string{"stack_name"}),
		nil,
	), t.handleDescribeCFNStack)

	mcpServer.AddTool(mcp.CreateTool(
		"scan_iac_workspace",
		"Scan workspace for Terraform, CloudFormation, and CDK declared resources.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"workspace_root": map[string]interface{}{"type": "string", "description": "Optional workspace root path"},
		}, nil),
		nil,
	), t.handleScanIaC)

	mcpServer.AddTool(mcp.CreateTool(
		"correlate_iac_resource",
		"Compare declared IaC resource vs live AWS state (read-only drift check).",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"resource_type": map[string]interface{}{"type": "string", "description": "Resource type e.g. aws_instance"},
			"id_or_arn":     map[string]interface{}{"type": "string", "description": "Physical ID or ARN"},
			"declared":      map[string]interface{}{"description": "Optional declared resource snapshot"},
		}, []string{"resource_type", "id_or_arn"}),
		nil,
	), t.handleCorrelateIaC)

	mcpServer.AddTool(mcp.CreateTool(
		"get_cost_summary",
		"Cost Explorer summary grouped by dimension (requires ce:GetCostAndUsage).",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"start_date":  map[string]interface{}{"type": "string", "description": "Start date YYYY-MM-DD"},
			"end_date":    map[string]interface{}{"type": "string", "description": "End date YYYY-MM-DD"},
			"group_by":    map[string]interface{}{"type": "string", "description": "SERVICE or LINKED_ACCOUNT"},
			"granularity": map[string]interface{}{"type": "string", "description": "DAILY or MONTHLY"},
		}, nil),
		nil,
	), t.handleCostSummary)

	mcpServer.AddTool(mcp.CreateTool(
		"list_security_hub_findings",
		"List Security Hub findings (paginated, optional severity filter).",
		pageSchema(map[string]interface{}{
			"severity": map[string]interface{}{"type": "string", "description": "CRITICAL, HIGH, MEDIUM, LOW"},
		}, nil),
		nil,
	), t.handleSecurityHub)

	mcpServer.AddTool(mcp.CreateTool(
		"list_guardduty_findings",
		"List GuardDuty findings in the configured region.",
		pageSchema(nil, nil),
		nil,
	), t.handleGuardDuty)

	mcpServer.AddTool(mcp.CreateTool(
		"analyze_iam_policy",
		"Analyze IAM policy document or role for allowed/denied actions.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"role_name":       map[string]interface{}{"type": "string", "description": "IAM role name"},
			"policy_document": map[string]interface{}{"description": "Optional policy JSON"},
			"action_names": map[string]interface{}{
				"type": "array", "items": map[string]interface{}{"type": "string"},
				"description": "Actions to evaluate",
			},
		}, nil),
		nil,
	), t.handleAnalyzeIAMPolicy)

	mcpServer.AddTool(mcp.CreateTool(
		"list_organization_accounts",
		"List AWS Organization accounts (filtered by allowed_accounts if set).",
		pageSchema(nil, nil),
		nil,
	), t.handleListOrgAccounts)

	mcpServer.AddTool(mcp.CreateTool(
		"assume_account_context",
		"Validate access to a member account in the organization allowlist.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"account_id": map[string]interface{}{"type": "string", "description": "12-digit AWS account ID"},
		}, []string{"account_id"}),
		nil,
	), t.handleAssumeAccount)

	mcpServer.AddTool(mcp.CreateTool(
		"ec2_stop_instance",
		"Stop an EC2 instance (requires write_enabled and confirm_token from user approval).",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"instance_id":   map[string]interface{}{"type": "string", "description": "EC2 instance ID"},
			"confirm_token": map[string]interface{}{"type": "string", "description": "User approval string from chat"},
		}, []string{"instance_id", "confirm_token"}),
		nil,
	), t.handleEC2Stop)

	mcpServer.AddTool(mcp.CreateTool(
		"lambda_update_function_configuration",
		"Update Lambda configuration (requires write_enabled and confirm_token).",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"function_name": map[string]interface{}{"type": "string", "description": "Lambda function name"},
			"timeout":       map[string]interface{}{"type": "number", "description": "Optional timeout seconds"},
			"memory_size":   map[string]interface{}{"type": "number", "description": "Optional memory MB"},
			"environment":   map[string]interface{}{"description": "Optional {Variables: {...}}"},
			"confirm_token": map[string]interface{}{"type": "string", "description": "User approval string from chat"},
		}, []string{"function_name", "confirm_token"}),
		nil,
	), t.handleLambdaUpdate)
}

func (t *sidecarTools) sidecarPost(ctx context.Context, path string, body map[string]any) (*mcpgo.CallToolResult, error) {
	client := t.client()
	if client == nil {
		return mcp.HandleToolError(fmt.Errorf("aws sidecar not configured"), path), nil
	}
	out, err := client.PostJSON(ctx, path, body)
	if err != nil {
		return mcp.HandleToolError(err, path), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (t *sidecarTools) bodyFromRequest(request mcpgo.CallToolRequest) map[string]any {
	body := map[string]any{}
	args := request.GetArguments()
	for k, v := range args {
		if v != nil {
			body[k] = v
		}
	}
	return body
}

func (t *sidecarTools) handleDescribeEC2(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/describe-ec2-instances", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleListS3(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/list-s3-buckets", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleGetLambda(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/get-lambda-config", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleListLambda(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/list-lambda-functions", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleDescribeIAMRole(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/describe-iam-role", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleDescribeCFNStack(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/describe-cloudformation-stack", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleScanIaC(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/scan-iac-workspace", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleCorrelateIaC(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/correlate-iac-resource", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleCostSummary(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/get-cost-summary", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleSecurityHub(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/list-security-hub-findings", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleGuardDuty(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/list-guardduty-findings", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleAnalyzeIAMPolicy(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/analyze-iam-policy", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleListOrgAccounts(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/list-organization-accounts", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleAssumeAccount(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/assume-account-context", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleEC2Stop(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/ec2-stop-instance", t.bodyFromRequest(request))
}

func (t *sidecarTools) handleLambdaUpdate(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return t.sidecarPost(ctx, "/api/aws/lambda-update-function-configuration", t.bodyFromRequest(request))
}

// sidecarCallerIdentity uses sidecar when available, else falls back to CLI.
func sidecarCallerIdentity(ctx context.Context) (string, error) {
	client := awssidecar.DefaultSidecarClient
	if client == nil && awssidecar.SidecarBaseURL != nil {
		client = awssidecar.NewSidecarClient(awssidecar.SidecarBaseURL)
	}
	if client != nil {
		out, err := client.PostJSON(ctx, "/api/aws/get-caller-identity", nil)
		if err == nil {
			return out, nil
		}
		if !strings.Contains(err.Error(), "not running") {
			return "", err
		}
	}
	return "", fmt.Errorf("aws sidecar unavailable")
}
