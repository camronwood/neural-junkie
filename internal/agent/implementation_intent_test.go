package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserAffirmsPendingImplementation_expanded(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"approved", true},
		{"its approved", true},
		{"please keep going", true},
		{"yes keep going", true},
		{"yes that sounds good", true},
		{"can you review the code for issues?", false},
	}
	for _, tc := range cases {
		if got := userAffirmsPendingImplementation(tc.in); got != tc.want {
			t.Fatalf("userAffirmsPendingImplementation(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestUserRequestsImplementation_debugPhrases(t *testing.T) {
	t.Parallel()
	if !userRequestsImplementation("can you review the code for issues?") {
		t.Fatal("expected code review to request implementation")
	}
	if !userRequestsImplementation("ok the app does not seem to be working now") {
		t.Fatal("expected broken-app report to request implementation")
	}
}

func TestChannelHasRecentImplementationActivity(t *testing.T) {
	t.Parallel()
	agentID := "fe-1"
	history := []*protocol.Message{
		{
			ID:   "u1",
			Type: protocol.MessageTypeQuestion,
			From: protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
			Content: "add a settings button with themes",
		},
		{
			ID:   "fc1",
			Type: protocol.MessageTypeFileChange,
			From: protocol.AgentInfo{ID: agentID, Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
			Content: "edit src/App.tsx",
		},
	}
	if !channelHasRecentImplementationActivity(history, "u2", agentID) {
		t.Fatal("expected active thread after file change")
	}
}

func TestChannelHasRecentImplementationActivity_stopsAtClosure(t *testing.T) {
	t.Parallel()
	agentID := "asst"
	history := []*protocol.Message{
		{
			ID:      "u1",
			Type:    protocol.MessageTypeQuestion,
			From:    protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
			Content: "how would you add a theme toggle?",
		},
		{
			ID:   "a1",
			Type: protocol.MessageTypeChat,
			From: protocol.AgentInfo{ID: agentID, Name: "Assistant", Type: protocol.AgentTypeAssistant},
			Content: "Put it in the settings header.",
		},
		{
			ID:      "u2",
			Type:    protocol.MessageTypeQuestion,
			From:    protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
			Content: "ok thanks",
		},
		{
			ID:   "a2",
			Type: protocol.MessageTypeChat,
			From: protocol.AgentInfo{ID: agentID, Name: "Assistant", Type: protocol.AgentTypeAssistant},
			Content: "You're welcome! Let me know if you need anything else.",
		},
	}
	if channelHasRecentImplementationActivity(history, "u3", agentID) {
		t.Fatal("closure should terminate implementation thread")
	}
}

func TestIsAdvisoryImplementationQuestion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"how would you add a light/dark theme toggle in a React settings page?", true},
		{"One more thing — where should the theme toggle live in the settings UI?", true},
		{"please add theme support to settings", false},
		{"implement the theme toggle now", false},
	}
	for _, tc := range cases {
		if got := isAdvisoryImplementationQuestion(tc.in); got != tc.want {
			t.Fatalf("isAdvisoryImplementationQuestion(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestUserRequestsImplementationForMessage_afterUIApproval(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-fe": {
					{
						ID:      "u1",
						Type:    protocol.MessageTypeQuestion,
						From:    protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
						Content: "blank screen can you fix it?",
					},
					{
						ID:   "fc1",
						Type: protocol.MessageTypeFileChange,
						From: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
						Content: "edit src/App.tsx",
					},
					{
						ID:      "u-approve",
						Type:    protocol.MessageTypeChat,
						From:    protocol.AgentInfo{ID: "human-user", Name: "User", Type: "human"},
						Content: "Approved and applied your edit change to `src/App.tsx`. Continue with the implementation — do not ask me to approve again.",
						Metadata: map[string]interface{}{
							protocol.MetaFileChangeApproved: true,
							protocol.MetaFileChangeAgentID:  "fe-1",
						},
					},
				},
			},
		},
	}
	msg := &protocol.Message{
		ID:      "u2",
		Channel: "dm-u-fe",
		From:    protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
		Content: "what's next?",
	}
	if !userRequestsImplementationForMessage(a, msg) {
		t.Fatal("expected follow-up after UI approval to continue implementation")
	}
}

