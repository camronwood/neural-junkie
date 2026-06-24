package agent

import "strings"

// messageAsksAboutNJApp reports whether the user is asking about the Neural Junkie
// desktop app, its UI, settings, shortcuts, packs, or platform features.
func messageAsksAboutNJApp(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}

	explicit := []string{
		"neural junkie", "neural-junkie", "nj app", "nj desktop",
		"this app", "this application", "the desktop app",
		"command palette", "keyboard shortcut", "keyboard shortcuts",
		"hotkey", "hotkeys", "key binding", "key bindings",
		"domain pack", "domain packs", "dev pack", "software development pack",
		"life sciences pack", "team chat pack",
		"ide layout", "ide mode", "team layout", "layout preset",
		"workspace switcher", "runbook", "runbooks",
		"settings modal", "settings →", "settings->",
		"help-assistant", "/help-assistant",
		"how does neural junkie", "what is neural junkie",
		"how do i use neural junkie", "how to use neural junkie",
		"toggle terminal", "toggle the terminal", "toggle sidebar", "toggle explorer",
		"open settings",
		"file explorer panel", "git panel", "problems panel",
		"task management panel", "my agents panel",
		"quick open", "go to symbol", "fast edit",
		"tauri app", "desktop client",
	}
	for _, marker := range explicit {
		if strings.Contains(lower, marker) {
			return true
		}
	}

	// "settings" + app-context words (avoid generic project settings questions).
	if strings.Contains(lower, "settings") {
		appContext := []string{
			"keyboard", "layout", "pack", "provider", "ollama", "shortcut",
			"open settings", "in the app", "in neural", "nj ",
		}
		for _, ctx := range appContext {
			if strings.Contains(lower, ctx) {
				return true
			}
		}
	}

	// "how do i" / "how to" + common in-app actions.
	if strings.Contains(lower, "how do i") || strings.Contains(lower, "how to") {
		actions := []string{
			"create channel", "create a channel", "new dm", "new channel",
			"switch workspace", "open settings", "command palette",
			"toggle terminal", "toggle git", "toggle explorer",
			"create agent", "create-repo-agent", "mention an agent",
			"@mention", "switch provider", "switch layout",
			"save file", "close tab", "editor tab",
		}
		for _, action := range actions {
			if strings.Contains(lower, action) {
				return true
			}
		}
	}

	return false
}

// appendNJAppKnowledgeBrief adds a compact always-on guide to the Neural Junkie product.
func appendNJAppKnowledgeBrief(b *strings.Builder) {
	b.WriteString("=== NEURAL JUNKIE APP (brief) ===\n")
	b.WriteString("Neural Junkie (NJ) is a multi-agent collaboration desktop app (Tauri + React) with a Go hub sidecar.\n")
	b.WriteString("Users chat in channels/DMs with specialist agents (@mention by name) and you (Assistant).\n")
	b.WriteString("Domain packs enable role-specific experts: Software development (IDE, Git, repo agents), Life sciences, Team chat & productivity.\n")
	b.WriteString("Desktop UI: channel sidebar, main chat, optional IDE panels (explorer, Monaco editor, terminal, Git, problems, tasks, My Agents).\n")
	b.WriteString("Layout presets: Team (chat-first) vs IDE (project-first, dev pack). Toggle via toolbar or mod+shift+i (dev pack).\n")
	b.WriteString("Settings (mod+,): providers/models, domain packs, layout/trust, integrations, keyboard reference table.\n")
	b.WriteString("Command palette: mod+shift+p. Model library: mod+shift+m. Workspace switcher (dev): mod+shift+w.\n")
	b.WriteString("Common shortcuts: mod+b sidebar, mod+shift+e explorer, mod+j terminal, mod+f find in chat, escape closes top overlay.\n")
	b.WriteString("IDE/dev: mod+p quick open, mod+k fast edit (terminal clear when terminal focused), mod+l focus composer, mod+s save.\n")
	b.WriteString("Runbooks (mod+shift+r) orchestrate multi-step agent collaborations. Threads use mod+shift+] to close panel.\n")
	b.WriteString("Integrations (when configured): Slack, Confluence, Google Calendar/Meet, email sync for meeting notes.\n")
	b.WriteString("You are the primary in-app guide for NJ product questions — use NEURAL JUNKIE APP KNOWLEDGE and SYSTEM COMMANDS below.\n")
	b.WriteString("For deep shortcut/layout questions, the full reference loads when the user asks about the app.\n\n")
}

