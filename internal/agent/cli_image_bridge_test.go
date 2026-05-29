package agent

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaterializeUserImagesForCLI_WritesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a}
	imgs := []protocol.UserImagePart{{MIME: "image/png", Data: png}}

	paths, err := MaterializeUserImagesForCLI(dir, "msg-abc-123", imgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths: %v", paths)
	}
	wantRel := filepath.ToSlash(filepath.Join(cliChatAttachmentsDir, "msg-abc-123", "img-1.png"))
	if paths[0] != wantRel {
		t.Fatalf("rel path = %q want %q", paths[0], wantRel)
	}
	abs := filepath.Join(dir, filepath.FromSlash(paths[0]))
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(png) {
		t.Fatalf("bytes mismatch")
	}
}

func TestShouldRespond_CLIAgentWithUserImages_WhenMentioned(t *testing.T) {
	hubStub := shouldRespondTestHub{}
	workDir := t.TempDir()
	provider := ai.NewCursorCLIProvider(workDir, "")
	ag := NewAgent(protocol.AgentTypeCLI, "Cursor", []string{"code"}, provider, hubStub)
	ag.Info.ID = "cursor-id"
	ag.Info.AIProvider = "cursor-cli"

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "human", Name: "Camron", Type: "human"},
		"@Cursor what is in this screenshot?",
	)
	msg.Mention("cursor-id")
	msg.Metadata = map[string]interface{}{
		protocol.MetadataUserImages: []interface{}{
			map[string]interface{}{
				"mime": "image/png",
				"data": base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47}),
			},
		},
	}

	if !ag.shouldRespond(msg) {
		t.Fatal("expected CLI agent to respond when mentioned with user_images")
	}
}

func TestShouldRespond_NonCLIAgentWithUserImages_Blocked(t *testing.T) {
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeRust, "RustExpert", []string{"rust"}, mockAI, hubStub)
	ag.Info.ID = "rust-id"

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "human", Name: "Camron", Type: "human"},
		"@RustExpert review this",
	)
	msg.Mention("rust-id")
	msg.Metadata = map[string]interface{}{
		protocol.MetadataUserImages: []interface{}{
			map[string]interface{}{
				"mime": "image/png",
				"data": base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47}),
			},
		},
	}

	if ag.shouldRespond(msg) {
		t.Fatal("expected non-vision specialist to ignore user_images")
	}
}

func TestAugmentPromptWithCLIImages_AppendsPaths(t *testing.T) {
	workDir := t.TempDir()
	provider := ai.NewCursorCLIProvider(workDir, "")
	ag := NewAgent(protocol.AgentTypeCLI, "Cursor", []string{"code"}, provider, hubStub{})
	ag.Info.AIProvider = "cursor-cli"

	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"general",
		protocol.AgentInfo{ID: "human", Name: "Camron", Type: "human"},
		"look at this",
	)
	msg.ID = "test-msg-id"
	msg.Metadata = map[string]interface{}{
		protocol.MetadataUserImages: []interface{}{
			map[string]interface{}{
				"mime": "image/png",
				"data": base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a}),
			},
		},
	}

	out := ag.augmentPromptWithCLIImages(msg, "base prompt")
	if !strings.Contains(out, cliChatAttachmentsDir) {
		t.Fatalf("expected attachment dir in prompt, got: %s", out)
	}
	if !strings.Contains(out, "img-1.png") {
		t.Fatalf("expected img path in prompt, got: %s", out)
	}
}

type hubStub struct{}

func (hubStub) SendMessage(*protocol.Message) error       { return nil }
func (hubStub) BroadcastDirect(string, *protocol.Message) {}
func (hubStub) Subscribe(string) (chan *protocol.Message, error) {
	ch := make(chan *protocol.Message, 1)
	return ch, nil
}
func (hubStub) GetMessages(string, int) ([]*protocol.Message, error)  { return nil, nil }
func (hubStub) GetChannelAgents(string) ([]protocol.AgentInfo, error) { return nil, nil }
func (hubStub) GetThreadParentAuthor(string) string                   { return "" }
func (hubStub) GetCommandHandler() CommandHandlerInterface            { return nil }
func (hubStub) GetAgentChannels(string) []string                      { return nil }
func (hubStub) GetChannelType(string) protocol.ChannelType            { return protocol.ChannelTypePublic }
func (hubStub) GetChannelSessionSummary(string) string                { return "" }
func (hubStub) GetThreadMessages(string, int) ([]*protocol.Message, error) { return nil, nil }
func (hubStub) IsChannelHeld(string) bool                               { return false }
func (hubStub) ImageGenerationEnabled() bool                          { return false }
func (hubStub) GenerateAndPostImage(context.Context, string, protocol.AgentInfo, string, string) error {
	return nil
}
