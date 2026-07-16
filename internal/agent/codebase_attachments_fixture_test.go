package agent

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMergeCodebaseAttachments_minimalRepoFixture(t *testing.T) {
	t.Parallel()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "scenarios", "fixtures", "minimal-repo")
	msg := &protocol.Message{
		Content: "@codebase What does ComputeObscureWidget return?",
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": root,
			},
		},
	}
	MergeCodebaseAttachments(msg)
	raw, ok := msg.Metadata[MetadataPromptAttachments]
	if !ok {
		t.Fatal("expected prompt_attachments")
	}
	arr, ok := raw.([]interface{})
	if !ok || len(arr) == 0 {
		t.Fatalf("expected chunks, got %v", raw)
	}
	found := false
	for _, item := range arr {
		fm, _ := item.(map[string]interface{})
		content, _ := fm["content"].(string)
		if strings.Contains(content, "ComputeObscureWidget") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ComputeObscureWidget not in attachments: %#v", arr)
	}
}
