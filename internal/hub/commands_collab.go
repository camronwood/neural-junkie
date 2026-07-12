package hub

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// collaborationCwdForChannel returns source repo or sandbox path for a collab channel terminal.
func (ch *CommandHandler) collaborationCwdForChannel(channel string) string {
	channel = strings.TrimSpace(channel)
	if channel == "" || !strings.HasPrefix(channel, "collab-") {
		return ""
	}
	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ""
	}
	for _, c := range cm.ListActive() {
		if c.Channel != channel {
			continue
		}
		if p := strings.TrimSpace(c.SourceRepoPath); p != "" && !ch.hub.pathUnderCollabAssets(p) {
			return p
		}
		if p := strings.TrimSpace(c.WorkingDirectory); p != "" {
			return p
		}
	}
	return ""
}

// SetAssistantAgent sets the assistant agent reference for meeting notes functionality

// ── Collaboration Command Handlers ──────────────────────────────────

// handleCollaborate starts a multi-agent collaboration.
// Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 @Agent3 build a CLI tool that encrypts files
func (ch *CommandHandler) handleCollaborate(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	flagParse, tail, flagErr := parseCollaborateLeadFlags(parts)
	if flagErr != "" {
		return ch.systemResponse(msg.Channel, flagErr), nil
	}
	discussionCfg := flagParse.Discussion
	if len(tail) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /collaborate [--rounds N] [--messages M] [--workspace] [--worktree] [--allow-agent-adds] @Agent1 @Agent2 ... description\nAt least 2 agents (max 3) and a description are required."), nil
	}

	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration manager is not available."), nil
	}

	// Parse agent mentions and description
	mentionStrings := protocol.ParseMentions(strings.Join(tail, " "))
	if len(mentionStrings) < 2 {
		// Fallback: scan the full command in case flag parsing left @mentions in an unexpected segment.
		mentionStrings = protocol.ParseMentions(strings.Join(parts[1:], " "))
	}
	if len(mentionStrings) < 2 && msg != nil {
		mentionStrings = protocol.ParseMentions(msg.Content)
	}
	if len(mentionStrings) < 2 {
		return ch.systemResponse(msg.Channel, "❌ At least 2 agents must be @mentioned.\nUsage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description"), nil
	}

	// Resolve mentions to agent IDs
	resolved := make(map[string]bool)
	agentIDs := ch.hub.ResolveMentionsWithValidation(mentionStrings, resolved, msg.Channel)
	if len(agentIDs) < 2 {
		unresolved := []string{}
		for _, m := range mentionStrings {
			if !resolved[m] {
				unresolved = append(unresolved, "@"+m)
			}
		}
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Could not resolve enough agents. Unresolved: %s\nAvailable agents: %s",
			strings.Join(unresolved, ", "), ch.hub.getAgentListString())), nil
	}
	if len(agentIDs) > collaboration.HardMaxAgentsPerCollaboration {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ At most %d agents can join a collaboration (you mentioned %d).\nPick 2–%d agents for /collaborate.",
			collaboration.HardMaxAgentsPerCollaboration, len(agentIDs), collaboration.HardMaxAgentsPerCollaboration)), nil
	}

	// Extract description (everything after mentions)
	description := strings.Join(tail, " ")
	for _, m := range mentionStrings {
		description = strings.Replace(description, "@"+m, "", 1)
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return ch.systemResponse(msg.Channel, "❌ A description is required after the agent mentions."), nil
	}

	ch.clearCollaborateRedirect()

	var createOpts collaboration.CreateOptions
	createOpts.AllowAgentParticipantRequests = flagParse.AllowAgentParticipantRequests
	sourceWorkspacePath, sourceWorkspaceWarn := ch.hub.resolveCollaborateSourceRepoPath(msg, flagParse)
	if sourceWorkspacePath != "" {
		createOpts.SourceRepoPath = sourceWorkspacePath
	}
	createOpts.SourceWorkspaceContext = workspaceContextForCollaboration(msg, flagParse, sourceWorkspacePath, description)
	createOpts.AttachWorkspaceContext = flagParse.AttachWorkspace
	if flagParse.Worktree {
		createOpts.ExecutionMode = collaboration.ExecutionModeWorktree
		if createOpts.SourceRepoPath != "" {
			if err := ch.hub.validateGitRepoForCollaboration(createOpts.SourceRepoPath); err != nil {
				if flagParse.AttachWorkspace {
					return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ --worktree with --workspace: %v", err)), nil
				}
				createOpts.SourceRepoPath = ""
			}
		}
	}

	collab, err := cm.CreateCollaboration(description, agentIDs, msg.Channel, msg.From.Name, discussionCfg, createOpts)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to create collaboration: %v", err)), nil
	}

	collabChannelName := "collab-" + collab.ID
	ch.hub.CreateChannelWithType(
		collabChannelName,
		collab.Title,
		msg.Channel,
		protocol.ChannelTypeCollaboration,
		msg.From.Name,
	)
	if err := cm.BindCollaborationChannel(collab.ID, collabChannelName); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to bind collaboration channel: %v", err)), nil
	}

	// Join hub membership and subscribe so agents receive collab messages (DM-spawned
	// agents disable channel discovery and would otherwise never hear this channel).
	var setupFailures []string
	for _, participant := range collab.Agents {
		liveID := ch.resolveLiveAgentID(participant.AgentID, participant.AgentName, participant.AgentType)
		if err := ch.hub.AddAgentToChannel(liveID, collabChannelName); err != nil {
			setupFailures = append(setupFailures, fmt.Sprintf("join %s: %v", shortID(liveID), err))
			log.Printf("[Collaboration] Warning: failed to add participant %s to channel %s: %v", liveID, collabChannelName, err)
			continue
		}
		if err := ch.ensureAgentSubscribedToChannel(ctx, liveID, participant.AgentName, participant.AgentType, collabChannelName); err != nil {
			setupFailures = append(setupFailures, fmt.Sprintf("subscribe %s: %v", shortID(liveID), err))
			log.Printf("[Collaboration] Warning: failed to subscribe participant %s to %s: %v", liveID, collabChannelName, err)
		}
	}
	if len(setupFailures) > 0 {
		ch.setCollaborateRedirect(collabChannelName, collab.ID)
		return ch.systemResponse(msg.Channel, fmt.Sprintf(
			"⚠️ Collaboration `%s` created in **#%s** but some participants could not join: %s\n\nSwitching you to the collaboration channel.",
			collab.ID[:8], collabChannelName, strings.Join(setupFailures, "; "),
		)), nil
	}

	// Build agent list for display
	var agentListStr strings.Builder
	for i, a := range collab.Agents {
		if i > 0 {
			agentListStr.WriteString(", ")
		}
		agentListStr.WriteString(fmt.Sprintf("**@%s** (%s)", a.AgentName, a.Role))
	}

	seedBody := fmt.Sprintf("🤝 **Collaboration Started** (ID: `%s`)\n\n**Goal:** %s\n\n**Participants:** %s\n\n**Discussion limits:** %d rounds, %d agent messages (server hard max: %d rounds, %d messages)\n\n**Phase:** Planning (agents will discuss and propose a plan)\n\nAgents, please discuss and create a structured plan with tasks assigned to the agent best suited for each task. Use `- Task N: @AgentName - description` format for tasks.",
		collab.ID[:8], description, agentListStr.String(),
		collab.Discussion.MaxRounds, collab.Discussion.MaxTotalMessages,
		collaboration.HardMaxRounds, collaboration.HardMaxTotalMessages)
	if flagParse.Worktree {
		seedBody += "\n\n**Execution mode:** Git worktree (isolated branch). "
		if collab.SourceRepoPath != "" {
			seedBody += fmt.Sprintf("Source repo: `%s`.", collab.SourceRepoPath)
		} else {
			seedBody += "Source repo will be chosen at execution (desktop active workspace or `--workspace` at start)."
		}
	} else if collab.SourceRepoPath != "" {
		rel := collaboration.ProjectCollabRelPath(collab.ID)
		seedBody += fmt.Sprintf(
			"\n\n**Source workspace:** `%s`.\n\n**Deliverables folder:** `%s/` under the project (plans, research, and outputs). Inspect the repo at the source workspace; write files under the deliverables folder.",
			collab.SourceRepoPath, rel,
		)
	} else {
		seedBody += "\n\n**Research-only** — no project repository is bound. Do not invent repo file paths; use the collaboration sandbox for outputs when execution starts."
	}
	if sourceWorkspaceWarn != "" {
		seedBody += "\n\n" + sourceWorkspaceWarn
	}
	if collab.AllowAgentParticipantRequests {
		seedBody += "\n\n**Agent expansion requests:** Enabled. Agents may suggest adding other agents, but Camron must approve each request."
	}

	// Send the seed message to kick off the planning discussion
	seedMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collabChannelName,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		seedBody,
	)
	seedMsg.SetCollaborationID(collab.ID)
	seedMsg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	if seedMsg.Metadata == nil {
		seedMsg.Metadata = map[string]interface{}{}
	}
	seedMsg.Metadata["collab_internal_event"] = true
	if flagParse.AttachWorkspace {
		ch.attachCollaborationWorkspaceMetadata(msg, seedMsg, false)
	}

	if err := ch.hub.SendMessage(seedMsg); err != nil {
		log.Printf("[Collaboration] Failed to send seed message: %v", err)
		ch.setCollaborateRedirect(collabChannelName, collab.ID)
		return ch.systemResponse(msg.Channel, fmt.Sprintf("⚠️ Collaboration created in **#%s** but failed to start discussion: %v", collabChannelName, err)), nil
	}

	// Set the Collab field on participating agents so they can check collaboration state
	collabClient := ch.hub.NewCollaborationClientAdapter()
	for _, a := range collab.Agents {
		liveID := ch.resolveLiveAgentID(a.AgentID, a.AgentName, a.AgentType)
		ch.setCollabClientOnAgent(liveID, a.AgentName, collabClient)
	}

	if len(collab.Agents) == 0 {
		return ch.systemResponse(msg.Channel, "❌ Collaboration has no participants."), nil
	}

	// Send the first turn prompt to the first agent
	firstAgent := collab.Agents[0]
	turnMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collabChannelName,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("@%s -- You're up first for: %s\n\nPropose a **minimal** task list (3–6 lines) with concrete deliverable paths (`- Task N: @Agent - Write collabs/<id>/file.md …`). Defer debate until tasks are drafted; use each participant's lane.",
			firstAgent.AgentName, description),
	)
	turnMsg.SetCollaborationID(collab.ID)
	turnMsg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	turnMsg.Mentions = []string{ch.resolveLiveAgentID(firstAgent.AgentID, firstAgent.AgentName, firstAgent.AgentType)}
	if turnMsg.Metadata == nil {
		turnMsg.Metadata = map[string]interface{}{}
	}
	turnMsg.Metadata["collab_internal_event"] = true
	if flagParse.AttachWorkspace {
		ch.attachCollaborationWorkspaceMetadata(msg, turnMsg, false)
	}

	if err := ch.hub.SendMessage(turnMsg); err != nil {
		log.Printf("[Collaboration] Failed to send first turn message: %v", err)
		ch.setCollaborateRedirect(collabChannelName, collab.ID)
		return ch.systemResponse(msg.Channel, fmt.Sprintf("⚠️ Collaboration created in **#%s** but failed to prompt first agent: %v", collabChannelName, err)), nil
	}

	// Re-dispatch if the first participant stays silent (15s throttle; no immediate kick — avoids duplicate first-turn prompts).
	go func(collabID string) {
		time.Sleep(collabPlanningHandoffRedispatchAfter)
		ch.hub.KickPlanningDiscussionWatchdog(collabID)
	}(collab.ID)

	ch.setCollaborateRedirect(collabChannelName, collab.ID)

	if sourceWorkspaceWarn != "" {
		warnMsg := ch.systemResponse(msg.Channel, sourceWorkspaceWarn)
		if err := ch.hub.SendMessage(warnMsg); err != nil {
			log.Printf("[Collaboration] workspace warning on %s: %v", msg.Channel, err)
		}
	}

	return nil, nil
}