func TestChannelHasPendingImplementationPlan_falseAfterUIApproval(t *testing.T) {
	t.Parallel()
	agentID := "fe-1"
	history := []*protocol.Message{
		{
			ID:   "fc1",
			Type: protocol.MessageTypeFileChange,
			From: protocol.AgentInfo{ID: agentID, Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
			Content: "edit src/App.tsx",
		},
		{
			ID:      "u-approve",
			Type:    protocol.MessageTypeChat,
			From:    protocol.AgentInfo{ID: "human-user", Name: "User", Type: "human"},
			Content: "Approved and applied your edit change to `src/App.tsx`.",
			Metadata: map[string]interface{}{
				protocol.MetaFileChangeApproved: true,
				protocol.MetaFileChangeAgentID:  agentID,
			},
		},
	}
	if channelHasPendingImplementationPlan(history, "", agentID) {
		t.Fatal("expected no pending plan after UI approval")
	}
}

func TestUserRequestsImplementationForMessage_afterApproval(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-fe": {
					{
						ID:   "fc1",
						Type: protocol.MessageTypeFileChange,
						From: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
						Content: "edit src/App.tsx",
					},
				},
			},
		},
	}
	msg := &protocol.Message{
		ID:      "u2",
		Channel: "dm-u-fe",
		From:    protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
		Content: "approved",
	}
	if !userRequestsImplementationForMessage(a, msg) {
		t.Fatal("expected approved after file change to continue implementation")
	}
}

func TestUserRequestsImplementationForMessage_weakAffirmAfterFailedSession(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-fe": {
					{
						ID:      "u1",
						Type:    protocol.MessageTypeQuestion,
						From:    protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
						Content: "blank screen can you fix it?",
					},
					{
						ID:      "a1",
						Type:    protocol.MessageTypeChat,
						From:    protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
						Content: "Implementation session finished without file changes.\n\nPlease share vite.config.ts",
					},
				},
			},
		},
	}
	msg := &protocol.Message{
		ID:      "u2",
		Channel: "dm-u-fe",
		From:    protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
		Content: "looks good",
	}
	if userRequestsImplementationForMessage(a, msg) {
		t.Fatal("expected looks good after failed session not to continue implementation")
	}
}

func TestUserRequestsImplementation_workspaceDirective(t *testing.T) {
	t.Parallel()
	if !userRequestsImplementation("use the open workspace it has all the files you need") {
		t.Fatal("expected workspace directive to request implementation")
	}
}

func TestShouldForceSessionSummaryRefreshOnAgentResponse(t *testing.T) {
	t.Parallel()
	if !ShouldForceSessionSummaryRefreshOnAgentResponse("### vite.config.ts\n```ts\nexport default {}\n```") {
		t.Fatal("file dump should force summary refresh")
	}
	if !ShouldForceSessionSummaryRefreshOnAgentResponse("Implementation session finished without file changes.") {
		t.Fatal("failed session should force summary refresh")
	}
}

func TestSanitizeFailedImplementationResponse(t *testing.T) {
	t.Parallel()
	state := &ImplementationSessionState{SeedsLoaded: 3}
	out := sanitizeFailedImplementationResponse("Please provide the content of vite.config.ts", state)
	if strings.Contains(out, "Please provide") {
		t.Fatalf("expected paste request to be replaced, got %q", out)
	}
	if !strings.Contains(out, "src/main.tsx") {
		t.Fatalf("expected diagnostic guidance, got %q", out)
	}
}

func TestShouldUseFileChangeFenceFallback_implementationSession(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend}}
	msg := &protocol.Message{
		Content: "hello",
		Metadata: map[string]interface{}{
			protocol.IdeMetaImplementationSession: true,
		},
	}
	if !a.shouldUseFileChangeFenceFallback(msg) {
		t.Fatal("implementation_session metadata should allow fence fallback")
	}
	msg2 := &protocol.Message{Content: "what is react?"}
	if a.shouldUseFileChangeFenceFallback(msg2) {
		t.Fatal("generic chat should not allow fence fallback")
	}
}

func TestShouldUseFileChangeFenceFallback_contentDelivery(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{ID: "cr-1", Type: protocol.AgentTypeCodeReview}}
	msg := &protocol.Message{
		Content: "Can you create a linkedin article about this app for me?",
		Metadata: map[string]interface{}{
			protocol.IdeMetaImplementationSession: true,
		},
	}
	if a.shouldUseFileChangeFenceFallback(msg) {
		t.Fatal("linkedin article should not allow fence fallback")
	}
}

