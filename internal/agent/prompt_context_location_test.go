package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAppendGrantedDeviceLocation(t *testing.T) {
	var b strings.Builder
	msg := &protocol.Message{Metadata: map[string]interface{}{
		MetadataGrantedDeviceLocation: map[string]interface{}{
			"lat":          30.2672,
			"lon":          -97.7431,
			"display_name": "Austin, Texas",
			"accuracy_m":   20.0,
			"age_s":        12.0,
			"source":       "session",
		},
	}}
	if n := AppendGrantedDeviceLocation(&b, msg); n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
	out := b.String()
	if !strings.Contains(out, "Austin, Texas") || !strings.Contains(out, "30.267200") {
		t.Fatalf("missing location details:\n%s", out)
	}
	if !strings.Contains(out, "web_search") {
		t.Fatal("expected web_search rewrite guidance")
	}
}

func TestAppendGrantedDeviceLocationSkipsIncomplete(t *testing.T) {
	var b strings.Builder
	msg := &protocol.Message{Metadata: map[string]interface{}{
		MetadataGrantedDeviceLocation: map[string]interface{}{"display_name": "Austin"},
	}}
	if n := AppendGrantedDeviceLocation(&b, msg); n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}
