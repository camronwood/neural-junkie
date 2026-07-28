package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldRunImplementationSession(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeBackend, Name: "BackendEngineer"}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{ID: "u1", Name: "User"}, "please implement a health check handler")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"ide_route_agent_type":   "backend",
		"implementation_session": true,
	}
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("expected implementation session")
	}
	msg.Metadata["editor_mode"] = "ask"
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("ask mode should not run session")
	}
	msg.Metadata["editor_mode"] = "plan"
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("plan mode should not run session")
	}
}

func TestShouldRunImplementationSession_explicitSessionWinsOverAnswerDecision(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "implement-scenarios",
		protocol.AgentInfo{ID: "u1", Name: "User"},
		"the app won't boot — make start-all fails:\n```\n$ make start-all\nmake: *** No rule to make target 'start-all'.  Stop.\n```")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"ide_route_agent_type":   "frontend",
		"implementation_session": true,
		"editor_agent_trust":     "auto_apply_edits",
		"conversation_mode":      "code",
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Interaction:     intent.InteractionTask,
		RequestedAction: intent.ActionAnswer,
		Action:          intent.ActionAnswer,
		Mutation:        intent.MutationNone,
		Confidence:      0.9,
		Source:          intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("implementation_session metadata must win over semantic ActionAnswer")
	}
}

// TestShouldRunImplementationSession_continuationAfterFileChange documents the stamp-first
// replacement for the old phrase-based "approved re-enters an active implementation thread"
// heuristic: a classifier that stamps the approval turn ActionContinue is trusted directly —
// routing no longer re-scans channel history for affirmation phrasing without a stamp.
func TestShouldRunImplementationSession_continuationAfterFileChange(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-fe": {
					{
						ID:      "fc1",
						Type:    protocol.MessageTypeFileChange,
						From:    protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
						Content: "edit src/App.tsx",
					},
				},
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-fe", protocol.AgentInfo{ID: "u2", Name: "User"}, "approved")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"implementation_session": true,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionContinue, Action: intent.ActionContinue,
		Mutation: intent.MutationWorkspace, Confidence: 0.9, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("expected implementation session after a stamped ActionContinue approval")
	}
}

func TestShouldRunImplementationSession_vagueContinuationWithoutThread(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-fe": {
					{
						ID:      "join",
						Type:    protocol.MessageTypeAgentJoin,
						From:    protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
						Content: "FrontendEngineer has joined the channel",
					},
				},
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-fe", protocol.AgentInfo{ID: "u1", Name: "User"}, "can you pick up where you left off?")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"conversation_mode":      "code",
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("vague continuation without prior thread should not run implementation session")
	}
}

func TestShouldRunImplementationSession_scenarioChannelForce(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"user-flow-scenarios": {
					{
						ID:      "join",
						Type:    protocol.MessageTypeAgentJoin,
						From:    protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
						Content: "FrontendEngineer has joined the channel",
					},
				},
			},
		},
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"user-flow-scenarios",
		protocol.AgentInfo{ID: "u1", Name: "User"},
		"can you pick up where you left off?",
	)
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"implementation_session": true,
		"conversation_mode":      "code",
	}
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("scenario channel with explicit implementation_session should force session")
	}

	msg2 := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "u1", Name: "User"},
		"can you pick up where you left off?",
	)
	msg2.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"implementation_session": true,
		"conversation_mode":      "code",
	}
	if shouldRunImplementationSession(a, msg2) {
		t.Fatal("non-scenario channel should still block vague continuation without thread")
	}
}

// Note: the old TestShouldRunImplementationSession_weakAffirmAfterFailedSession relied on a
// phrase-based "weak affirmation after a failed session" veto (userAffirmsPendingImplementation
// combined with channel-history phrase scanning) to keep a bare "looks good" from re-entering
// the file-edit loop. That heuristic is deprecated — see implementation_intent.go. Routing now
// trusts the stamped TurnDecision (or, with no stamp, the explicit implementation_session flag)
// instead of re-deriving continuation intent from affirmation phrasing.

func TestShouldRunImplementationSession_respectsChatMode(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "asst", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"general": {
					{
						Type:    protocol.MessageTypeQuestion,
						From:    protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
						Content: "how would you add a light/dark theme toggle in a React settings page?",
					},
					{
						Type:    protocol.MessageTypeChat,
						From:    protocol.AgentInfo{ID: "asst", Name: "Assistant", Type: protocol.AgentTypeAssistant},
						Content: "Use useState and a theme context provider.",
					},
					{
						Type:    protocol.MessageTypeQuestion,
						From:    protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
						Content: "ok thanks",
					},
					{
						Type:    protocol.MessageTypeChat,
						From:    protocol.AgentInfo{ID: "asst", Name: "Assistant", Type: protocol.AgentTypeAssistant},
						Content: "You're welcome! Let me know if you need anything else.",
					},
				},
			},
		},
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
		"One more thing — where should the theme toggle live in the settings UI?",
	)
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		MetadataContextScope:     ContextScopeNone,
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("chat-mode theme advice should not run implementation session")
	}
}

