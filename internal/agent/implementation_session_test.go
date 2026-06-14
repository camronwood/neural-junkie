package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
}

func TestShouldRunImplementationSession_continuationAfterFileChange(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
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
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-fe", protocol.AgentInfo{ID: "u2", Name: "User"}, "approved")
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("expected implementation session after approval in active thread")
	}
}

func TestShouldRunImplementationSession_vagueContinuationWithoutThread(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-fe": {
					{
						ID:   "join",
						Type: protocol.MessageTypeAgentJoin,
						From: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
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

func TestShouldRunImplementationSession_weakAffirmAfterFailedSession(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "fe-1", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-fe": {
					{
						ID:      "u1",
						Type:    protocol.MessageTypeQuestion,
						From:    protocol.AgentInfo{ID: "u0", Name: "User"},
						Content: "blank screen can you fix it?",
					},
					{
						ID:      "a1",
						Type:    protocol.MessageTypeChat,
						From:    protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
						Content: "Implementation session finished without file changes.",
					},
				},
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-fe", protocol.AgentInfo{ID: "u2", Name: "User"}, "looks good")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"implementation_session":   true,
		"ide_route_agent_type":   "frontend",
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("expected looks good after failed session not to run implementation session")
	}
}

func TestShouldRunImplementationSession_respectsChatMode(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "asst", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"general": {
					{
						Type:    protocol.MessageTypeQuestion,
						From:    protocol.AgentInfo{ID: "u", Name: "User"},
						Content: "how would you add a light/dark theme toggle in a React settings page?",
					},
					{
						Type:    protocol.MessageTypeChat,
						From:    protocol.AgentInfo{ID: "asst", Name: "Assistant", Type: protocol.AgentTypeAssistant},
						Content: "Use useState and a theme context provider.",
					},
				},
			},
		},
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "u", Name: "User"},
		"One more thing — where should the theme toggle live in the settings UI?",
	)
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		MetadataContextScope:   ContextScopeNone,
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("chat-mode theme advice should not run implementation session")
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
	for _, c := range cmds {
		if strings.Contains(c, "npm run build") {
			t.Fatalf("should not npm build without node_modules, got %v", cmds)
		}
	}
	if len(cmds) == 0 {
		t.Fatal("expected tsc/npm exec fallback when node_modules missing but TS dep declared")
	}
}

func TestGroundingSatisfied(t *testing.T) {
	t.Parallel()
	st := &ImplementationSessionState{StackManifest: &StackManifest{EntryPoint: "src/App.tsx"}}
	if !st.groundingSatisfied() {
		t.Fatal("entry point should satisfy grounding")
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

func TestAppendUnique(t *testing.T) {
	got := appendUnique([]string{"a"}, []string{"b", "a"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
}
