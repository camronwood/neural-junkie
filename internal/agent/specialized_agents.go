package agent

import (
	"log"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/contextcompress"
	"github.com/camronwood/neural-junkie/internal/mcp/architecture"
	"github.com/camronwood/neural-junkie/internal/mcp/backend"
	"github.com/camronwood/neural-junkie/internal/mcp/biology"
	"github.com/camronwood/neural-junkie/internal/mcp/cad"
	"github.com/camronwood/neural-junkie/internal/mcp/aws"
	"github.com/camronwood/neural-junkie/internal/mcp/incident"
	"github.com/camronwood/neural-junkie/internal/mcp/browser"
	"github.com/camronwood/neural-junkie/internal/mcp/codereview"
	"github.com/camronwood/neural-junkie/internal/mcp/database"
	"github.com/camronwood/neural-junkie/internal/mcp/devops"
	"github.com/camronwood/neural-junkie/internal/mcp/frontend"
	"github.com/camronwood/neural-junkie/internal/mcp/rust"
	"github.com/camronwood/neural-junkie/internal/mcp/security"
	"github.com/camronwood/neural-junkie/internal/mcp/workspace"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func attachWorkspaceTools(agent *Agent, srv MCPServerInterface) {
	if agent == nil || srv == nil {
		return
	}
	workspace.AttachTools(srv.GetMCPServer(), func() string {
		return agent.WorkspacePath
	})
}

func attachContextCompressTools(srv MCPServerInterface) {
	if srv == nil {
		return
	}
	contextcompress.AttachRetrieveTool(srv.GetMCPServer(), contextcompress.DefaultStore())
}

func startAgentMCP(agent *Agent, label string, srv MCPServerInterface) {
	startAgentMCPWithOptions(agent, label, srv, true)
}

// startDomainAgentMCP starts a domain-pack MCP server without workspace file tools.
// Domain tools take explicit paths from chat/workspace context; listing the repo on
// every turn causes spurious list_dir failures when no folder is open.
func startDomainAgentMCP(agent *Agent, label string, srv MCPServerInterface) {
	startAgentMCPWithOptions(agent, label, srv, false)
}

func startAgentMCPWithOptions(agent *Agent, label string, srv MCPServerInterface, attachWorkspace bool) {
	if agent == nil || srv == nil {
		return
	}
	agent.MCPServer = srv
	if attachWorkspace {
		attachWorkspaceTools(agent, srv)
	}
	attachContextCompressTools(srv)
	if err := srv.Start(); err != nil {
		log.Printf("Failed to start %s MCP server: %v", label, err)
	} else {
		log.Printf("%s MCP server started for agent: %s", label, agent.Info.Name)
	}
}

// NewFrontendAgent creates a frontend development agent
func NewFrontendAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Web UI", "Desktop UI", "TypeScript",
		"JavaScript", "CSS", "HTML",
		"UI/UX Design", "Accessibility",
		"Performance Optimization", "State Management",
		"Design Systems", "Component Architecture",
		"Visual QA", "Responsive Design",
	}

	agent := NewAgent(protocol.AgentTypeFrontend, name, expertise, ai, hub)
	agent.SupportsVision = true
	agent.Info.SupportsVision = true

	if frontendMCP, err := frontend.NewFrontendMCP(); err != nil {
		log.Printf("Failed to create Frontend MCP server: %v", err)
	} else {
		startAgentMCP(agent, "Frontend", frontendMCP)
	}

	return agent
}

// NewBackendAgent creates a backend development agent
func NewBackendAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"APIs", "Services", "Data Modeling",
		"REST", "GraphQL", "gRPC",
		"Microservices", "Event-Driven Architecture",
		"Business Logic", "API Design",
		"Performance", "Caching",
		"Message Queues", "Integration Patterns",
	}

	agent := NewAgent(protocol.AgentTypeBackend, name, expertise, ai, hub)

	backendMCP, err := backend.NewBackendMCP()
	if err != nil {
		log.Printf("Failed to create Backend MCP server: %v", err)
	} else {
		startAgentMCP(agent, "Backend", backendMCP)
	}

	return agent
}

// NewDevOpsAgent creates a DevOps agent
func NewDevOpsAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Docker", "Kubernetes", "Helm",
		"CI/CD", "GitHub Actions", "Jenkins",
		"AWS", "GCP", "Azure",
		"Terraform", "Infrastructure as Code",
		"Monitoring", "Logging", "Prometheus",
		"Load Balancing", "Nginx",
		"kubectl", "cluster context",
		"secrets management",
		"deployment", "environment management",
	}

	agent := NewAgent(protocol.AgentTypeDevOps, name, expertise, ai, hub)

	devopsMCP, err := devops.NewDevOpsMCP()
	if err != nil {
		log.Printf("Failed to create DevOps MCP server: %v", err)
	} else {
		startAgentMCP(agent, "DevOps", devopsMCP)
	}

	return agent
}