func TestShouldRunImplementationSession_chatModeBlocksSemanticEdit(t *testing.T) {
	a := &Agent{
		Info:    protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
		Context: &ConversationContext{History: map[string][]*protocol.Message{}},
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-chatscenario-frontendengineer",
		protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
		"Design a theme settings flow. Keep the toggle in an Appearance section and call the component ThemeSettings.",
	)
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		MetadataContextScope:     ContextScopeNone,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionEdit, Action: intent.ActionEdit,
		Mutation: intent.MutationWorkspace, Confidence: 0.9, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: true, CanRunImplSession: true, RequiresWorkspace: true,
	})
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("conversation_mode=chat must block semantic ActionEdit implementation sessions")
	}
}

func TestShouldRunImplementationSession_statusCheckInChatMode(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "sa-1", Type: protocol.AgentTypeArchitecture, Name: "SoftwareArchitect"},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-sa": {
					{
						ID:      "u1",
						Type:    protocol.MessageTypeQuestion,
						From:    protocol.AgentInfo{ID: "u0", Name: "User"},
						Content: "the app is not booting can you fix it?",
					},
					{
						ID:      "a1",
						Type:    protocol.MessageTypeChat,
						From:    protocol.AgentInfo{ID: "sa-1", Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture},
						Content: "Implementation session complete — proposals submitted for approval (changes to: src/App.js).",
					},
				},
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-sa", protocol.AgentInfo{ID: "u2", Name: "User"}, "is it fixed?")
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		"editor_mode":            "agent",
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("status check follow-up should use conversational reply, not implementation session")
	}
}

func TestLooksLikeListDirToolEcho(t *testing.T) {
	t.Parallel()
	echo := "Implementation session finished without file changes.\n\nApp.js (file)\nApp.tsx (file)\nmain.tsx (file)\ncomponents (dir)"
	if !looksLikeListDirToolEcho(echo) {
		t.Fatal("expected list_dir echo detection")
	}
}

func TestShouldRunImplementationSession_exportMode(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "asst", Name: "Assistant", Type: protocol.AgentTypeAssistant}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u", protocol.AgentInfo{ID: "u1", Name: "User"}, "please save it")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "export",
		"implementation_session": true,
	}
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("expected export composer mode to run implementation session")
	}
}

func TestShouldRunImplementationSession_assistantFlightExportMetadata(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "asst", Name: "Assistant", Type: protocol.AgentTypeAssistant}}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "u1", Name: "User"},
		"Can you check flight times? I need to plan a trip from St. Louis, MO to Chicago, IL.",
	)
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "export",
		"composer_mode":          "export",
		"implementation_session": true,
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("personal travel question must not run implementation session from export metadata alone")
	}
}

func TestDetectVerifyCommands_go(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmds := detectVerifyCommands(dir)
	if len(cmds) != 1 || cmds[0] != "go test ./..." {
		t.Fatalf("got %v", cmds)
	}
}

func TestDetectVerifyCommands_nodeBuild(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts":{"build":"vite build","test":"node test.js"}}`)
	writeFile(t, dir, "tsconfig.json", `{}`)
	if err := os.Mkdir(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmds := detectVerifyCommands(dir)
	if len(cmds) < 2 {
		t.Fatalf("expected build + test, got %v", cmds)
	}
	if cmds[0] != "npm run build" {
		t.Fatalf("first cmd: got %q", cmds[0])
	}
}

func TestDetectVerifyCommands_nodeWithoutModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts":{"build":"vite build"},"devDependencies":{"typescript":"5.0.0"}}`)
	writeFile(t, dir, "tsconfig.json", `{}`)
	cmds := detectVerifyCommands(dir)
	if len(cmds) != 0 {
		t.Fatalf("should skip node verify without node_modules, got %v", cmds)
	}
}

func TestDetectVerifyCommands_cargoBuild(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname=\"test\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmds := detectVerifyCommands(dir)
	if len(cmds) != 1 || cmds[0] != "cargo build" {
		t.Fatalf("got %v", cmds)
	}
}

