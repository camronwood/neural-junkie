package protocol

import (
	"encoding/base64"
	"testing"
)

func TestExtractUserImages_JSONStringBase64(t *testing.T) {
	raw := []byte{0x89, 0x50, 0x4e, 0x47} // fake png header
	b64 := base64.StdEncoding.EncodeToString(raw)
	msg := &Message{
		Metadata: map[string]interface{}{
			MetadataImageData: b64,
			MetadataImageType: "image/png",
		},
	}
	parts := ExtractUserImages(msg)
	if len(parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(parts))
	}
	if parts[0].MIME != "image/png" {
		t.Fatalf("mime %q", parts[0].MIME)
	}
	if string(parts[0].Data) != string(raw) {
		t.Fatalf("data mismatch")
	}
}

func TestExtractUserImages_UserImagesArray(t *testing.T) {
	raw := []byte("hello")
	b64 := base64.StdEncoding.EncodeToString(raw)
	msg := &Message{
		Metadata: map[string]interface{}{
			MetadataUserImages: []interface{}{
				map[string]interface{}{"mime": "image/jpeg", "data": b64},
			},
		},
	}
	parts := ExtractUserImages(msg)
	if len(parts) != 1 || string(parts[0].Data) != "hello" {
		t.Fatalf("got %+v", parts)
	}
}

func TestSanitizeUserImagesMetadata_NormalizesLegacy(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("x"))
	msg := &Message{Metadata: map[string]interface{}{
		MetadataImageData: b64,
		MetadataImageType: "image/gif",
	}}
	SanitizeUserImagesMetadata(msg)
	if _, ok := msg.Metadata[MetadataImageData]; ok {
		t.Fatal("expected legacy keys removed")
	}
	arr, ok := msg.Metadata[MetadataUserImages].([]interface{})
	if !ok || len(arr) != 1 {
		t.Fatalf("user_images: %v", msg.Metadata[MetadataUserImages])
	}
}
