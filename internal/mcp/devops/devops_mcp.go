package devops

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var k8sNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

var allowedK8sResources = map[string]bool{
	"pods": true, "pod": true, "services": true, "service": true, "svc": true,
	"deployments": true, "deployment": true, "deploy": true,
	"namespaces": true, "namespace": true, "ns": true,
	"nodes": true, "node": true, "configmaps": true, "configmap": true,
	"secrets": true, "secret": true, "ingresses": true, "ingress": true,
	"statefulsets": true, "statefulset": true, "daemonsets": true, "daemonset": true,
	"jobs": true, "job": true, "cronjobs": true, "cronjob": true,
}

func isValidK8sName(name string) bool {
	if name == "" || len(name) > 253 {
		return false
	}
	return k8sNamePattern.MatchString(name)
}

func isAllowedK8sResource(resource string) bool {
	return allowedK8sResources[strings.ToLower(strings.TrimSpace(resource))]
}

func isSafeYamlPath(path string) bool {
	if path == "" || strings.Contains(path, "..") {
		return false
	}
	clean := filepath.Clean(path)
	return clean == path || !strings.HasPrefix(clean, "..")
}

// DevOpsMCP provides MCP tools for DevOps operations
type DevOpsMCP struct {
	mcpServer   *server.MCPServer
	httpServer  *server.StreamableHTTPServer
	config      *mcp.MCPServerConfig
}

// NewDevOpsMCP creates a new DevOps MCP server
func NewDevOpsMCP() (*DevOpsMCP, error) {
	config := mcp.GetMCPServerConfig("DEVOPS")

	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	d := &DevOpsMCP{
		mcpServer:   mcpServer,
		httpServer:  httpServer,
		config:      config,
	}

	d.registerTools()

	return d, nil
}

// Start starts the DevOps MCP server
func (d *DevOpsMCP) Start() error {
	return mcp.StartMCPServer(d.httpServer, d.config.Port)
}

// GetMCPServer returns the underlying MCP server
func (d *DevOpsMCP) GetMCPServer() *server.MCPServer {
	return d.mcpServer
}

// registerTools registers all DevOps MCP tools
func (d *DevOpsMCP) registerTools() {
	// Tool 1: kubectl_query
	d.mcpServer.AddTool(mcp.CreateTool(
		"kubectl_query",
		"Query Kubernetes cluster using kubectl",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"resource":  "Kubernetes resource type (e.g., pods, services, deployments)",
			"namespace": "Kubernetes namespace (optional, defaults to current context)",
		}),
		nil,
	), d.handleKubectlQuery)

	// Tool 3: check_docker_image
	d.mcpServer.AddTool(mcp.CreateTool(
		"check_docker_image",
		"Analyze Docker image for size, layers, and vulnerabilities",
		mcp.CreateStringInputSchema("image_name", "Docker image name to analyze"),
		nil,
	), d.handleCheckDockerImage)

	// Tool 4: validate_yaml
	d.mcpServer.AddTool(mcp.CreateTool(
		"validate_yaml",
		"Validate Kubernetes or Helm YAML files for syntax and best practices",
		mcp.CreateStringInputSchema("yaml_file", "Path to YAML file to validate"),
		nil,
	), d.handleValidateYaml)

	// Tool 5: check_pod_logs
	d.mcpServer.AddTool(mcp.CreateTool(
		"check_pod_logs",
		"Fetch and analyze logs from Kubernetes pods",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"pod_name":  "Name of the pod to check logs for",
			"namespace": "Kubernetes namespace (optional)",
		}),
		nil,
	), d.handleCheckPodLogs)

	// Tool 6: query_prometheus
	d.mcpServer.AddTool(mcp.CreateTool(
		"query_prometheus",
		"Query Prometheus metrics for monitoring data",
		mcp.CreateStringInputSchema("query", "Prometheus query to execute"),
		nil,
	), d.handleQueryPrometheus)

	log.Printf("Registered %d DevOps MCP tools", len(d.mcpServer.ListTools()))
}

// handleKubectlQuery queries Kubernetes cluster
func (d *DevOpsMCP) handleKubectlQuery(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"resource"}); err != nil {
		return mcp.HandleToolError(err, "kubectl_query"), nil
	}

	resource := request.GetString("resource", "")
	namespace := request.GetString("namespace", "")

	if !isAllowedK8sResource(resource) {
		return mcp.HandleToolError(fmt.Errorf("unsupported kubernetes resource: %s", resource), "kubectl_query"), nil
	}
	if namespace != "" && !isValidK8sName(namespace) {
		return mcp.HandleToolError(fmt.Errorf("invalid namespace: %s", namespace), "kubectl_query"), nil
	}

	// Build kubectl command
	cmd := exec.CommandContext(ctx, "kubectl", "get", resource)
	if namespace != "" {
		cmd.Args = append(cmd.Args, "-n", namespace)
	}
	cmd.Args = append(cmd.Args, "-o", "wide")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("kubectl query failed: %w\nOutput: %s", err, string(output)), "kubectl_query"), nil
	}

	result := fmt.Sprintf("Kubernetes %s in namespace %s:\n%s", resource, namespace, string(output))
	return mcp.HandleToolSuccess(result), nil
}

