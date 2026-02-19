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
- `config.json` - Assistant configuration

## Configuration

The assistant can be configured through the `config.json` file:

```json
{
  "timezone": "UTC",
  "default_channel": "general",
  "reminder_advance": 15,
  "keywords": ["meeting", "deadline", "review", "deploy", "release"]
}
```

### Configuration Options

- **timezone**: Timezone for reminder scheduling
- **default_channel**: Default channel for notifications
- **reminder_advance**: Minutes before meetings to send reminders
- **keywords**: Keywords to watch for proactive suggestions

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
- **Helper Agents**: Can save notes from expert knowledge
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
