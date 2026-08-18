package agent

import (
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
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
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		return turnIntentFromSemantic(decision.Interaction)
	}

	if channelType == protocol.ChannelTypeCollaboration {
		return IntentSubstantive
	}

	if userAffirmsPendingImplementation(content) && (channelHasRecentImplementationAsk(history, msg.ID) ||
		channelHasRecentImplementationActivity(history, msg.ID, agentID)) {
		return IntentTask
	}

	mode := EffectiveConversationMode(msg, channelType)

	// Client Auto inference was ambiguous: ask in chat, do not treat weak verbs as tasks yet.
	if mode == ConversationModeClarify {
		return IntentSubstantive
	}

	if hasCodeTaskSignals(content) {
		return IntentTask
	}

	if greetingRE.MatchString(content) {
		return IntentLowSignal
	}
	if isSocialOrStatusPing(content) || intent.LooksLikePresenceCheck(content) {
		return IntentLowSignal
	}

	// Open thread follow-ups stay substantive so ConversationWindow is used
	// (capability notices, pronouns, "why?", "go on") instead of casual demotion.
	if channelHasOpenDialogue(history, agentID) && looksLikeThreadFollowUp(content) {
		return IntentSubstantive
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
		if decision, ok := protocol.ExtractTurnDecision(msg); ok {
			return turnIntentFromSemantic(decision.Interaction)
		}
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
	if msg != nil {
		if decision, ok := protocol.ExtractTurnDecision(msg); ok {
			return turnIntentFromSemantic(decision.Interaction)
		}
	}
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
	b.WriteString("\nContinue this conversation. Recent exchanges are ground truth for the open thread; ")
	b.WriteString("treat this summary as background only. Do not ignore the last exchanges or treat the latest line as an isolated new topic. ")
	b.WriteString("Do not paste prior assistant replies verbatim as your entire answer.\n\n")
	return b.String()
}

// turnLedgerProvider is optional on HubClient implementations.
type turnLedgerProvider interface {
	GetChannelTurnLedger(channel string, limit int) []TurnLedgerRow
}

// TurnLedgerRow is a compact durable turn used in the Memory-stage overlay.
type TurnLedgerRow struct {
	Speaker     string
	SpeakerType string
	Excerpt     string
	Entities    []string
}

const turnLedgerPromptRows = 12