// handleRunbook creates a draft runbook collaboration (user-built task DAG).
// Usage: /runbook [--workspace] [--worktree] @Agent1 @Agent2 goal description

// handleRunbook creates a draft runbook collaboration (user-built task DAG).
// Usage: /runbook [--workspace] [--worktree] @Agent1 @Agent2 goal description
func (ch *CommandHandler) handleRunbook(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	flagParse, tail, flagErr := parseCollaborateLeadFlags(parts)
	if flagErr != "" {
		return ch.systemResponse(msg.Channel, flagErr), nil
	}
	if len(tail) < 1 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /runbook [--workspace] [--worktree] @Agent1 ... description\nAt least 1 agent and a description are required."), nil
	}

	mentionStrings := protocol.ParseMentions(strings.Join(tail, " "))
	if len(mentionStrings) < 1 {
		return ch.systemResponse(msg.Channel, "❌ At least 1 agent must be @mentioned.\nUsage: /runbook @Agent1 ... description"), nil
	}

	resolved := make(map[string]bool)
	agentIDs := ch.hub.ResolveMentionsWithValidation(mentionStrings, resolved, msg.Channel)
	if len(agentIDs) < 1 {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Could not resolve agents. Available: %s", ch.hub.getAgentListString())), nil
	}

	description := strings.Join(tail, " ")
	for _, m := range mentionStrings {
		description = strings.Replace(description, "@"+m, "", 1)
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return ch.systemResponse(msg.Channel, "❌ A description is required after the agent mentions."), nil
	}

	ch.clearCollaborateRedirect()

	execMode := ""
	if flagParse.Worktree {
		execMode = string(collaboration.ExecutionModeWorktree)
	}
	sourceRepo, _ := ch.hub.resolveCollaborateSourceRepoPath(msg, flagParse)
	if flagParse.Worktree && sourceRepo != "" {
		if err := ch.hub.validateGitRepoForCollaboration(sourceRepo); err != nil {
			if flagParse.AttachWorkspace {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ --worktree with --workspace: %v", err)), nil
			}
			sourceRepo = ""
		}
	}

	result, err := ch.hub.CreateRunbookSession(RunbookCreateRequest{
		Description:    description,
		AgentIDs:       agentIDs,
		Channel:        msg.Channel,
		CreatedBy:      msg.From.Name,
		ExecutionMode:  execMode,
		SourceRepoPath: sourceRepo,
	})
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to create runbook: %v", err)), nil
	}

	collabChannelName := result.CollaborationChannel
	for _, participantID := range agentIDs {
		_ = ch.hub.AddAgentToChannel(participantID, collabChannelName)
		_ = ch.EnsureAgentSubscribedToChannel(ctx, participantID, collabChannelName)
	}

	seedBody := fmt.Sprintf("📋 **Runbook created** (ID: `%s`)\n\n**Goal:** %s\n\n**Phase:** Draft — open **Runbook builder** in the desktop app (or PUT `/api/runbooks/%s`) to add tasks, dependencies, and assignees. Then **Submit** → **Start** to execute.",
		result.CollaborationID[:8], description, result.CollaborationID)
	seedMsg := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		collabChannelName,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		seedBody,
	)
	seedMsg.SetCollaborationID(result.CollaborationID)
	seedMsg.SetCollaborationPhase(string(collaboration.PhaseDraft))
	if seedMsg.Metadata == nil {
		seedMsg.Metadata = map[string]interface{}{}
	}
	seedMsg.Metadata["collab_internal_event"] = true
	_ = ch.hub.SendMessage(seedMsg)

	ch.setCollaborateRedirect(collabChannelName, result.CollaborationID)

	out := ch.systemResponse(msg.Channel, fmt.Sprintf("📋 **Runbook** `%s` created in #%s. Open the Runbook builder to define tasks.", result.CollaborationID[:8], collabChannelName))
	out.SetCollaborationID(result.CollaborationID)
	return out, nil
}

