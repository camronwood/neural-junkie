package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

func (a *Agent) generateResponse(ctx context.Context, msg *protocol.Message, eff ai.AIProvider) (string, error) {
	// Check if this is a design analysis request
	if designAnalysis, ok := msg.Metadata["design_analysis"].(bool); ok && designAnalysis {
		return a.generateDesignAnalysisResponse(ctx, msg)
	}
	intent := turnIntentForContext(ctx, a, msg)
	a.logTurnIntent(intent, msg)
	if intent == IntentClosure {
		if resp, ok := tryConversationalClosure(a, msg); ok {
			log.Printf("[%s] Conversational closure (no LLM): %q", a.Info.Name, truncateForLog(msg.Content, 60))
			return resp, nil
		}
	}
	if resp, ok := a.tryWorkspaceVisibilityResponse(msg); ok {
		log.Printf("[%s] Workspace visibility (no LLM): %q", a.Info.Name, truncateForLog(msg.Content, 60))
		return resp, nil
	}
	if resp, ok := a.tryPriorReferenceResponse(msg); ok {
		log.Printf("[%s] Prior reference missing from history (no LLM): %q", a.Info.Name, truncateForLog(msg.Content, 60))
		return resp, nil
	}
	if eff == nil {
		eff = a.GetAIProvider()
	}
	restoreCLI := a.prepareCLIInvocation(msg)
	defer restoreCLI()

	// Track files already included in the prompt so the workspace scanner
	// doesn't duplicate them.
	includedFiles := collectIncludedFilePaths(msg)

	prompt := a.buildPromptForIntent(msg, intent)
	prompt = a.appendDelegationContext(ctx, msg, prompt)
	prompt = a.appendRepoConsultContext(ctx, msg, prompt, intent)

	// Auto-detect and load file paths referenced in the user's message.
	wsPath := a.resolveWorkspacePath(msg)
	referencedLoaded := 0
	if a.shouldAugmentPromptWithWorkspace(intent, msg) && wsPath != "" {
		var referencedFiles strings.Builder
		referencedLoaded = AppendReferencedFiles(&referencedFiles, msg.Content, wsPath)
		if userRequestsImplementationForMessage(a, msg) || messageNeedsWorkspaceFileLoad(a, msg) {
			referencedLoaded += AppendImplementationSeedFiles(&referencedFiles, a, msg, wsPath, a.Info.Type, includedFiles)
		}
		if userRequestsContentDelivery(msg.Content) {
			referencedLoaded += AppendContentDeliverySeedFiles(&referencedFiles, wsPath, includedFiles)
		}
		if referencedFiles.Len() > 0 {
			prompt += referencedFiles.String()
			for _, p := range DetectFilePaths(msg.Content) {
				includedFiles[p] = true
			}
		}
	}

	// Proactively scan the workspace for domain-relevant source files.
	// This lets specialist agents (BackendEngineer, CodeReviewer, etc.) see project
	// code even when the user doesn't mention specific file paths.
	scannedLoaded := 0
	collabInfo := a.getCollaborationContext(msg)
	if a.shouldAugmentPromptWithWorkspace(intent, msg) && wsPath != "" && !a.agentHasDedicatedContext() && !a.hasWorkspaceTools() && shouldProactiveScanWorkspaceForMessage(a, msg) && collaborationProactiveWorkspaceScan(msg, collabInfo) {
		existingContextSize := len(prompt) - len(a.buildPromptForIntent(msg, intent))
		if existingContextSize < maxScanChars/2 {
			scanQuery := BuildWorkspaceScanQuery(msg.Content, a.channelHistory(msg.Channel))
			scannedFiles, loadedCount, err := ScanWorkspaceFiles(wsPath, a.Info.Type, scanQuery, maxScanChars, includedFiles)
			if err != nil {
				log.Printf("[%s] Workspace scan failed: %v", a.Info.Name, err)
			} else if scannedFiles != "" {
				prompt += scannedFiles
				scannedLoaded = loadedCount
			}
		}
	}

	if shouldForceWorkspaceGroundingOpener(msg) && collaborationWorkspaceGroundingLine(msg, collabInfo) {
		openFileLoaded := len(collectIncludedFilePaths(msg))
		totalLoaded := openFileLoaded + referencedLoaded + scannedLoaded
		if g := workspaceGroundingRequirement(totalLoaded, msg.Content, userRequestsImplementationForMessage(a, msg)); g != "" {
			prompt += g
		}
	}

	history := a.conversationHistoryForIntent(msg, intent)
	prompt = a.appendPriorReferenceGuidance(prompt, msg, history)

	prompt = a.augmentPromptWithCLIImages(msg, prompt)

	var budgetStats ContextBudgetStats
	prompt, budgetStats = applyContextBudgetForMessage(msg, prompt)
	stampContextBudgetStats(msg, budgetStats)
	if budgetStats.Truncated {
		log.Printf("[%s] context budget applied: %d -> %d bytes", a.Info.Name, budgetStats.OriginalBytes, budgetStats.FinalBytes)
	}

	imgs := protocol.ExtractUserImages(msg)
	if len(imgs) > 0 && a.Info.SupportsVision {
		approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
		if mp, ok := eff.(ai.MultimodalProvider); ok {
			return mp.GenerateMultimodal(approvalCtx, prompt, imgs, historyToMessages(history))
		}
		if len(imgs) == 1 {
			return eff.GenerateVisionResponse(approvalCtx, prompt, imgs[0].Data, imgs[0].MIME, historyToMessages(history))
		}
		return "", fmt.Errorf("multiple images require a multimodal-capable provider")
	}

	approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
	if resp, ok := a.tryBiologyScanToolShortcut(approvalCtx, msg); ok {
		return resp, nil
	}
	if resp, ok := a.tryOpenCanvasMetaAnswer(msg); ok {
		return resp, nil
	}
	if resp, ok := a.tryMapsRouteShortcut(approvalCtx, msg); ok {
		return resp, nil
	}
	if resp, ok := a.tryNeuralCanvasMarkdownShortcut(approvalCtx, msg, prompt, eff); ok {
		return resp, nil
	}
	if resp, ok := a.tryNeuralCanvasMermaidShortcut(approvalCtx, msg, prompt, eff); ok {
		return resp, nil
	}
	if resp, ok := a.tryHubImageGenerationShortcut(approvalCtx, msg); ok {
		return a.completeMixedImageResponse(approvalCtx, msg, prompt, history, eff, resp), nil
	}
	if resp, ok := a.tryHubMusicGenerationShortcut(approvalCtx, msg); ok {
		return resp, nil
	}
	if len(a.agentToolDefinitions(msg)) > 0 {
		response, err := a.generateWithAgentTools(approvalCtx, msg, prompt, history, eff)
		if err != nil {
			return "", err
		}
		if retry := a.maybeRetryConversationalQuality(ctx, msg, response, history, eff); retry != response {
			return retry, nil
		}
		return a.finalizeWorkspaceVisibilityReply(msg, response), nil
	}
	response, err := eff.GenerateResponse(approvalCtx, prompt, historyToMessages(history))
	if err != nil {
		return "", err
	}

	if a.useOllamaContextGuardrails(msg) &&
		(looksLikeOllamaPromptLeak(response) || looksLikeContextStackEcho(msg, response)) {
		retryPrompt := a.buildUltraCompactOllamaPrompt(msg)
		if a.useCompactAssistantOllamaPrompt(msg) {
			retryPrompt = a.buildUltraCompactAssistantOllamaPrompt(msg)
		}
		retryHistory := shortenedConversationWindow(history, 4)
		retry, err2 := eff.GenerateResponse(approvalCtx, retryPrompt, historyToMessages(retryHistory))
		if err2 == nil && strings.TrimSpace(retry) != "" &&
			!looksLikeOllamaPromptLeak(retry) && !looksLikeContextStackEcho(msg, retry) {
			log.Printf("[%s] Ollama context-stack echo; used compact retry", a.Info.Name)
			return retry, nil
		}
	}
	if retry := a.maybeRetryConversationalQuality(ctx, msg, response, history, eff); retry != response {
		return retry, nil
	}
	return a.finalizeWorkspaceVisibilityReply(msg, response), nil
}

