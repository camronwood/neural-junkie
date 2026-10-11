# Assistant Agent Documentation

## Overview

The Assistant Agent is a comprehensive personal assistant that auto-loads with the chat system and provides productivity features including reminders, task management, note-taking, scheduling, and conversation summarization. It's designed to be proactive, helpful, and context-aware.

## Features

### 🔔 Reminder System
- **Time-based reminders**: Set reminders for specific times
- **Recurring reminders**: Daily, weekly, monthly schedules
- **Context-triggered reminders**: Fire when certain keywords are mentioned
- **Proactive notifications**: Automatic reminders sent to appropriate channels

### 📝 Task Management
- **Create tasks**: Add tasks with titles and descriptions
- **Priority levels**: 1-5 priority system (5 being highest)
- **Status tracking**: todo, in_progress, done
- **Due dates**: Optional deadline tracking
- **Per-channel organization**: Tasks organized by channel

### 📋 Note Taking
- **Save notes**: Capture important information from conversations
- **Tagging system**: Organize notes with tags
- **Search functionality**: Find notes by content or tags
- **Message linking**: Link notes to specific messages

### 📅 Meeting & Schedule Management
- **Meeting tracking**: Add meetings with times and descriptions
- **Recurring meetings**: Set up recurring events
- **Upcoming notifications**: Get notified of upcoming meetings
- **Calendar integration**: Basic scheduling capabilities
- **Google Meet notes**: Connect Google to sync Gemini meeting notes from Gmail. See [GOOGLE_MEET_NOTES.md](GOOGLE_MEET_NOTES.md).

### 📄 Conversation Summarization
- **Automatic summarization**: Summarize long discussions on request
- **Action item extraction**: Identify tasks and decisions
- **Key point identification**: Highlight important information
- **Context preservation**: Maintain conversation context

### 🤖 Proactive Assistance
- **Smart suggestions**: Suggest actions based on conversation content
- **Meeting detection**: Offer to create reminders for mentioned meetings
- **Task suggestions**: Propose creating tasks from action items
- **Note suggestions**: Suggest saving important information

### 🧠 Personal learning (Specialist tuning)

When the **Specialist tuning** pack and **personal learning** opt-in are enabled, you can save user-approved notes scoped to the Assistant (or any expert):

- `/learn [draft]` — open the approval dialog
- `/learning-list` — list saved learnings
- Natural phrases like “remember that I prefer …” propose a save (nothing stored until you confirm)

See [PERSONAL_LEARNING_V2.md](PERSONAL_LEARNING_V2.md).

## Ability packs

When these domain packs are enabled, Assistant gains their tools directly (no separate expert agent):

| Pack | Tools |
|------|--------|
| **Maps** | `maps_geocode`, `maps_route`, `maps_create`, `maps_update` + Neural Canvas maps. Device location (`maps_locate`, session share) is a separate sensitive grant. |
| **Music creation** | `generate_music`, stems (hub-native) |
| **Web browser** | Playwright `browser_*` automation (web_search / fetch_url are always available when web search is configured) |

Custom experts can receive the same tools via Composition Model pack grants (`/api/mcp/pack-tools`).

## Commands

### Reminder Commands

#### `/remind <time> <message>`
Set a one-time reminder.

**Examples:**
```
/remind in 30 minutes Review the PR
/remind at 3pm Standup meeting
/remind tomorrow 9am Team sync
/remind Dec 25 10am Christmas planning
```

**Time Formats Supported:**
- Relative: `in 30 minutes`, `in 2 hours`, `in 3 days`
- Today: `at 3pm`, `at 14:30`, `at 9am`
- Tomorrow: `tomorrow at 2pm`, `tomorrow 9am`
- Specific dates: `Dec 25 10am`, `January 15, 2024 3pm`

#### `/remind-recurring <schedule> <message>`
Set a recurring reminder.

**Examples:**
```
/remind-recurring daily 9am Daily standup
/remind-recurring weekly Monday 10am Weekly planning
/remind-recurring monthly 1st 9am Monthly review
```

**Schedule Formats:**
- Daily: `daily`, `every 2 days`
- Weekly: `weekly`, `every 2 weeks`
- Monthly: `monthly`, `every 2 months`

### Task Commands

#### `/task-add <title>`
Create a new task.

**Examples:**
```
/task-add Fix the login bug
/task-add Review the security audit
/task-add Update documentation
```

#### `/task-list`
List all active tasks.

**Output:**
```
📋 Task List:
1. Fix the login bug (Priority: 3, Status: todo)
2. Review the security audit (Priority: 5, Status: in_progress)
3. Update documentation (Priority: 2, Status: todo)
```

#### `/task-done <id>`
Mark a task as complete.