// handleRunbookRun instantiates and starts a library definition.
// Usage: /runbook-run <definition-id> @Agent1 [--input key=value]
func (ch *CommandHandler) handleRunbookRun(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /runbook-run <definition-id> @Agent1 [@Agent2 ...] [--input key=value]"), nil
	}
	defID := strings.TrimSpace(parts[1])
	inputs := map[string]string{}
	var mentionParts []string
	for _, p := range parts[2:] {
		if strings.HasPrefix(p, "--input") {
			kv := strings.TrimPrefix(p, "--input")
			kv = strings.TrimSpace(strings.TrimPrefix(kv, "="))
			if idx := strings.Index(kv, "="); idx > 0 {
				inputs[kv[:idx]] = kv[idx+1:]
			}
			continue
		}
		if strings.HasPrefix(p, "--input=") {
			kv := strings.TrimPrefix(p, "--input=")
			if idx := strings.Index(kv, "="); idx > 0 {
				inputs[kv[:idx]] = kv[idx+1:]
			}
			continue
		}
		mentionParts = append(mentionParts, p)
	}
	mentionStrings := protocol.ParseMentions(strings.Join(mentionParts, " "))
	if len(mentionStrings) < 1 {
		return ch.systemResponse(msg.Channel, "❌ At least one @agent is required.\nUsage: /runbook-run health-check-alert @Assistant"), nil
	}
	resolved := make(map[string]bool)
	agentIDs := ch.hub.ResolveMentionsWithValidation(mentionStrings, resolved, msg.Channel)
	if len(agentIDs) < 1 {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Could not resolve agents. Available: %s", ch.hub.getAgentListString())), nil
	}
	result, err := ch.hub.InstantiateDefinition(defID, 0, RunbookCreateRequest{
		AgentIDs:  agentIDs,
		Channel:   msg.Channel,
		CreatedBy: msg.From.Name,
		RunInputs: inputs,
	})
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to instantiate runbook: %v", err)), nil
	}
	if _, err := ch.hub.SubmitRunbookForReview(result.CollaborationID); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to submit runbook: %v", err)), nil
	}
	if _, err := ch.hub.StartRunbook(result.CollaborationID, inputs); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start runbook: %v", err)), nil
	}
	out := ch.systemResponse(msg.Channel, fmt.Sprintf("▶️ **Runbook run started** — `%s` from definition **%s** in #%s", result.CollaborationID[:8], defID, result.CollaborationChannel))
	out.SetCollaborationID(result.CollaborationID)
	return out, nil
}

