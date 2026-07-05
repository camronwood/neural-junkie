package hub

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
)

// SetProviderRegistry wires persisted provider config for expert/DM agent creation.
func (ch *CommandHandler) SetProviderRegistry(cfg *config.Config, cache *ai.ProviderCache) {
	if ch == nil {
		return
	}
	ch.appConfig = cfg
	ch.providerCache = cache
}

func (ch *CommandHandler) clearCollaborateRedirect() {
	if ch == nil {
		return
	}
	ch.collabRedirectMu.Lock()
	defer ch.collabRedirectMu.Unlock()
	ch.collabRedirectChannel = ""
	ch.collabRedirectID = ""
}

func (ch *CommandHandler) setCollaborateRedirect(channelName, collabID string) {
	ch.collabRedirectMu.Lock()
	defer ch.collabRedirectMu.Unlock()
	ch.collabRedirectChannel = channelName
	ch.collabRedirectID = collabID
}

// TakeCollaborateRedirect returns redirect metadata from the last successful /collaborate and clears it.

// TakeCollaborateRedirect returns redirect metadata from the last successful /collaborate and clears it.
func (ch *CommandHandler) TakeCollaborateRedirect() (channelName, collabID string, ok bool) {
	if ch == nil {
		return "", "", false
	}
	ch.collabRedirectMu.Lock()
	defer ch.collabRedirectMu.Unlock()
	if ch.collabRedirectChannel == "" {
		return "", "", false
	}
	channelName, collabID = ch.collabRedirectChannel, ch.collabRedirectID
	ch.collabRedirectChannel = ""
	ch.collabRedirectID = ""
	return channelName, collabID, true
}

func (ch *CommandHandler) setDMRedirect(channelName string) {
	if ch == nil {
		return
	}
	ch.dmRedirectMu.Lock()
	defer ch.dmRedirectMu.Unlock()
	ch.dmRedirectChannel = strings.TrimSpace(channelName)
}

// TakeDMRedirect returns the DM channel from the last successful /create-expert and clears it.

// TakeDMRedirect returns the DM channel from the last successful /create-expert and clears it.
func (ch *CommandHandler) TakeDMRedirect() (channelName string, ok bool) {
	if ch == nil {
		return "", false
	}
	ch.dmRedirectMu.Lock()
	defer ch.dmRedirectMu.Unlock()
	if ch.dmRedirectChannel == "" {
		return "", false
	}
	channelName = ch.dmRedirectChannel
	ch.dmRedirectChannel = ""
	return channelName, true
}

// ProcessCommand processes a command from a message.
func (ch *CommandHandler) ProcessCommand(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	content := strings.TrimSpace(msg.Content)

	// Check if it's a command (starts with /)
	if !strings.HasPrefix(content, "/") {
		return nil, nil // Not a command
	}

	parts := strings.Fields(content)
	if len(parts) == 0 {
		return nil, nil
	}

	command := strings.ToLower(parts[0])
	if command == "/collaborate" || command == "/runbook" || command == "/runbook-run" {
		parts = tokenizeSlashCommand(content)
		if len(parts) == 0 {
			return nil, nil
		}
		command = strings.ToLower(parts[0])
	}
	executor, ok := ch.commandExecutors()[command]
	if !ok {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Unknown command: %s\nUse /help to see available commands.", command)), nil
	}
	return executor(ctx, msg, parts)
}

func (ch *CommandHandler) commandExecutors() map[string]commandExecutor {
	return map[string]commandExecutor{
		"/create-repo-agent":        ch.handleCreateRepoAgent,
		"/create-confluence-agent":  ch.handleCreateConfluenceAgent,
		"/create-expert":            ch.handleCreateExpert,
		"/delete-agent":             ch.handleDeleteAgent,
		"/reindex-agent":            ch.handleReindexAgent,
		"/reindex-confluence-agent": ch.handleReindexConfluenceAgent,
		"/pause-agent":              ch.handlePauseAgent,
		"/unpause-agent":            ch.handleUnpauseAgent,
		"/enable-watch":             ch.handleEnableWatch,
		"/disable-watch":            ch.handleDisableWatch,
		"/list-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListAgents(ctx, msg)
		},
		"/tools-list": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleToolsList(ctx, msg)
		},
		"/list-confluence-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListConfluenceAgents(ctx, msg)
		},
		"/remove-agent": ch.handleRemoveAgent,
		"/recall-agent": ch.handleRecallAgent,
		"/list-removed-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListRemovedAgents(ctx, msg)
		},
		"/export-agent-mcp": ch.handleExportAgentMCP,
		"/list-exports": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListExports(ctx, msg)
		},
		"/delete-export":    ch.handleDeleteExport,
		"/import-agent-mcp": ch.handleImportAgentMCP,
		"/export-all-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleExportAllAgents(ctx, msg)
		},
		"/test-anthropic-connection": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleTestAnthropicConnection(ctx, msg)
		},
		"/test-github-connection": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleTestGitHubConnection(ctx, msg)
		},
		"/test-confluence-connection": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleTestConfluenceConnection(ctx, msg)
		},
		"/switch-provider":      ch.handleSwitchProvider,
		"/switch-all-providers": ch.handleSwitchAllProviders,
		"/create-channel":       ch.handleCreateChannelCmd,
		"/add-to-channel":       ch.handleAddToChannel,
		"/remove-from-channel":  ch.handleRemoveFromChannel,
		"/list-channels": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListChannels(ctx, msg)
		},
		"/delete-channel":   ch.handleDeleteChannelCmd,
		"/open-terminal":    ch.handleOpenTerminal,
		"/create-cli-agent": ch.handleCreateCLIAgent,
		"/list-cli-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListCLIAgents(ctx, msg)
		},
		"/help": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleHelp(ctx, msg)
		},
		"/migrate-agent-names": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleMigrateAgentNames(ctx, msg)
		},
		"/open-file":     ch.handleOpenFile,
		"/add-workspace": ch.handleAddWorkspace,
		"/list-workspaces": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListWorkspaces(ctx, msg)
		},
		"/remind":           ch.handleReminder,
		"/remind-recurring": ch.handleReminder,
		"/task-add":         ch.handleTask,
		"/task-list":        ch.handleTask,
		"/task-done":        ch.handleTask,
		"/note-save":        ch.handleNote,
		"/note-search":      ch.handleNote,
		"/learn":            ch.handleLearn,
		"/learning-list":    ch.handleLearningList,
		"/learning-forget":  ch.handleLearningForget,
		"/meeting-add":      ch.handleMeeting,
		"/ingest-meetings": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleIngestMeetings(ctx, msg)
		},
		"/search-meetings": ch.handleSearchMeetings,
		"/meeting-summary": ch.handleMeetingSummary,
		"/action-items": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleActionItems(ctx, msg)
		},
		"/list-meetings": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListMeetings(ctx, msg)
		},
		"/summarize": ch.handleSummarize,
		"/help-assistant": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleAssistantHelp(ctx, msg)
		},
		"/analyze-design": ch.handleAnalyzeDesign,
		"/generate-image": ch.handleGenerateImage,
		"/generate-music": ch.handleGenerateMusic,
		"/approve-file":   ch.handleApproveFile,
		"/reject-file":    ch.handleRejectFile,
		"/approve-delete": ch.handleApproveDelete,
		"/list-file-changes": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListFileChanges(ctx, msg)
		},
		"/collaborate":          ch.handleCollaborate,
		"/runbook":              ch.handleRunbook,
		"/runbook-run":          ch.handleRunbookRun,
		"/approve-plan":         ch.handleApprovePlan,
		"/submit-plan":          ch.handleSubmitPlan,
		"/ack-collab-workspace": ch.handleAckCollabWorkspace,
		"/resume-plan":          ch.handleResumePlan,
		"/revise-plan":          ch.handleRevisePlan,
		"/cancel-plan":          ch.handleCancelPlan,
		"/complete-collab":      ch.handleCompleteCollab,
		"/collab-task-done":     ch.handleCollabTaskDone,
		"/collab-extend":        ch.handleCollabExtend,
		"/collab-rename":        ch.handleCollabRename,
		"/collab-status":        ch.handleCollabStatus,
	}
}