**Examples:**
```
/task-done 1
/task-done 3
```

### Note Commands

#### `/note-save <content>`
Save a note for future reference.

**Examples:**
```
/note-save Important: API key is abc123
/note-save Meeting notes: Discussed new architecture
/note-save Decision: Use React for frontend
```

#### `/note-search <query>`
Search through saved notes.

**Examples:**
```
/note-search API key
/note-search architecture
/note-search decision
```

### Meeting Commands

#### `/meeting-add <time> <title>`
Add a meeting to your schedule.

**Examples:**
```
/meeting-add tomorrow 2pm Team standup
/meeting-add next Monday 10am Sprint planning
/meeting-add Dec 15 3pm Architecture review
```

#### Desktop Listen (local STT)

Capture meeting audio from the desktop mic (or a virtual audio device) and save notes locally. Speech-to-text runs on the hub via **whisper.cpp** only — no cloud STT.

**Toolbar:** use the Listen control (input device dropdown + mic button) in the channel toolbar.

**Commands:**
```
/listen-start [title]
/listen-stop
/listen-status
```

Audio chunks are uploaded only while the toolbar capture is running. Prefer the toolbar for a full start/stop flow; slash commands manage the hub session and status.

### Summarization Commands

#### `/summarize [last N messages]`
Summarize recent conversation.

**Examples:**
```
/summarize
/summarize last 10 messages
/summarize last 20 messages
```

### Help Commands

#### `/help-assistant`
Show comprehensive help for all assistant commands.

## Usage Examples

### Setting Up Daily Reminders
```
User: /remind-recurring daily 9am Daily standup
Assistant: ⏰ Recurring reminder set: 'Daily standup' at daily 9am
```

### Managing Tasks
```
User: /task-add Fix the authentication bug
Assistant: 📝 Task added: 'Fix the authentication bug'

User: /task-list
Assistant: 📋 Task List:
1. Fix the authentication bug (Priority: 3, Status: todo)
```

### Saving Important Information
```
User: /note-save Database password: securepass123
Assistant: 📝 Note saved: 'Database password: securepass123'

User: /note-search password
Assistant: 🔍 Found 1 note:
• Database password: securepass123
```

### Meeting Management
```
User: /meeting-add tomorrow 2pm Sprint planning
Assistant: 📅 Meeting added: 'Sprint planning' at tomorrow 2pm
```

### Conversation Summarization
```
User: /summarize last 15 messages
Assistant: 📄 Summary of last 15 messages:
• Discussed authentication implementation
• Decided to use JWT tokens
• Action items: Research OAuth2, Update documentation
• Next meeting: Tomorrow 2pm for architecture review
```

## Proactive Features

### Smart Suggestions
The assistant monitors conversations and provides helpful suggestions:

**Meeting Detection:**
```
User: "We should have a standup meeting tomorrow"
Assistant: 💡 I noticed you mentioned a meeting. Would you like me to help you set up a reminder or add it to your schedule? Use `/meeting-add` or `/remind` to get started!
```

**Action Item Detection:**
```
User: "We need to fix the bug in the login system"
Assistant: 📝 I noticed some action items in your message. Would you like me to help you track these as tasks? Use `/task-add` to create a task!
```

**Important Information Detection:**
```
User: "This is important: the API key is abc123"
Assistant: 📋 This seems like important information. Would you like me to save this as a note for future reference? Use `/note-save` to save it!
```

## Data Persistence

All assistant data is stored persistently in `~/.neural-junkie/assistant/`:

- `reminders.json` - All active and completed reminders
- `tasks.json` - Task list with status and priorities
- `notes.json` - Saved notes with tags and search
- `meetings.json` - Meeting and schedule data
- `meeting_notes.json` - Ingested meeting notes (Google sync and Desktop Listen)
- `google_oauth.json` - Google OAuth tokens (Gmail + Drive read-only)
- `google_sync_state.json` - Processed Gmail message IDs
- `config.json` - Assistant configuration

Whisper models live under `~/.neural-junkie/models/whisper/` (default `ggml-base.en.bin`). Binary: `~/.neural-junkie/bin/whisper-cli` or `NJ_WHISPER_CMD`.

## Configuration

The assistant can be configured through the `config.json` file:

```json
{
  "timezone": "UTC",
  "default_channel": "general",
  "reminder_advance": 15,
  "keywords": ["meeting", "deadline", "review", "deploy", "release"],
  "google_meet_notes_enabled": true,
  "google_sync_interval_minutes": 15,
  "google_sender_email": "gemini-notes@google.com",
  "google_backfill_days": 90,
  "desktop_listen_enabled": true,
  "whisper_command": "",
  "whisper_model_path": ""
}
```

### Google Meet notes