// setCollabClientOnAgent sets the CollaborationClient on any agent type
// that embeds the base Agent struct. It searches known agent registries.

func (ch *CommandHandler) setCollabClientOnAgent(agentID, agentName string, client agent.CollaborationClient) {
	liveID := ch.resolveLiveAgentID(agentID, agentName, protocol.AgentType(""))
	if ch.setCollabClientOnAgentID(liveID, client) {
		return
	}
	log.Printf("[Collaboration] Warning: could not set collab client on agent %s (%s)", agentName, shortID(liveID))
}

func (ch *CommandHandler) setCollabClientOnAgentID(agentID string, client agent.CollaborationClient) bool {
	if a := ch.findAgentByID(agentID); a != nil {
		a.SetCollabClient(client)
		return true
	}
	ch.agentsMu.RLock()
	defer ch.agentsMu.RUnlock()
	if ch.assistantAgent != nil && ch.assistantAgent.Info.ID == agentID {
		ch.assistantAgent.SetCollabClient(client)
		return true
	}
	for _, ra := range ch.repoAgents {
		if ra != nil && ra.GetAgentInfo().ID == agentID {
			ra.SetCollabClient(client)
			return true
		}
	}
	for _, ca := range ch.confluenceAgents {
		if ca != nil && ca.GetAgentInfo().ID == agentID && ca.Agent != nil {
			ca.Agent.SetCollabClient(client)
			return true
		}
	}
	return false
}

func (ch *CommandHandler) handleSubmitPlan(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /submit-plan <collab-id>"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	snap, err := cm.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found."), nil
	}

	switch snap.Phase {
	case collaboration.PhaseReviewing, collaboration.PhaseApproved:
		out := ch.systemResponse(msg.Channel, fmt.Sprintf(
			"📋 **Already in review** (`%s`) — read the plan and session summary, then **Approve & start** (`/approve-plan %s`) or **Revise** (`/revise-plan`).",
			collabID[:8], collabID[:8],
		))
		out.SetCollaborationID(collabID)
		return out, nil
	case collaboration.PhaseExecuting, collaboration.PhaseCompleted, collaboration.PhaseCancelled:
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Collaboration is **%s** — cannot submit for review.", snap.Phase)), nil
	case collaboration.PhasePlanning:
		if snap.Discussion != nil && snap.Discussion.Status == collaboration.DiscussionActive {
			return ch.systemResponse(msg.Channel, fmt.Sprintf(
				"⏳ **Planning still in progress** (`%s`) — agent discussion is active (round %d/%d, %d/%d messages). **Submit for review** unlocks when planning finishes (consensus, message limit, or round limit).",
				collabID[:8],
				snap.Discussion.CurrentRound, snap.Discussion.MaxRounds,
				snap.Discussion.TotalMessageCount, snap.Discussion.MaxTotalMessages,
			)), nil
		}
		// fall through
	default:
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Collaboration is in **%s** — submit for review only applies during planning.", snap.Phase)), nil
	}

	if _, err := cm.TransitionToReviewing(collabID); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}

	ch.hub.persistCollaborationReviewAssets(collabID)

	submitBody := fmt.Sprintf(
		"📋 **Submitted for your review** (`%s`)\n\nAgent planning is paused. A facilitator will post a **session summary**; when it appears, review the plan and use **Approve & start** (`/approve-plan %s`) or **Revise** (`/revise-plan %s …`).",
		collabID[:8], collabID[:8], collabID[:8],
	)
	if reviewSnap, err := cm.GetCollaborationSnapshot(collabID); err == nil && reviewSnap != nil {
		submitBody += collaboration.FormatTaskPathWarnings(
			collaboration.ValidateCollaborationPaths(reviewSnap),
			reviewSnap.SourceRepoPath,
		)
	}

	out := ch.systemResponse(msg.Channel, submitBody)
	out.SetCollaborationID(collabID)
	out.SetCollaborationPhase(string(collaboration.PhaseReviewing))
	return out, nil
}

