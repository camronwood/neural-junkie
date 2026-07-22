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
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{name: "existing conversion phrase", content: "can you make it an image?", want: true},
		{name: "direct image request", content: "can you generate an image of a logo?", want: true},
		{name: "direct cover art request", content: "generate cover art for the book", want: true},
		{
			name:    "indirect sample cover request",
			content: "ok lets see what a sample outline and cover art image will look like",
			want:    true,
		},
		{name: "preview sample image", content: "I'd like to preview a sample cover image", want: true},
		{name: "descriptive cover question", content: "what would the cover art look like do you think?", want: false},
		{name: "negated capability statement", content: "you can not generate images", want: false},
		{name: "negative request", content: "please don't generate an image yet", want: false},
		{name: "unrelated question", content: "what time is it?", want: false},
		{name: "delivery boilerplate", content: "🖼️ Generated image.", want: false},
		{name: "past tense status", content: "generated image attached for review", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := UserRequestsGeneratedImage(tc.content); got != tc.want {
				t.Fatalf("UserRequestsGeneratedImage(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestUserRequestsImageWithCompanionText(t *testing.T) {
	if !UserRequestsImageWithCompanionText("Let's see a sample outline and cover art image.") {
		t.Fatal("expected mixed outline and image request")
	}
	if UserRequestsImageWithCompanionText("Generate an image of a written story outline.") {
		t.Fatal("an image subject must not become a separate text task")
	}
	if UserRequestsImageWithCompanionText("Generate cover art for the book.") {
		t.Fatal("image-only request must not require a companion response")
	}
}