func (a *Agent) completeMixedImageResponse(
	ctx context.Context,
	msg *protocol.Message,
	prompt string,
	history []*protocol.Message,
	eff ai.AIProvider,
	imageResponse string,
) string {
	if msg == nil || eff == nil || !UserRequestsImageWithCompanionText(msg.Content) {
		return imageResponse
	}
	companionPrompt := prompt +
		"\n\n=== COMPLETED IMAGE ACTION ===\n" +
		"The requested image has already been generated and posted successfully. " +
		"Answer only the remaining non-image portion of the user's request now. " +
		"Do not deny image capability, ask whether to proceed, or claim that the image still needs to be generated."
	companion, err := eff.GenerateResponse(ctx, companionPrompt, historyToMessages(history))
	companion = strings.TrimSpace(sanitizeInternalToolNames(companion))
	if err != nil || companion == "" {
		return imageResponse
	}
	return companion + "\n\n" + imageResponse
}

// generateResponseStreaming builds the same prompt as generateResponse but
// streams the AI response token-by-token. Each token is broadcast to
// subscribers as a stream_delta message. Returns the full accumulated text
// and the stable stream message ID so the caller can reuse it for the
// final chat message (allowing the frontend to correlate streaming with
// the persisted message).
func (a *Agent) generateResponseStreaming(ctx context.Context, msg *protocol.Message, eff ai.AIProvider) (string, string, string, error) {
	if designAnalysis, ok := msg.Metadata["design_analysis"].(bool); ok && designAnalysis {
		resp, err := a.generateDesignAnalysisResponse(ctx, msg)
		return resp, "", "", err
	}
	intent := turnIntentForContext(ctx, a, msg)
	a.logTurnIntent(intent, msg)
	if intent == IntentClosure {
		if resp, ok := tryConversationalClosure(a, msg); ok {
			log.Printf("[%s] Conversational closure (no LLM stream): %q", a.Info.Name, truncateForLog(msg.Content, 60))
			return resp, "", "", nil
		}
	}
	if resp, ok := a.tryWorkspaceVisibilityResponse(msg); ok {
		log.Printf("[%s] Workspace visibility (no LLM stream): %q", a.Info.Name, truncateForLog(msg.Content, 60))
		return resp, "", "", nil
	}
	if resp, ok := a.tryPriorReferenceResponse(msg); ok {
		log.Printf("[%s] Prior reference missing from history (no LLM stream): %q", a.Info.Name, truncateForLog(msg.Content, 60))
		return resp, "", "", nil
	}
	if eff == nil {
		eff = a.GetAIProvider()
	}
	restoreCLI := a.prepareCLIInvocation(msg)
	defer restoreCLI()

	prompt := a.buildPromptForIntent(msg, intent)
	prompt = a.appendDelegationContext(ctx, msg, prompt)
	prompt = a.appendRepoConsultContext(ctx, msg, prompt, intent)

	includedFiles := collectIncludedFilePaths(msg)
	collabInfo := a.getCollaborationContext(msg)

	wsPath := a.resolveWorkspacePath(msg)
	referencedLoaded := 0
	if a.shouldAugmentPromptWithWorkspace(intent, msg) && wsPath != "" {
		var referencedFiles strings.Builder
		referencedLoaded = AppendReferencedFiles(&referencedFiles, msg.Content, wsPath)
		if userRequestsImplementationForMessage(a, msg) || messageNeedsWorkspaceFileLoad(a, msg) {
			referencedLoaded += AppendImplementationSeedFiles(&referencedFiles, a, msg, wsPath, a.Info.Type, includedFiles)
		}
		if userRequestsContentDelivery(msg.Content) {
			referencedLoaded += AppendContentDeliverySeedFiles(&referencedFiles, wsPath, includedFiles)
		}
		if referencedFiles.Len() > 0 {
			prompt += referencedFiles.String()
			for _, p := range DetectFilePaths(msg.Content) {
				includedFiles[p] = true
			}
		}
	}

	scannedLoaded := 0
	if a.shouldAugmentPromptWithWorkspace(intent, msg) && wsPath != "" && !a.agentHasDedicatedContext() && !a.hasWorkspaceTools() && shouldProactiveScanWorkspaceForMessage(a, msg) && collaborationProactiveWorkspaceScan(msg, collabInfo) {
		basePrompt := a.buildPromptForIntent(msg, intent)
		existingContextSize := len(prompt) - len(basePrompt)
		if existingContextSize < maxScanChars/2 {
			scanQuery := BuildWorkspaceScanQuery(msg.Content, a.channelHistory(msg.Channel))
			scannedFiles, loadedCount, scanErr := ScanWorkspaceFiles(wsPath, a.Info.Type, scanQuery, maxScanChars, includedFiles)
			if scanErr != nil {
				log.Printf("[%s] Workspace scan failed: %v", a.Info.Name, scanErr)
			} else if scannedFiles != "" {
				prompt += scannedFiles
				scannedLoaded = loadedCount
			}
		}
	}

	if shouldForceWorkspaceGroundingOpener(msg) && collaborationWorkspaceGroundingLine(msg, collabInfo) {
		openFileLoaded := len(collectIncludedFilePaths(msg))
		totalLoaded := openFileLoaded + referencedLoaded + scannedLoaded
		if g := workspaceGroundingRequirement(totalLoaded, msg.Content, userRequestsImplementationForMessage(a, msg)); g != "" {
			prompt += g
		}
	}

	history := a.conversationHistoryForIntent(msg, intent)
	prompt = a.appendPriorReferenceGuidance(prompt, msg, history)

	// Pre-create a stable message ID for the stream so the frontend can
	// correlate deltas with the final message.
	streamMsgID := uuid.New().String()

	approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
	approvalCtx = WithStreamMessageID(approvalCtx, streamMsgID)

	prompt = a.augmentPromptWithCLIImages(msg, prompt)

	var streamBudgetStats ContextBudgetStats
	prompt, streamBudgetStats = applyContextBudgetForMessage(msg, prompt)
	stampContextBudgetStats(msg, streamBudgetStats)
	if streamBudgetStats.Truncated {
		log.Printf("[%s] context budget applied (stream): %d -> %d bytes", a.Info.Name, streamBudgetStats.OriginalBytes, streamBudgetStats.FinalBytes)
	}

	imgs := protocol.ExtractUserImages(msg)
	if len(imgs) > 0 && a.Info.SupportsVision {
		if mp, ok := eff.(ai.MultimodalProvider); ok {
			tokenCh, err := mp.GenerateMultimodalStream(approvalCtx, prompt, imgs, historyToMessages(history))
			if err == nil {
				return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
			}
			log.Printf("[%s] Multimodal stream failed (%v), falling back to batch multimodal", a.Info.Name, err)
			text, err := mp.GenerateMultimodal(approvalCtx, prompt, imgs, historyToMessages(history))
			return text, "", "", err
		}
		if len(imgs) == 1 {
			text, err := eff.GenerateVisionResponse(approvalCtx, prompt, imgs[0].Data, imgs[0].MIME, historyToMessages(history))
			return text, "", "", err
		}
		return "", "", "", fmt.Errorf("multiple images require a multimodal-capable provider")
	}

	// Tool loop (MCP / image generation) uses batch API; stream the final answer as one chunk.
	// Run whenever tools exist — generateWithAgentTools falls back to qwen when the chat model (e.g. koesn) lacks native tools.
	if resp, ok := a.tryBiologyScanToolShortcut(approvalCtx, msg); ok {
		tokenCh := make(chan ai.StreamToken, 2)
		tokenCh <- ai.StreamToken{Content: resp}
		tokenCh <- ai.StreamToken{Done: true}
		close(tokenCh)
		return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
	}
	if resp, ok := a.tryOpenCanvasMetaAnswer(msg); ok {
		tokenCh := make(chan ai.StreamToken, 2)
		tokenCh <- ai.StreamToken{Content: resp}
		tokenCh <- ai.StreamToken{Done: true}
		close(tokenCh)
		return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
	}
	if resp, ok := a.tryMapsRouteShortcut(approvalCtx, msg); ok {
		tokenCh := make(chan ai.StreamToken, 2)
		tokenCh <- ai.StreamToken{Content: resp}
		tokenCh <- ai.StreamToken{Done: true}
		close(tokenCh)
		return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
	}
	if resp, ok := a.tryNeuralCanvasMarkdownShortcut(approvalCtx, msg, prompt, eff); ok {
		tokenCh := make(chan ai.StreamToken, 2)
		tokenCh <- ai.StreamToken{Content: resp}
		tokenCh <- ai.StreamToken{Done: true}
		close(tokenCh)
		return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
	}
	if resp, ok := a.tryNeuralCanvasMermaidShortcut(approvalCtx, msg, prompt, eff); ok {
		tokenCh := make(chan ai.StreamToken, 2)
		tokenCh <- ai.StreamToken{Content: resp}
		tokenCh <- ai.StreamToken{Done: true}
		close(tokenCh)
		return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
	}
	if resp, ok := a.tryHubImageGenerationShortcut(approvalCtx, msg); ok {
		resp = a.completeMixedImageResponse(approvalCtx, msg, prompt, history, eff, resp)
		tokenCh := make(chan ai.StreamToken, 2)
		tokenCh <- ai.StreamToken{Content: resp}
		tokenCh <- ai.StreamToken{Done: true}
		close(tokenCh)
		return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
	}
	if resp, ok := a.tryHubMusicGenerationShortcut(approvalCtx, msg); ok {
		tokenCh := make(chan ai.StreamToken, 2)
		tokenCh <- ai.StreamToken{Content: resp}
		tokenCh <- ai.StreamToken{Done: true}
		close(tokenCh)
		return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
	}
	if len(a.agentToolDefinitions(msg)) > 0 {
		outerObserver := ai.ToolStepObserverFromContext(approvalCtx)
		toolCtx := ai.WithToolStepObserver(approvalCtx, func(ev ai.ToolStepEvent) {
			if outerObserver != nil {
				outerObserver(ev)
			} else {
				a.broadcastToolStep(approvalCtx, msg, streamMsgID, ev)
			}
		})
		text, err := a.generateWithAgentTools(toolCtx, msg, prompt, history, eff)
		if err != nil {
			return "", "", "", err
		}
		text = a.maybeRetryConversationalQuality(approvalCtx, msg, text, history, eff)
		text = a.finalizeWorkspaceVisibilityReply(msg, text)
		return a.streamTextAsTokens(approvalCtx, msg, streamMsgID, text)
	}

	sp, ok := eff.(ai.StreamingProvider)
	if !ok || !sp.SupportsStreaming() {
		return "", "", "", fmt.Errorf("internal: expected streaming-capable provider")
	}

	maxAttempts := 1
	if a.useOllamaContextGuardrails(msg) {
		maxAttempts = 3
	}
	var lastErr error
	attemptPrompt := prompt
	streamProvider := eff
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 && a.useOllamaContextGuardrails(msg) {
			if a.useCompactAssistantOllamaPrompt(msg) {
				attemptPrompt = a.buildUltraCompactAssistantOllamaPrompt(msg)
			} else {
				attemptPrompt = a.buildUltraCompactOllamaPrompt(msg)
			}
			history = shortenedConversationWindow(history, 4)
			log.Printf("[%s] Retrying Ollama stream (attempt %d/%d, prompt %d bytes)", a.Info.Name, attempt+1, maxAttempts, len(attemptPrompt))
		}
		attemptSP, ok := streamProvider.(ai.StreamingProvider)
		if !ok {
			return "", "", "", fmt.Errorf("internal: expected streaming-capable provider")
		}
		tokenCh, err := attemptSP.GenerateResponseStream(approvalCtx, attemptPrompt, historyToMessages(history))
		if err != nil {
			return "", "", "", err
		}
		text, id, reasoning, err := a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
		if err != nil {
			if strings.TrimSpace(text) != "" &&
				a.Info.Type == protocol.AgentTypeAssistant &&
				(messageAsksAboutMeetings(msg.Content) || messageAsksAboutEmail(msg.Content) || messageAsksAboutNJApp(msg.Content)) {
				log.Printf("[%s] Keeping partial grounded Assistant stream for meeting/email query despite stream error: %v", a.Info.Name, err)
				return text, id, reasoning, nil
			}
			lastErr = err
			if errors.Is(err, ai.ErrOllamaNoContent) && attempt+1 < maxAttempts {
				continue
			}
			// Reasoning-only is a retryable Ollama failure mode (model streamed thinking but no
			// visible answer). Retry before surfacing as a "silent" turn in collab.
			if errors.Is(err, ai.ErrOllamaReasoningOnly) && attempt+1 < maxAttempts {
				continue
			}
			break
		}
		if (looksLikeOllamaPromptLeak(text) || looksLikeContextStackEcho(msg, text)) && attempt+1 < maxAttempts {
			log.Printf("[%s] Ollama reply looked like prompt/context echo; retrying", a.Info.Name)
			continue
		}
		text = a.maybeRetryConversationalQuality(approvalCtx, msg, text, history, eff)
		text = a.finalizeWorkspaceVisibilityReply(msg, text)
		return text, id, reasoning, nil
	}

	if a.useOllamaContextGuardrails(msg) && errors.Is(lastErr, ai.ErrOllamaNoContent) {
		if fb := ollamaFallbackProvider(eff, ai.OllamaBiologyFallbackModel); fb != nil {
			log.Printf("[%s] nj-bio returned empty; trying fallback model %q", a.Info.Name, ai.OllamaBiologyFallbackModel)
			fbSP, ok := fb.(ai.StreamingProvider)
			if ok && fbSP.SupportsStreaming() {
				fbPrompt := a.buildCompactOllamaPrompt(msg)
				tokenCh, err := fbSP.GenerateResponseStream(approvalCtx, fbPrompt, nil)
				if err == nil {
					text, id, reasoning, err := a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
					if err == nil && strings.TrimSpace(text) != "" && !looksLikeOllamaPromptLeak(text) {
						return text, id, reasoning, nil
					}
					if err != nil {
						lastErr = err
					}
				}
			}
		}
	}

	if lastErr != nil {
		return "", "", "", lastErr
	}
	return "", "", "", ai.ErrOllamaNoContent
}