// appendNJAppKnowledgeFull adds detailed product reference (injected on NJ-related queries).
func appendNJAppKnowledgeFull(b *strings.Builder) {
	b.WriteString("=== NEURAL JUNKIE APP KNOWLEDGE (full reference) ===\n")
	b.WriteString("Use this section to answer questions about the Neural Junkie application itself — UI, settings, shortcuts, packs, and workflows.\n\n")

	b.WriteString("**Architecture**\n")
	b.WriteString("• Desktop app bundles the Go hub (default http://localhost:18765) and starts specialists from enabled domain packs.\n")
	b.WriteString("• Assistant auto-starts with the hub. CLI agents (Cursor, Claude Code, etc.) join when binaries are on PATH.\n")
	b.WriteString("• Repository expert agents index codebases; /create-repo-agent, /reindex-agent, /enable-watch manage them.\n\n")

	b.WriteString("**Domain packs** (Settings → Domain packs)\n")
	b.WriteString("• Software development — six engineering specialists, IDE layout, Git/problems panels, semantic @codebase search, fast edit.\n")
	b.WriteString("• Life sciences — biology/chemistry specialists and scan tooling.\n")
	b.WriteString("• Team chat & productivity — collaboration-focused defaults without full IDE.\n")
	b.WriteString("• Specialist tuning (optional) — LoRA training from repo expert sessions.\n\n")

	b.WriteString("**Layout & panels**\n")
	b.WriteString("• Team layout: chat-first. IDE layout: explorer + editor + chat (dev pack).\n")
	b.WriteString("• Panels: channel sidebar (mod+b), explorer (mod+shift+e), terminal (mod+j), Git (mod+shift+g), problems (mod+shift+d),\n")
	b.WriteString("  pending changes (mod+shift+u), tasks (mod+shift+t), My Agents (mod+shift+a), chat panel (mod+shift+c), toolbar sidebar (mod+shift+\\).\n")
	b.WriteString("• Editor trust (Settings → Layout): interactive (approve edits), auto_apply_edits, yolo (reserved).\n")
	b.WriteString("• IDE routing: with dev pack + IDE layout, sends include active file context; implicit route by file type unless user @mentions someone.\n\n")

	b.WriteString("**Keyboard shortcuts** (mod = Cmd on macOS, Ctrl on Windows/Linux; live list in Settings → Keyboard)\n")
	b.WriteString("Global: mod+, settings | mod+shift+p command palette | mod+shift+m model library | mod+f find in chat | escape close overlay/stop agents | mod+j terminal\n")
	b.WriteString("Layout: mod+b sidebar | mod+shift+e explorer | mod+shift+t tasks | mod+shift+g git | mod+shift+d problems | mod+shift+u pending | mod+shift+a my agents | mod+shift+c chat | mod+shift+\\ toolbar | mod+shift+i IDE vs team | mod+shift+r new runbook\n")
	b.WriteString("Navigation: alt+up/down prev/next channel | mod+0 focus sidebar search | mod+n create channel | mod+shift+n new DM | mod+shift+w workspace switcher\n")
	b.WriteString("IDE: mod+p quick open | mod+shift+o go to symbol | mod+k fast edit (terminal clear when terminal focused) | mod+l focus composer | mod+s save | mod+shift+s save all | mod+w close tab | mod+shift+f code editor | mod+tab / mod+shift+tab cycle tabs\n")
	b.WriteString("Approvals: mod+enter approve first pending | mod+backspace reject first pending | mod+shift+] close thread\n\n")

	b.WriteString("**Chat & agents**\n")
	b.WriteString("• @mention agents by name (e.g. @BackendEngineer). /list-agents shows channel roster.\n")
	b.WriteString("• /remove-agent, /recall-agent, /pause-agent, /switch-provider manage agents. /help lists slash commands.\n")
	b.WriteString("• Collaborations/runbooks: structured multi-agent tasks with recap messages; use for complex workflows.\n")
	b.WriteString("• Ask vs Agent toggle (IDE): Ask is read-only guidance; Agent allows file-change proposals.\n\n")

	b.WriteString("**Getting help**\n")
	b.WriteString("• Ask you (Assistant) about reminders, tasks, notes, meetings, email, and how to use Neural Junkie.\n")
	b.WriteString("• /help-assistant — assistant-specific commands. Settings → Keyboard — shortcut reference.\n")
	b.WriteString("• Docs: GETTING_STARTED.md, KEYBOARD_SHORTCUTS.md, IDE_V3.md, ARCHITECTURE.md in the repo.\n\n")
}