func TestShouldUseFileChangeFenceFallback_bareWorkspaceDirective(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{ID: "cr-1", Type: protocol.AgentTypeCodeReview}}
	msg := &protocol.Message{
		Content:  "use the workspace",
		Metadata: map[string]interface{}{protocol.IdeMetaImplementationSession: true},
	}
	if a.shouldUseFileChangeFenceFallback(msg) {
		t.Fatal("bare workspace directive should not allow fence fallback")
	}
	msg2 := &protocol.Message{Content: "use the workspace to implement dark mode"}
	if !a.shouldUseFileChangeFenceFallback(msg2) {
		t.Fatal("workspace directive with implement ask should allow fence fallback")
	}
}

func TestImplementationSeedCandidates_workspaceDirectiveLoadsReadme(t *testing.T) {
	t.Parallel()
	paths := implementationSeedCandidates("", "use the workspace", nil, nil)
	found := false
	for _, p := range paths {
		if p == "README.md" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected README.md in seeds, got %v", paths)
	}
}

func TestUserRequestsContentDelivery(t *testing.T) {
	t.Parallel()
	if !userRequestsContentDelivery("Can you create a linkedin article about this app?") {
		t.Fatal("expected linkedin article")
	}
	if userRequestsContentDelivery("use the workspace") {
		t.Fatal("workspace directive is not content delivery")
	}
}

func TestIsBareWorkspaceDirective(t *testing.T) {
	t.Parallel()
	if !isBareWorkspaceDirective("use the workspace") {
		t.Fatal("expected bare workspace directive")
	}
	if !isBareWorkspaceDirective("can you use the workspace for this?") {
		t.Fatal("expected deictic follow-up to count as bare")
	}
	if isBareWorkspaceDirective("use the workspace to implement dark mode") {
		t.Fatal("implement tail should not be bare")
	}
}

func TestResolveImplementationToolModel_prefersAgentCoderModel(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{AIModel: "qwen3.5:27b"}}
	if got := a.resolveImplementationToolModel("qwen3.5:9b"); got != "qwen3.5:9b" {
		t.Fatalf("got %q want 9b tool model for general qwen3.5 specialist", got)
	}
	aCoder := &Agent{Info: protocol.AgentInfo{AIModel: "qwen2.5-coder:14b"}}
	if got := aCoder.resolveImplementationToolModel("qwen3.5:9b"); got != "qwen2.5-coder:14b" {
		t.Fatalf("got %q want 14b coder", got)
	}
}

func TestShouldForceSessionSummaryRefresh(t *testing.T) {
	t.Parallel()
	if !ShouldForceSessionSummaryRefresh("approved") {
		t.Fatal("approval should force summary refresh")
	}
	if !ShouldForceSessionSummaryRefresh("can you review the code for issues?") {
		t.Fatal("code review should force summary refresh")
	}
}

func TestUserRequestsImplementation_settingsModalSession(t *testing.T) {
	t.Parallel()
	msg := "yesterday we were working on adding a settings modal and button and some options under it for font size and themes dark/light, can we pick up where we left off and finish that work?"
	if !userRequestsImplementation(msg) {
		t.Fatal("expected settings modal continuation to request implementation")
	}
	if !userAffirmsPendingImplementation("ok goahead") {
		t.Fatal("expected ok goahead to affirm implementation")
	}
}

func TestShouldProactiveScanWorkspace_implementationWithoutPaths(t *testing.T) {
	t.Parallel()
	if !shouldProactiveScanWorkspace("finish the settings modal with dark/light themes") {
		t.Fatal("expected proactive scan on implementation turn without explicit paths")
	}
}

func TestShouldProactiveScanWorkspaceForMessage_affirmationFollowUp(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-fe": {
					{
						ID:      "u1",
						Type:    protocol.MessageTypeQuestion,
						From:    protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
						Content: "add a settings modal with dark/light themes",
					},
				},
			},
		},
	}
	msg := &protocol.Message{
		ID:      "u2",
		Channel: "dm-u-fe",
		From:    protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
		Content: "ok goahead",
	}
	if !shouldProactiveScanWorkspaceForMessage(a, msg) {
		t.Fatal("expected affirmation follow-up to trigger workspace scan")
	}
}

func TestChannelRecentlyAppliedFilePaths(t *testing.T) {
	t.Parallel()
	agentID := "fe-1"
	history := []*protocol.Message{
		{
			ID:      "u-approve",
			Type:    protocol.MessageTypeChat,
			From:    protocol.AgentInfo{ID: "human-user", Name: "User", Type: "human"},
			Content: "Approved and applied your edit change to `/Users/camronwood/projects/dickory-docs/tailwind.config.js`. Continue with the implementation — do not ask me to approve again.",
			Metadata: map[string]interface{}{
				protocol.MetaFileChangeApproved: true,
				protocol.MetaFileChangePath:     "/Users/camronwood/projects/dickory-docs/tailwind.config.js",
				protocol.MetaFileChangeAgentID:  agentID,
			},
		},
	}
	got := channelRecentlyAppliedFilePaths(history, "", agentID)
	if len(got) == 0 {
		t.Fatal("expected applied paths")
	}
	found := false
	for _, p := range got {
		if p == "tailwind.config.js" || strings.HasSuffix(p, "tailwind.config.js") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tailwind.config.js in %v", got)
	}
}

