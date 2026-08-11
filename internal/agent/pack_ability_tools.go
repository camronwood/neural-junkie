package agent

import (
	"log"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp/browser"
	mapsmcp "github.com/camronwood/neural-junkie/internal/mcp/maps"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/mark3labs/mcp-go/server"
)

// attachEnabledPackToolsToAssistant registers ability-pack MCP tools on Assistant
// when the corresponding pack is enabled (maps geocode/route, browser Playwright).
func attachEnabledPackToolsToAssistant(mcpServer *server.MCPServer) {
	if mcpServer == nil {
		return
	}
	cfg := config.AppConfig()
	if cfg == nil {
		return
	}
	if cfg.IsPackEnabled(config.PackMaps) || cfg.HasPackCapability("maps-tools") {
		mapsmcp.AttachGeocodeRouteTools(mcpServer)
		log.Printf("Assistant: attached maps geocode/route tools (maps pack enabled)")
	}
	if cfg.IsPackEnabled(config.PackMaps) || cfg.HasPackCapability("maps-location") {
		mapsmcp.AttachLocateTool(mcpServer)
		log.Printf("Assistant: attached maps_locate (maps-location capability; sensitive grant required)")
	}
	if cfg.IsPackEnabled(config.PackWebBrowser) || cfg.HasPackCapability("web-browser") {
		browser.AttachAutomationTools(mcpServer)
		log.Printf("Assistant: attached browser automation tools (web-browser pack enabled)")
	}
}

func mapsPackToolsAvailable() bool {
	cfg := config.AppConfig()
	return cfg != nil && (cfg.IsPackEnabled(config.PackMaps) || cfg.HasPackCapability("maps-tools"))
}

func webBrowserPackToolsAvailable() bool {
	cfg := config.AppConfig()
	return cfg != nil && (cfg.IsPackEnabled(config.PackWebBrowser) || cfg.HasPackCapability("web-browser"))
}

// agentSupportsMapsTools reports whether this agent may use maps native/MCP tools.
func (a *Agent) agentSupportsMapsTools() bool {
	if a == nil {
		return false
	}
	switch a.Info.Type {
	case protocol.AgentTypeMaps:
		return true
	case protocol.AgentTypeAssistant:
		return mapsPackToolsAvailable()
	case protocol.AgentTypeExpert:
		return packToolGrantedToAgent(a.Info.Name, "maps-tools")
	default:
		return false
	}
}

// agentSupportsBrowserAutomation reports whether Playwright automation tools apply.
// Assistant gets them via attach when pack is on; custom experts via Composition grants.
func (a *Agent) agentSupportsBrowserAutomation() bool {
	if a == nil {
		return false
	}
	switch a.Info.Type {
	case protocol.AgentTypeBrowser:
		return true
	case protocol.AgentTypeAssistant:
		return webBrowserPackToolsAvailable()
	case protocol.AgentTypeExpert:
		return packToolGrantedToAgent(a.Info.Name, "web-browser")
	default:
		return false
	}
}