// collectStreamTokens drains a stream channel, broadcasts deltas, emits stream_end, and returns full text.
func (a *Agent) collectStreamTokens(ctx context.Context, msg *protocol.Message, streamMsgID string, tokenCh <-chan ai.StreamToken) (string, string, string, error) {
	var fullResponse strings.Builder
	var fullReasoning strings.Builder
	var streamErr error
	bufferForValidation := false
	if goal, ok := turnGoalFromContext(ctx); ok {
		bufferForValidation = goal.RequiresActionEvidence()
	}

	for {
		select {
		case <-ctx.Done():
			streamErr = ctx.Err()
			goto finishStream
		case token, ok := <-tokenCh:
			if !ok {
				goto finishStream
			}
			if token.Error != nil {
				if fullResponse.Len() > 0 || fullReasoning.Len() > 0 {
					streamErr = token.Error
					goto finishStream
				}
				return "", "", "", token.Error
			}
			if token.Thinking != "" {
				fullReasoning.WriteString(token.Thinking)
				delta := protocol.NewMessage(
					protocol.MessageTypeStreamDelta,
					msg.Channel,
					a.Info,
					"",
				)
				delta.ID = streamMsgID
				delta.ReplyTo = msg.ID
				if delta.Metadata == nil {
					delta.Metadata = make(map[string]interface{})
				}
				delta.Metadata["reasoning_delta"] = true
				delta.Metadata["reasoning_append"] = token.Thinking
				if msg.IsInThread() {
					delta.ThreadID = msg.ThreadID
					delta.IsThreadReply = true
				}
				if !bufferForValidation {
					a.Hub.BroadcastDirect(msg.Channel, delta)
				}
			}
			if token.Content != "" {
				fullResponse.WriteString(token.Content)
				delta := protocol.NewMessage(
					protocol.MessageTypeStreamDelta,
					msg.Channel,
					a.Info,
					token.Content,
				)
				delta.ID = streamMsgID
				delta.ReplyTo = msg.ID
				if msg.IsInThread() {
					delta.ThreadID = msg.ThreadID
					delta.IsThreadReply = true
				}
				if !bufferForValidation {
					a.Hub.BroadcastDirect(msg.Channel, delta)
				}
			}
			if token.Done {
				goto finishStream
			}
		}
	}

finishStream:
	if streamErr != nil {
		if errors.Is(streamErr, context.Canceled) {
			if fullResponse.Len() > 0 {
				fullResponse.WriteString("\n\n[stopped]")
			}
			endMsg := protocol.NewMessage(protocol.MessageTypeStreamEnd, msg.Channel, a.Info, "")
			endMsg.ID = streamMsgID
			endMsg.ReplyTo = msg.ID
			if msg.IsInThread() {
				endMsg.ThreadID = msg.ThreadID
				endMsg.IsThreadReply = true
			}
			a.Hub.BroadcastDirect(msg.Channel, endMsg)
			return "", "", "", context.Canceled
		}
		// Fail-fast: 0-byte deadline/timeouts must not look like a successful empty turn.
		// Callers treat the error as generation_error and can retry/nudge.
		if fullResponse.Len() == 0 && (errors.Is(streamErr, context.DeadlineExceeded) ||
			strings.Contains(strings.ToLower(streamErr.Error()), "deadline exceeded") ||
			strings.Contains(strings.ToLower(streamErr.Error()), "timeout")) {
			log.Printf("[%s] Stream error with partial content (0 bytes): %v — failing turn", a.Info.Name, streamErr)
			endMsg := protocol.NewMessage(protocol.MessageTypeStreamEnd, msg.Channel, a.Info, "")
			endMsg.ID = streamMsgID
			endMsg.ReplyTo = msg.ID
			if msg.IsInThread() {
				endMsg.ThreadID = msg.ThreadID
				endMsg.IsThreadReply = true
			}
			a.Hub.BroadcastDirect(msg.Channel, endMsg)
			return "", streamMsgID, "", streamErr
		}
		log.Printf("[%s] Stream error with partial content (%d bytes): %v", a.Info.Name, fullResponse.Len(), streamErr)
		fullResponse.WriteString("\n\n[")
		fullResponse.WriteString(truncationLabelForError(streamErr))
		fullResponse.WriteString("]")
	}

	endMsg := protocol.NewMessage(
		protocol.MessageTypeStreamEnd,
		msg.Channel,
		a.Info,
		"",
	)
	endMsg.ID = streamMsgID
	endMsg.ReplyTo = msg.ID
	if msg.IsInThread() {
		endMsg.ThreadID = msg.ThreadID
		endMsg.IsThreadReply = true
	}
	a.Hub.BroadcastDirect(msg.Channel, endMsg)

	return fullResponse.String(), streamMsgID, fullReasoning.String(), nil
}

