package protocol

import "testing"

func TestRedactImageBinaryMetadataPreservesGeneratedImagePath(t *testing.T) {
	msg := &Message{
		Metadata: map[string]interface{}{
			"generated_image": map[string]interface{}{
				"mime": "image/png",
				"data": "aGVsbG8=",
				"path": "/Users/me/.neural-junkie/generated-images/msg-1.png",
			},
		},
	}
	RedactImageBinaryMetadata(msg)
	raw, ok := msg.Metadata["generated_image"].(map[string]interface{})
	if !ok {
		t.Fatal("expected generated_image map")
	}
	if raw["data_redacted"] != true {
		t.Fatal("expected data_redacted")
	}
	if _, has := raw["data"]; has {
		t.Fatal("data should be stripped")
	}
	if raw["path"] != "/Users/me/.neural-junkie/generated-images/msg-1.png" {
		t.Fatalf("path not preserved: %+v", raw)
	}
}

func TestIsGeneratedImageDelivery(t *testing.T) {
	delivery := NewMessage(MessageTypeChat, "general", AgentInfo{Name: "Assistant"}, GeneratedImageDeliveryContent)
	delivery.Metadata = map[string]interface{}{
		"generated_image": map[string]interface{}{"mime": "image/png"},
	}
	if !IsGeneratedImageDelivery(delivery) {
		t.Fatal("expected standard delivery message to be detected")
	}
	request := NewMessage(MessageTypeQuestion, "general", AgentInfo{Name: "camron"}, "please generate an image of a logo")
	if IsGeneratedImageDelivery(request) {
		t.Fatal("expected user request not to be delivery")
	}
}
