package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserRequestsCodeReview_projectPath(t *testing.T) {
	t.Parallel()
	if !userRequestsCodeReview("Ok code review this project please: /Users/me/dickory-docs") {
		t.Fatal("expected project code review")
	}
	if !userRequestsCodeReview("Can you review the code in the workspace?") {
		t.Fatal("expected workspace code review")
	}
	if userRequestsCodeReview("can you review the code for issues?") {
		t.Fatal("targeted fix review should not be whole-project code review")
	}
}

func TestUserRequestsImplementation_excludesProjectCodeReview(t *testing.T) {
	t.Parallel()
	if userRequestsImplementation("code review this project please") {
		t.Fatal("project code review should not route to implementation")
	}
	if userRequestsImplementation("Can you review the code in the workspace?") {
		t.Fatal("workspace code review should not route to implementation")
	}
	if userRequestsImplementation("can you review the code for issues?") {
		t.Fatal("deprecated userRequestsImplementation must be false")
	}
}

func TestShouldRunImplementationSession_skipsCodeReview(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "User"}, "code review this project")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"implementation_session": true,
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("code review should not run implementation session")
	}
}

func TestShouldRunImplementationSession_skipsCodeReviewIntent(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{Type: protocol.AgentTypeBackend, Name: "BackendEngineer"},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "User"}, "code review this project")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"implementation_session": true,
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("code review intent should never run implementation session")
	}
}

func TestMessageDefersToRepoExpert_pendingReview(t *testing.T) {
	repoPath := t.TempDir()
	hub := &deferRepoTestHub{pending: map[string]bool{normalizeRepoPath(repoPath): true}}
	a := &Agent{
		Info: protocol.AgentInfo{Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
		Hub:  hub,
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "User"},
		"code review this project: "+repoPath)
	if !messageDefersToRepoExpert(a, msg) {
		t.Fatal("expected defer to repo expert")
	}
}

type deferRepoTestHub struct {
	hubArenaNoop
	pending map[string]bool
}

func (h *deferRepoTestHub) SendMessage(*protocol.Message) error       { return nil }
func (h *deferRepoTestHub) BroadcastDirect(string, *protocol.Message) {}
func (h *deferRepoTestHub) Subscribe(string) (chan *protocol.Message, error) {
	return make(chan *protocol.Message), nil
}
func (h *deferRepoTestHub) GetMessages(string, int) ([]*protocol.Message, error)  { return nil, nil }
func (h *deferRepoTestHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) { return nil, nil }
func (h *deferRepoTestHub) GetThreadParentAuthor(string) string                   { return "" }
func (h *deferRepoTestHub) GetCommandHandler() CommandHandlerInterface {
	return &deferRepoCommandHandler{pending: h.pending}
}
func (h *deferRepoTestHub) GetAgentChannels(string) []string { return nil }
func (h *deferRepoTestHub) GetChannelType(string) protocol.ChannelType {
	return protocol.ChannelTypePublic
}
func (h *deferRepoTestHub) GetChannelSessionSummary(string) string { return "" }
func (h *deferRepoTestHub) GetThreadMessages(string, int) ([]*protocol.Message, error) {
	return nil, nil
}
func (h *deferRepoTestHub) IsChannelHeld(string) bool    { return false }
func (h *deferRepoTestHub) ImageGenerationEnabled() bool { return false }
func (h *deferRepoTestHub) GenerateAndPostImage(ctx context.Context, channel string, from protocol.AgentInfo, prompt, size string) error {
	return nil
}
func (h *deferRepoTestHub) MusicGenerationEnabled() bool { return false }
func (h *deferRepoTestHub) GenerateAndPostMusic(context.Context, string, protocol.AgentInfo, MusicGenerateRequest) error {
	return nil
}
func (h *deferRepoTestHub) ExtractAndPostMusicStems(context.Context, string, protocol.AgentInfo, MusicExtractRequest) error {
	return nil
}
func (h *deferRepoTestHub) AskUserQuestion(string, string, string, string, []string) (string, error) {
	return "", nil
}
func (h *deferRepoTestHub) RequestToolApproval(string, string, string, string, map[string]interface{}) (bool, error) {
	return true, nil
}

type deferRepoCommandHandler struct {
	pending map[string]bool
}

func (c *deferRepoCommandHandler) AddPendingReview(string, *protocol.Message, string) {}
func (c *deferRepoCommandHandler) GetPendingReview(string) *protocol.PendingReview    { return nil }
func (c *deferRepoCommandHandler) RemovePendingReview(string)                         {}
func (c *deferRepoCommandHandler) HasPendingReview(path string) bool {
	return c.pending[normalizeRepoPath(path)]
}
func (c *deferRepoCommandHandler) ConsultRepoForPath(context.Context, string, string, string) (string, string, error) {
	return "", "", nil
}
func (c *deferRepoCommandHandler) ConsultReposForPaths(context.Context, []WorkspaceRef, string, string) ([]RepoConsultBlock, error) {
	return nil, nil
}