// handleCreateRepoAgent creates a new repository expert agent

// handleHelp shows available commands
func (ch *CommandHandler) handleHelp(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	help := "**Available Commands:**\n\n" +
		"**Repository Agents:**\n" +
		"• `/create-repo-agent <path> [name]` - Create a repository expert agent\n" +
		"• `/reindex-agent <name>` - Reindex a repository agent\n" +
		"• `/enable-watch <name>` - Enable automatic file watching and reindexing\n" +
		"• `/disable-watch <name>` - Disable automatic file watching\n\n" +
		"**Expert Agents:**\n" +
		"• `/create-expert <type> [name] [provider] [model]` - Create a specialist in a **private DM** (invite them to a channel when needed)\n\n" +
		"**Agent Management:**\n" +
		"• `/remove-agent <name>` - Remove agent from conversation (can recall later)\n" +
		"• `/recall-agent <name>` - Recall a removed agent back to conversation\n" +
		"• `/list-removed-agents` - List removed agents available for recall\n" +
		"• `/delete-agent <name>` - Delete an agent permanently\n" +
		"• `/pause-agent <name>` - Pause an agent (stops responding)\n" +
		"• `/unpause-agent <name>` - Resume a paused agent\n" +
		"• `/list-agents` - List all agents in the channel\n" +
		"• `/tools-list` - List MCP tools available to agents in this channel\n\n" +
		"**MCP Exports:**\n" +
		"• `/export-agent-mcp <name>` - Export an agent to MCP format\n" +
		"• `/list-exports` - List all exported agents\n" +
		"• `/delete-export <name>` - Delete an export\n" +
		"• `/import-agent-mcp <path>` - Import an agent from file\n" +
		"• `/export-all-agents` - Export all agents at once\n\n" +
		"**Migration:**\n" +
		"• `/migrate-agent-names` - Check and migrate agent names for @mention compatibility\n\n" +
		"**Help:**\n" +
		"• `/help` - Show this help message\n\n" +
		"**Examples:**\n" +
		"```\n" +
		"/create-repo-agent /path/to/my-project MyProjectExpert\n" +
		"/enable-watch MyProjectExpert\n" +
		"```\n"

	return ch.systemResponse(msg.Channel, help), nil
}

// systemResponse creates a system message response

// systemResponse creates a system message response
func (ch *CommandHandler) systemResponse(channel, content string) *protocol.Message {
	msg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		protocol.AgentInfo{
			ID:   "system",
			Name: "System",
			Type: protocol.AgentTypeGeneral,
		},
		content,
	)
	msg.Mentions = []string{}
	return msg
}

// handleCreateConfluenceAgent creates a new Confluence space expert agent