// agentHasDedicatedContext returns true for agent types that already have their
// own file-context strategy (repo agents use their index, CLI agents have shell access).
func (a *Agent) agentHasDedicatedContext() bool {
	switch a.Info.Type {
	case protocol.AgentTypeRepo, protocol.AgentTypeCLI,
		protocol.AgentTypeModerator, protocol.AgentTypeAssistant, protocol.AgentTypeConfluence,
		protocol.AgentTypeMaps, protocol.AgentTypeMusic, protocol.AgentTypeArena:
		return true
	default:
		return false
	}
}

func truncationLabelForError(err error) string {
	_, code, _ := classifyUserFacingError(err)
	switch code {
	case "timeout":
		return "Response truncated due to timeout"
	case "rate_limit":
		return "Response truncated due to provider rate limit"
	default:
		return "Response truncated due to provider error"
	}
}

// appendFileChangeMachineBlockDocs writes the canonical [FILE_CHANGE] spec parsed by
// maybeSubmitFileChangeFromResponse. Shared by normal chat and collaboration execution.
func appendFileChangeMachineBlockDocs(sb *strings.Builder) {
	sb.WriteString("[FILE_CHANGE]\n")
	sb.WriteString("operation: create|edit|delete|move\n")
	sb.WriteString("path: relative/path/from/workspace (must include a file extension, e.g. tailwind.config.js or src/App.tsx — never a label like \"File:\")\n")
	sb.WriteString("old_path: relative/path (move only)\n")
	sb.WriteString("new_path: relative/path (move only)\n")
	sb.WriteString("```new\n<new content for create/edit>\n```\n")
	sb.WriteString("```old\n<old content for edit>\n```\n")
	sb.WriteString("[/FILE_CHANGE]\n")
	sb.WriteString("If no file change should be proposed, do not include a FILE_CHANGE block.\n")
}

