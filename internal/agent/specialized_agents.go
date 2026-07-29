package agent

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/contextcompress"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/aws"
	"github.com/camronwood/neural-junkie/internal/mcp/biology"
	"github.com/camronwood/neural-junkie/internal/mcp/browser"
	"github.com/camronwood/neural-junkie/internal/mcp/cad"
	"github.com/camronwood/neural-junkie/internal/mcp/externalmedia"
	"github.com/camronwood/neural-junkie/internal/mcp/incident"
	"github.com/camronwood/neural-junkie/internal/mcp/manufacturing"
	mapsmcp "github.com/camronwood/neural-junkie/internal/mcp/maps"
	"github.com/camronwood/neural-junkie/internal/mcp/usertools"
	"github.com/camronwood/neural-junkie/internal/mcp/workspace"
	webmcp "github.com/camronwood/neural-junkie/internal/mcp/web"
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/mark3labs/mcp-go/server"
)

var sharedBiologyMCPStartOnce sync.Once

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

func attachWebSearchTools(srv MCPServerInterface) {
	if srv == nil {
		return
	}
	webmcp.AttachTools(srv.GetMCPServer())
}

// ensureAgentWebSearchTools attaches shared web_search / fetch_url tools so every
// agent can use hub web search when configured (not Assistant-only).
func ensureAgentWebSearchTools(agent *Agent) {
	if agent == nil {
		return
	}
	if agent.MCPServer != nil {
		attachWebSearchTools(agent.MCPServer)
		return
	}
	mcpServer, err := mcp.NewInProcessMCPServer("shared-web-mcp", "1.0.0")
	if err != nil {
		log.Printf("Failed to create shared web MCP for %s: %v", agent.Info.Name, err)
		return
	}
	webmcp.AttachTools(mcpServer)
	agent.MCPServer = &rawMCPServer{srv: mcpServer}
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
	attachWebSearchTools(srv)
	if err := srv.Start(); err != nil {
		log.Printf("Failed to start %s MCP server: %v", label, err)
	} else {
		log.Printf("%s MCP server started for agent: %s", label, agent.Info.Name)
	}
}

// NewFrontendAgent creates a frontend development agent
func NewFrontendAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Web UI", "Desktop UI", "Mobile UI", "Terminal/TUI",
		"TypeScript", "JavaScript", "Swift", "Kotlin",
		"CSS", "HTML", "React", "Vue", "Svelte",
		"Tauri", "Electron", "iOS", "Android",
		"UI/UX Design", "Accessibility",
		"Performance Optimization", "State Management",
		"Design Systems", "Component Architecture",
		"Visual QA", "Responsive Design",
	}

	agent := NewAgent(protocol.AgentTypeFrontend, name, expertise, ai, hub)
	agent.SupportsVision = true
	agent.Info.SupportsVision = true

	attachSDDomainMCP(agent, "frontend", "Frontend", true, nil)

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

	attachSDDomainMCP(agent, "backend", "Backend", true, nil)

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

	attachSDDomainMCP(agent, "devops", "DevOps", true, nil)

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

	attachSDDomainMCP(agent, "database", "Database", true, nil)

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

	attachSDDomainMCP(agent, "security", "Security", true, nil)

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

	attachSDDomainMCP(agent, "rust", "Rust", true, nil)

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

	attachSDDomainMCP(agent, "architecture", "Architecture", true, nil)

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

	attachSDDomainMCP(agent, "code-review", "CodeReview", true, nil)

	return agent
}

// NewSREAgent creates an SRE / observability specialist.
func NewSREAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Observability", "Prometheus", "Grafana", "OpenTelemetry",
		"Alerting", "SLOs", "Incident response", "On-call",
		"Log analysis", "Distributed tracing", "Capacity planning",
	}
	agent := NewAgent(protocol.AgentTypeSRE, name, expertise, ai, hub)
	attachSDDomainMCP(agent, "sre", "SRE", true, nil)
	return agent
}

// NewMobileAgent creates a mobile development specialist.
func NewMobileAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"React Native", "iOS", "Android", "Swift", "Kotlin",
		"Mobile CI", "App store releases", "Push notifications",
		"Mobile performance", "Device testing",
	}
	agent := NewAgent(protocol.AgentTypeMobile, name, expertise, ai, hub)
	attachSDDomainMCP(agent, "mobile", "Mobile", true, nil)
	return agent
}