func (ch *CommandHandler) handleApprovePlan(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /approve-plan <collab-id>"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	collab, err := cm.ApprovePlan(collabID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}
	snapPre, _ := cm.GetCollaborationSnapshot(collabID)
	if snapPre != nil && snapPre.PlanningRecapStatus == collaboration.RecapStatusFailed {
		log.Printf("[Collaboration] Approving %s after failed planning recap (user override)", collabID[:8])
	}
	snapForTasks, _ := cm.GetCollaborationSnapshot(collabID)
	if snapForTasks == nil {
		snapForTasks = collab
	}
	var approveWarnings []string
	if snapForTasks.Plan != nil && strings.TrimSpace(snapForTasks.Plan.Content) != "" {
		extractedTasks, warnings := collaboration.NormalizeAndValidateTasksForExecution(snapForTasks)
		approveWarnings = warnings
		if len(extractedTasks) > 0 {
			if err := cm.SetTasks(collabID, extractedTasks); err != nil {
				log.Printf("[Collaboration] Failed to set extracted tasks for %s: %v", collabID[:8], err)
			}
		}
		if err := cm.SetApproveWarnings(collabID, approveWarnings); err != nil {
			log.Printf("[Collaboration] SetApproveWarnings for %s: %v", collabID[:8], err)
		}
	}

	pathWarnSnap, _ := cm.GetCollaborationSnapshot(collabID)
	repoPath := ""
	if pathWarnSnap != nil {
		repoPath = pathWarnSnap.SourceRepoPath
	}
	pathWarnings := collaboration.FormatTaskPathWarnings(
		collaboration.ValidateCollaborationPaths(pathWarnSnap),
		repoPath,
	)

	// Transition to executing
	_, err = cm.TransitionToExecuting(collabID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start execution: %v", err)), nil
	}

	if _, err := cm.EnsureExecutionTasks(collabID); err != nil {
		log.Printf("[Collaboration] EnsureExecutionTasks for %s: %v", collabID[:8], err)
	}

	collabSnap, err := cm.GetCollaborationSnapshot(collabID)
	if err != nil || collabSnap == nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Could not load collaboration %s after execution start", collabID[:8])), nil
	}

	var autoAckErr error
	if collaboration.ShouldAutoAckWorkspaceOnApprove(collabSnap) && !collabSnap.WorkspaceAcknowledged {
		if err := ch.hub.AcknowledgeCollaborationWorkspace(collabID, ""); err != nil {
			log.Printf("[Collaboration] Auto workspace ack for %s: %v", collabID[:8], err)
			autoAckErr = err
		} else {
			collabSnap, err = cm.GetCollaborationSnapshot(collabID)
			if err != nil || collabSnap == nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Could not reload collaboration %s after workspace ack", collabID[:8])), nil
			}
		}
	}

	// Notify agents about their assigned tasks
	var taskSummary strings.Builder
	taskSummary.WriteString(fmt.Sprintf("✅ **Plan Approved** (Collaboration `%s`)\n\n", collabID[:8]))
	taskSummary.WriteString("**Assigned Tasks:**\n\n")

	if len(collabSnap.Tasks) == 0 {
		taskSummary.WriteString("_No tasks to assign (no participants)._\n\n")
	}

	for i, task := range collabSnap.Tasks {
		status := "⬜"
		assigneeLabel := task.AssignedName
		if assigneeLabel == "" {
			assigneeLabel = "unassigned"
		}
		taskSummary.WriteString(fmt.Sprintf("%s **Task %d:** %s\n   Assigned to: **@%s**\n\n", status, i+1, task.Description, assigneeLabel))
	}

	canDispatch := ch.hub.CollaborationCanDispatchTasks(collabSnap)
	if canDispatch {
		if collaboration.ShouldAutoAckWorkspaceOnApprove(collabSnap) && collabSnap.WorkspaceAcknowledged {
			taskSummary.WriteString("\n**Tasks dispatching** — workspace was auto-confirmed (bound project repo).\n")
		} else {
			taskSummary.WriteString("\n**Tasks dispatching** to assigned agents.\n")
		}
	} else {
		if autoAckErr != nil {
			taskSummary.WriteString(fmt.Sprintf(
				"\n⚠️ **Auto workspace confirmation failed** — %v. Use **Continue** or `/ack-collab-workspace %s` before tasks can run.\n",
				autoAckErr, collabID[:8],
			))
		}
		if collabSnap.ExecutionMode == collaboration.ExecutionModeWorktree {
			if strings.TrimSpace(collabSnap.WorkingDirectory) != "" {
				taskSummary.WriteString(fmt.Sprintf("\n**Git worktree:** `%s` (branch `%s`)\n", collabSnap.WorkingDirectory, collabSnap.WorktreeBranch))
			} else {
				taskSummary.WriteString("\n**Git worktree:** will be created from your active workspace when you confirm.\n")
			}
		}
		chName := strings.TrimSpace(collabSnap.Channel)
		if chName == "" {
			chName = "the collaboration channel"
		} else {
			chName = "#" + chName
		}
		taskSummary.WriteString(fmt.Sprintf("\n⏸ **Waiting for workspace confirmation** — agents will receive their task prompts after you click **Continue** on %s in the desktop app, or run `/ack-collab-workspace %s` here.\n", chName, collabID[:8]))
		if collabSnap.ExecutionMode == collaboration.ExecutionModeWorktree && collabSnap.WorktreeBranch != "" && strings.TrimSpace(collabSnap.WorkingDirectory) != "" {
			taskSummary.WriteString(fmt.Sprintf("_After completion, merge branch `%s` from your main checkout._\n", collabSnap.WorktreeBranch))
		}
	}
	if pathWarnings != "" {
		taskSummary.WriteString(pathWarnings)
	}
	if planWarn := collaboration.FormatApproveWarnings(approveWarnings); planWarn != "" {
		taskSummary.WriteString(planWarn)
	}

	out := ch.systemResponse(msg.Channel, taskSummary.String())
	out.SetCollaborationID(collabID)

	inheritMsg := msg
	ch.hub.collabAsyncWG.Add(1)
	go func() {
		defer ch.hub.collabAsyncWG.Done()
		ch.hub.persistCollaborationReviewAssets(collabID)
		if !canDispatch {
			return
		}
		fresh, err := cm.GetCollaborationSnapshot(collabID)
		if err != nil || fresh == nil {
			log.Printf("[Collaboration] approve-plan async dispatch snapshot for %s: %v", collabID[:8], err)
			return
		}
		ch.hub.dispatchCollabTaskMessages(fresh, inheritMsg, false)
	}()

	return out, nil
}

func (ch *CommandHandler) handleAckCollabWorkspace(_ context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /ack-collab-workspace <collab-id>"), nil
	}
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}
	if err := ch.hub.AcknowledgeCollaborationWorkspace(collabID, ""); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}
	out := ch.systemResponse(msg.Channel, fmt.Sprintf("✅ **Workspace confirmed** (`%s`). Task prompts are with the agents.", collabID[:8]))
	out.SetCollaborationID(collabID)
	return out, nil
}