// handleMigrateAgentNames migrates existing agents with problematic names to @mention-compatible format
func (ch *CommandHandler) handleMigrateAgentNames(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	var response strings.Builder
	response.WriteString("🔄 Migrating agent names for @mention compatibility...\n\n")

	// Track migrations
	migrated := []string{}
	errors := []string{}

	// 1. Migrate repository agents
	response.WriteString("**Repository Agents:**\n")
	repoStorage, err := repo.NewStorage()
	if err == nil {
		cachedRepos, err := repoStorage.GetAllCachedRepos()
		if err == nil {
			for _, repoData := range cachedRepos {
				if name, ok := repoData["name"].(string); ok {
					normalized := protocol.NormalizeAgentName(name)
					if name != normalized {
						response.WriteString(fmt.Sprintf("  • %s → %s\n", name, normalized))
						migrated = append(migrated, fmt.Sprintf("%s (repo)", name))
					}
				}
			}
		} else {
			errors = append(errors, fmt.Sprintf("Failed to load cached repos: %v", err))
		}
	} else {
		errors = append(errors, fmt.Sprintf("Failed to initialize repo storage: %v", err))
	}

	// 2. Migrate Confluence agents (check exports)
	response.WriteString("\n**Confluence Agents:**\n")
	exports, err := ch.exportStorage.ListExports()
	if err == nil {
		for _, export := range exports {
			if export.Type == "confluence" {
				normalized := protocol.NormalizeAgentName(export.Name)
				if export.Name != normalized {
					response.WriteString(fmt.Sprintf("  • %s → %s\n", export.Name, normalized))
					migrated = append(migrated, fmt.Sprintf("%s (confluence)", export.Name))
				}
			}
		}
	} else {
		errors = append(errors, fmt.Sprintf("Failed to load exports: %v", err))
	}

	// Summary
	response.WriteString("\n**Summary:**\n")
	if len(migrated) > 0 {
		response.WriteString(fmt.Sprintf("✅ Found %d agents with names that need migration:\n", len(migrated)))
		for _, agent := range migrated {
			response.WriteString(fmt.Sprintf("  • %s\n", agent))
		}
		response.WriteString("\n⚠️  **Note:** This is a read-only check. Agent names will be automatically normalized when agents are recreated.\n")
		response.WriteString("To apply migrations, restart the agents or recreate them.\n")
	} else {
		response.WriteString("✅ All agent names are already @mention-compatible!\n")
	}

	if len(errors) > 0 {
		response.WriteString(fmt.Sprintf("\n❌ %d errors during migration check:\n", len(errors)))
		for _, err := range errors {
			response.WriteString(fmt.Sprintf("  • %s\n", err))
		}
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// AddPendingReview adds a pending review to track

// Assistant command handlers

// handleReminder handles reminder-related commands
func (ch *CommandHandler) handleReminder(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel,
			"❌ Usage: `/remind <time> <message>` or `/remind-recurring <schedule> <message>`\n"+
				"Examples:\n"+
				"• `/remind in 30 minutes Review the PR`\n"+
				"• `/remind in 30s check the deploy`\n"+
				"• `/remind at 3pm Standup meeting`\n"+
				"• `/remind-recurring daily 9am Daily standup`"), nil
	}

	command := parts[0]
	rest := strings.TrimSpace(strings.TrimPrefix(msg.Content, command))

	storage, err := agent.NewAssistantStorage()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to initialize assistant storage: %v", err)), nil
	}

	if command == "/remind-recurring" {
		schedule, reminderText, err := splitRecurringArgs(rest)
		if err != nil {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/remind-recurring <schedule> <message>`\nExample: `/remind-recurring daily 9am Daily standup`"), nil
		}
		recurring, triggerTime, err := parseRecurringSchedule(schedule)
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid recurring schedule: %v", err)), nil
		}
		reminder := &agent.Reminder{
			ID:          fmt.Sprintf("reminder_%d", time.Now().UnixNano()),
			Content:     reminderText,
			TriggerTime: triggerTime,
			Recurring:   recurring,
			Channel:     msg.Channel,
			CreatedBy:   msg.From.Name,
			Active:      true,
			CreatedAt:   time.Now(),
		}
		if err := storage.SaveReminder(reminder); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save recurring reminder: %v", err)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("⏰ Recurring reminder set: '%s' (%s)", reminderText, schedule)), nil
	}

	timeExpr, reminderText, err := splitOneTimeReminderArgs(rest)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Usage: `/remind <time> <message>`\nExamples: `/remind in 30m Review PR`, `/remind in 30s check logs`, `/remind at 3pm Standup`"), nil
	}

	triggerTime, err := parseReminderTimeExpression(timeExpr)
	if err != nil {
		if assistant := ch.findAssistantAgent(); assistant != nil {
			parsed, parseErr := assistant.ParseTime(timeExpr)
			if parseErr == nil {
				triggerTime = parsed
			} else {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid reminder time: %v", err)), nil
			}
		} else {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid reminder time: %v", err)), nil
		}
	}

	reminder := &agent.Reminder{
		ID:          fmt.Sprintf("reminder_%d", time.Now().UnixNano()),
		Content:     reminderText,
		TriggerTime: triggerTime,
		Channel:     msg.Channel,
		CreatedBy:   msg.From.Name,
		Active:      true,
		CreatedAt:   time.Now(),
	}
	if err := storage.SaveReminder(reminder); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save reminder: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("⏰ Reminder set: '%s' at %s", reminderText, triggerTime.Format(time.RFC1123))), nil
}

// handleTask handles task-related commands

// handleTask handles task-related commands
func (ch *CommandHandler) handleTask(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	command := parts[0]
	storage, err := agent.NewAssistantStorage()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to initialize assistant storage: %v", err)), nil
	}

	switch command {
	case "/task-add":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/task-add <title>`"), nil
		}
		title := strings.Join(parts[1:], " ")
		task := &agent.Task{
			ID:        fmt.Sprintf("task_%d", time.Now().UnixNano()),
			Title:     title,
			Priority:  3,
			Status:    "todo",
			CreatedAt: time.Now(),
			Channel:   msg.Channel,
			CreatedBy: msg.From.Name,
		}
		if err := storage.SaveTask(task); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save task: %v", err)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("📝 Task added: [%s] %s", shortID(task.ID), title)), nil

	case "/task-list":
		tasks, err := storage.LoadTasks()
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to load tasks: %v", err)), nil
		}
		var pending []*agent.Task
		var done []*agent.Task
		for _, task := range tasks {
			if task.Channel != msg.Channel {
				continue
			}
			if task.Status == "done" {
				done = append(done, task)
			} else {
				pending = append(pending, task)
			}
		}
		if len(pending) == 0 && len(done) == 0 {
			return ch.systemResponse(msg.Channel, "📋 Task List:\nNo tasks found in this channel."), nil
		}
		sort.Slice(pending, func(i, j int) bool { return pending[i].CreatedAt.After(pending[j].CreatedAt) })
		sort.Slice(done, func(i, j int) bool { return done[i].CreatedAt.After(done[j].CreatedAt) })
		var b strings.Builder
		b.WriteString("📋 Task List:\n")
		if len(pending) > 0 {
			b.WriteString("Pending:\n")
			for i, task := range pending {
				b.WriteString(fmt.Sprintf("%d. [%s] %s (priority: %d)\n", i+1, shortID(task.ID), task.Title, task.Priority))
			}
		}
		if len(done) > 0 {
			if len(pending) > 0 {
				b.WriteString("\n")
			}
			b.WriteString("Done:\n")
			for i, task := range done {
				b.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, shortID(task.ID), task.Title))
			}
		}
		return ch.systemResponse(msg.Channel, strings.TrimSpace(b.String())), nil

	case "/task-done":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/task-done <task-id>`"), nil
		}
		taskID := parts[1]
		tasks, err := storage.LoadTasks()
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to load tasks: %v", err)), nil
		}
		var matched *agent.Task
		for _, task := range tasks {
			if task.Channel != msg.Channel {
				continue
			}
			if task.ID == taskID || strings.HasPrefix(task.ID, taskID) || shortID(task.ID) == taskID {
				matched = task
				break
			}
		}
		if matched == nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Task '%s' not found in this channel", taskID)), nil
		}
		if matched.Status == "done" {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("ℹ️ Task '%s' is already marked done", shortID(matched.ID))), nil
		}
		matched.Status = "done"
		if err := storage.SaveTask(matched); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to update task: %v", err)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("✅ Task '%s' marked as done", shortID(matched.ID))), nil

	default:
		return ch.systemResponse(msg.Channel, "❌ Unknown task command"), nil
	}
}

// handleNote handles note-related commands

// handleNote handles note-related commands
func (ch *CommandHandler) handleNote(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	command := parts[0]
	storage, err := agent.NewAssistantStorage()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to initialize assistant storage: %v", err)), nil
	}

	switch command {
	case "/note-save":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/note-save <content>`"), nil
		}
		content := strings.Join(parts[1:], " ")
		note := &agent.Note{
			ID:        fmt.Sprintf("note_%d", time.Now().UnixNano()),
			Content:   content,
			Tags:      []string{},
			Channel:   msg.Channel,
			CreatedAt: time.Now(),
			CreatedBy: msg.From.Name,
		}
		if err := storage.SaveNote(note); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save note: %v", err)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("📝 Note saved: [%s] %s", shortID(note.ID), content)), nil

	case "/note-search":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/note-search <query>`"), nil
		}
		query := strings.Join(parts[1:], " ")
		results, err := storage.SearchNotes(query)
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to search notes: %v", err)), nil
		}
		var b strings.Builder
		count := 0
		for _, note := range results {
			if note.Channel != msg.Channel {
				continue
			}
			count++
			b.WriteString(fmt.Sprintf("• [%s] %s\n", shortID(note.ID), note.Content))
		}
		if count == 0 {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("🔍 No notes found for '%s' in this channel", query)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("🔍 Found %d note(s) for '%s':\n%s", count, query, strings.TrimSpace(b.String()))), nil

	default:
		return ch.systemResponse(msg.Channel, "❌ Unknown note command"), nil
	}
}

// handleMeeting handles meeting-related commands

// handleMeeting handles meeting-related commands
func (ch *CommandHandler) handleMeeting(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel,
			"❌ Usage: `/meeting-add <time> <title>`\n"+
				"Example: `/meeting-add tomorrow 2pm Team standup`"), nil
	}

	rest := strings.TrimSpace(strings.TrimPrefix(msg.Content, "/meeting-add"))
	timeExpr, title, err := splitOneTimeReminderArgs(rest)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Usage: `/meeting-add <time> <title>`\nExample: `/meeting-add tomorrow 2pm Team standup`"), nil
	}

	startTime, err := parseReminderTimeExpression(timeExpr)
	if err != nil {
		if assistant := ch.findAssistantAgent(); assistant != nil {
			parsed, parseErr := assistant.ParseTime(timeExpr)
			if parseErr == nil {
				startTime = parsed
			} else {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid meeting time: %v", err)), nil
			}
		} else {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid meeting time: %v", err)), nil
		}
	}

	storage, err := agent.NewAssistantStorage()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to initialize assistant storage: %v", err)), nil
	}
	meeting := &agent.Meeting{
		ID:        fmt.Sprintf("meeting_%d", time.Now().UnixNano()),
		Title:     title,
		StartTime: startTime,
		Channel:   msg.Channel,
		CreatedBy: msg.From.Name,
		CreatedAt: time.Now(),
	}
	if err := storage.SaveMeeting(meeting); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save meeting: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("📅 Meeting added: '%s' at %s", title, startTime.Format(time.RFC1123))), nil
}

// handleSummarize handles conversation summarization

// handleSummarize handles conversation summarization
func (ch *CommandHandler) handleSummarize(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	limit := 20
	if len(parts) > 1 {
		arg := strings.ToLower(parts[1])
		if arg == "last" && len(parts) > 2 {
			arg = parts[2]
		}
		if n, err := strconv.Atoi(arg); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}

	// Get recent messages from the channel
	messages, err := ch.hub.GetMessages(msg.Channel, limit)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get messages: %v", err)), nil
	}

	if len(messages) == 0 {
		return ch.systemResponse(msg.Channel, "❌ No messages to summarize"), nil
	}

	var transcript strings.Builder
	for _, m := range messages {
		if m.Type == protocol.MessageTypeSystemInfo || m.Type == protocol.MessageTypeAgentStatus {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(m.Content), "/") {
			continue
		}
		transcript.WriteString(fmt.Sprintf("[%s][%s] %s\n", m.Timestamp.Format("15:04"), m.From.Name, m.Content))
	}
	if transcript.Len() == 0 {
		return ch.systemResponse(msg.Channel, "❌ No non-command messages to summarize"), nil
	}

	prompt := fmt.Sprintf("Summarize this channel conversation into concise bullets with action items and decisions.\n\n%s", transcript.String())
	aiProvider := ch.aiProvider
	if assistant := ch.findAssistantAgent(); assistant != nil && assistant.AI != nil {
		aiProvider = assistant.AI
	}
	summary, err := aiProvider.GenerateResponse(ctx, prompt, nil)
	if err != nil {
		// Fallback to deterministic summary if AI call fails.
		lines := strings.Split(strings.TrimSpace(transcript.String()), "\n")
		if len(lines) > 8 {
			lines = lines[len(lines)-8:]
		}
		return ch.systemResponse(msg.Channel, fmt.Sprintf("📄 Summary fallback (AI unavailable):\n• %s", strings.Join(lines, "\n• "))), nil
	}
	return ch.systemResponse(msg.Channel, fmt.Sprintf("📄 Summary of last %d messages:\n%s", len(messages), summary)), nil
}

// handleAssistantHelp shows help for assistant commands

// handleAssistantHelp shows help for assistant commands
func (ch *CommandHandler) handleAssistantHelp(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	help := "🤖 **Assistant Commands**\n\n" +
		"**Reminders:**\n" +
		"• `/remind <time> <message>` - Set a reminder\n" +
		"• `/remind-recurring <schedule> <message>` - Set recurring reminder\n\n" +
		"**Tasks:**\n" +
		"• `/task-add <title>` - Add a task\n" +
		"• `/task-list` - List all tasks\n" +
		"• `/task-done <id>` - Mark task complete\n\n" +
		"**Notes:**\n" +
		"• `/note-save <content>` - Save a note\n" +
		"• `/note-search <query>` - Search notes\n\n" +
		"**Personal learning** (Specialist tuning + opt-in):\n" +
		"• `/learn [draft]` - Save a note for the active expert\n" +
		"• `/learning-list [@agent]` - List saved learnings\n" +
		"• `/learning-forget <id-prefix>` - Forget a learning\n\n" +
		"**Meetings:**\n" +
		"• `/meeting-add <time> <title>` - Add meeting\n\n" +
		"**Other:**\n" +
		"• `/summarize [count]` - Summarize recent channel conversation\n\n" +
		"**Examples:**\n" +
		"• `/remind in 30 minutes Review the PR`\n" +
		"• `/remind at 3pm Standup meeting`\n" +
		"• `/task-add Fix the bug in login`\n" +
		"• `/note-save Important: API key is abc123`\n" +
		"• `/meeting-add tomorrow 2pm Team standup`"

	return ch.systemResponse(msg.Channel, help), nil
}

func splitOneTimeReminderArgs(input string) (string, string, error) {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("not enough arguments")
	}
	maxTimeParts := 5
	if len(parts)-1 < maxTimeParts {
		maxTimeParts = len(parts) - 1
	}
	for i := 1; i <= maxTimeParts; i++ {
		candidate := strings.Join(parts[:i], " ")
		if _, err := parseReminderTimeExpression(candidate); err == nil {
			message := strings.TrimSpace(strings.Join(parts[i:], " "))
			if message != "" {
				return candidate, message, nil
			}
		}
	}
	return "", "", fmt.Errorf("unable to parse time expression")
}

func splitRecurringArgs(input string) (string, string, error) {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("not enough arguments")
	}
	scheduleType := strings.ToLower(parts[0])
	switch scheduleType {
	case "daily", "weekly", "monthly":
	default:
		return "", "", fmt.Errorf("unsupported schedule type")
	}

	if len(parts) >= 3 && likelyClockToken(parts[1]) {
		schedule := scheduleType + " " + parts[1]
		message := strings.TrimSpace(strings.Join(parts[2:], " "))
		if message == "" {
			return "", "", fmt.Errorf("missing reminder message")
		}
		return schedule, message, nil
	}

	message := strings.TrimSpace(strings.Join(parts[1:], " "))
	if message == "" {
		return "", "", fmt.Errorf("missing reminder message")
	}
	return scheduleType, message, nil
}

func parseRecurringSchedule(schedule string) (*agent.RecurringSchedule, time.Time, error) {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(schedule)))
	if len(parts) == 0 {
		return nil, time.Time{}, fmt.Errorf("empty schedule")
	}

	recurring := &agent.RecurringSchedule{
		Type:     parts[0],
		Interval: 1,
		Time:     "09:00",
	}

	now := time.Now()
	switch recurring.Type {
	case "daily":
		trigger := now.Add(24 * time.Hour)
		if len(parts) > 1 {
			parsed, err := parseClockTime(parts[1], now)
			if err != nil {
				return nil, time.Time{}, err
			}
			trigger = parsed
			recurring.Time = parsed.Format("15:04")
		}
		return recurring, trigger, nil
	case "weekly":
		trigger := now.Add(7 * 24 * time.Hour)
		if len(parts) > 1 {
			parsed, err := parseClockTime(parts[1], now)
			if err != nil {
				return nil, time.Time{}, err
			}
			trigger = parsed
			recurring.Time = parsed.Format("15:04")
		}
		return recurring, trigger, nil
	case "monthly":
		trigger := now.AddDate(0, 1, 0)
		if len(parts) > 1 {
			parsed, err := parseClockTime(parts[1], now)
			if err != nil {
				return nil, time.Time{}, err
			}
			trigger = parsed
			recurring.Time = parsed.Format("15:04")
		}
		return recurring, trigger, nil
	default:
		return nil, time.Time{}, fmt.Errorf("unsupported recurring type")
	}
}

func parseReminderTimeExpression(input string) (time.Time, error) {
	trimmed := strings.ToLower(strings.TrimSpace(input))
	now := time.Now()

	relativeRe := regexp.MustCompile(`^(?:in\s+)?(\d+)\s*(s|sec|secs|second|seconds|m|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days)$`)
	if m := relativeRe.FindStringSubmatch(trimmed); len(m) == 3 {
		amount, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "s", "sec", "secs", "second", "seconds":
			return now.Add(time.Duration(amount) * time.Second), nil
		case "m", "min", "mins", "minute", "minutes":
			return now.Add(time.Duration(amount) * time.Minute), nil
		case "h", "hr", "hrs", "hour", "hours":
			return now.Add(time.Duration(amount) * time.Hour), nil
		case "d", "day", "days":
			return now.AddDate(0, 0, amount), nil
		}
	}

	if strings.HasPrefix(trimmed, "in ") {
		return parseReminderTimeExpression(strings.TrimSpace(strings.TrimPrefix(trimmed, "in ")))
	}

	if strings.HasPrefix(trimmed, "tomorrow") {
		clock := strings.TrimSpace(strings.TrimPrefix(trimmed, "tomorrow"))
		clock = strings.TrimSpace(strings.TrimPrefix(clock, "at"))
		if clock == "" {
			return now.AddDate(0, 0, 1), nil
		}
		tomorrow := now.AddDate(0, 0, 1)
		return parseClockTime(clock, tomorrow)
	}

	if strings.HasPrefix(trimmed, "at ") {
		return parseClockTime(strings.TrimSpace(strings.TrimPrefix(trimmed, "at ")), now)
	}

	return parseClockTime(trimmed, now)
}

func parseClockTime(timeExpr string, day time.Time) (time.Time, error) {
	layouts := []string{"15:04", "3:04pm", "3:04 pm", "3pm", "3 pm", "3:04PM", "3PM"}
	normalized := strings.ToLower(strings.TrimSpace(timeExpr))
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, normalized); err == nil {
			candidate := time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0, day.Location())
			// If user gave a clock time that's already passed today, schedule for tomorrow.
			if day.Year() == time.Now().Year() && day.YearDay() == time.Now().YearDay() && candidate.Before(time.Now()) {
				candidate = candidate.AddDate(0, 0, 1)
			}
			return candidate, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time expression")
}

func likelyClockToken(v string) bool {
	token := strings.ToLower(strings.TrimSpace(v))
	return strings.Contains(token, ":") || strings.HasSuffix(token, "am") || strings.HasSuffix(token, "pm")
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// handleIngestMeetings manually triggers Google meet notes sync
func (ch *CommandHandler) handleIngestMeetings(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	go func() {
		n, err := assistantAgent.SyncGoogleMeetNotes(ctx)
		if err != nil {
			ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Meet notes sync failed: %v", err))
			return
		}
		ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Synced %d meeting note(s) from Google.", n))
	}()

	return ch.systemResponse(msg.Channel, "🔄 Syncing meeting notes from Google..."), nil
}

// handleSearchMeetings searches meeting notes by query

// handleSearchMeetings searches meeting notes by query
func (ch *CommandHandler) handleSearchMeetings(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: `/search-meetings <query>`"), nil
	}

	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	query := strings.Join(parts[1:], " ")
	notes, err := assistantAgent.SearchMeetingNotes(query)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to search meeting notes: %v", err)), nil
	}

	if len(notes) == 0 {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("🔍 No meeting notes found for query: '%s'", query)), nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("🔍 **Found %d meeting notes for '%s':**\n\n", len(notes), query))

	for i, note := range notes {
		if i >= 5 { // Limit to 5 results
			response.WriteString(fmt.Sprintf("... and %d more results\n", len(notes)-5))
			break
		}

		response.WriteString(fmt.Sprintf("**%s**\n", note.Title))
		response.WriteString(fmt.Sprintf("📅 %s\n", note.MeetingDate.Format("2006-01-02 15:04")))
		if len(note.Attendees) > 0 {
			response.WriteString(fmt.Sprintf("👥 %s\n", strings.Join(note.Attendees, ", ")))
		}
		response.WriteString(fmt.Sprintf("📝 %s\n\n", note.Summary[:minInt(100, len(note.Summary))]))
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleMeetingSummary gets a summary of a specific meeting

// handleMeetingSummary gets a summary of a specific meeting
func (ch *CommandHandler) handleMeetingSummary(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: `/meeting-summary <date>` (e.g., 2025-10-21)"), nil
	}

	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	dateStr := parts[1]
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Invalid date format. Use YYYY-MM-DD"), nil
	}

	// Search for meetings on that date
	notes, err := assistantAgent.GetMeetingNotesByDate(date)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get meeting notes: %v", err)), nil
	}

	if len(notes) == 0 {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("📅 No meetings found for %s", dateStr)), nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📅 **Meetings on %s:**\n\n", dateStr))

	for _, note := range notes {
		response.WriteString(fmt.Sprintf("**%s**\n", note.Title))
		response.WriteString(fmt.Sprintf("🕐 %s\n", note.MeetingDate.Format("15:04")))
		if len(note.Attendees) > 0 {
			response.WriteString(fmt.Sprintf("👥 %s\n", strings.Join(note.Attendees, ", ")))
		}
		response.WriteString(fmt.Sprintf("📝 %s\n", note.Summary))
		if len(note.ActionItems) > 0 {
			response.WriteString("✅ **Action Items:**\n")
			for _, item := range note.ActionItems {
				response.WriteString(fmt.Sprintf("   • %s\n", item))
			}
		}
		if note.GoogleDocLink != "" {
			response.WriteString(fmt.Sprintf("🔗 [View Full Notes](%s)\n", note.GoogleDocLink))
		}
		response.WriteString("\n")
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleActionItems lists all pending action items from meeting notes

// handleActionItems lists all pending action items from meeting notes
func (ch *CommandHandler) handleActionItems(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	actionItems, err := assistantAgent.GetPendingActionItems()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get action items: %v", err)), nil
	}

	if len(actionItems) == 0 {
		return ch.systemResponse(msg.Channel, "✅ No pending action items found in meeting notes."), nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📋 **Pending Action Items (%d):**\n\n", len(actionItems)))

	for i, item := range actionItems {
		if i >= 10 { // Limit to 10 items
			response.WriteString(fmt.Sprintf("... and %d more items\n", len(actionItems)-10))
			break
		}
		response.WriteString(fmt.Sprintf("• %s\n", item))
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleListMeetings lists recent meeting notes

// handleListMeetings lists recent meeting notes
func (ch *CommandHandler) handleListMeetings(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	notes, err := assistantAgent.GetRecentMeetingNotes(10)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get meeting notes: %v", err)), nil
	}

	if len(notes) == 0 {
		return ch.systemResponse(msg.Channel, "📅 No meeting notes found."), nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📅 **Recent Meeting Notes (%d):**\n\n", len(notes)))

	for _, note := range notes {
		response.WriteString(fmt.Sprintf("**%s**\n", note.Title))
		response.WriteString(fmt.Sprintf("📅 %s\n", note.MeetingDate.Format("2006-01-02 15:04")))
		if len(note.Attendees) > 0 {
			response.WriteString(fmt.Sprintf("👥 %s\n", strings.Join(note.Attendees, ", ")))
		}
		response.WriteString(fmt.Sprintf("📝 %s\n\n", note.Summary[:minInt(100, len(note.Summary))]))
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// findAssistantAgent finds the assistant agent in the hub

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleSwitchProvider handles /switch-provider command

// GetCommandDefinitions returns metadata for every registered slash command.
func (ch *CommandHandler) GetCommandDefinitions() []protocol.CommandDefinition {
	return ch.buildCommandDefinitions()
}

func (ch *CommandHandler) buildCommandDefinitions() []protocol.CommandDefinition {
	providerOpts := []string{"ollama", "claude", "lmstudio", "huggingface"}

	return []protocol.CommandDefinition{
		// ── Repository Agents ──────────────────────────────────────────
		{
			Name:        "/create-repo-agent",
			Description: "Create a new repository expert agent",
			Category:    "Repository Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "repo-path", Description: "Path to the repository", Type: "path", Required: true},
				{Name: "agent-name", Description: "Custom name for the agent", Type: "string", Required: false},
				{Name: "provider", Description: "AI provider", Type: "provider", Required: false, Options: providerOpts, Default: "ollama"},
				{Name: "model", Description: "AI model name or composed LoRA tag", Type: "model", Required: false},
				{Name: "adapter-repo", Description: "Hugging Face LoRA repo to compose (with --adapter-repo flag)", Type: "string", Required: false},
			},
		},
		{
			Name:        "/reindex-agent",
			Description: "Re-index a repository agent",
			Category:    "Repository Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the repo agent", Type: "repo-agent-name", Required: true},
			},
		},
		{
			Name:        "/enable-watch",
			Description: "Enable file watching for a repo agent",
			Category:    "Repository Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the repo agent", Type: "repo-agent-name", Required: true},
			},
		},
		{
			Name:        "/disable-watch",
			Description: "Disable file watching for a repo agent",
			Category:    "Repository Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the repo agent", Type: "repo-agent-name", Required: true},
			},
		},

		// ── Confluence ─────────────────────────────────────────────────
		{
			Name:        "/create-confluence-agent",
			Description: "Create a Confluence documentation agent",
			Category:    "Confluence",
			Arguments: []protocol.CommandArgument{
				{Name: "space-key", Description: "Confluence space key", Type: "string", Required: true},
				{Name: "agent-name", Description: "Custom name for the agent", Type: "string", Required: false},
				{Name: "provider", Description: "AI provider", Type: "provider", Required: false, Options: providerOpts, Default: "ollama"},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
		},
		{
			Name:        "/reindex-confluence-agent",
			Description: "Re-index a Confluence agent",
			Category:    "Confluence",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the Confluence agent", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-confluence-agents",
			Description: "List all Confluence agents",
			Category:    "Confluence",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Expert Agents ─────────────────────────────────────────────
		{
			Name:        "/create-expert",
			Description: "Create a specialist agent (backend, frontend, devops, security, architecture, code-review, biology, cad, assistant)",
			Category:    "Expert Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "type", Description: "Expert type: preset (backend, frontend, devops, security, architecture, code-review, biology, cad, assistant) or any custom slug (e.g. guitar, legal-advice)", Type: "string", Required: true},
				{Name: "name", Description: "Custom name for the agent", Type: "string", Required: false},
				{Name: "provider", Description: "AI provider", Type: "provider", Required: false, Options: providerOpts, Default: "ollama"},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
		},

		// ── Agent Management ───────────────────────────────────────────
		{
			Name:        "/delete-agent",
			Description: "Permanently delete an agent and its data",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to delete", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/pause-agent",
			Description: "Pause an agent so it stops responding",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to pause", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/unpause-agent",
			Description: "Unpause a paused agent",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to unpause", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-agents",
			Description: "List all agents (active, available, removed)",
			Category:    "Agent Management",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/tools-list",
			Description: "List hub MCP tools for agents in the current channel",
			Category:    "Agent Management",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/remove-agent",
			Description: "Remove an agent from the current conversation",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to remove", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/recall-agent",
			Description: "Recall a previously removed agent",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to recall", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-removed-agents",
			Description: "List agents that have been removed from channels",
			Category:    "Agent Management",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── MCP Export/Import ──────────────────────────────────────────
		{
			Name:        "/export-agent-mcp",
			Description: "Export an agent to MCP format",
			Category:    "MCP Export/Import",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to export", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-exports",
			Description: "List all MCP exports",
			Category:    "MCP Export/Import",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/delete-export",
			Description: "Delete an MCP export",
			Category:    "MCP Export/Import",
			Arguments: []protocol.CommandArgument{
				{Name: "export-name", Description: "Name of the export to delete", Type: "string", Required: true},
			},
		},
		{
			Name:        "/import-agent-mcp",
			Description: "Import an agent from an MCP file",
			Category:    "MCP Export/Import",
			Arguments: []protocol.CommandArgument{
				{Name: "file-path", Description: "Path to the MCP export file", Type: "path", Required: true},
			},
		},
		{
			Name:        "/export-all-agents",
			Description: "Export all agents to MCP format",
			Category:    "MCP Export/Import",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Connection Tests ───────────────────────────────────────────
		{
			Name:        "/test-anthropic-connection",
			Description: "Test the Anthropic API connection",
			Category:    "Connection Tests",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/test-github-connection",
			Description: "Test the GitHub API connection",
			Category:    "Connection Tests",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/test-confluence-connection",
			Description: "Test the Confluence API connection",
			Category:    "Connection Tests",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Provider ───────────────────────────────────────────────────
		{
			Name:        "/switch-provider",
			Description: "Switch one agent's AI provider",
			Category:    "Provider",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent", Type: "agent-name", Required: true},
				{Name: "provider", Description: "New AI provider", Type: "provider", Required: true, Options: providerOpts},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
		},
		{
			Name:        "/switch-all-providers",
			Description: "Switch all agents to the same AI provider",
			Category:    "Provider",
			Arguments: []protocol.CommandArgument{
				{Name: "provider", Description: "New AI provider", Type: "provider", Required: true, Options: providerOpts},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
		},

		// ── Meetings ───────────────────────────────────────────────────
		{
			Name:        "/ingest-meetings",
			Description: "Sync meeting notes from Google (Gmail + Docs)",
			Category:    "Meetings",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/search-meetings",
			Description: "Search through ingested meeting notes",
			Category:    "Meetings",
			Arguments: []protocol.CommandArgument{
				{Name: "query", Description: "Search query", Type: "string", Required: true},
			},
		},
		{
			Name:        "/meeting-summary",
			Description: "Get a summary of meetings for a specific date",
			Category:    "Meetings",
			Arguments: []protocol.CommandArgument{
				{Name: "date", Description: "Date (e.g. 2025-01-15 or today)", Type: "string", Required: false, Default: "today"},
			},
		},
		{
			Name:        "/action-items",
			Description: "List pending action items from meetings",
			Category:    "Meetings",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/list-meetings",
			Description: "List recent meeting notes",
			Category:    "Meetings",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Files & Workspace ──────────────────────────────────────────
		{
			Name:        "/open-file",
			Description: "Open a file in the code editor",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "file-path", Description: "Path to the file to open", Type: "path", Required: true},
			},
		},
		{
			Name:        "/add-workspace",
			Description: "Add a workspace directory",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "path", Description: "Path to the workspace directory", Type: "path", Required: true},
			},
		},
		{
			Name:        "/list-workspaces",
			Description: "List configured workspaces",
			Category:    "Files & Workspace",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/remind",
			Description: "Set a one-time reminder",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "time", Description: "Reminder time (e.g. in 30m, at 3pm)", Type: "string", Required: true},
				{Name: "message", Description: "Reminder content", Type: "string", Required: true},
			},
		},
		{
			Name:        "/remind-recurring",
			Description: "Set a recurring reminder",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "schedule", Description: "daily|weekly|monthly", Type: "string", Required: true, Options: []string{"daily", "weekly", "monthly"}},
				{Name: "message", Description: "Reminder content", Type: "string", Required: true},
			},
		},
		{
			Name:        "/task-add",
			Description: "Add a task",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "title", Description: "Task title", Type: "string", Required: true},
			},
		},
		{
			Name:        "/task-list",
			Description: "List tasks in this channel",
			Category:    "Assistant",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/task-done",
			Description: "Mark a task as complete",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "task-id", Description: "Task id or short id prefix", Type: "assistant-task-id", Required: true},
			},
		},
		{
			Name:        "/note-save",
			Description: "Save a note",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "content", Description: "Note content", Type: "string", Required: true},
			},
		},
		{
			Name:        "/note-search",
			Description: "Search saved notes",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "query", Description: "Search query", Type: "string", Required: true},
			},
		},
		{
			Name:        "/learn",
			Description: "Open dialog to save a personal learning for the active expert",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "draft", Description: "Optional draft text to prefill", Type: "string", Required: false},
			},
		},
		{
			Name:        "/learning-list",
			Description: "List saved personal learnings (optional @agent filter)",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "agent", Description: "Optional agent name", Type: "string", Required: false},
			},
		},
		{
			Name:        "/learning-forget",
			Description: "Forget a saved learning by id prefix",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "id-prefix", Description: "Learning id prefix", Type: "string", Required: true},
			},
		},
		{
			Name:        "/meeting-add",
			Description: "Add a meeting to schedule",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "time", Description: "Start time (e.g. tomorrow 2pm)", Type: "string", Required: true},
				{Name: "title", Description: "Meeting title", Type: "string", Required: true},
			},
		},
		{
			Name:        "/summarize",
			Description: "Summarize recent channel messages",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "count", Description: "Optional message count, e.g. 10", Type: "string", Required: false},
			},
		},
		{
			Name:        "/approve-file",
			Description: "Approve a pending file change",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "change-id", Description: "ID of the file change to approve", Type: "file-change-id", Required: true},
			},
		},
		{
			Name:        "/reject-file",
			Description: "Reject a pending file change",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "change-id", Description: "ID of the file change to reject", Type: "file-change-id", Required: true},
			},
		},
		{
			Name:        "/approve-delete",
			Description: "Approve a pending file delete operation",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "change-id", Description: "ID of the delete operation to approve", Type: "file-change-id", Required: true},
			},
		},
		{
			Name:        "/list-file-changes",
			Description: "List all pending file changes",
			Category:    "Files & Workspace",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Design ─────────────────────────────────────────────────────
		{
			Name:        "/analyze-design",
			Description: "Analyze an uploaded design image with vision agents",
			Category:    "Design",
			Arguments: []protocol.CommandArgument{
				{Name: "image-url", Description: "URL or path to the design image", Type: "string", Required: true},
			},
		},
		{
			Name:        "/generate-image",
			Description: "Generate an image (local Ollama by default; optional OpenAI via NEURAL_JUNKIE_IMAGE_PROVIDER=openai)",
			Category:    "Design",
			Arguments: []protocol.CommandArgument{
				{Name: "prompt", Description: "What to generate", Type: "string", Required: true},
			},
		},
		{
			Name:        "/generate-music",
			Description: "Generate a song via the Music creation pack (ACE-Step sidecar). Style tags required; optional lyrics after |",
			Category:    "Design",
			Arguments: []protocol.CommandArgument{
				{Name: "style", Description: "Genre, mood, tempo, instruments", Type: "string", Required: true},
				{Name: "lyrics", Description: "Optional lyrics after |", Type: "string", Required: false},
			},
		},

		// ── Channels ───────────────────────────────────────────────────
		{
			Name:        "/create-channel",
			Description: "Create a custom channel with optional description",
			Category:    "Channels",
			Arguments: []protocol.CommandArgument{
				{Name: "name", Description: "Channel name (slug)", Type: "string", Required: true},
				{Name: "description", Description: "Channel description", Type: "string", Required: false},
			},
		},
		{
			Name:        "/add-to-channel",
			Description: "Add an agent to a channel",
			Category:    "Channels",
			Arguments: []protocol.CommandArgument{
				{Name: "channel", Description: "Channel name", Type: "channel-name", Required: true},
				{Name: "agent-name", Description: "Agent to add", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/remove-from-channel",
			Description: "Remove an agent from a channel",
			Category:    "Channels",
			Arguments: []protocol.CommandArgument{
				{Name: "channel", Description: "Channel name", Type: "channel-name", Required: true},
				{Name: "agent-name", Description: "Agent to remove", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-channels",
			Description: "List all channels with member counts",
			Category:    "Channels",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/delete-channel",
			Description: "Delete a custom or DM channel",
			Category:    "Channels",
			Arguments: []protocol.CommandArgument{
				{Name: "name", Description: "Channel name to delete", Type: "channel-name", Required: true},
			},
		},

		// ── Terminal ───────────────────────────────────────────────────
		{
			Name:        "/open-terminal",
			Description: "Open a new terminal tab (optionally in a given directory)",
			Category:    "Terminal",
			Arguments: []protocol.CommandArgument{
				{Name: "cwd", Description: "Working directory for the terminal", Type: "path", Required: false},
			},
		},

		// ── CLI Agents ────────────────────────────────────────────────
		{
			Name:        "/create-cli-agent",
			Description: "Create a CLI proxy agent (see /list-cli-agents for types)",
			Category:    "CLI Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "type", Description: "CLI agent type", Type: "string", Required: true, Options: agent.ListCLIAgentTypes()},
				{Name: "name", Description: "Custom agent name", Type: "string", Required: false},
				{Name: "work-dir", Description: "Working directory for the CLI", Type: "path", Required: false},
			},
		},
		{
			Name:        "/list-cli-agents",
			Description: "List available CLI agent types and their install status",
			Category:    "CLI Agents",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Migration ──────────────────────────────────────────────────
		{
			Name:        "/migrate-agent-names",
			Description: "Check and migrate agent names for @mention compatibility",
			Category:    "Migration",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Help ───────────────────────────────────────────────────────
		{
			Name:        "/help",
			Description: "Show all available commands",
			Category:    "Help",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/help-assistant",
			Description: "Show assistant-specific commands and features",
			Category:    "Help",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Collaboration ─────────────────────────────────────────────
		{
			Name:        "/runbook",
			Description: "Create a draft runbook (user-built task DAG). Optional `--workspace` / `--worktree` before mentions. At least 1 agent; open the Runbook builder in the desktop app to add tasks.",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "agents", Description: "At least one @mentioned agent for the pool", Type: "string", Required: true},
				{Name: "description", Description: "Runbook goal", Type: "string", Required: true},
			},
		},
		{
			Name:        "/runbook-run",
			Description: "Instantiate and start a library runbook definition. Optional `--input key=value` flags.",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "definition-id", Description: "Runbook definition id (e.g. health-check-alert)", Type: "string", Required: true},
				{Name: "agents", Description: "At least one @mentioned agent", Type: "string", Required: true},
			},
		},
		{
			Name:        "/collaborate",
			Description: "Start a multi-agent collaboration. Optional `--rounds` / `--messages` / `--workspace` / `--no-workspace` / `--repo <path>` / `--worktree` / `--allow-agent-adds` before mentions (defaults 3 / 20). Desktop command form can pick active workspace, a folder, or research-only (no repo).",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "description", Description: "[--rounds N] [--messages M] [--workspace] [--worktree] @Agent1 @Agent2 ... description", Type: "string", Required: true},
			},
		},
		{
			Name:        "/approve-plan",
			Description: "Approve a collaboration plan and begin execution",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (first 8 chars is enough)", Type: "collaboration-id", Required: true},
			},
		},
		{
			Name:        "/submit-plan",
			Description: "End planning and move to user review (plan + session summary before approve)",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (first 8 chars is enough)", Type: "collaboration-id", Required: true},
			},
		},
		{
			Name:        "/ack-collab-workspace",
			Description: "Confirm the collaboration sandbox so agents receive task prompts (after /approve-plan)",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (prefix ok)", Type: "collaboration-id", Required: true},
			},
		},
		{
			Name:        "/resume-plan",
			Description: "Resume a collaboration: approve/retry execution when reviewing or approved, or re-send open task prompts when executing",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (first 8 chars is enough)", Type: "collaboration-id", Required: true},
			},
		},
		{
			Name:        "/revise-plan",
			Description: "Send feedback to revise a collaboration plan",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID", Type: "collaboration-id", Required: true},
				{Name: "feedback", Description: "Revision feedback for the agents", Type: "string", Required: true},
			},
		},
		{
			Name:        "/cancel-plan",
			Description: "Cancel an active collaboration",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID", Type: "collaboration-id", Required: true},
			},
		},
		{
			Name:        "/complete-collab",
			Description: "Mark a collaboration complete (use --force to close open tasks)",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (prefix ok)", Type: "collaboration-id", Required: true},
				{Name: "force", Description: "Pass --force to mark open tasks done and close", Type: "string", Required: false, Options: []string{"--force"}},
			},
		},
		{
			Name:        "/collab-task-done",
			Description: "Mark one collaboration task complete (1-based task number or task id prefix)",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (prefix ok)", Type: "collaboration-id", Required: true},
				{Name: "task", Description: "Task number (1-based) or task id prefix", Type: "collaboration-task", Required: true},
			},
		},
		{
			Name:        "/collab-extend",
			Description: "Raise planning/review discussion limits after budget_exhausted (or bump caps while active)",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (prefix ok)", Type: "collaboration-id", Required: true},
				{Name: "rounds", Description: "Add this many to max rounds (optional)", Type: "string", Required: false},
				{Name: "messages", Description: "Add this many to max agent messages (optional)", Type: "string", Required: false},
			},
		},
		{
			Name:        "/collab-rename",
			Description: "Set a short display title for a collaboration",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (prefix ok)", Type: "collaboration-id", Required: true},
				{Name: "title", Description: "New title", Type: "string", Required: true},
			},
		},
		{
			Name:        "/collab-status",
			Description: "Show status of active collaborations",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (optional, shows all if omitted)", Type: "collaboration-id", Required: false},
			},
		},
	}
}

func (ch *CommandHandler) validateCommandDefinitions() {
	executors := ch.commandExecutors()
	defs := ch.buildCommandDefinitions()

	defSet := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		defSet[strings.ToLower(def.Name)] = struct{}{}
	}

	var missingInDefs []string
	for name := range executors {
		if _, ok := defSet[name]; !ok {
			missingInDefs = append(missingInDefs, name)
		}
	}

	execSet := make(map[string]struct{}, len(executors))
	for name := range executors {
		execSet[name] = struct{}{}
	}

	var missingInExecutors []string
	for _, def := range defs {
		name := strings.ToLower(def.Name)
		if _, ok := execSet[name]; !ok {
			missingInExecutors = append(missingInExecutors, name)
		}
	}

	sort.Strings(missingInDefs)
	sort.Strings(missingInExecutors)

	if len(missingInDefs) > 0 {
		log.Printf("⚠️  Command parity mismatch: command handlers missing from definitions: %s", strings.Join(missingInDefs, ", "))
	}
	if len(missingInExecutors) > 0 {
		log.Printf("⚠️  Command parity mismatch: command definitions missing handlers: %s", strings.Join(missingInExecutors, ", "))
	}
}

// ── Collaboration Command Handlers ──────────────────────────────────

// handleCollaborate starts a multi-agent collaboration.
// Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 @Agent3 build a CLI tool that encrypts files