// NewDataMLAgent creates a data / ML engineering specialist.
func NewDataMLAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Data pipelines", "ML training", "Feature stores", "Model serving",
		"Jupyter", "Pandas", "scikit-learn", "Experiment tracking",
		"Dataset profiling", "MLOps",
	}
	agent := NewAgent(protocol.AgentTypeDataML, name, expertise, ai, hub)
	attachSDDomainMCP(agent, "data-ml", "DataML", true, nil)
	return agent
}

// NewBiologyAgent creates a life-sciences agent with Bio MCP tools (v1 compat — all tools).
func NewBiologyAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Molecular Biology", "Genomics", "Protein Biochemistry",
		"Lab Protocols", "Sequence Analysis", "Structure Prediction",
		"Assay Design", "CRISPR", "Cell Culture", "Research Literature",
	}

	agent := NewAgent(protocol.AgentTypeBiology, name, expertise, ai, hub)
	attachSharedBiologyMCP(agent, "Biology", "biology")
	return agent
}

// NewGenomicsAgent creates a sequence-focused life-sciences specialist.
func NewGenomicsAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Genomics", "Sequence Analysis", "FASTA", "Mutations",
		"Pathway Context", "DNA", "RNA", "Research Literature",
	}
	agent := NewAgent(protocol.AgentTypeGenomics, name, expertise, ai, hub)
	attachSharedBiologyMCP(agent, "Genomics", "genomics")
	return agent
}

// NewStructuralBiologyAgent creates a structure-focused life-sciences specialist.
func NewStructuralBiologyAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Protein Structure", "ESMFold", "PDB", "Structure Prediction",
		"pLDDT", "Structural Biology", "Research Literature",
	}
	agent := NewAgent(protocol.AgentTypeStructuralBiology, name, expertise, ai, hub)
	attachSharedBiologyMCP(agent, "StructuralBiology", "structural-biology")
	return agent
}

// NewCheminformaticsAgent creates a small-molecule cheminformatics specialist.
func NewCheminformaticsAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"SMILES", "RDKit", "Molecular Descriptors", "Cheminformatics",
		"Medicinal Chemistry Triage", "Research Literature",
	}
	agent := NewAgent(protocol.AgentTypeCheminformatics, name, expertise, ai, hub)
	attachSharedBiologyMCP(agent, "Cheminformatics", "cheminformatics")
	return agent
}

func attachSharedBiologyMCP(agent *Agent, label, agentType string) {
	bioMCP, err := biology.SharedBiologyMCP()
	if err != nil {
		log.Printf("Failed to create Biology MCP server: %v", err)
		return
	}
	if allowlist := biology.ToolAllowlistForAgentType(agentType); allowlist != nil {
		agent.MCPToolAllowlist = allowlist
	}
	agent.MCPServer = bioMCP
	attachWorkspaceTools(agent, bioMCP)
	sharedBiologyMCPStartOnce.Do(func() {
		attachContextCompressTools(bioMCP)
		if err := bioMCP.Start(); err != nil {
			log.Printf("Failed to start %s MCP server: %v", label, err)
		} else {
			log.Printf("Biology MCP server started (shared)")
		}
	})
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

// NewManufacturingAgent creates a manufacturing agent with print/export MCP tools.
func NewManufacturingAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"FDM 3D Printing", "Printability", "Overhangs", "Wall Thickness",
		"Mesh Repair", "Slicer Presets", "PrusaSlicer", "Orca Slicer",
		"G-code", "STEP Export", "Technical Drawings",
	}

	agent := NewAgent(protocol.AgentTypeManufacturing, name, expertise, ai, hub)

	mfgMCP, err := manufacturing.NewManufacturingMCP()
	if err != nil {
		log.Printf("Failed to create Manufacturing MCP server: %v", err)
	} else {
		startDomainAgentMCP(agent, "Manufacturing", mfgMCP)
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
		startAgentMCPWithOptions(agent, "AWS", awsMCP, true)
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

// NewMusicAgent creates a songwriting agent with hub music generation tools.
func NewMusicAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Songwriting", "Lyrics", "Music production", "Genre and mood",
		"Instrumental beds", "Jingles", "Soundtracks", "ACE-Step",
	}
	return NewAgent(protocol.AgentTypeMusic, name, expertise, ai, hub)
}

