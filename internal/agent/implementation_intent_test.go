package agent

import (
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

func TestResolveImplementationToolModel_prefersAgentCoderModel(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{AIModel: "qwen2.5-coder:14b"}}
	if got := a.resolveImplementationToolModel("qwen2.5-coder:7b"); got != "qwen2.5-coder:14b" {
		t.Fatalf("got %q want 14b", got)
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
	paths := implementationSeedCandidates(protocol.AgentTypeFrontend, "continue please", history, mergeAppliedPathsIntoExclude(nil, applied))
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
	paths := implementationSeedCandidates(protocol.AgentTypeFrontend, "ok goahead", history, nil)
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
