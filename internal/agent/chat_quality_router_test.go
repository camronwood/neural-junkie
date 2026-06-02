package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// chatQualityCase exercises the Conversation Context Stack router (mode, intent, closure, tooling, history, persona).
type chatQualityCase struct {
	name        string
	content     string
	channel     string
	channelType protocol.ChannelType
	metadata    map[string]interface{}
	history     []*protocol.Message
	agentID     string

	checkIntent     bool
	wantIntent      TurnIntent
	checkClosure    bool
	wantClosure     ClosureKind
	checkMode       bool
	wantMode        string
	checkTooling    bool
	wantTooling     bool
	toolingIntent   TurnIntent
	checkHistoryCap bool
	wantHistoryCap  int
	hasSummary      bool
	historyIntent   TurnIntent
	checkPersona       bool
	wantPersona        PromptPersonaTier
	useMultiAgentHub   bool
}

type multiAgentChannelHub struct {
	shouldRespondTestHub
}

func (multiAgentChannelHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) {
	return []protocol.AgentInfo{
		{Name: "Assistant", Status: "active"},
		{Name: "BackendEngineer", Status: "active"},
	}, nil
}

func buildChatHistory(agentID, priorUser, priorAgent string) []*protocol.Message {
	if priorUser == "" && priorAgent == "" {
		return nil
	}
	var out []*protocol.Message
	if priorUser != "" {
		out = append(out, protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test", protocol.AgentInfo{ID: "u", Name: "User"}, priorUser))
	}
	if priorAgent != "" {
		m := protocol.NewMessage(protocol.MessageTypeChat, "dm-test", protocol.AgentInfo{ID: agentID, Name: "Assistant"}, priorAgent)
		out = append(out, m)
	}
	return out
}

func chatMsg(content, channel string, chType protocol.ChannelType, meta map[string]interface{}) *protocol.Message {
	ch := channel
	if ch == "" {
		ch = "general"
	}
	msg := protocol.NewMessage(protocol.MessageTypeChat, ch, protocol.AgentInfo{ID: "u", Name: "User"}, content)
	if chType == protocol.ChannelTypeDM || chType == "" && len(ch) > 3 && ch[:3] == "dm-" {
		// channel name only; caller sets channelType explicitly
	}
	if meta != nil {
		msg.Metadata = meta
	}
	return msg
}

func metaChat(scope string) map[string]interface{} {
	m := map[string]interface{}{MetadataConversationMode: ConversationModeChat}
	if scope != "" {
		m[MetadataContextScope] = scope
	}
	return m
}

func TestChatQualityRouter(t *testing.T) {
	agentID := "asst-1"
	cases := []chatQualityCase{
		// --- Closure ---
		{name: "closure_thanks", content: "ok thanks", channel: "dm-u-a", channelType: protocol.ChannelTypeDM, agentID: agentID,
			checkIntent: true, wantIntent: IntentClosure, checkClosure: true, wantClosure: ClosureThanks},
		{name: "closure_thank_you", content: "Thank you!", channel: "dm-u-a", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentClosure, checkClosure: true, wantClosure: ClosureThanks},
		{name: "closure_already_said", content: "I know you said that already", channel: "dm-u-a", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentClosure, checkClosure: true, wantClosure: ClosureAlreadyAnswered},
		{name: "closure_already_said_with_question", content: "you already told me — what about traffic?", channel: "dm-u-a", channelType: protocol.ChannelTypeDM,
			checkClosure: true, wantClosure: ClosureNone, checkIntent: true, wantIntent: IntentSubstantive},
		{name: "closure_brief_no_prior", content: "cool", channel: "dm-u-a", channelType: protocol.ChannelTypeDM, agentID: agentID,
			checkIntent: true, wantIntent: IntentLowSignal, checkClosure: true, wantClosure: ClosureBriefAck},
		{name: "closure_brief_with_prior", content: "cool", channel: "dm-u-a", channelType: protocol.ChannelTypeDM, agentID: agentID,
			history: buildChatHistory(agentID, "distance?", "39 miles"),
			checkIntent: true, wantIntent: IntentClosure, checkClosure: true, wantClosure: ClosureBriefAck},
		{name: "closure_got_it_after_answer", content: "got it", channel: "dm-u-a", channelType: protocol.ChannelTypeDM, agentID: agentID,
			history: buildChatHistory(agentID, "explain REST", "REST is an architectural style"),
			checkIntent: true, wantIntent: IntentClosure},

		// --- Chat mode ---
		{name: "chat_mode_opinion", content: "what do you think about golang?", channel: "general", channelType: protocol.ChannelTypePublic,
			metadata: metaChat(ContextScopeNone),
			checkIntent: true, wantIntent: IntentLowSignal, checkMode: true, wantMode: ConversationModeChat},
		{name: "chat_mode_long_question", content: "What are the main tradeoffs between PostgreSQL and SQLite for a small side project?", channel: "general", channelType: protocol.ChannelTypePublic,
			metadata: metaChat(ContextScopeNone),
			checkIntent: true, wantIntent: IntentSubstantive, checkMode: true, wantMode: ConversationModeChat},
		{name: "chat_mode_explicit_metadata", content: "review main.go", channel: "general", channelType: protocol.ChannelTypePublic,
			metadata: map[string]interface{}{MetadataConversationMode: ConversationModeChat},
			checkMode: true, wantMode: ConversationModeChat, checkIntent: true, wantIntent: IntentTask},

		// --- Code / task ---
		{name: "task_review_path", content: "review cmd/server/main.go", channel: "general", channelType: protocol.ChannelTypePublic,
			checkIntent: true, wantIntent: IntentTask, checkMode: true, wantMode: ConversationModeCode},
		{name: "task_codebase_mention", content: "scan @codebase for TODOs", channel: "dm-u-a", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentTask, checkMode: true, wantMode: ConversationModeCode},
		{name: "task_refactor_verb", content: "refactor the hub package", channel: "general", channelType: protocol.ChannelTypePublic,
			checkIntent: true, wantIntent: IntentTask},
		{name: "task_dm_review", content: "fix internal/agent/turn_intent.go", channel: "dm-u-a", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentTask,
			checkTooling: true, wantTooling: true, toolingIntent: IntentTask},

		// --- Meta ---
		{name: "meta_prompt_context", content: "what information do you get when I send you a prompt?", channel: "dm-u-a", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentSubstantive, checkHistoryCap: true, wantHistoryCap: 8, historyIntent: IntentSubstantive},
		{name: "workspace_visibility", content: "can you see my workspace?", channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentSubstantive},
		{name: "workspace_visibility_long_phrase", content: "can you see my workspace I have open?", channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentSubstantive},
		{name: "chat_short_question_what", content: "What?", channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			metadata: metaChat(ContextScopeNone),
			checkIntent: true, wantIntent: IntentSubstantive},
		{name: "chat_mid_opinion_stays_casual", content: "what do you think about golang?", channel: "general", channelType: protocol.ChannelTypePublic,
			metadata: metaChat(ContextScopeNone),
			checkIntent: true, wantIntent: IntentLowSignal},
		{name: "theme_add_task", content: "I want to add theme support to this app", channel: "dm-u-be", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentTask},
		{name: "meta_model_identity", content: "what model are you?", channel: "general", channelType: protocol.ChannelTypePublic,
			checkIntent: true, wantIntent: IntentMeta},
		{name: "meta_not_reminder", content: "remind me in 5 minutes", channel: "dm-u-a", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentLowSignal},

		// --- Channel type ---
		{name: "dm_greeting_casual", content: "hello", channel: "dm-u-a", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentLowSignal, checkPersona: true, wantPersona: PersonaDirect},
		{name: "dm_distance_substantive", content: "How far is it from Collinsville IL to St Louis MO?", channel: "dm-u-a", channelType: protocol.ChannelTypeDM,
			checkIntent: true, wantIntent: IntentSubstantive},
		{name: "public_whats_up", content: "what's up?", channel: "general", channelType: protocol.ChannelTypePublic,
			checkIntent: true, wantIntent: IntentLowSignal},
		{name: "collab_ok_substantive", content: "ok", channel: "collab-1", channelType: protocol.ChannelTypeCollaboration,
			checkIntent: true, wantIntent: IntentSubstantive},
		{name: "slash_command", content: "/summarize", channel: "general", channelType: protocol.ChannelTypePublic,
			checkIntent: true, wantIntent: IntentSlashCommand},

		// --- Mode inference ---
		{name: "mode_infer_greeting", content: "hey", channel: "general", channelType: protocol.ChannelTypePublic,
			checkMode: true, wantMode: ConversationModeChat},
		{name: "mode_infer_review", content: "review cmd/server/main.go", channel: "general", channelType: protocol.ChannelTypePublic,
			checkMode: true, wantMode: ConversationModeCode},
		{name: "mode_metadata_code", content: "hello", channel: "general", channelType: protocol.ChannelTypePublic,
			metadata: map[string]interface{}{MetadataConversationMode: ConversationModeCode},
			checkMode: true, wantMode: ConversationModeCode},

		// --- Tooling ---
		{name: "tooling_casual_chat_off", content: "hello", channel: "dm-u-b", channelType: protocol.ChannelTypeDM,
			metadata: metaChat(ContextScopeNone),
			checkTooling: true, wantTooling: false, toolingIntent: IntentLowSignal},
		{name: "tooling_substantive_dm_on", content: "Explain how JWT refresh tokens work in detail.", channel: "dm-u-b", channelType: protocol.ChannelTypeDM,
			checkTooling: true, wantTooling: true, toolingIntent: IntentSubstantive},
		{name: "tooling_task_code_mode_on", content: "review cmd/server/main.go", channel: "general", channelType: protocol.ChannelTypePublic,
			metadata: map[string]interface{}{MetadataConversationMode: ConversationModeCode},
			checkTooling: true, wantTooling: true, toolingIntent: IntentTask},

		// --- History caps ---
		{name: "history_casual", checkHistoryCap: true, wantHistoryCap: 2, historyIntent: IntentLowSignal},
		{name: "history_meta", checkHistoryCap: true, wantHistoryCap: 2, historyIntent: IntentMeta},
		{name: "history_task", checkHistoryCap: true, wantHistoryCap: 8, historyIntent: IntentTask},
		{name: "history_substantive_with_summary", checkHistoryCap: true, wantHistoryCap: 4, hasSummary: true, historyIntent: IntentSubstantive},
		{name: "history_substantive_no_summary", checkHistoryCap: true, wantHistoryCap: 8, hasSummary: false, historyIntent: IntentSubstantive},

		// --- Persona ---
		{name: "persona_dm_slug", content: "hi", channel: "dm-user-backendengineer", channelType: protocol.ChannelTypeDM,
			checkPersona: true, wantPersona: PersonaDirect},
		{name: "persona_public_multi_agent", content: "hi", channel: "general", channelType: protocol.ChannelTypePublic,
			checkPersona: true, wantPersona: PersonaChannel, useMultiAgentHub: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			channel := tc.channel
			if channel == "" {
				channel = "general"
			}
			chType := tc.channelType
			if chType == "" {
				chType = protocol.ChannelTypePublic
			}
			aid := tc.agentID
			if aid == "" {
				aid = agentID
			}

			msg := chatMsg(tc.content, channel, chType, tc.metadata)
			if tc.content != "" {
				msg.Content = tc.content
			}
			if tc.channelType != "" {
				_ = chType
			}

			if tc.checkClosure {
				got := classifyConversationalClosure(tc.content)
				if got != tc.wantClosure {
					t.Fatalf("closure: got %v want %v", got, tc.wantClosure)
				}
			}

			if tc.checkIntent {
				got := ClassifyTurnIntentPublic(msg, chType, aid, tc.history)
				if got != tc.wantIntent {
					t.Fatalf("intent: got %v (%s) want %v", got, got.String(), tc.wantIntent)
				}
			}

			if tc.checkMode {
				got := EffectiveConversationMode(msg, chType)
				if got != tc.wantMode {
					t.Fatalf("mode: got %q want %q", got, tc.wantMode)
				}
			}

			if tc.checkHistoryCap {
				got := maxHistoryForIntent(tc.historyIntent, tc.hasSummary)
				if got != tc.wantHistoryCap {
					t.Fatalf("history cap: got %d want %d", got, tc.wantHistoryCap)
				}
			}

			if tc.checkTooling {
				hub := shouldRespondTestHub{}
				ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", []string{"general"}, ai.NewMockProvider(), hub)
				intent := tc.toolingIntent
				if intent == 0 && tc.checkIntent {
					intent = tc.wantIntent
				}
				got := ag.shouldIncludeToolingInPrompt(msg, intent)
				if got != tc.wantTooling {
					t.Fatalf("tooling: got %v want %v", got, tc.wantTooling)
				}
			}

			if tc.checkPersona {
				var hub HubClient = shouldRespondTestHub{}
				if tc.useMultiAgentHub {
					hub = multiAgentChannelHub{}
				}
				ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, ai.NewMockProvider(), hub)
				got := ag.promptPersonaTier(msg)
				if got != tc.wantPersona {
					t.Fatalf("persona: got %v want %v", got, tc.wantPersona)
				}
			}
		})
	}
}

// TestChatQualityRouter_tryClosure integrates canned closure responses with history.
func TestChatQualityRouter_tryClosure(t *testing.T) {
	ag := &Agent{
		Info: protocol.AgentInfo{ID: "asst-1", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm": buildChatHistory("asst-1", "distance?", "39 miles"),
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{ID: "u", Name: "Camron"}, "ok thanks")
	resp, ok := tryConversationalClosure(ag, msg)
	if !ok || resp == "" {
		t.Fatal("expected closure for ok thanks")
	}
}