// NewMapsAgent creates a maps/routing agent with geocode, route, and canvas map tools.
func NewMapsAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Maps", "Geocoding", "Walking routes", "Driving routes",
		"OpenStreetMap", "OSRM", "Nominatim", "Itineraries", "Directions",
	}

	agent := NewAgent(protocol.AgentTypeMaps, name, expertise, ai, hub)
	// Native maps_create/maps_update publish artifacts; MCP keeps geocode/route only.
	agent.MCPToolAllowlist = []string{"maps_geocode", "maps_route"}

	if mapsMCP, err := mapsmcp.NewMapsMCP(); err != nil {
		log.Printf("Failed to create Maps MCP server: %v", err)
	} else {
		startDomainAgentMCP(agent, "Maps", mapsMCP)
	}

	return agent
}

// NewArenaAgent creates a model comparison agent for chess, Connect Four, and logic puzzles.
func NewArenaAgent(name string, ai ai.AIProvider, hub HubClient) *Agent {
	expertise := []string{
		"Chess", "Connect Four", "Logic puzzles", "Model comparison", "Benchmarks",
	}
	return NewAgent(protocol.AgentTypeArena, name, expertise, ai, hub)
}

// NewCustomExpertAgent creates a user-defined domain expert (any slug/persona).
// When the MCP Tool Wizard has granted this agent (by name) one or more
// user-defined tools, they're attached here so the tool loop can call them —
// custom experts otherwise have no MCP server at all.
func NewCustomExpertAgent(name string, expertise []string, aiProvider ai.AIProvider, hub HubClient) *Agent {
	if len(expertise) == 0 {
		expertise = []string{"General"}
	}
	agent := NewAgent(protocol.AgentTypeExpert, name, expertise, aiProvider, hub)
	attachUserToolsMCP(agent, name)
	return agent
}

// attachUserToolsMCP wires granted MCP Tool Wizard user tools and/or granted
// external-media tools (media_submit/media_status/media_fetch) onto agent,
// plus Composition Model pack-tool grants (maps/browser). Sharing one
// in-process MCP server so a custom expert with both kinds of grants gets a
// single unified tool call surface. No-ops (and leaves agent.MCPServer nil)
// when nothing is granted, so custom experts without tools keep their current
// zero-overhead behavior.
func attachUserToolsMCP(agent *Agent, agentName string) {
	mcpServer, err := mcp.NewInProcessMCPServer("custom-expert-tools-mcp", "1.0.0")
	if err != nil {
		log.Printf("Failed to create custom expert tools MCP for %s: %v", agentName, err)
		return
	}

	attached := usertools.AttachGranted(mcpServer, agentName) > 0
	if media, err := externalmedia.AttachGranted(mcpServer, agentName); err != nil {
		log.Printf("Failed to attach external media tools for %s: %v", agentName, err)
	} else if media != nil {
		attached = true
	}
	if attachPackToolGrantsToServer(mcpServer, agentName) {
		attached = true
	}

	if !attached {
		return
	}
	startDomainAgentMCP(agent, "Custom Tools", &rawMCPServer{srv: mcpServer})
}

// ReattachGrantedTools rebuilds a custom expert's MCP from current grants
// (user tools, external media, pack ability tools). Safe to call after a grant API update.
func ReattachGrantedTools(a *Agent) {
	if a == nil || a.Info.Type != protocol.AgentTypeExpert {
		return
	}
	a.MCPServer = nil
	a.MCPToolAllowlist = nil
	attachUserToolsMCP(a, a.Info.Name)
}

// attachPackToolGrantsToServer registers maps/browser MCP tools when Composition
// granted them to agentName and the owning pack is still enabled.
func attachPackToolGrantsToServer(mcpServer *server.MCPServer, agentName string) bool {
	if mcpServer == nil || strings.TrimSpace(agentName) == "" {
		return false
	}
	attached := false
	if packToolGrantedToAgent(agentName, "maps-tools") {
		mapsmcp.AttachGeocodeRouteTools(mcpServer)
		attached = true
	}
	if packToolGrantedToAgent(agentName, "web-browser") {
		browser.AttachAutomationTools(mcpServer)
		attached = true
	}
	return attached
}