func TestDetectVerifyCommands_cargoBuildWithRustHints(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", minimalCargoTomlBody("demo"))
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "src/main.rs", "fn main() {}\n")
	state := &ImplementationSessionState{FilesChanged: []string{"src/main.rs"}, RegisteredFiles: []string{"src/main.rs"}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "user-flow-scenarios", protocol.AgentInfo{}, "Rust CLI with src/main.rs")
	cmds := detectVerifyCommandsForSession(dir, state, msg)
	if len(cmds) != 1 || cmds[0] != "cargo build" {
		t.Fatalf("got %v", cmds)
	}
}

func TestDetectVerifyCommands_hybridGoWithoutMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts":{"build":"react-scripts build","test":"react-scripts test"}}`)
	writeFile(t, dir, "core/sample/main.go", "package main\nfunc main() {}\n")
	writeFile(t, dir, "core/server/main.go", "package main\nfunc main() {}\n")
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	state := &ImplementationSessionState{FilesChanged: []string{"core/sample/main.go"}, RegisteredFiles: []string{"core/sample/main.go"}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios", protocol.AgentInfo{}, "please implement a HelloWorld function in core/sample/main.go and call it from main")
	cmds := detectVerifyCommandsForSession(dir, state, msg)
	if len(cmds) != 1 || cmds[0] != "go build ./core/sample" {
		t.Fatalf("expected go build for edited package only, got %v", cmds)
	}
}

func TestDetectVerifyCommands_nodeSafeTestFlags(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts":{"build":"vite build","test":"node test.js"}}`)
	if err := os.Mkdir(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmds := detectVerifyCommands(dir)
	foundTest := false
	for _, c := range cmds {
		if strings.Contains(c, "npm test") {
			foundTest = true
			if !strings.Contains(c, "CI=true") || !strings.Contains(c, "watchAll=false") || !strings.Contains(c, "passWithNoTests") {
				t.Fatalf("npm test should use CI-safe flags, got %q", c)
			}
		}
	}
	if !foundTest {
		t.Fatalf("expected npm test command, got %v", cmds)
	}
}

func TestShouldSkipVerifyRepairAfterAutoApply(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "core/sample/main.go", "package main\nfunc main() {}\n")
	a := &Agent{WorkspacePath: dir}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios", protocol.AgentInfo{ID: "be", Name: "BackendEngineer"}, "edit go")
	msg.Metadata = map[string]interface{}{"editor_agent_trust": editorTrustAutoApply}
	state := &ImplementationSessionState{
		VerifyFailed:              true,
		FilesChanged:              []string{"core/sample/main.go"},
		DeterministicFallbackUsed: true,
		TrustMode:                 editorTrustAutoApply,
	}
	if !a.shouldSkipVerifyRepairAfterAutoApply(msg, state) {
		t.Fatal("expected skip after deterministic auto-apply")
	}
	state.DeterministicFallbackUsed = false
	if !a.shouldSkipVerifyRepairAfterAutoApply(msg, state) {
		t.Fatal("expected skip for Go-only auto-apply without fix-like intent")
	}
	state.FixLikeIntent = true
	if a.shouldSkipVerifyRepairAfterAutoApply(msg, state) {
		t.Fatal("fix-like intent should not skip repair")
	}
}

func TestGroundingSatisfied(t *testing.T) {
	t.Parallel()
	st := &ImplementationSessionState{StackManifest: &StackManifest{EntryPoint: "src/App.tsx"}}
	if st.groundingSatisfied() {
		t.Fatal("manifest detection alone must not satisfy grounding")
	}
	st2 := &ImplementationSessionState{}
	if st2.groundingSatisfied() {
		t.Fatal("empty state should not satisfy grounding")
	}
	st3 := &ImplementationSessionState{SeedsLoaded: 1}
	if !st3.groundingSatisfied() {
		t.Fatal("one seed file should satisfy grounding")
	}
}

func TestBuildImplementationSessionOutcome(t *testing.T) {
	a := &Agent{}
	state := &ImplementationSessionState{
		RepairUsed:    true,
		VerifyFailed:  false,
		VerifySkipped: false,
		FilesChanged:  []string{"core/sample/math.go"},
	}
	outcome := a.buildImplementationSessionOutcome(nil, state, true)
	if outcome["repair_used"] != true {
		t.Fatalf("repair_used=%v", outcome["repair_used"])
	}
	if outcome["verify_failed"] != false {
		t.Fatalf("verify_failed=%v", outcome["verify_failed"])
	}
	if outcome["outcome"] == "no_changes" {
		t.Fatalf("expected non-no_changes outcome, got %v", outcome["outcome"])
	}
	empty := a.buildImplementationSessionOutcome(nil, nil, false)
	if empty["outcome"] != "no_changes" {
		t.Fatalf("empty outcome=%v", empty["outcome"])
	}
}