// collectIncludedFilePaths extracts file paths that are already present in
// the prompt via workspace context (open editor tabs) so the scanner can
// skip them.
func collectIncludedFilePaths(msg *protocol.Message) map[string]bool {
	paths := make(map[string]bool)
	if msg.Metadata == nil {
		return paths
	}
	if raw, ok := msg.Metadata["task_context_paths"]; ok {
		switch v := raw.(type) {
		case []string:
			for _, p := range v {
				if p != "" {
					paths[p] = true
				}
			}
		case []interface{}:
			for _, item := range v {
				if p, ok := item.(string); ok && p != "" {
					paths[p] = true
				}
			}
		}
	}
	wsCtx, ok := msg.Metadata["workspace_context"]
	if !ok {
		return paths
	}
	ctxMap, ok := wsCtx.(map[string]interface{})
	if !ok {
		return paths
	}
	if files, ok := ctxMap["open_files"].([]interface{}); ok {
		for _, f := range files {
			if fm, ok := f.(map[string]interface{}); ok {
				if p, ok := fm["path"].(string); ok && p != "" {
					paths[p] = true
				}
			}
		}
	}
	return paths
}

// resolveWorkspacePath determines the workspace root from available sources.
// Priority: 1) workspace context metadata, 2) agent's stored WorkspacePath
