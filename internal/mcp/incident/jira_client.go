package incident

import (
	"fmt"

	jiraclient "github.com/camronwood/neural-junkie/internal/integrations/jira"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
)

// Client is the Jira Cloud REST client (alias for integrations/jira).
type Client = jiraclient.Client

// CreateIssueRequest re-exports jira create request.
type CreateIssueRequest = jiraclient.CreateIssueRequest

// NewClient creates a Jira client from settings.
var NewClient = jiraclient.NewClient

func clientFromHub() (*Client, error) {
	if cfg := mcp.AppConfig(); cfg != nil {
		return NewClient(cfg.JiraSettings())
	}
	return nil, fmt.Errorf("hub config unavailable")
}