func (a *Agent) turnLedgerBlock(channel string) string {
	if a == nil || a.Hub == nil || strings.TrimSpace(channel) == "" {
		return ""
	}
	provider, ok := a.Hub.(turnLedgerProvider)
	if !ok {
		return ""
	}
	rows := provider.GetChannelTurnLedger(channel, turnLedgerPromptRows)
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("=== TURN LEDGER (recent) ===\n")
	b.WriteString("Structured recent turns (speaker-attributed). Recent exchanges remain ground truth.\n")
	for _, e := range rows {
		who := strings.TrimSpace(e.Speaker)
		if who == "" {
			who = "?"
		}
		if kind := strings.TrimSpace(e.SpeakerType); kind != "" {
			who = who + "/" + kind
		}
		line := strings.TrimSpace(e.Excerpt)
		if line == "" {
			line = "(empty)"
		}
		// Bound overlay line length for context budget safety.
		if runes := []rune(line); len(runes) > 180 {
			line = string(runes[:179]) + "…"
		}
		b.WriteString("- ")
		b.WriteString(who)
		b.WriteString(": ")
		b.WriteString(line)
		if len(e.Entities) > 0 {
			b.WriteString(" [")
			b.WriteString(strings.Join(e.Entities, ", "))
			b.WriteString("]")
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return b.String()
}

func (a *Agent) injectSessionSummary(prompt string, msg *protocol.Message) string {
	if msg == nil {
		return prompt
	}
	block := a.sessionSummaryBlock(msg.Channel)
	ledger := a.turnLedgerBlock(msg.Channel)
	if block == "" && ledger == "" {
		return prompt
	}
	// Ledger first (timeline), then rolled-up summary — never replaces ConversationWindow in prompt body.
	return ledger + block + prompt
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
	if !userAsksAboutModelIdentity(msg.Content) {
		// Ledger first (timeline), then rolled-up summary — matches injectSessionSummary.
		if ledger := a.turnLedgerBlock(msg.Channel); ledger != "" {
			b.WriteString(ledger)
		}
		if block := a.sessionSummaryBlock(msg.Channel); block != "" {
			b.WriteString(block)
		}
	}
	fallback := ResolveUserRulesHubFallback(msg)
	AppendUserAndAgentRules(&b, msg, &a.Info, fallback, 0)
	AppendMemoryForMessage(&b, msg, a.channelHistory(msg.Channel), a.effectiveKnowledgePlanFromMessage(msg))
	AppendLearningsForMessage(&b, msg, &a.Info)
	b.WriteString("Continue the conversation briefly and naturally. Prefer the open thread over treating this as a new isolated question.\n")
	b.WriteString("Do not paste long prior answers verbatim or re-derive facts already covered in the session summary.\n")
	b.WriteString("Never quote or restate the USER MESSAGE verbatim as your entire reply. ")
	b.WriteString("Do not claim work was completed unless you actually did it this turn.\n\n")
	b.WriteString("USER MESSAGE:\n")
	b.WriteString(strings.TrimSpace(msg.Content))
	return b.String()
}

// assistantPersonalContextQuery is true for Assistant turns that need synced
// meeting/email/app context. These stay casual conversation (LowSignal) but
// must not use the minimal prompt that omits that context.
func assistantPersonalContextQuery(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	return messageAsksAboutMeetings(msg.Content) ||
		messageAsksAboutEmail(msg.Content) ||
		messageAsksAboutNJApp(msg.Content)
}

func (a *Agent) buildPromptForIntent(msg *protocol.Message, intent TurnIntent) (prompt string) {
	envelope := a.selectTurnContext(msg)
	defer func() {
		prompt = appendDurableConversationContext(prompt, envelope, msg)
	}()
	if a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeCollaboration {
		return a.injectSessionSummary(a.buildPrompt(msg, intent), msg)
	}
	// Meeting/email/app questions are casual chat for Assistant, but still need
	// enriched prompts even when the classifier marks the turn LowSignal/Meta.
	if a.Info.Type == protocol.AgentTypeAssistant &&
		assistantPersonalContextQuery(msg) &&
		a.customPromptBuilder != nil {
		return a.injectSessionSummary(a.customPromptBuilder(msg), msg)
	}
	switch intent {
	case IntentLowSignal, IntentMeta:
		return a.buildMinimalPrompt(msg)
	default:
		if a.constrainedIDETurn(msg) {
			return a.injectSessionSummary(a.buildConstrainedComposerPrompt(msg, intent), msg)
		}
		if msg != nil && msg.IdeEditorModeIsPlan() {
			return a.injectSessionSummary(a.buildComposerPlanPrompt(msg), msg)
		}
		if a.Info.Type == protocol.AgentTypeAssistant && a.shouldUseDialogueAssistantPrompt(msg) {
			return a.injectSessionSummary(a.buildDialogueAssistantPrompt(msg), msg)
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

// conversationHistoryForIntent returns the dialogue-first ConversationWindow:
// recent complete user↔assistant exchanges (both roles). Summary is an overlay
// and must not shrink or strip the window.
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
	envelope := a.selectTurnContext(msg)
	exchanges := recentCompleteExchanges(raw, msg, envelope.SupersededMessageIDs, max)
	return messagesFromExchanges(pinGoalMessageInExchanges(raw, envelope, exchanges))
}

// maxHistoryForIntent returns a message budget sized for complete exchanges.
// hasSummary must not shrink the window (summary is overlay-only).
func maxHistoryForIntent(intent TurnIntent, hasSummary bool) int {
	_ = hasSummary
	switch intent {
	case IntentLowSignal, IntentMeta:
		return 4 // ~2 exchanges
	case IntentTask:
		return 16 // ~8 exchanges
	case IntentSubstantive:
		return 12 // ~6 exchanges
	default:
		return maxLLMHistoryMessages()
	}
}

// channelHasOpenDialogue reports a recent agent answer in channel history.
func channelHasOpenDialogue(history []*protocol.Message, agentID string) bool {
	return recentAgentAnsweredInChannel(history, agentID)
}

// looksLikeThreadFollowUp detects short continuations of an open thread.
func looksLikeThreadFollowUp(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	markers := []string{
		"turned on", "turned off", "i enabled", "i've enabled", "i have enabled",
		"websearch", "web search", "go on", "and then", "what about", "how about",
		"why did", "why would", "why are", "continue", "tell me more", "one more",
		"that one", "the second", "the first", "from that", "about that", "about it",
		"move it", "make it", "do that", "do it",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	if lower == "yeah" || lower == "yep" || lower == "yes" || lower == "sure" ||
		lower == "ok" || lower == "okay" || lower == "go ahead" {
		return true
	}
	// Pronoun-heavy short follow-ups.
	if len(content) <= 100 {
		for _, p := range []string{" it ", " that ", " this ", " those ", " them "} {
			if strings.Contains(" "+lower+" ", p) {
				return true
			}
		}
		if strings.HasPrefix(lower, "why") || strings.HasPrefix(lower, "and ") {
			return true
		}
	}
	return false
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
	// MapsExpert answers via geocode/route tools — shared IDE workspace must not
	// flip the turn into codebase grounding / FILE_CHANGE mode.
	if a != nil && a.Info.Type == protocol.AgentTypeMaps {
		if msg != nil && (UserRequestsMapOrRoute(msg.Content) || mapEndpointsLookGeographic(msg.Content)) {
			return false
		}
		if msg == nil || (!userRequestsImplementationForMessage(a, msg) && len(DetectFilePaths(msg.Content)) == 0) {
			return false
		}
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
	// Explicit none-scope wins even for IntentTask / InteractionCorrection — otherwise
	// chat design corrections dump README seeds and hang on codebase search.
	if ResolveContextScope(msg) == ContextScopeNone {
		return false
	}
	mode := ToolingConversationMode(msg, a.effectiveChannelType(msg.Channel))
	if mode == ConversationModeCode || intent == IntentTask || messageNeedsWorkspaceFileLoad(a, msg) {
		return true
	}
	if mode == ConversationModeChat && intent != IntentTask {
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
	system.WriteString("Do NOT claim the context window is empty, that you lack project details, or ask them to paste files — use the loaded files below.\n")
	system.WriteString("Stay on the user's topic (e.g. theme/dark/light/CSS if that is the thread) and answer in 3-8 sentences.\n")
	if intent.LooksLikeProjectOverviewAsk(msg.Content) {
		system.WriteString("The user asked for a project review/summary. Lead with what the project is, its stack, and main layout from README/package manifests/source roots. ")
		system.WriteString("Do not invent a hollow Grounding-only stub or a Changes section unless they asked for edits.\n")
	}
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