func (ch *CommandHandler) handleResumePlan(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /resume-plan <collab-id>"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	snap, err := cm.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found."), nil
	}

	switch snap.Phase {
	case collaboration.PhaseReviewing, collaboration.PhaseApproved:
		return ch.handleApprovePlan(ctx, msg, []string{"/approve-plan", parts[1]})
	case collaboration.PhaseExecuting:
		if _, err := cm.EnsureExecutionTasks(collabID); err != nil {
			log.Printf("[Collaboration] resume-plan EnsureExecutionTasks for %s: %v", collabID[:8], err)
		}
		snap, err = cm.GetCollaborationSnapshot(collabID)
		if err != nil || snap == nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Could not load collaboration %s after resume", collabID[:8])), nil
		}
		open := func(t collaboration.CollaborationTask) bool {
			return t.Status == collaboration.TaskPending ||
				t.Status == collaboration.TaskInProgress ||
				t.Status == collaboration.TaskBlocked
		}
		n := 0
		for _, t := range snap.Tasks {
			if open(t) {
				n++
			}
		}
		if n == 0 {
			out := ch.systemResponse(msg.Channel, fmt.Sprintf("↻ **Resume** (`%s`) — all tasks are already finished. Nothing to re-send.", collabID[:8]))
			out.SetCollaborationID(collabID)
			return out, nil
		}
		if !ch.hub.CollaborationCanDispatchTasks(snap) {
			out := ch.systemResponse(msg.Channel, fmt.Sprintf("⏸ **Workspace not confirmed yet** (`%s`) — use the desktop **Continue** prompt or `/ack-collab-workspace %s` before re-sending task prompts.", collabID[:8], collabID[:8]))
			out.SetCollaborationID(collabID)
			return out, nil
		}
		ch.hub.dispatchCollabTaskMessagesFilter(snap, msg, open, true)
		out := ch.systemResponse(msg.Channel, fmt.Sprintf("↻ **Resumed** (`%s`) — re-sent **%d** open task prompt(s) to assignees.", collabID[:8], n))
		out.SetCollaborationID(collabID)
		return out, nil
	case collaboration.PhasePlanning:
		return ch.systemResponse(msg.Channel, fmt.Sprintf("⏳ **Still planning** (`%s`) — use **Submit for review** (`/submit-plan %s`) when ready, then **Approve & start** after the session summary.", collabID[:8], collabID[:8])), nil
	default:
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Collaboration is **%s** — start a new session with `/collaborate`.", snap.Phase)), nil
	}
}

func (ch *CommandHandler) handleRevisePlan(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /revise-plan <collab-id> <feedback>"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	feedback := strings.Join(parts[2:], " ")

	collab, err := cm.RevisePlan(collabID, feedback)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}

	revisionChannel := strings.TrimSpace(collab.Channel)
	if revisionChannel == "" {
		revisionChannel = msg.Channel
	}

	// Send feedback to the collaboration channel
	revisionMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		revisionChannel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("📝 **Plan Revision Requested** (Collaboration `%s`)\n\n**Feedback:** %s\n\nAgents, please revise the plan based on this feedback.",
			collabID[:8], feedback),
	)
	revisionMsg.SetCollaborationID(collabID)
	revisionMsg.SetCollaborationPhase(string(collaboration.PhasePlanning))

	// Mention all agents to notify them
	for _, a := range collab.Agents {
		revisionMsg.Mentions = append(revisionMsg.Mentions, a.AgentID)
	}

	if err := ch.hub.SendMessage(revisionMsg); err != nil {
		log.Printf("[Collaboration] Failed to send revision message: %v", err)
	}

	return nil, nil
}

func (ch *CommandHandler) handleCancelPlan(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /cancel-plan <collab-id>"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	snap, err := cm.CancelCollaboration(collabID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}
	ch.hub.syncRunbookRunIndex(snap)
	ch.hub.cancelCollaborationRecaps(collabID)

	// Abort any in-flight agent generations on the collaboration channel so cancelled
	// runs don't keep consuming Ollama/CLI capacity and cascading timeouts.
	cancelChannel := strings.TrimSpace(snap.Channel)
	if cancelChannel == "" {
		cancelChannel = msg.Channel
	}
	ch.AbortRuntimeAgentsOnChannel(cancelChannel)
	ch.hub.broadcastChannelInterjectAbort(cancelChannel)

	out := ch.systemResponse(msg.Channel, fmt.Sprintf("🛑 **Collaboration Cancelled** (`%s`)", collabID[:8]))
	out.SetCollaborationID(collabID)
	return out, nil
}

func (ch *CommandHandler) handleCompleteCollab(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	_ = ctx
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /complete-collab <collab-id> [--force]"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration manager is not available."), nil
	}

	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	force := false
	for _, p := range parts[2:] {
		if strings.EqualFold(strings.TrimSpace(p), "--force") {
			force = true
		}
	}

	snap, err := cm.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found."), nil
	}
	if snap.Phase == collaboration.PhaseCompleted {
		out := ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Collaboration `%s` is already completed.", collabID[:8]))
		out.SetCollaborationID(collabID)
		out.SetCollaborationPhase(string(collaboration.PhaseCompleted))
		return out, nil
	}
	if snap.Phase == collaboration.PhaseCancelled {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Collaboration `%s` was cancelled. Start a new session with `/collaborate`.", collabID[:8])), nil
	}

	if collaboration.HasOpenTasks(snap) && !force {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("❌ Collaboration `%s` still has open tasks:\n", collabID[:8]))
		for i, title := range collaboration.OpenTaskTitles(snap) {
			sb.WriteString(fmt.Sprintf("- Task %d: %s\n", i+1, title))
		}
		sb.WriteString("\nUse `/complete-collab ")
		sb.WriteString(collabID[:8])
		sb.WriteString(" --force` after you confirm, or mark tasks done with `/collab-task-done`.")
		return ch.systemResponse(msg.Channel, sb.String()), nil
	}

	opts := collaboration.FinalizeOptions{MarkOpenTasksComplete: force}
	reason := "Marked complete by user."
	if force && collaboration.HasOpenTasks(snap) {
		reason = "Closed by user (open tasks marked done)."
	}
	ch.hub.requestFinalRecapAndFinalize(collabID, msg.Channel, reason, opts)

	out := ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Collaboration `%s` closing — session summary in progress.", collabID[:8]))
	out.SetCollaborationID(collabID)
	if snap, _ := cm.GetCollaborationSnapshot(collabID); snap != nil {
		out.SetCollaborationPhase(string(snap.Phase))
	}
	ch.hub.attachCollaborationData(out)
	return out, nil
}

