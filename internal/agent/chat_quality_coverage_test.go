package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// agentChannelCase exercises intent/mode for each builtin specialist across channel types.
type agentChannelCase struct {
	name        string
	agentType   protocol.AgentType
	channel     string
	channelType protocol.ChannelType
	content     string
	metadata    map[string]interface{}
	wantIntent  TurnIntent
	wantMode    string // empty = skip mode check
}

func TestChatQualityCoverage_agentChannels(t *testing.T) {
	cases := []agentChannelCase{
		// DM + code task
		{name: "backend_dm_review", agentType: protocol.AgentTypeBackend, channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			content: "review internal/hub/hub.go", wantIntent: IntentTask, wantMode: ConversationModeCode},
		{name: "frontend_dm_component", agentType: protocol.AgentTypeFrontend, channel: "dm-u-fe", channelType: protocol.ChannelTypeDM,
			content: "fix the React hook in src/App.tsx", wantIntent: IntentTask, wantMode: ConversationModeCode},
		{name: "security_dm_audit", agentType: protocol.AgentTypeSecurity, channel: "dm-u-sec", channelType: protocol.ChannelTypeDM,
			content: "audit cmd/server/main.go for auth issues", wantIntent: IntentTask},
		{name: "architecture_dm_outline", agentType: protocol.AgentTypeArchitecture, channel: "dm-u-arch", channelType: protocol.ChannelTypeDM,
			content: "What is the architecture of this repo?", wantIntent: IntentLowSignal},
		{name: "architecture_dm_outline_long", agentType: protocol.AgentTypeArchitecture, channel: "dm-u-arch", channelType: protocol.ChannelTypeDM,
			content: "What is the high-level architecture of this repository and how are packages organized?", wantIntent: IntentSubstantive},
		{name: "code_review_dm_path", agentType: protocol.AgentTypeCodeReview, channel: "dm-u-cr", channelType: protocol.ChannelTypeDM,
			content: "review internal/agent/agent.go", wantIntent: IntentTask},
		{name: "devops_dm_deploy", agentType: protocol.AgentTypeDevOps, channel: "dm-u-pe", channelType: protocol.ChannelTypeDM,
			content: "debug the Dockerfile in deploy/", wantIntent: IntentTask},
		{name: "database_dm_schema", agentType: protocol.AgentTypeDatabase, channel: "dm-u-db", channelType: protocol.ChannelTypeDM,
			content: "review migrations/schema.sql", wantIntent: IntentTask},
		{name: "biology_dm_sequence", agentType: protocol.AgentTypeBiology, channel: "dm-u-bio", channelType: protocol.ChannelTypeDM,
			content: "summarize_scan_analysis on reports/results.json", wantIntent: IntentTask},

		// Public channel
		{name: "backend_public_task", agentType: protocol.AgentTypeBackend, channel: "general", channelType: protocol.ChannelTypePublic,
			content: "refactor internal/agent/turn_intent.go", wantIntent: IntentTask},
		{name: "assistant_public_chat", agentType: protocol.AgentTypeAssistant, channel: "general", channelType: protocol.ChannelTypePublic,
			content: "what do you think about rust?", metadata: metaChat(ContextScopeNone), wantIntent: IntentSubstantive, wantMode: ConversationModeChat},
		{name: "assistant_public_chat_mid_opinion", agentType: protocol.AgentTypeAssistant, channel: "general", channelType: protocol.ChannelTypePublic,
			content: "what do you think about go vs rust?", metadata: metaChat(ContextScopeNone), wantIntent: IntentLowSignal, wantMode: ConversationModeChat},

		// Regression: workspace + confusion (all agent types should classify the same)
		{name: "workspace_visibility_backend", agentType: protocol.AgentTypeBackend, channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			content: "can you see my workspace I have open?", wantIntent: IntentSubstantive},
		{name: "workspace_visibility_frontend", agentType: protocol.AgentTypeFrontend, channel: "dm-u-fe", channelType: protocol.ChannelTypeDM,
			content: "can you see my workspace I have open?", wantIntent: IntentSubstantive},
		{name: "workspace_visibility_security", agentType: protocol.AgentTypeSecurity, channel: "dm-u-sec", channelType: protocol.ChannelTypeDM,
			content: "can you see my workspace I have open?", wantIntent: IntentSubstantive},
		{name: "workspace_visibility_architecture", agentType: protocol.AgentTypeArchitecture, channel: "dm-u-arch", channelType: protocol.ChannelTypeDM,
			content: "can you see my workspace I have open?", wantIntent: IntentSubstantive},
		{name: "workspace_visibility_code_review", agentType: protocol.AgentTypeCodeReview, channel: "dm-u-cr", channelType: protocol.ChannelTypeDM,
			content: "can you see my workspace I have open?", wantIntent: IntentSubstantive},
		{name: "workspace_visibility_devops", agentType: protocol.AgentTypeDevOps, channel: "dm-u-pe", channelType: protocol.ChannelTypeDM,
			content: "can you see my workspace I have open?", wantIntent: IntentSubstantive},
		{name: "workspace_visibility_database", agentType: protocol.AgentTypeDatabase, channel: "dm-u-db", channelType: protocol.ChannelTypeDM,
			content: "can you see my workspace I have open?", wantIntent: IntentSubstantive},
		{name: "short_confusion_dm", agentType: protocol.AgentTypeFrontend, channel: "dm-u-fe", channelType: protocol.ChannelTypeDM,
			content: "What?", wantIntent: IntentSubstantive},
		{name: "short_confusion_backend_dm", agentType: protocol.AgentTypeBackend, channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			content: "What?", wantIntent: IntentSubstantive},
		{name: "theme_task_dm", agentType: protocol.AgentTypeBackend, channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			content: "I want to add theme support to this app", wantIntent: IntentTask},
		{name: "deep_continuation_dm", agentType: protocol.AgentTypeBackend, channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			content: "go deeper on the approach — what would you implement first?", wantIntent: IntentTask},
		{name: "topic_switch_chat_turn_dm", agentType: protocol.AgentTypeBackend, channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			content: "what do you think about go vs rust for backend services?", metadata: metaChat(ContextScopeNone), wantIntent: IntentSubstantive, wantMode: ConversationModeChat},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := protocol.NewMessage(protocol.MessageTypeQuestion, tc.channel, protocol.AgentInfo{ID: "u", Name: "User"}, tc.content)
			if tc.metadata != nil {
				msg.Metadata = tc.metadata
			}
			got := ClassifyTurnIntentPublic(msg, tc.channelType, "agent-1", nil)
			if got != tc.wantIntent {
				t.Fatalf("intent: got %s want %s", got.String(), tc.wantIntent.String())
			}
			if tc.wantMode != "" {
				mode := EffectiveConversationMode(msg, tc.channelType)
				if mode != tc.wantMode {
					t.Fatalf("mode: got %q want %q", mode, tc.wantMode)
				}
			}
			_ = tc.agentType
		})
	}
}

func TestChatQualityCoverage_scanShortcutNoBareSummarize(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "BiologyExpert", Type: protocol.AgentTypeBiology}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Name: "User"}, "can you summarize IL-6?")
	msg.Metadata = map[string]interface{}{
		MetadataContextScope: ContextScopeFocus,
		"workspace_context": map[string]interface{}{
			"scan_analysis": map[string]interface{}{"analysis_dir": "/tmp/export"},
		},
	}
	if got := a.resolveBiologyScanToolForTurn(msg); got != "" {
		t.Fatalf("got %q want no shortcut", got)
	}
}