// NewDatabaseAgent creates a database agent
func NewDatabaseAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"PostgreSQL", "MySQL", "MongoDB",
		"Redis", "Elasticsearch",
		"Schema Design", "Query Optimization",
		"Indexing", "Database Migrations",
		"Transactions", "ACID", "CAP Theorem",
		"Replication", "Sharding",
	}

	agent := NewAgent(protocol.AgentTypeDatabase, name, expertise, ai, hub)

	databaseMCP, err := database.NewDatabaseMCP()
	if err != nil {
		log.Printf("Failed to create Database MCP server: %v", err)
	} else {
		startAgentMCP(agent, "Database", databaseMCP)
	}

	return agent
}

// NewSecurityAgent creates a security agent
func NewSecurityAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Authentication", "Authorization", "OAuth",
		"JWT", "Session Management",
		"Encryption", "HTTPS", "TLS",
		"XSS Prevention", "CSRF Protection",
		"SQL Injection", "Input Validation",
		"Security Best Practices", "OWASP",
		"Compliance", "GDPR", "SOC2",
	}

	agent := NewAgent(protocol.AgentTypeSecurity, name, expertise, ai, hub)

	if securityMCP, err := security.NewSecurityMCP(); err != nil {
		log.Printf("Failed to create Security MCP server: %v", err)
	} else {
		startAgentMCP(agent, "Security", securityMCP)
	}

	return agent
}

// NewRustAgent creates a Rust development agent
func NewRustAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Rust", "Cargo", "Tokio",
		"Ownership", "Borrowing", "Lifetimes",
		"Traits", "Generics", "Enums",
		"Error Handling", "Result", "Option",
		"Async/Await", "Futures",
		"Unsafe Rust", "FFI",
		"WASM", "WebAssembly",
		"Serde", "Serialization",
		"Concurrency", "Send", "Sync",
		"Macros", "Procedural Macros",
		"no_std", "Embedded",
		"Performance", "Zero-Cost Abstractions",
	}

	agent := NewAgent(protocol.AgentTypeRust, name, expertise, ai, hub)

	if rustMCP, err := rust.NewRustMCP(); err != nil {
		log.Printf("Failed to create Rust MCP server: %v", err)
	} else {
		startAgentMCP(agent, "Rust", rustMCP)
	}

	return agent
}

// NewArchitectureAgent creates a broad software architecture agent.
func NewArchitectureAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"System Design", "Architecture Review", "Service Boundaries",
		"Scalability", "Reliability", "Maintainability",
		"Technical Strategy", "Tradeoff Analysis", "Migration Planning",
		"Integration Design", "Data Flow", "Operational Readiness",
	}

	agent := NewAgent(protocol.AgentTypeArchitecture, name, expertise, ai, hub)

	if archMCP, err := architecture.NewArchitectureMCP(); err != nil {
		log.Printf("Failed to create Architecture MCP server: %v", err)
	} else {
		startAgentMCP(agent, "Architecture", archMCP)
	}

	return agent
}

// NewCodeReviewAgent creates a broad code review agent.
func NewCodeReviewAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Code Review", "Correctness", "Maintainability",
		"Testing", "Refactoring", "Error Handling",
		"Performance", "Readability", "API Contracts",
		"Regression Risk", "Dependency Hygiene", "Documentation",
	}

	agent := NewAgent(protocol.AgentTypeCodeReview, name, expertise, ai, hub)

	if reviewMCP, err := codereview.NewCodeReviewMCP(); err != nil {
		log.Printf("Failed to create Code Review MCP server: %v", err)
	} else {
		startAgentMCP(agent, "CodeReview", reviewMCP)
	}

	return agent
}

// NewBiologyAgent creates a life-sciences agent with Bio MCP tools.
func NewBiologyAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Molecular Biology", "Genomics", "Protein Biochemistry",
		"Lab Protocols", "Sequence Analysis", "Structure Prediction",
		"Assay Design", "CRISPR", "Cell Culture", "Research Literature",
	}

	agent := NewAgent(protocol.AgentTypeBiology, name, expertise, ai, hub)

	bioMCP, err := biology.NewBiologyMCP()
	if err != nil {
		log.Printf("Failed to create Biology MCP server: %v", err)
	} else {
		startAgentMCP(agent, "Biology", bioMCP)
	}

	return agent
}

// NewCADAgent creates a CAD agent with OpenSCAD MCP tools.
func NewCADAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"OpenSCAD", "Parametric Modeling", "Mechanical Design",
		"3D Printing", "CSG Modeling", "CAD Workbench",
	}

	agent := NewAgent(protocol.AgentTypeCAD, name, expertise, ai, hub)

	cadMCP, err := cad.NewCADMCP()
	if err != nil {
		log.Printf("Failed to create CAD MCP server: %v", err)
	} else {
		startDomainAgentMCP(agent, "CAD", cadMCP)
	}

	return agent
}