1. In the desktop app: **Settings → Integrations → Google Meet notes** — enter OAuth Client ID, Secret, and Redirect URI, then **Save OAuth credentials**.
2. **Connect Google** to authorize your account.
3. Optional: set `NEURAL_JUNKIE_GOOGLE_OAUTH_*` env vars on the hub to override saved credentials (see `env.example`).
3. Notes sync automatically every `google_sync_interval_minutes`, or use **Sync now** / `/ingest-meetings`.

### Desktop Listen and Zoom via virtual audio

Desktop Listen captures whatever **audio input device** you select (default mic or a virtual device). Neural Junkie does not auto-wire Zoom.

Typical macOS path for Zoom (or similar):

1. Install a virtual device such as [BlackHole](https://existential.audio/blackhole/).
2. Create an **Aggregate Device** (Audio MIDI Setup) if you need your mic + BlackHole together.
3. Route Zoom speaker output into BlackHole (or use a multi-output device).
4. In Neural Junkie, open the Listen device dropdown and select BlackHole / Aggregate Device, then Start.

Local STT env overrides on the hub:

- `NJ_WHISPER_CMD` — whisper CLI or CI stub (`scripts/whisper-stub.sh`)
- `NJ_WHISPER_MODEL` — path to ggml model

Smoke (hub running with stub): `scripts/smoke-desktop-listen.sh`

Google Meet note sync remains complementary — see [GOOGLE_MEET_NOTES.md](GOOGLE_MEET_NOTES.md).

### Configuration Options

- **timezone**: Timezone for reminder scheduling
- **default_channel**: Default channel for notifications
- **reminder_advance**: Minutes before meetings to send reminders
- **keywords**: Keywords to watch for proactive suggestions
- **google_meet_notes_enabled**: Enable periodic Gmail/Docs sync
- **google_sync_interval_minutes**: Poll interval (default 15)
- **google_sender_email**: Gmail sender filter (default `gemini-notes@google.com`)
- **google_backfill_days**: How far back to search on sync (default 90)
- **desktop_listen_enabled**: Allow Desktop Listen sessions (default true)
- **whisper_command** / **whisper_model_path**: Optional overrides for local whisper.cpp

## Best Practices

### Reminder Management
1. **Use specific times**: "3pm" is better than "afternoon"
2. **Include context**: "Review PR #123" is better than "Review"
3. **Set recurring reminders**: Use for regular meetings and tasks
4. **Use appropriate channels**: Set reminders in relevant channels

### Task Organization
1. **Use descriptive titles**: Clear, actionable task names
2. **Set priorities**: Use 1-5 scale consistently
3. **Include due dates**: For time-sensitive tasks
4. **Mark complete promptly**: Keep task list current

### Note Taking
1. **Tag consistently**: Use consistent tag names
2. **Include context**: Add relevant details
3. **Search regularly**: Use search to find information
4. **Link to messages**: Reference specific conversations

### Meeting Management
1. **Set clear titles**: Descriptive meeting names
2. **Include descriptions**: Add agenda or purpose
3. **Use recurring meetings**: For regular events
4. **Set reminders**: Get notified before meetings

## Troubleshooting

### Common Issues

**Reminders not firing:**
- Check timezone configuration
- Verify reminder is active
- Ensure server is running

**Tasks not persisting:**
- Check storage directory permissions
- Verify JSON file format
- Restart assistant agent

**Notes not searchable:**
- Check search query format
- Verify note tags
- Ensure content is saved

**Meetings not showing:**
- Check time format
- Verify meeting is saved
- Check upcoming meetings query

### Debug Commands

Use these commands to troubleshoot:

```
/help-assistant          # Show all commands
/task-list              # Check task status
/note-search <query>    # Test note search
```

## Integration with Other Agents

The Assistant Agent works seamlessly with other agents:

- **Moderator Agent**: Provides help with assistant commands
- **Repository Agents**: Can create tasks from code review discussions
- **Custom domain experts**: Can save notes from expert knowledge
- **All Agents**: Can benefit from meeting reminders and task tracking

## Future Enhancements

Planned features for future versions:

- **Calendar integration**: Connect with external calendars
- **Email reminders**: Send email notifications
- **Mobile notifications**: Push notifications for mobile
- **Team collaboration**: Shared tasks and reminders
- **Advanced scheduling**: Conflict detection and resolution
- **AI-powered suggestions**: Machine learning for better recommendations

## Support

For issues or questions about the Assistant Agent:

1. Check this documentation first
2. Use `/help-assistant` for command help
3. Ask the Moderator Agent for assistance
4. Check the server logs for errors

The Assistant Agent is designed to be helpful, proactive, and easy to use. It will continue to learn and improve based on your usage patterns and feedback.
