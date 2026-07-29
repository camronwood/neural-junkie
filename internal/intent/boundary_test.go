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
	// Post-hoc quality / claim validators — not turn routing.
	"internal/agent/response_validation.go": true,
	// Chat/code mode inference and advisory prompt NL cues — not stamp-first routing.
	"internal/agent/conversation_mode.go": true,
	// Conversation trust / playbook helpers still hold NL cues (not stamp overrides).
	"internal/agent/conversation_trust.go":       true,
	"internal/agent/implementation_fallback.go":  true,
	"internal/agent/implementation_intent.go":    true, // unused RE vars pending delete
	"internal/agent/implementation_session.go":   true, // export continuation RE
	"internal/routing/knowledge_router.go":       true,
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

// TestAgentTestsAvoidUserRequestsPhraseHelpers fails when agent *_test.go files call
// UserRequests* without an explicit // phrase-migration-shim comment on the same line
// or the line above. Prefer StampTurnDecision fixtures instead.
func TestAgentTestsAvoidUserRequestsPhraseHelpers(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "internal", "agent"))
	helperRE := regexp.MustCompile(`\bUserRequests(Artifact|MapOrRoute|GeneratedImage|GeneratedMusic)\b`)
	shimRE := regexp.MustCompile(`phrase-migration-shim`)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if !helperRE.MatchString(line) {
				continue
			}
			if shimRE.MatchString(line) {
				continue
			}
			if i > 0 && shimRE.MatchString(lines[i-1]) {
				continue
			}
			rel, _ := filepath.Rel(filepath.Join(root, "..", ".."), path)
			t.Errorf("%s:%d: prefer StampTurnDecision over UserRequests*; add // phrase-migration-shim only while migrating", filepath.ToSlash(rel), i+1)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