// rawMCPServer adapts a bare mcp-go server.MCPServer to MCPServerInterface,
// so multiple tool-registration helpers can share one in-process server.
type rawMCPServer struct {
	srv *server.MCPServer
}

func (r *rawMCPServer) GetMCPServer() *server.MCPServer { return r.srv }
func (r *rawMCPServer) Start() error                    { return nil }

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
	var agent *Agent
	switch agentType {
	case protocol.AgentTypeFrontend:
		agent = NewFrontendAgent(name, ai, hub)
	case protocol.AgentTypeBackend:
		agent = NewBackendAgent(name, ai, hub)
	case protocol.AgentTypeDevOps:
		agent = NewDevOpsAgent(name, ai, hub)
	case protocol.AgentTypeDatabase:
		agent = NewDatabaseAgent(name, ai, hub)
	case protocol.AgentTypeSecurity:
		agent = NewSecurityAgent(name, ai, hub)
	case protocol.AgentTypeRust:
		agent = NewRustAgent(name, ai, hub)
	case protocol.AgentTypeArchitecture:
		agent = NewArchitectureAgent(name, ai, hub)
	case protocol.AgentTypeSRE:
		agent = NewSREAgent(name, ai, hub)
	case protocol.AgentTypeMobile:
		agent = NewMobileAgent(name, ai, hub)
	case protocol.AgentTypeDataML:
		agent = NewDataMLAgent(name, ai, hub)
	case protocol.AgentTypeBiology:
		agent = NewBiologyAgent(name, ai, hub)
	case protocol.AgentTypeGenomics:
		agent = NewGenomicsAgent(name, ai, hub)
	case protocol.AgentTypeStructuralBiology:
		agent = NewStructuralBiologyAgent(name, ai, hub)
	case protocol.AgentTypeCheminformatics:
		agent = NewCheminformaticsAgent(name, ai, hub)
	case protocol.AgentTypeCAD:
		agent = NewCADAgent(name, ai, hub)
	case protocol.AgentTypeManufacturing:
		agent = NewManufacturingAgent(name, ai, hub)
	case protocol.AgentTypeAWS:
		agent = NewAWSAgent(name, ai, hub)
	case protocol.AgentTypeIncident:
		agent = NewIncidentAgent(name, ai, hub)
	case protocol.AgentTypeArena:
		agent = NewArenaAgent(name, ai, hub)
	case protocol.AgentTypeBrowser, protocol.AgentTypeMusic, protocol.AgentTypeMaps, protocol.AgentTypeCodeReview:
		return nil, fmt.Errorf("%s agents were removed — enable the ability pack for Assistant tools, or use Composition grants for custom experts", agentType)
	case protocol.AgentTypeRepo:
		agent = NewRepoAgentWrapper(name, ai, hub)
	case protocol.AgentTypeAssistant:
		agent = NewAssistantAgent(name, ai, hub).Agent
	case protocol.AgentTypeCLI:
		agent = NewCursorCLIAgent(name, ai, hub)
	default:
		agent = NewAgent(agentType, name, []string{}, ai, hub)
	}
	ensureAgentWebSearchTools(agent)
	return agent, nil
}

// ResolveAgentTypeFromPackSpec picks the runtime agent type from a pack AgentSpec.
// A valid implementation: builtin/<slug> wins over type (including empty or mismatched type).
func ResolveAgentTypeFromPackSpec(spec packs.AgentSpec) protocol.AgentType {
	if at, ok := packs.ParseBuiltinImplementation(spec.Implementation); ok {
		return protocol.AgentType(at)
	}
	return protocol.AgentType(strings.TrimSpace(spec.Type))
}

// AgentFactoryFromPackSpec creates an in-process agent from a pack AgentSpec.
// Pilot: implementation builtin/music -> NewMusicAgent even when type is empty or wrong.
func AgentFactoryFromPackSpec(spec packs.AgentSpec, name string, aiProvider ai.AIProvider, hub HubClient) (*Agent, error) {
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSpace(spec.Name)
	}
	return AgentFactory(ResolveAgentTypeFromPackSpec(spec), name, aiProvider, hub)
}