func TestImplementationContinuation_skipsRecentlyAppliedPaths(t *testing.T) {
	t.Parallel()
	agentID := "fe-1"
	history := []*protocol.Message{
		{
			ID:      "u-approve",
			Type:    protocol.MessageTypeChat,
			From:    protocol.AgentInfo{ID: "human-user", Name: "User", Type: "human"},
			Content: "Approved and applied your edit change to `tailwind.config.js`. Continue with the implementation — do not ask me to approve again.",
			Metadata: map[string]interface{}{
				protocol.MetaFileChangeApproved: true,
				protocol.MetaFileChangePath:     "tailwind.config.js",
				protocol.MetaFileChangeAgentID:  agentID,
			},
		},
	}
	applied := channelRecentlyAppliedFilePaths(history, "u-next", agentID)
	dir := t.TempDir()
	writeImplementationReactFixture(t, dir)
	paths := implementationSeedCandidates(dir, "continue please", history, mergeAppliedPathsIntoExclude(nil, applied))
	for _, p := range paths {
		if p == "tailwind.config.js" {
			t.Fatalf("tailwind.config.js should be excluded after approval, got %v", paths)
		}
	}
	if len(paths) == 0 {
		t.Fatal("expected other frontend seed paths after excluding tailwind.config.js")
	}
}

func TestIsVagueImplementationContinuation(t *testing.T) {
	t.Parallel()
	if !isVagueImplementationContinuation("can you pick up where you left off?") {
		t.Fatal("expected vague continuation")
	}
	if isVagueImplementationContinuation("yesterday we were working on a settings modal, can we pick up where we left off and finish that work?") {
		t.Fatal("concrete continuation should not be vague-only")
	}
}

func TestImplementationSeedCandidates_fromHistory(t *testing.T) {
	t.Parallel()
	history := []*protocol.Message{
		{
			Type:    protocol.MessageTypeChat,
			From:    protocol.AgentInfo{Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
			Content: "src/components/SettingsModal.tsx src/components/MermaidModal.tsx",
		},
	}
	paths := implementationSeedCandidates("", "ok goahead", history, nil)
	found := false
	for _, p := range paths {
		if p == "src/components/SettingsModal.tsx" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected SettingsModal from history, got %v", paths)
	}
}

func TestAppendContentDeliverySeedFiles_loadsReadme(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Neural Junkie\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	n := AppendContentDeliverySeedFiles(&b, dir, nil)
	if n != 1 {
		t.Fatalf("expected 1 file loaded, got %d", n)
	}
	if !strings.Contains(b.String(), "Neural Junkie") {
		t.Fatalf("expected README content in prompt:\n%s", b.String())
	}
}

func TestShouldAugmentPromptWithWorkspace_contentDelivery(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Assistant", Type: protocol.AgentTypeAssistant},
	}
	msg := &protocol.Message{
		Content: "Can you write me an article about this app?",
		Metadata: map[string]interface{}{
			MetadataContextScope:   ContextScopeOutline,
			MetadataConversationMode: ConversationModeChat,
			"workspace_context": map[string]interface{}{
				"workspace_path": dirOrSkip(t),
			},
		},
	}
	if !a.shouldAugmentPromptWithWorkspace(IntentSubstantive, msg) {
		t.Fatal("content delivery with workspace context should augment prompt")
	}
}

func dirOrSkip(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	return d
}

func writeImplementationReactFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
  "name": "fixture",
  "scripts": { "build": "vite build", "dev": "vite" },
  "dependencies": { "react": "^18.2.0", "react-dom": "^18.2.0" },
  "devDependencies": { "vite": "^5.0.0", "typescript": "^5.0.0", "@vitejs/plugin-react": "^4.0.0" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ path, body string }{
		{"index.html", "<!doctype html><html><body><div id=\"root\"></div></body></html>"},
		{"src/main.tsx", "import App from './App'\nexport {}"},
		{"src/App.tsx", "export default function App() { return null }\n"},
		{"tailwind.config.js", "module.exports = { content: ['./src/**/*.{tsx,ts}'] }\n"},
	} {
		if err := os.WriteFile(filepath.Join(dir, item.path), []byte(item.body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestImplementationSeedCandidates_stackSeedsForAllAgents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeImplementationReactFixture(t, dir)
	paths := implementationSeedCandidates(dir, "the app is not booting can you fix it?", nil, nil)
	want := map[string]bool{"package.json": false, "src/main.tsx": false, "src/App.tsx": false}
	for _, p := range paths {
		if _, ok := want[p]; ok {
			want[p] = true
		}
	}
	for p, found := range want {
		if !found {
			t.Fatalf("expected stack seed %q in %v", p, paths)
		}
	}
}

func TestImplementationSeedCandidates_bootErrorAddsAppJS(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeImplementationReactFixture(t, dir)
	corrupt := "diff --git a/tailwind.config.js b/tailwind.config.js\n"
	if err := os.WriteFile(filepath.Join(dir, "src", "App.js"), []byte(corrupt), 0o644); err != nil {
		t.Fatal(err)
	}
	errLog := "✘ [ERROR] Expected \";\" but found \"git\"\n    src/App.js:1:7:\n      1 │ diff --git"
	paths := implementationSeedCandidates(dir, errLog, nil, nil)
	found := false
	for _, p := range paths {
		if p == "src/App.js" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected src/App.js in seeds, got %v", paths)
	}
}

func TestMessageHasBootOrBuildError(t *testing.T) {
	t.Parallel()
	if !messageHasBootOrBuildError("the app is not booting up can you help?") {
		t.Fatal("expected booting phrase")
	}
	if !messageHasBootOrBuildError("✘ [ERROR] Expected \";\" but found \"git\" in src/App.js") {
		t.Fatal("expected esbuild error")
	}
}

func TestShouldForceSessionSummaryRefresh_bootError(t *testing.T) {
	t.Parallel()
	if !ShouldForceSessionSummaryRefresh("vite dev failed with esbuild error in src/App.js") {
		t.Fatal("expected boot error log to force summary refresh")
	}
}

func TestTryImplementationStatusCheckShortcut_appJSRemoved(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "src/App.tsx", "export default function App() {}\n")
	a := &Agent{
		Info: protocol.AgentInfo{ID: "sa-1", Type: protocol.AgentTypeArchitecture, Name: "SoftwareArchitect"},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"implement-scenarios": {
					{
						ID:      "a1",
						Channel: "implement-scenarios",
						Type:    protocol.MessageTypeChat,
						From:    protocol.AgentInfo{ID: "sa-1", Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture},
						Content: "Implementation session complete — proposals submitted (changes to: src/App.js).",
						Metadata: map[string]interface{}{
							"workspace_context": map[string]interface{}{
								"workspace_path": dir,
							},
						},
					},
				},
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios", protocol.AgentInfo{ID: "u2", Name: "User"}, "is it fixed?")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": dir,
		},
	}
	resp, ok := a.tryImplementationStatusCheckShortcut(msg)
	if !ok || !strings.Contains(resp, "src/App.js") {
		t.Fatalf("expected shortcut reply, ok=%v resp=%q", ok, resp)
	}
}

func TestUserRequestsImplementationStatusCheck(t *testing.T) {
	t.Parallel()
	if !userRequestsImplementationStatusCheck("is it fixed?") {
		t.Fatal("expected status check")
	}
	if userRequestsImplementationStatusCheck("hello") {
		t.Fatal("greeting should not be status check")
	}
}

func TestScrubStaleSessionSummary(t *testing.T) {
	t.Parallel()
	transcript := "user: vite error\n✘ [ERROR] Expected ; but found git\nsrc/App.js:1:7"
	summary := "- Goal: fix boot\n- Key facts still needed: Specific error messages\n- Open questions: What happens when you try to start?"
	out := ScrubStaleSessionSummary(summary, transcript)
	if strings.Contains(strings.ToLower(out), "still needed") {
		t.Fatalf("expected stale bullets removed, got %q", out)
	}
	if !strings.Contains(out, "fix boot") {
		t.Fatalf("expected goal kept, got %q", out)
	}
}

func TestDetectFilePaths_esbuildLocation(t *testing.T) {
	t.Parallel()
	paths := DetectFilePaths("src/App.js:1:7:\n  1 │ diff --git")
	found := false
	for _, p := range paths {
		if p == "src/App.js" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected src/App.js from esbuild location, got %v", paths)
	}
}
