package agent

import (
	// "log" // Temporarily unused due to MCP disable

	// MCP imports temporarily disabled
	// "github.com/camronwood/neural-junkie/internal/mcp/backend"
	// "github.com/camronwood/neural-junkie/internal/mcp/database"
	// "github.com/camronwood/neural-junkie/internal/mcp/devops"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// NewFrontendAgent creates a frontend development agent
func NewFrontendAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"React", "Vue.js", "Angular",
		"TypeScript", "JavaScript",
		"CSS", "HTML", "Tailwind",
		"UI/UX Design", "Accessibility",
		"Performance Optimization",
		"State Management", "Redux",
		"Design Analysis", "CSS Generation",
		"Style Guide Creation", "Design Tokens",
		"Component Recreation", "Visual Design",
	}

	agent := NewAgent(protocol.AgentTypeFrontend, name, expertise, ai, hub)
	agent.SupportsVision = true
	agent.Info.SupportsVision = true
	return agent
}

// NewBackendAgent creates a backend development agent
func NewBackendAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Go", "Node.js", "Python",
		"REST APIs", "GraphQL", "gRPC",
		"Microservices", "Event-Driven Architecture",
		"Business Logic", "API Design",
		"Performance", "Caching",
		"Message Queues", "Redis",
	}

	agent := NewAgent(protocol.AgentTypeBackend, name, expertise, ai, hub)

	// MCP temporarily disabled
	// backendMCP, err := backend.NewBackendMCP()
	// if err != nil {
	// 	log.Printf("Failed to create Backend MCP server: %v", err)
	// } else {
	// 	agent.MCPServer = backendMCP
	// 	if err := backendMCP.Start(); err != nil {
	// 		log.Printf("Failed to start Backend MCP server: %v", err)
	// 	} else {
	// 		log.Printf("Backend MCP server started for agent: %s", name)
	// 	}
	// }

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

	// MCP temporarily disabled
	// devopsMCP, err := devops.NewDevOpsMCP()
	// if err != nil {
	// 	log.Printf("Failed to create DevOps MCP server: %v", err)
	// } else {
	// 	agent.MCPServer = devopsMCP
	// 	if err := devopsMCP.Start(); err != nil {
	// 		log.Printf("Failed to start DevOps MCP server: %v", err)
	// 	} else {
	// 		log.Printf("DevOps MCP server started for agent: %s", name)
	// 	}
	// }

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

	// MCP temporarily disabled
	// databaseMCP, err := database.NewDatabaseMCP()
	// if err != nil {
	// 	log.Printf("Failed to create Database MCP server: %v", err)
	// } else {
	// 	agent.MCPServer = databaseMCP
	// 	if err := databaseMCP.Start(); err != nil {
	// 		log.Printf("Failed to start Database MCP server: %v", err)
	// 	} else {
	// 		log.Printf("Database MCP server started for agent: %s", name)
	// 	}
	// }

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

	return NewAgent(protocol.AgentTypeSecurity, name, expertise, ai, hub)
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

	return NewAgent(protocol.AgentTypeRust, name, expertise, ai, hub)
}

// NewRepoAgentWrapper creates a repository expert agent wrapper
// Note: The actual RepoAgent is created with NewRepoAgent which requires a repo path
// This is just a placeholder for the factory pattern
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
	case protocol.AgentTypeRepo:
		// For repo agents, use the wrapper - actual repo agents should be created with NewRepoAgent
		return NewRepoAgentWrapper(name, ai, hub), nil
	case protocol.AgentTypeModerator:
		// Return the base Agent from the ModeratorAgent
		moderator := NewModeratorAgent(name, ai, hub)
		return moderator.Agent, nil
	case protocol.AgentTypeAssistant:
		// Return the base Agent from the AssistantAgent
		assistant := NewAssistantAgent(name, ai, hub)
		return assistant.Agent, nil
	case protocol.AgentTypeCLI:
		return NewCursorCLIAgent(name, ai, hub), nil
	default:
		return NewAgent(agentType, name, []string{}, ai, hub), nil
	}
}