// NewAWSAgent creates an AWS infrastructure agent with AWS CLI MCP tools.
func NewAWSAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"AWS", "IAM", "EC2", "S3", "Lambda", "CloudFormation",
		"SSO", "Organizations", "VPC", "RDS", "ECS", "EKS",
		"Terraform", "Infrastructure", "Cost", "Security",
	}

	agent := NewAgent(protocol.AgentTypeAWS, name, expertise, ai, hub)

	if awsMCP, err := aws.NewAWSMCP(); err != nil {
		log.Printf("Failed to create AWS MCP server: %v", err)
	} else {
		startDomainAgentMCP(agent, "AWS", awsMCP)
	}

	return agent
}

// NewIncidentAgent creates an incident triage agent with Jira MCP tools.
func NewIncidentAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Incident response", "Bug triage", "Root cause analysis",
		"Reproduction steps", "Severity assessment", "Jira",
		"Stack traces", "Regression analysis", "Handoff",
	}

	agent := NewAgent(protocol.AgentTypeIncident, name, expertise, ai, hub)

	if incMCP, err := incident.NewIncidentMCP(); err != nil {
		log.Printf("Failed to create Incident MCP server: %v", err)
	} else {
		startDomainAgentMCP(agent, "Incident", incMCP)
	}

	return agent
}

// NewBrowserAgent creates a web browsing agent with fetch_url and web_search MCP tools.
func NewBrowserAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"HTML", "CSS", "Web pages", "DOM", "Responsive design",
		"Local dev servers", "localhost", "Page preview", "Web fetch",
		"Frontend QA", "Accessibility", "SEO basics",
	}

	agent := NewAgent(protocol.AgentTypeBrowser, name, expertise, ai, hub)

	if browserMCP, err := browser.NewBrowserMCP(); err != nil {
		log.Printf("Failed to create Browser MCP server: %v", err)
	} else {
		startDomainAgentMCP(agent, "Browser", browserMCP)
	}

	return agent
}

// NewCustomExpertAgent creates a user-defined domain expert (any slug/persona).
func NewCustomExpertAgent(name string, expertise []string, aiProvider ai.AIProvider, hub HubClient) *Agent {
	if len(expertise) == 0 {
		expertise = []string{"General"}
	}
	return NewAgent(protocol.AgentTypeExpert, name, expertise, aiProvider, hub)
}

// NewRepoAgentWrapper creates a repository expert agent wrapper
func NewRepoAgentWrapper(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Repository Analysis",
		"Code Structure",
		"Project Architecture",
	}
	return NewAgent(protocol.AgentTypeRepo, name, expertise, ai, hub)
}

// AgentFactory creates specialized agents based on type
func AgentFactory(agentType protocol.AgentType, name string, ai ai.AIProvider, hub HubClient) (*Agent, error) {
	switch agentType {
	case protocol.AgentTypeFrontend:
		return NewFrontendAgent(name, ai, hub), nil
	case protocol.AgentTypeBackend:
		return NewBackendAgent(name, ai, hub), nil
	case protocol.AgentTypeDevOps:
		return NewDevOpsAgent(name, ai, hub), nil
	case protocol.AgentTypeDatabase:
		return NewDatabaseAgent(name, ai, hub), nil
	case protocol.AgentTypeSecurity:
		return NewSecurityAgent(name, ai, hub), nil
	case protocol.AgentTypeRust:
		return NewRustAgent(name, ai, hub), nil
	case protocol.AgentTypeArchitecture:
		return NewArchitectureAgent(name, ai, hub), nil
	case protocol.AgentTypeCodeReview:
		return NewCodeReviewAgent(name, ai, hub), nil
	case protocol.AgentTypeBiology:
		return NewBiologyAgent(name, ai, hub), nil
	case protocol.AgentTypeCAD:
		return NewCADAgent(name, ai, hub), nil
	case protocol.AgentTypeAWS:
		return NewAWSAgent(name, ai, hub), nil
	case protocol.AgentTypeIncident:
		return NewIncidentAgent(name, ai, hub), nil
	case protocol.AgentTypeBrowser:
		return NewBrowserAgent(name, ai, hub), nil
	case protocol.AgentTypeRepo:
		return NewRepoAgentWrapper(name, ai, hub), nil
	case protocol.AgentTypeAssistant:
		assistant := NewAssistantAgent(name, ai, hub)
		return assistant.Agent, nil
	case protocol.AgentTypeCLI:
		return NewCursorCLIAgent(name, ai, hub), nil
	default:
		return NewAgent(agentType, name, []string{}, ai, hub), nil
	}
}