func (ch *CommandHandler) handleCollabTaskDone(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	_ = ctx
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /collab-task-done <collab-id> <task#|task-id-prefix>"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration manager is not available."), nil
	}

	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	taskID, err := cm.ResolveTaskIndex(collabID, parts[2])
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}

	snap, err := cm.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found."), nil
	}
	if snap.Phase == collaboration.PhaseCompleted || snap.Phase == collaboration.PhaseCancelled {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Collaboration is **%s**.", snap.Phase)), nil
	}

	effects, err := cm.UpdateTaskStatusWithEffects(collabID, taskID, collaboration.TaskCompleted, "Marked complete by user")
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}

	var taskTitle string
	for _, t := range snap.Tasks {
		if t.ID == taskID {
			taskTitle = t.Title
			break
		}
	}
	if taskTitle == "" {
		taskTitle = taskID[:8]
	}

	out := ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Task **%s** marked complete (`%s`).", taskTitle, collabID[:8]))
	out.SetCollaborationID(collabID)
	out.SetTaskID(taskID)
	out.SetTaskStatus(string(collaboration.TaskCompleted))

	if cm.AllTasksComplete(collabID) {
		ch.hub.requestFinalRecapAndFinalize(collabID, msg.Channel, "All tasks are done.", collaboration.FinalizeOptions{})
		out.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	} else {
		out.SetCollaborationPhase(string(snap.Phase))
		if effects.ShouldDispatchWave {
			if fresh, err := cm.GetCollaborationSnapshot(collabID); err == nil && fresh != nil && fresh.Phase == collaboration.PhaseExecuting {
				ch.hub.dispatchReadyCollabTasks(fresh, msg, false)
			}
		}
	}
	ch.hub.attachCollaborationData(out)
	return out, nil
}

func (ch *CommandHandler) handleCollabExtend(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	id, extraRounds, extraMessages, errMsg := parseCollabExtendArgs(parts)
	if errMsg != "" {
		return ch.systemResponse(msg.Channel, errMsg), nil
	}

	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration manager is not available."), nil
	}
	collabID := ch.resolveCollabID(id)
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to list active IDs."), nil
	}

	collab, err := cm.ExtendDiscussionLimits(collabID, extraRounds, extraMessages)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}
	if collab.Discussion == nil {
		return ch.systemResponse(msg.Channel, "❌ Internal error: missing discussion after extend."), nil
	}

	d := collab.Discussion
	out := ch.systemResponse(msg.Channel, fmt.Sprintf(
		"✅ **Discussion limits extended** (`%s`)\n\n**New caps:** %d rounds, %d agent messages (hard max %d / %d).\n**Status:** %s",
		collabID[:8], d.MaxRounds, d.MaxTotalMessages, collaboration.HardMaxRounds, collaboration.HardMaxTotalMessages, d.Status,
	))
	out.SetCollaborationID(collabID)
	return out, nil
}

func (ch *CommandHandler) handleCollabRename(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	_ = ctx
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /collab-rename <collab-id> <title>"), nil
	}
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to list active IDs."), nil
	}
	title := strings.TrimSpace(strings.Join(parts[2:], " "))
	if title == "" {
		return ch.systemResponse(msg.Channel, "❌ Title cannot be empty."), nil
	}

	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration manager is not available."), nil
	}
	collab, err := cm.SetCollaborationTitle(collabID, title)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}
	if collab.Channel != "" {
		if err := ch.hub.SetChannelDescription(collab.Channel, collab.Title); err != nil {
			log.Printf("[Collaboration] rename: channel description not updated: %v", err)
		}
	}
	out := ch.systemResponse(msg.Channel, fmt.Sprintf("✅ **Renamed** (`%s`) → **%s**", collabID[:8], collab.Title))
	out.SetCollaborationID(collabID)
	return out, nil
}

// attachCollaborationWorkspaceMetadata copies workspace context onto collab channel messages
// when /collaborate was invoked with --workspace (outline scope; no open file bodies).

// attachCollaborationWorkspaceMetadata copies workspace context onto collab channel messages
// when /collaborate was invoked with --workspace (outline scope; no open file bodies).
func (ch *CommandHandler) attachCollaborationWorkspaceMetadata(src, dst *protocol.Message, fullOpenFiles bool) {
	if workspacePathFromMessageMetadata(src) == "" {
		return
	}
	inheritWorkspaceContextMetadata(src, dst, !fullOpenFiles)
}

// workspaceContextFromMessageMetadata returns a copy of workspace_context from outbound metadata.

// workspaceContextFromMessageMetadata returns a copy of workspace_context from outbound metadata.
func workspaceContextFromMessageMetadata(msg *protocol.Message) map[string]interface{} {
	if msg == nil || msg.Metadata == nil {
		return nil
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return nil
	}
	ctxMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	out := map[string]interface{}{}
	for k, v := range ctxMap {
		out[k] = v
	}
	if p, _ := out["workspace_path"].(string); strings.TrimSpace(p) == "" {
		return nil
	}
	return out
}

// inheritWorkspaceContextMetadata copies workspace metadata from src to dst.
// When outlineOnly is true, only name/path/tree are copied — no open file bodies.

