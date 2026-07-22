package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func ambientTestMessage(content string, state interface{}) *protocol.Message {
	return &protocol.Message{
		Content:  content,
		Metadata: map[string]interface{}{MetadataAmbientState: state},
	}
}

func TestSanitizeAmbientStateMetadataRelevanceRedactionAndCap(t *testing.T) {
	diagnostics := make([]interface{}, 0, 100)
	for i := 0; i < 100; i++ {
		diagnostics = append(diagnostics, map[string]interface{}{
			"path": "/repo/main.go", "line": i + 1, "column": 1, "severity": "error",
			"message": "password=hunter2 " + strings.Repeat("x", 1000),
		})
	}
	msg := ambientTestMessage("fix the diagnostics", map[string]interface{}{
		"active_editor": map[string]interface{}{
			"path":      "/repo/.env",
			"selection": map[string]interface{}{"start_line": 1, "end_line": 1, "text": "api_key=secret"},
		},
		"diagnostics": diagnostics,
		"terminal": map[string]interface{}{
			"cwd": "/repo", "failed_tail": "\x1b[31merror client_secret=secret\x1b[0m",
		},
	})
	SanitizeAmbientStateMetadata(msg)

	state, ok := msg.Metadata[MetadataAmbientState].(map[string]interface{})
	if !ok {
		t.Fatal("expected sanitized ambient state")
	}
	encoded, _ := json.Marshal(state)
	if len(encoded) > maxAmbientStateServerBytes {
		t.Fatalf("ambient state is %d bytes, cap is %d", len(encoded), maxAmbientStateServerBytes)
	}
	if strings.Contains(string(encoded), "hunter2") || strings.Contains(string(encoded), "client_secret=secret") || strings.Contains(string(encoded), "\x1b") {
		t.Fatalf("unsafe text remained: %s", encoded)
	}
	editor := state["active_editor"].(map[string]interface{})
	selection := editor["selection"].(map[string]interface{})
	if _, exists := selection["text"]; exists {
		t.Fatal("sensitive-file selection must be path/coordinates only")
	}
}

func TestSanitizeAmbientStateMetadataDropsIrrelevantState(t *testing.T) {
	msg := ambientTestMessage("What is the weather?", map[string]interface{}{
		"terminal": map[string]interface{}{"failed_tail": "error"},
	})
	SanitizeAmbientStateMetadata(msg)
	if _, exists := msg.Metadata[MetadataAmbientState]; exists {
		t.Fatal("irrelevant ambient state should be removed")
	}
}

func TestAppendAmbientStateRendersEphemeralSection(t *testing.T) {
	msg := ambientTestMessage("debug this", map[string]interface{}{
		"recent_edits": []interface{}{map[string]interface{}{"path": "/repo/main.go", "edited_at": 1}},
	})
	SanitizeAmbientStateMetadata(msg)
	var prompt strings.Builder
	AppendAmbientState(&prompt, msg)
	if !strings.Contains(prompt.String(), "AMBIENT IDE STATE (EPHEMERAL)") ||
		!strings.Contains(prompt.String(), "/repo/main.go") {
		t.Fatalf("unexpected prompt section: %q", prompt.String())
	}
}
