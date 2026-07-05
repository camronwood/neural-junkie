package agent

import (
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TurnIntent classifies a user turn for prompt tiering and history caps.
type TurnIntent int

const (
	IntentClosure TurnIntent = iota
	IntentSubstantive
	IntentMeta
	IntentSlashCommand
	IntentLowSignal
	IntentTask
)

func (i TurnIntent) String() string {
	switch i {
	case IntentClosure:
		return "closure"
	case IntentSubstantive:
		return "substantive"
	case IntentMeta:
		return "meta"
	case IntentSlashCommand:
		return "slash"
	case IntentLowSignal:
		return "casual"
	case IntentTask:
		return "task"
	default:
		return "unknown"
	}
}

func classifyTurnIntent(msg *protocol.Message, channelType protocol.ChannelType, agentID string, history []*protocol.Message) TurnIntent {
	if msg == nil {
		return IntentSubstantive
	}
	content := strings.TrimSpace(msg.Content)
	if content != "" && content[0] == '/' {
		return IntentSlashCommand
	}

	kind := classifyConversationalClosure(content)
	if kind != ClosureNone {
		if kind == ClosureBriefAck && !recentAgentAnsweredInChannel(history, agentID) {
			// Not a true closure turn; treat as casual minimal prompt.
		} else {
			return IntentClosure
		}
	}

	if userAsksAboutModelIdentity(content) {
		return IntentMeta
	}
	if userAsksAboutPromptContext(content) || userAsksAboutWorkspaceVisibility(content) {
		return IntentSubstantive
	}

	if hasScanOrEditorTaskSignals(content) || requestedBiologyScanTool(content) != "" {
		return IntentTask
	}
	if UserRequestsGeneratedImage(content) || UserRequestsGeneratedMusic(content) {
		return IntentTask
	}

	if channelType == protocol.ChannelTypeCollaboration {
		return IntentSubstantive
	}

	if userAffirmsPendingImplementation(content) && (channelHasRecentImplementationAsk(history, msg.ID) ||
		channelHasRecentImplementationActivity(history, msg.ID, agentID)) {
		return IntentTask
	}

	mode := EffectiveConversationMode(msg, channelType)

	if hasCodeTaskSignals(content) {
		return IntentTask
	}

	if greetingRE.MatchString(content) {
		return IntentLowSignal
	}

	if mode == ConversationModeChat {
		if strings.Contains(content, "?") {
			// Short clarifications (What?, see my workspace?) vs mid-length casual opinions.
			if len(content) < 30 || len(content) >= 40 {
				return IntentSubstantive
			}
			return IntentLowSignal
		}
		return IntentLowSignal
	}

	if ContextScopeFromMessage(msg) == ContextScopeNone && !strings.Contains(content, "?") && len(content) < 80 {
		return IntentLowSignal
	}

	if len(content) < 25 && !strings.Contains(content, "?") {
		return IntentLowSignal
	}
	return IntentSubstantive
}

func (a *Agent) classifyTurnIntentForMessage(msg *protocol.Message) TurnIntent {
	if msg != nil {
		caps := protocol.ResolveTurnCapabilities(msg)
		if caps.ComposerMode == "ask" {
			return IntentSubstantive
		}
		taskFromComposer := caps.ComposerMode == "export" || (caps.CanRunImplSession && msg.ImplementationSession())
		if taskFromComposer {
			if a.Info.Type != protocol.AgentTypeAssistant || assistantAllowsImplementationSession(a, msg) {
				return IntentTask
			}
			return IntentSubstantive
		}
	}
	history := a.channelHistory(msg.Channel)
	return classifyTurnIntent(msg, a.effectiveChannelType(msg.Channel), a.Info.ID, history)
}

// ClassifyTurnIntentPublic exposes turn intent classification for debug tooling.
func ClassifyTurnIntentPublic(msg *protocol.Message, channelType protocol.ChannelType, agentID string, history []*protocol.Message) TurnIntent {
	return classifyTurnIntent(msg, channelType, agentID, history)
}

func (a *Agent) sessionSummaryBlock(channel string) string {
	if a.Hub == nil {
		return ""
	}
	chType := a.effectiveChannelType(channel)
	if !shouldMaintainSessionSummary(chType, channel) {
		return ""
	}
	summary := strings.TrimSpace(a.Hub.GetChannelSessionSummary(channel))
	if summary == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("=== SESSION SUMMARY ===\n")
	b.WriteString(summary)
	b.WriteString("\nAnswer ONLY the user's latest message. ")
	b.WriteString("Do not re-answer earlier questions or repeat assistant replies from the summary.\n\n")
	return b.String()
}

func (a *Agent) injectSessionSummary(prompt string, msg *protocol.Message) string {
	block := a.sessionSummaryBlock(msg.Channel)
	if block == "" {
		return prompt
	}
	return block + prompt
}

func (a *Agent) buildMinimalPrompt(msg *protocol.Message) string {
	var b strings.Builder
	tier := a.promptPersonaTier(msg)
	specialty := string(a.Info.Type)
	if a.Info.Type == protocol.AgentTypeExpert && len(a.Info.Expertise) > 0 {
		specialty = a.Info.Expertise[0]
	}
	if tier == PersonaDirect {
		fmt.Fprintf(&b, "You are %s, a %s specialist speaking directly with the user.\n\n", a.Info.Name, specialty)
	} else {
		fmt.Fprintf(&b, "You are %s, a %s specialist in a multi-agent chat.\n\n", a.Info.Name, specialty)
	}
	if a.Info.Type == protocol.AgentTypeAssistant {
		fmt.Fprintf(&b, "You are powered by the %q model via the %q provider.\n", a.Info.AIModel, a.Info.AIProvider)
		if userAsksAboutModelIdentity(msg.Content) {
			b.WriteString("If asked about your model, state only the model and provider above.\n\n")
		} else {
			b.WriteString("\n")
		}
	}
	if block := a.sessionSummaryBlock(msg.Channel); block != "" && !userAsksAboutModelIdentity(msg.Content) {
		b.WriteString(block)
	}
	fallback := ResolveUserRulesHubFallback(msg)
	AppendUserAndAgentRules(&b, msg, &a.Info, fallback, 0)
	AppendMemoryForMessage(&b, msg, a.channelHistory(msg.Channel), a.effectiveKnowledgePlan(msg))
	AppendLearningsForMessage(&b, msg, &a.Info)
	b.WriteString("Respond briefly and naturally to the user's latest message only.\n")
	b.WriteString("Do not repeat long prior answers or re-derive facts already covered in the session summary.\n")
	b.WriteString("Never quote or restate the USER MESSAGE verbatim as your entire reply. ")
	b.WriteString("Do not claim work was completed unless you actually did it this turn.\n\n")
	b.WriteString("USER MESSAGE:\n")
	b.WriteString(strings.TrimSpace(msg.Content))
	return b.String()
}

func (a *Agent) buildPromptForIntent(msg *protocol.Message, intent TurnIntent) string {
	if a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeCollaboration {
		return a.injectSessionSummary(a.buildPrompt(msg, intent), msg)
	}
	switch intent {
	case IntentLowSignal, IntentMeta:
		return a.buildMinimalPrompt(msg)
	default:
		// Meeting/email turns need enriched prompts even on small Ollama models.
		if a.Info.Type == protocol.AgentTypeAssistant &&
			(messageAsksAboutMeetings(msg.Content) || messageAsksAboutEmail(msg.Content) || messageAsksAboutNJApp(msg.Content)) &&
			a.customPromptBuilder != nil {
			return a.injectSessionSummary(a.customPromptBuilder(msg), msg)
		}
		if a.useCompactAssistantOllamaPrompt(msg) {
			return a.injectSessionSummary(a.buildCompactAssistantOllamaPrompt(msg), msg)
		}
		if a.customPromptBuilder != nil {
			return a.injectSessionSummary(a.customPromptBuilder(msg), msg)
		}
		return a.injectSessionSummary(a.buildPrompt(msg, intent), msg)
	}
}

func (a *Agent) conversationHistoryForIntent(msg *protocol.Message, intent TurnIntent) []*protocol.Message {
	hasSummary := a.sessionSummaryBlock(msg.Channel) != ""
	max := maxHistoryForIntent(intent, hasSummary)
	if msg != nil && msg.IdeRouteAgentType() != "" && max < 12 {
		max = 12
	}
	var raw []*protocol.Message
	if msg.IsInThread() && a.Hub != nil {
		threadID := msg.GetThreadID()
		if threadID != "" {
			if threadMsgs, err := a.Hub.GetThreadMessages(threadID, maxHistoryForIntent(intent, false)+2); err == nil && len(threadMsgs) > 0 {
				raw = threadMsgs
			}
		}
	}
	if len(raw) == 0 {
		raw = a.channelHistory(msg.Channel)
	}
	var base []*protocol.Message
	if a.Info.Type == protocol.AgentTypeAssistant {
		base = filterAssistantHistory(raw, msg)
		if a.useCompactAssistantOllamaPrompt(msg) {
			base = recentUserHistoryOnly(base, max)
		}
	} else {
		base = historyForGeneration(raw, msg.ID)
		if a.useOllamaContextGuardrails(msg) || hasSummary {
			base = recentUserHistoryOnly(base, max)
		}
	}
	return trimHistoryTail(base, max)
}

func maxHistoryForIntent(intent TurnIntent, hasSummary bool) int {
	switch intent {
	case IntentLowSignal, IntentMeta:
		return 2
	case IntentTask:
		return 8
	case IntentSubstantive:
		if hasSummary {
			return 4
		}
		return 8
	default:
		return maxLLMHistoryMessages()
	}
}

func trimHistoryTail(history []*protocol.Message, max int) []*protocol.Message {
	if max <= 0 || len(history) <= max {
		return history
	}
	return history[len(history)-max:]
}

func (a *Agent) shouldAugmentPromptWithWorkspace(intent TurnIntent, msg *protocol.Message) bool {
	if intent == IntentLowSignal || intent == IntentMeta || intent == IntentClosure {
		return false
	}
	if a.Info.Type == protocol.AgentTypeAssistant && a.isDMChannel(msg.Channel) &&
		!assistantAllowsImplementationSession(a, msg) {
		return false
	}
	if a.useOllamaContextGuardrails(msg) {
		return false
	}
	wsPath := strings.TrimSpace(a.resolveWorkspacePath(msg))
	hasWorkspace := wsPath != "" || messageHasWorkspaceContext(msg)
	if !hasWorkspace {
		return false
	}
	if msg != nil && userRequestsContentDelivery(msg.Content) {
		return true
	}
	mode := EffectiveConversationMode(msg, a.effectiveChannelType(msg.Channel))
	if mode == ConversationModeCode || intent == IntentTask || messageNeedsWorkspaceFileLoad(a, msg) {
		if ContextScopeFromMessage(msg) == ContextScopeNone && !messageHasWorkspaceContext(msg) {
			return intent == IntentTask
		}
		return true
	}
	if mode == ConversationModeChat && intent != IntentTask {
		return false
	}
	if ContextScopeFromMessage(msg) == ContextScopeNone && intent != IntentTask {
		return false
	}
	return true
}

// messageNeedsWorkspaceFileLoad reports fix/debug/workspace turns that should preload disk files.
func messageNeedsWorkspaceFileLoad(a *Agent, msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if userRequestsContentDelivery(msg.Content) {
		return true
	}
	if userRequestsImplementationForMessage(a, msg) {
		return true
	}
	if userRequestsImplementation(msg.Content) || workspaceDirectiveRE.MatchString(msg.Content) {
		return true
	}
	if len(DetectFilePaths(msg.Content)) > 0 {
		return true
	}
	if a != nil && channelHasRecentImplementationAsk(a.channelHistory(msg.Channel), msg.ID) {
		return true
	}
	return false
}

// buildWorkspaceGroundedRetryPrompt preloads workspace seed files when the model asked the user to paste content.
func (a *Agent) buildWorkspaceGroundedRetryPrompt(msg *protocol.Message) string {
	var system strings.Builder
	system.WriteString(fmt.Sprintf("You are %s.\n", a.Info.Name))
	system.WriteString("The user has shared their project workspace on disk. ")
	system.WriteString("Do NOT ask them to paste or share file contents — use the loaded files below.\n")
	if a.hasWorkspaceTools() {
		system.WriteString("You also have read_file / grep / glob_file_search tools for additional paths.\n")
	}
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath != "" {
		AppendImplementationSeedFiles(&system, a, msg, wsPath, a.Info.Type, nil)
		if userRequestsContentDelivery(msg.Content) {
			AppendContentDeliverySeedFiles(&system, wsPath, nil)
		}
	}
	var user strings.Builder
	user.WriteString(strings.TrimSpace(msg.Content))
	return system.String() + ai.SystemPromptSeparator + user.String()
}

func (a *Agent) logTurnIntent(intent TurnIntent, msg *protocol.Message) {
	hasSummary := a.sessionSummaryBlock(msg.Channel) != ""
	chars := 0
	mode := ""
	if msg != nil {
		chars = len(strings.TrimSpace(msg.Content))
		mode = EffectiveConversationMode(msg, a.effectiveChannelType(msg.Channel))
	}
	log.Printf("[%s] intent=%s mode=%s channel=%s chars=%d summary=%t",
		a.Info.Name, intent.String(), mode, msg.Channel, chars, hasSummary)
}
