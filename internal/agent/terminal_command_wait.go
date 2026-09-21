package agent

import (
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// awaitingUserTerminalRunCue is appended when the host surfaces bash-block Run buttons.
const awaitingUserTerminalRunCue = "⏳ Waiting for you to run the command(s) above in the terminal. I will continue after the output is posted."

// TerminalBashSuggestionPrompt explains shell capability and when to use run_command vs host Run.
func TerminalBashSuggestionPrompt() string {
	return "SHELL CAPABILITY: You CAN run non-destructive shell commands yourself via the run_command tool " +
		"(user may need to approve once). Prefer run_command for: ls, cat, git status/diff/log, go test, npm test/lint/build, cargo check/test, etc.\n" +
		"When workspace context is shared (file tree / Project: / workspace shared), INSPECT the workspace with tools " +
		"before asking the user to paste files or claiming you cannot see the code.\n" +
		"NEVER say you cannot execute commands, open a terminal, or run tools on the user's machine when run_command is available. " +
		"Do not give manual copy-paste instructions for commands you could run with run_command.\n" +
		"Use ```bash fenced blocks``` ONLY when the host terminal panel must own the process: long-running/dev servers " +
		"(tauri dev, npm run dev, cargo run --watch), interactive prompts, GUI launches, or commands the user must supervise. " +
		"Those fences surface a **Run** button — they do not auto-execute. After emitting them: briefly say what to run, then STOP. " +
		"Do not invent stdout/stderr. Do not mark TASK_STATUS: completed while waiting.\n" +
		"Do NOT put illustrative or example-only shell in ```bash``` fences (use plain prose or a non-bash fence) — " +
		"example fences falsely show Run / wait cues.\n" +
		"When a command_output message arrives for your suggestion, treat it as the real result and continue from exit code + streams.\n"
}

func appendTerminalBashSuggestionPrompt(system *strings.Builder) {
	if system == nil {
		return
	}
	system.WriteString(TerminalBashSuggestionPrompt())
}

func appendCommandOutputContinuationPrompt(system *strings.Builder, msg *protocol.Message) {
	if system == nil {
		return
	}
	system.WriteString("\n=== TERMINAL COMMAND RESULT (authoritative) ===\n")
	system.WriteString("The host posted the result of a command you suggested (or started). The user message IS that result.\n")
	system.WriteString("Read exit code, stdout, and stderr carefully. Continue the original task using this evidence.\n")
	system.WriteString("If the result says the command was started in the terminal (long-running / no exit yet), acknowledge startup and what to watch for — do not claim you cannot run commands.\n")
	system.WriteString("If the result is a failure or 'not allowed', explain the error and propose a different allowlisted command via run_command or a corrected ```bash block.\n")
	if msg != nil && strings.TrimSpace(msg.Content) != "" {
		system.WriteString("Do not ask the user to paste the output — it is already in this turn.\n")
	}
}

func ensureAwaitingUserTerminalRunCue(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.Contains(content, awaitingUserTerminalRunCue) {
		return content
	}
	if trimmed == "" {
		return awaitingUserTerminalRunCue
	}
	return strings.TrimRight(content, "\n") + "\n\n" + awaitingUserTerminalRunCue
}

func suggestedByAgentName(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	switch v := msg.Metadata["suggested_by_agent"].(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

// shouldRespondToSuggestedCommandOutput wakes only the agent that suggested the command.
func shouldRespondToSuggestedCommandOutput(a *Agent, msg *protocol.Message) bool {
	if a == nil || msg == nil || msg.Type != protocol.MessageTypeCommandOutput {
		return false
	}
	if name := suggestedByAgentName(msg); name != "" {
		match := strings.EqualFold(name, a.Info.Name)
		if match {
			log.Printf("[%s] ✅ TERMINAL command_output for our suggestion — will respond", a.Info.Name)
		}
		return match
	}
	if msg.ReplyTo == "" {
		return false
	}
	for _, hist := range a.channelHistory(msg.Channel) {
		if hist == nil || hist.ID != msg.ReplyTo {
			continue
		}
		if hist.From.ID != a.Info.ID {
			return false
		}
		if hist.Metadata != nil {
			if _, ok := hist.Metadata["suggested_commands"]; ok {
				log.Printf("[%s] ✅ TERMINAL command_output reply_to our suggestion — will respond", a.Info.Name)
				return true
			}
		}
		log.Printf("[%s] ✅ TERMINAL command_output reply_to our message — will respond", a.Info.Name)
		return true
	}
	return false
}

// applySuggestedCommandsToResponse attaches Run suggestions and a wait cue; softens premature task completion.
func applySuggestedCommandsToResponse(
	a *Agent,
	msg *protocol.Message,
	responseMsg *protocol.Message,
	response string,
) string {
	if a == nil || responseMsg == nil {
		return response
	}
	commandDetector := protocol.NewCommandDetector(nil)
	suggestions := commandDetector.DetectCommands(response, a.Info.Name, responseMsg.ID)
	suggestions = filterCollabCommandSuggestions(msg, suggestions)
	if cwd := collaborationWorkingDirectoryForMessage(a, msg); cwd != "" && len(suggestions) > 0 {
		for i := range suggestions {
			suggestions[i].Cwd = cwd
		}
	}
	if len(suggestions) == 0 {
		return response
	}
	if responseMsg.Metadata == nil {
		responseMsg.Metadata = make(map[string]interface{})
	}
	responseMsg.Metadata["suggested_commands"] = suggestions
	response = ensureAwaitingUserTerminalRunCue(response)
	responseMsg.Content = response
	softenTaskStatusWhileAwaitingTerminal(responseMsg, response)
	return response
}

func softenTaskStatusWhileAwaitingTerminal(responseMsg *protocol.Message, response string) {
	if responseMsg == nil {
		return
	}
	status := strings.ToLower(strings.TrimSpace(responseMsg.GetTaskStatus()))
	if status == "completed" || strings.Contains(strings.ToLower(response), "task_status: completed") {
		responseMsg.SetTaskStatus("in_progress")
	}
}
