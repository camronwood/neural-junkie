package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAttachGeneratedImageFromResponse(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "diagram.png")
	if err := os.WriteFile(imgPath, []byte{0x89, 0x50, 0x4e, 0x47}, 0644); err != nil {
		t.Fatal(err)
	}
	msg := protocol.NewMessage(protocol.MessageTypeAnswer, "general", protocol.AgentInfo{Name: "Cursor"}, "Saved to "+imgPath)
	if AttachGeneratedImageFromResponse(msg, dir) != true {
		t.Fatal("expected image attachment")
	}
	raw, ok := msg.Metadata["generated_image"].(map[string]interface{})
	if !ok || raw["data"] == "" {
		t.Fatalf("missing generated_image data: %+v", msg.Metadata)
	}
}

func TestAttachGeneratedImageFromResponse_MarkdownPath(t *testing.T) {
	path := "/Users/camronwood/.cursor/projects/Users-camronwood-development-sandbox-neural-junkie/assets/neural-junkie-architecture.png"
	if _, err := os.Stat(path); err != nil {
		t.Skip("cursor asset not on this machine:", err)
	}
	content := "It's also on disk at **" + path + "** if you want to open it."
	msg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:C1", protocol.AgentInfo{Name: "Cursor"}, content)
	if !AttachGeneratedImageFromResponse(msg) {
		t.Fatal("expected image attachment from markdown-wrapped absolute path")
	}
	raw, ok := msg.Metadata["generated_image"].(map[string]interface{})
	if !ok || raw["data"] == "" {
		t.Fatalf("missing generated_image: %+v", msg.Metadata)
	}
}

func TestUserRequestsGeneratedImage(t *testing.T) {
	if !UserRequestsGeneratedImage("can you make it an image?") {
		t.Fatal("expected true")
	}
	if UserRequestsGeneratedImage("what time is it?") {
		t.Fatal("expected false")
	}
	if UserRequestsGeneratedImage("🖼️ Generated image.") {
		t.Fatal("delivery boilerplate should not count as a request")
	}
	if UserRequestsGeneratedImage("generated image attached for review") {
		t.Fatal("past-tense generated should not match generate verb")
	}
	if !UserRequestsGeneratedImage("can you generate an image of a logo?") {
		t.Fatal("expected generate request to match")
	}
}
