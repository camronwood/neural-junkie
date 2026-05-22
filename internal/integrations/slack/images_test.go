package slack

import (
	"encoding/base64"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestExtractGeneratedImageFromMetadata(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:C1", protocol.AgentInfo{}, "done")
	msg.Metadata["generated_image"] = map[string]interface{}{
		"mime": "image/png",
		"data": base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47}),
	}
	img := ExtractGeneratedImage(msg)
	if img == nil || len(img.Data) != 4 {
		t.Fatalf("expected payload, got %+v", img)
	}
}

func TestNJMessageForSlackTS(t *testing.T) {
	tm := &ThreadMap{
		njMessageTS: map[string]string{"nj-1": "100.0"},
	}
	if got := tm.NJMessageForSlackTS("100.0"); got != "nj-1" {
		t.Fatalf("got %q", got)
	}
}