// inheritWorkspaceContextMetadata copies workspace metadata from src to dst.
// When outlineOnly is true, only name/path/tree are copied — no open file bodies.
func inheritWorkspaceContextMetadata(src, dst *protocol.Message, outlineOnly bool) {
	if src == nil || dst == nil || src.Metadata == nil {
		return
	}
	rawCtx, ok := src.Metadata["workspace_context"]
	if !ok {
		return
	}
	ctxMap, ok := rawCtx.(map[string]interface{})
	if !ok {
		return
	}
	safeCtx := map[string]interface{}{}
	if workspaceName, ok := ctxMap["workspace_name"].(string); ok {
		safeCtx["workspace_name"] = workspaceName
	}
	if workspacePath, ok := ctxMap["workspace_path"].(string); ok {
		safeCtx["workspace_path"] = workspacePath
	}
	if fileTree, ok := ctxMap["file_tree"].(string); ok {
		if len(fileTree) > 12000 {
			fileTree = fileTree[:12000] + "\n... (truncated)"
		}
		safeCtx["file_tree"] = fileTree
	}
	if !outlineOnly {
		if openFiles, ok := ctxMap["open_files"].([]interface{}); ok {
			trimmedFiles := make([]map[string]interface{}, 0, len(openFiles))
			for _, entry := range openFiles {
				fileMeta, ok := entry.(map[string]interface{})
				if !ok {
					continue
				}
				trimmed := map[string]interface{}{}
				if path, ok := fileMeta["path"].(string); ok {
					trimmed["path"] = path
				}
				if language, ok := fileMeta["language"].(string); ok {
					trimmed["language"] = language
				}
				if isActive, ok := fileMeta["is_active"].(bool); ok {
					trimmed["is_active"] = isActive
				}
				if content, ok := fileMeta["content"].(string); ok && content != "" {
					trimmed["content"] = content
				}
				if len(trimmed) > 0 {
					trimmedFiles = append(trimmedFiles, trimmed)
				}
			}
			if len(trimmedFiles) > 0 {
				safeCtx["open_files"] = trimmedFiles
			}
		}
	}
	if len(safeCtx) == 0 {
		return
	}
	if dst.Metadata == nil {
		dst.Metadata = map[string]interface{}{}
	}
	dst.Metadata["workspace_context"] = safeCtx
	if outlineOnly {
		dst.Metadata[agent.MetadataContextScope] = agent.ContextScopeOutline
	}
}

func (ch *CommandHandler) handleCollabStatus(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	cm := ch.hub.GetCollaborationManager()

	if len(parts) >= 2 {
		collabID := ch.resolveCollabID(parts[1])
		if collabID == "" {
			return ch.systemResponse(msg.Channel, "❌ Collaboration not found."), nil
		}
		collab, err := cm.GetCollaboration(collabID)
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
		}
		return ch.systemResponse(msg.Channel, ch.formatCollabDetail(collab)), nil
	}

	// List all active collaborations
	active := cm.ListActive()
	if len(active) == 0 {
		return ch.systemResponse(msg.Channel, "No active collaborations. Use `/collaborate @Agent1 @Agent2 description` to start one."), nil
	}

	var sb strings.Builder
	sb.WriteString("**Active Collaborations:**\n\n")
	for _, c := range active {
		agentNames := make([]string, 0, len(c.Agents))
		for _, a := range c.Agents {
			agentNames = append(agentNames, "@"+a.AgentName)
		}
		sb.WriteString(fmt.Sprintf("- `%s` | **%s** | Phase: %s | Agents: %s\n",
			c.ID[:8], c.Title, c.Phase, strings.Join(agentNames, ", ")))
	}
	return ch.systemResponse(msg.Channel, sb.String()), nil
}

func (ch *CommandHandler) formatCollabDetail(c *collaboration.Collaboration) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Collaboration: %s** (`%s`)\n\n", c.Title, c.ID[:8]))
	sb.WriteString(fmt.Sprintf("**Phase:** %s\n", c.Phase))
	sb.WriteString(fmt.Sprintf("**Description:** %s\n", c.Description))
	sb.WriteString(fmt.Sprintf("**Created by:** %s\n", c.CreatedBy))
	sb.WriteString(fmt.Sprintf("**Created at:** %s\n\n", c.CreatedAt.Format(time.RFC822)))

	sb.WriteString("**Participants:**\n")
	for _, a := range c.Agents {
		sb.WriteString(fmt.Sprintf("- @%s (%s) -- %s\n", a.AgentName, a.AgentType, a.Role))
	}

	if c.Discussion != nil {
		sb.WriteString(fmt.Sprintf("\n**Discussion:** Round %d/%d | Messages: %d/%d | Status: %s\n",
			c.Discussion.CurrentRound, c.Discussion.MaxRounds,
			c.Discussion.TotalMessageCount, c.Discussion.MaxTotalMessages,
			c.Discussion.Status))
	}

	if len(c.Tasks) > 0 {
		sb.WriteString("\n**Tasks:**\n")
		for i, t := range c.Tasks {
			icon := "⬜"
			switch t.Status {
			case collaboration.TaskInProgress:
				icon = "🔄"
			case collaboration.TaskCompleted:
				icon = "✅"
			case collaboration.TaskBlocked:
				icon = "🚫"
			}
			sb.WriteString(fmt.Sprintf("%s **Task %d:** %s (assigned to @%s) - %s\n", icon, i+1, t.Title, t.AssignedName, t.Status))
		}
	}

	if c.Plan != nil && c.Plan.Content != "" {
		sb.WriteString(fmt.Sprintf("\n**Plan (v%d):**\n%s\n", c.Plan.Version, c.Plan.Content))
	}

	return sb.String()
}

// resolveCollabID accepts either a full UUID or a short prefix and
// returns the full collaboration ID if found.

// resolveCollabID accepts either a full UUID or a short prefix and
// returns the full collaboration ID if found.
func (ch *CommandHandler) resolveCollabID(input string) string {
	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ""
	}

	// Try exact match first
	if _, err := cm.GetCollaboration(input); err == nil {
		return input
	}

	// Try prefix match
	for _, c := range cm.ListActive() {
		if strings.HasPrefix(c.ID, input) {
			return c.ID
		}
	}

	return ""
}
