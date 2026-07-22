package intent

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// This is an explicit quarantine list for legacy rollback recognizers. New
// semantic phrase authorities must be added to the typed classifier instead of
// spreading into another desktop/server module.
var legacySemanticRecognizerFiles = map[string]bool{
	"internal/agent/artifact_tools.go":                true,
	"internal/agent/code_review_intent.go":            true,
	"internal/agent/conversation_mode.go":             true,
	"internal/agent/conversation_trust.go":            true,
	"internal/agent/implementation_intent.go":         true,
	"internal/agent/implementation_fallback.go":       true,
	"internal/agent/implementation_session.go":        true,
	"internal/agent/response_validation.go":           true,
	"internal/agent/response_images.go":               true,
	"internal/agent/response_music.go":                true,
	"internal/agent/turn_goal.go":                     true,
	"internal/routing/knowledge_router.go":            true,
	"desktop/src/constants/composerMode.ts":           true,
	"desktop/src/utils/bootFixRouting.ts":             true,
	"desktop/src/utils/codeReviewSignals.ts":          true,
	"desktop/src/utils/conversationMode.ts":           true,
	"desktop/src/utils/implementationContinuation.ts": true,
}

func TestSemanticPhraseRecognizersStayQuarantined(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	suspicious := regexp.MustCompile(`(?i)(intent|request|continuation|affirm|boot|artifact|conversation.?mode).{0,80}(MustCompile|RegExp|/\\b)`)
	for _, base := range []string{"internal", "desktop/src"} {
		err := filepath.WalkDir(filepath.Join(root, base), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".test.ts") || strings.HasSuffix(path, ".test.tsx") {
				return nil
			}
			relative, _ := filepath.Rel(root, path)
			relative = filepath.ToSlash(relative)
			if relative == "internal/intent/classifier.go" || legacySemanticRecognizerFiles[relative] {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if suspicious.Match(content) {
				t.Errorf("semantic phrase recognizer added outside canonical classifier or legacy quarantine: %s", relative)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