func TestAppendCollabExecutionTaskStatus(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-ch", protocol.AgentInfo{ID: "be", Name: "BackendEngineer"}, "Write collabs/x/findings.md")
	msg.SetCollaborationID("collab-1111")
	msg.SetCollaborationPhase("executing")
	msg.SetTaskID("task-1")
	state := &ImplementationSessionState{RegisteredFiles: []string{"collabs/x/findings.md"}}

	got := appendCollabExecutionTaskStatus("Implementation session complete — proposals submitted for approval.", msg, state, true)
	if !strings.Contains(got, "TASK_STATUS: completed") {
		t.Fatalf("expected TASK_STATUS line, got %q", got)
	}
	if !strings.Contains(got, "findings.md") {
		t.Fatalf("expected shipped paths in summary, got %q", got)
	}

	already := strings.TrimSpace("Done.\nTASK_STATUS: completed\n")
	if appendCollabExecutionTaskStatus(already, msg, state, true) != already {
		t.Fatal("should not duplicate TASK_STATUS when already present")
	}

	chat := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{}, "fix bug")
	if appendCollabExecutionTaskStatus("done", chat, state, true) != "done" {
		t.Fatal("non-collab message should be unchanged")
	}
}

func TestAppendUnique(t *testing.T) {
	got := appendUnique([]string{"a"}, []string{"b", "a"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
}

func TestShouldRunImplementationSession_planBeatsSemanticEdit(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeBackend, Name: "BackendEngineer"}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{ID: "u1", Name: "User"}, "Plan how to add HelloWorld")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "plan",
		"composer_mode":          "plan",
		"implementation_session": true,
	}
	_ = protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion,
		Action:        intent.ActionEdit,
		Mutation:      intent.MutationWorkspace,
		Interaction:   intent.InteractionTask,
		Confidence:    1,
		Source:        "test",
	})
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("plan mode must not run implementation session under semantic edit")
	}
	if !isAskModeReadOnly(msg) {
		t.Fatal("plan mode must be read-only for file proposals")
	}
}

func TestDeriveTurnGoalFromDecision_respectsExplicitImplementationSession(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{ID: "u1", Name: "User"}, "fix the Multiply bug")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"composer_mode":          "agent",
		"implementation_session": true,
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: true, CanRunImplSession: true, RequiresWorkspace: false,
		Provenance: "test",
	})
	decision := intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion,
		Action:        intent.ActionEdit,
		Mutation:      intent.MutationWorkspace,
		Interaction:   intent.InteractionTask,
		Confidence:    1,
		Source:        "test",
	}
	_ = protocol.StampTurnDecision(msg, decision)
	goal := deriveTurnGoalFromDecision(msg, decision)
	if !goal.ImplementationSession {
		t.Fatal("expected implementation session when metadata requests it and caps allow")
	}
}

// TestDeriveTurnGoalFromDecision_stampedInspectStaysInspectDespiteBootFix documents the
// stamp-first replacement for the old bootOrFix upgrade block: explicit implementation_session
// metadata only promotes a turn into a running implementation session when the classifier
// already selected a workspace-mutation action (Edit/Debug/Continue/Run). A stamped
// ActionInspect never gets phrase/reason-matched into Debug — even with a "runtime_failure"
// reason code and explicit session metadata.
func TestDeriveTurnGoalFromDecision_stampedInspectStaysInspectDespiteBootFix(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"implement-scenarios",
		protocol.AgentInfo{ID: "u1", Name: "User"},
		"the app won't boot — make start-all fails:\n\n```\n$ make start-all\nmake: *** No rule to make target 'start-all'.  Stop.\n```",
	)
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"composer_mode":          "agent",
		"implementation_session": true,
		"conversation_mode":      "code",
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: true, CanRunImplSession: true, RequiresWorkspace: true,
		Provenance: "test",
	})
	decision := intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Action:          intent.ActionInspect,
		RequestedAction: intent.ActionInspect,
		Mutation:        intent.MutationNone,
		Interaction:     intent.InteractionQuestion,
		Confidence:      1,
		Source:          "local_model",
		ReasonCodes:     []string{"runtime_failure"},
	}
	_ = protocol.StampTurnDecision(msg, decision)
	goal := deriveTurnGoalFromDecision(msg, decision)
	if goal.Action != ActionInspect {
		t.Fatalf("action=%s want ActionInspect (stamp authoritative)", goal.Action)
	}
	if goal.Mutation != MutationNone {
		t.Fatalf("mutation=%s want none — ActionInspect must not gain workspace mutation", goal.Mutation)
	}
	if goal.ImplementationSession {
		t.Fatal("ActionInspect must never be promoted into an implementation session")
	}
	if turnGoalRunsImplementationSession(goal) {
		t.Fatal("turnGoalRunsImplementationSession should be false for a stamped ActionInspect")
	}
}