// handleCheckDockerImage analyzes Docker image
func (d *DevOpsMCP) handleCheckDockerImage(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"image_name"}); err != nil {
		return mcp.HandleToolError(err, "check_docker_image"), nil
	}

	imageName := request.GetString("image_name", "")
	if imageName == "" {
		return mcp.HandleToolError(fmt.Errorf("empty image name"), "check_docker_image"), nil
	}
	if strings.HasPrefix(imageName, "-") {
		return mcp.HandleToolError(fmt.Errorf("invalid image name"), "check_docker_image"), nil
	}

	var results []string

	// Get image information
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("Failed to inspect image %s: %v", imageName, err))
	} else {
		results = append(results, fmt.Sprintf("=== Image Information for %s ===", imageName))
		results = append(results, string(output))
	}

	// Get image size
	cmd = exec.CommandContext(ctx, "docker", "images", "--format", "table {{.Repository}}\t{{.Tag}}\t{{.Size}}", imageName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("Failed to get image size: %v", err))
	} else {
		results = append(results, "\n=== Image Size ===")
		results = append(results, string(output))
	}

	// Check for vulnerabilities (if trivy is available)
	cmd = exec.CommandContext(ctx, "trivy", "image", imageName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("\nTrivy scan not available: %v", err))
	} else {
		results = append(results, "\n=== Security Scan ===")
		results = append(results, string(output))
	}

	result := strings.Join(results, "\n")
	return mcp.HandleToolSuccess(result), nil
}

// handleValidateYaml validates YAML files
func (d *DevOpsMCP) handleValidateYaml(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"yaml_file"}); err != nil {
		return mcp.HandleToolError(err, "validate_yaml"), nil
	}

	yamlFile := request.GetString("yaml_file", "")
	if !isSafeYamlPath(yamlFile) || !d.isValidFilePath(yamlFile) {
		return mcp.HandleToolError(fmt.Errorf("invalid file path: %s", yamlFile), "validate_yaml"), nil
	}

	var results []string

	// Check if file exists
	if _, err := os.Stat(yamlFile); os.IsNotExist(err) {
		return mcp.HandleToolError(fmt.Errorf("file does not exist: %s", yamlFile), "validate_yaml"), nil
	}

	// Validate with kubectl if it's a Kubernetes YAML
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "--dry-run=client", "-f", "--", yamlFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("Kubernetes validation failed: %v", err))
		results = append(results, string(output))
	} else {
		results = append(results, "=== Kubernetes Validation ===")
		results = append(results, "✓ YAML is valid Kubernetes configuration")
		results = append(results, string(output))
	}

	// Check YAML syntax with yamllint if available
	cmd = exec.CommandContext(ctx, "yamllint", yamlFile)
	output, err = cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("\nYAML linting not available: %v", err))
	} else {
		results = append(results, "\n=== YAML Linting ===")
		results = append(results, string(output))
	}

	result := strings.Join(results, "\n")
	return mcp.HandleToolSuccess(result), nil
}

// handleCheckPodLogs fetches pod logs
func (d *DevOpsMCP) handleCheckPodLogs(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"pod_name"}); err != nil {
		return mcp.HandleToolError(err, "check_pod_logs"), nil
	}

	podName := request.GetString("pod_name", "")
	namespace := request.GetString("namespace", "")

	if !isValidK8sName(podName) {
		return mcp.HandleToolError(fmt.Errorf("invalid pod name: %s", podName), "check_pod_logs"), nil
	}
	if namespace != "" && !isValidK8sName(namespace) {
		return mcp.HandleToolError(fmt.Errorf("invalid namespace: %s", namespace), "check_pod_logs"), nil
	}

	// Build kubectl logs command (-- before pod name prevents flag injection)
	cmd := exec.CommandContext(ctx, "kubectl", "logs", "--tail", "100")
	if namespace != "" {
		cmd.Args = append(cmd.Args, "-n", namespace)
	}
	cmd.Args = append(cmd.Args, "--", podName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to fetch pod logs: %w\nOutput: %s", err, string(output)), "check_pod_logs"), nil
	}

	result := fmt.Sprintf("Logs for pod %s in namespace %s:\n%s", podName, namespace, string(output))
	return mcp.HandleToolSuccess(result), nil
}

// handleQueryPrometheus queries Prometheus metrics
func (d *DevOpsMCP) handleQueryPrometheus(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"query"}); err != nil {
		return mcp.HandleToolError(err, "query_prometheus"), nil
	}

	query := request.GetString("query", "")
	if query == "" {
		return mcp.HandleToolError(fmt.Errorf("empty query"), "query_prometheus"), nil
	}

	// Get Prometheus endpoint from environment
	prometheusURL := os.Getenv("PROMETHEUS_URL")
	if prometheusURL == "" {
		prometheusURL = "http://localhost:9090" // Default Prometheus URL
	}

	// This is a simplified implementation - in practice, you'd use the Prometheus Go client
	result := fmt.Sprintf("Prometheus Query: %s\n", query)
	result += fmt.Sprintf("Prometheus URL: %s\n", prometheusURL)
	result += "Note: This is a placeholder implementation. Full Prometheus integration requires:\n"
	result += "1. Prometheus Go client library\n"
	result += "2. Authentication configuration\n"
	result += "3. Query execution and result parsing\n"
	result += "4. Time range and aggregation options"

	return mcp.HandleToolSuccess(result), nil
}

// Helper methods

func (d *DevOpsMCP) isValidFilePath(path string) bool {
	if path == "" {
		return false
	}

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}