// TestDeriveTurnGoalFromDecision_stampedAnswerStaysAnswerDespiteBootFix documents the
// stamp-first replacement for the old bootOrFix upgrade block: a stamped ActionAnswer
// never gets phrase-matched into Debug/Edit, even for boot-error message text with an
// active implementation_session flag.
func TestDeriveTurnGoalFromDecision_stampedAnswerStaysAnswerDespiteBootFix(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"implement-scenarios",
		protocol.AgentInfo{ID: "u1", Name: "User"},
		"the app in this workspace is not booting up can you help me fix it?\n\nvite dev error log:\n\n✘ [ERROR] Expected \";\" but found \"git\"\n\n    src/App.js:1:7:\n      1 │ diff --git a/tailwind.config.js b/tailwind.config.js",
	)
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"composer_mode":          "agent",
		"implementation_session": true,
		"conversation_mode":      "code",
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: true, CanRunImplSession: true, RequiresWorkspace: true,
		Provenance: "test",
	})
	decision := intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion,
		Action:        intent.ActionAnswer,
		Mutation:      intent.MutationNone,
		Interaction:   intent.InteractionQuestion,
		Confidence:    1,
		Source:        "local_model",
	}
	_ = protocol.StampTurnDecision(msg, decision)
	goal := deriveTurnGoalFromDecision(msg, decision)
	if goal.Action != ActionAnswer {
		t.Fatalf("action=%s want ActionAnswer (stamp authoritative)", goal.Action)
	}
	if goal.ImplementationSession || goal.Mutation != MutationNone {
		t.Fatalf("goal=%+v, ActionAnswer must never be promoted into an implementation session", goal)
	}
}

// TestDeriveTurnGoalFromDecision_stampedAnswerStaysAnswerForExtractRequest documents the
// stamp-first replacement for the old bootOrFix upgrade block applied to extract/refactor
// asks: a stamped ActionAnswer stays advisory even for an explicit implementation_session
// flag — the classifier must stamp ActionEdit directly to run the file-edit loop.
func TestDeriveTurnGoalFromDecision_stampedAnswerStaysAnswerForExtractRequest(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"implement-scenarios",
		protocol.AgentInfo{ID: "u1", Name: "User"},
		"@FrontendEngineer Extract the selected sidebar footer block from src/App.tsx into src/components/SidebarFooter.tsx and import it in App.",
	)
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"composer_mode":          "agent",
		"implementation_session": true,
		"conversation_mode":      "code",
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: true, CanRunImplSession: true, RequiresWorkspace: true,
		Provenance: "test",
	})
	decision := intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion,
		Action:        intent.ActionAnswer,
		Mutation:      intent.MutationNone,
		Interaction:   intent.InteractionQuestion,
		Confidence:    1,
		Source:        "local_model",
	}
	_ = protocol.StampTurnDecision(msg, decision)
	goal := deriveTurnGoalFromDecision(msg, decision)
	if goal.Action != ActionAnswer {
		t.Fatalf("action=%s want ActionAnswer (stamp authoritative)", goal.Action)
	}
	if goal.ImplementationSession {
		t.Fatal("ActionAnswer must never be promoted into an implementation session")
	}
}

func TestDeriveTurnGoalFromDecision_explicitSessionUpgradesRunBootFix(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"implement-scenarios",
		protocol.AgentInfo{ID: "u1", Name: "User"},
		"the app won't boot — make start-all fails:\n\n```\n$ make start-all\nmake: *** No rule to make target 'start-all'.  Stop.\n```",
	)
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"composer_mode":          "agent",
		"implementation_session": true,
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: true, CanRunImplSession: true, RequiresWorkspace: true,
		Provenance: "test",
	})
	decision := intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion,
		Action:        intent.ActionRun,
		Mutation:      intent.MutationNone,
		Interaction:   intent.InteractionCasual,
		Confidence:    1,
		Source:        "local_model",
		ReasonCodes:   []string{"runtime_failure"},
	}
	_ = protocol.StampTurnDecision(msg, decision)
	goal := deriveTurnGoalFromDecision(msg, decision)
	if goal.Action != ActionRun {
		t.Fatalf("action=%s want ActionRun (stamp authoritative)", goal.Action)
	}
	if !goal.ImplementationSession || goal.Mutation != MutationWorkspace {
		t.Fatalf("goal=%+v", goal)
	}
}
