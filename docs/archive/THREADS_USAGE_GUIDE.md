# Threads Feature - Quick Start Guide

## How to Use Threads

### Starting a Thread

1. **Hover over any message** in the channel
2. **Click "Reply in thread"** button (appears in top-right on hover)
3. Thread panel opens on the right side
4. Type your reply in the thread input box
5. Hit Enter or click Send

### Viewing Threads

**When a message has replies:**
- You'll see a thread indicator at the bottom of the message
- Shows: "X replies" and "Last reply 5m ago"
- Click "View thread →" to open the thread panel

**Thread Panel:**
- Appears on the right side (400px width)
- Shows parent message at top
- Shows all thread replies below
- Has its own input box for replies
- Close button (X) in top-right

### Thread Behavior

**Messages in Threads:**
- Thread replies ONLY appear in the thread panel
- Thread replies do NOT appear in the main channel
- Keeps conversations organized and focused
- Just like Slack!

**Agent Participation:**
- Agents can see thread messages
- Agents ONLY respond when @mentioned in threads
- Example: `@backend how should I structure this?`
- Prevents agents from flooding threads with responses

**Real-time Updates:**
- Thread messages appear instantly
- Reply counts update automatically
- Last reply time updates automatically
- Multiple people can participate in threads simultaneously

### Layout

**Without Thread Open:**
```
┌────────────────────────────┬───────────┐
│                            │           │
│     Main Channel           │  Agents   │
│                            │           │
└────────────────────────────┴───────────┘
```

**With Thread Open:**
```
┌─────────────────┬────────────┬───────────┐
│                 │            │           │
│  Main Channel   │  Thread    │  Agents   │
│                 │            │           │
└─────────────────┴────────────┴───────────┘
```

## Testing the Feature

### 1. Start Everything

```bash
# Terminal 1 - Start server
make server

# Terminal 2 - Start agents
make agents

# Terminal 3 - Start Tauri desktop app
make desktop
```

### 2. Create Your First Thread

1. In the desktop app, send a message:
   ```
   What's the best way to structure a React app?
   ```

2. Wait for agent responses

3. Hover over one of the responses

4. Click "Reply in thread" button

5. Send a follow-up in the thread:
   ```
   Can you explain that in more detail?
   ```

6. Notice the reply only appears in the thread panel!

### 3. Test Agent Mentions in Threads

1. In the thread, type:
   ```
   @frontend what do you think about this approach?
   ```

2. Frontend agent should respond IN THE THREAD

3. The response appears ONLY in the thread, not the main channel

### 4. Test Multiple Threads

1. Send another message in the main channel:
   ```
   How do I set up Docker for this project?
   ```

2. Start a new thread on this message

3. You can now have two active threads!

4. Switch between them by clicking "View thread" on each parent message

### 5. Verify Thread Indicators

1. Look at the parent message in the main channel

2. Should show:
   - "3 replies" (or however many you sent)
   - "Last reply 2m ago" (or similar)
   - "View thread →" button

3. Click "View thread" to reopen a closed thread

## Tips & Tricks

### When to Use Threads

✅ **Good for:**
- Follow-up questions on an agent response
- Side discussions that would clutter the main channel
- Detailed technical discussions
- Reviewing specific code or ideas
- Multi-step problem solving

❌ **Not needed for:**
- Simple one-off questions
- Questions that benefit from all agents seeing them
- Announcements or broadcasts

### Keyboard Shortcuts

Currently no keyboard shortcuts implemented, but could add:
- `T` - Reply in thread
- `Esc` - Close thread panel
- `Enter` - Send message (already works)

### Agent @Mention Syntax in Threads

Works the same as in main channel:
- `@backend` - Mention backend agent
- `@frontend` - Mention frontend agent
- `@GoExpert` - Mention by name (if you have a repo agent named GoExpert)

### Thread Etiquette

1. **Keep threads focused** - Stay on topic of parent message
2. **Use @mentions** - To get agent input when needed
3. **Close threads** - When discussion is complete
4. **Start fresh threads** - For new topics (don't hijack existing threads)

## Common Questions

**Q: Can I see all threads?**
A: Currently no thread list view. Open threads by clicking "View thread" on messages with replies.

**Q: What if I want agents to participate without @mentions?**
A: That's intentional! It prevents thread flooding. Just @mention the agents you want.

**Q: Can I reply to a message in a thread?**
A: Currently no nested replies. All thread messages are at the same level.

**Q: How do I know if I have new replies in a thread?**
A: Watch the "Last reply" timestamp on the parent message. Will update to recent times.

**Q: Can I delete thread messages?**
A: Not currently. This is an in-memory system for now.

**Q: Do threads persist after restart?**
A: No - messages are stored in-memory. Database persistence planned for future.

## Architecture Notes (For Developers)

### Message Structure

**Channel Message:**
```json
{
  "id": "msg-123",
  "content": "How do I use threads?",
  "is_thread_reply": false,
  "thread_id": ""
}
```

**Thread Reply:**
```json
{
  "id": "msg-456",
  "content": "Just click Reply in thread!",
  "is_thread_reply": true,
  "thread_id": "msg-123"
}
```

### API Endpoints

```
GET  /api/threads/{threadID}/messages?limit=50
POST /api/threads/{threadID}/reply
GET  /api/threads/{threadID}/metadata
WS   /ws?channel=general&thread={threadID}
```

### State Management

**Zustand Store:**
- `openThreadId` - Currently viewing thread
- `threadMessages` - Map of thread ID to messages
- `threadMetadata` - Map of thread ID to stats

**Actions:**
- `openThread(id)` - Open thread panel
- `closeThread()` - Close thread panel
- `addThreadMessage(msg)` - Add new thread message
- `setThreadMessages(id, msgs)` - Set all thread messages
- `updateThreadMetadata(id, meta)` - Update thread stats

## Troubleshooting

**Thread panel won't open:**
- Check browser console for errors
- Verify parent message exists in `messages` array
- Check that `openThreadId` is set in store

**Messages not appearing in thread:**
- Check that `is_thread_reply` is true
- Verify `thread_id` matches parent message ID
- Check WebSocket connection status

**Agent not responding in thread:**
- Make sure you @mentioned the agent
- Check that agent is active (not paused)
- Verify agent received the message (check server logs)

**Thread indicator not showing:**
- Verify thread metadata exists for message ID
- Check that `reply_count > 0`
- Look for JavaScript errors in console

## Future Enhancements

Potential features for future versions:
- Thread notifications
- Unread thread counts
- Thread search
- Thread permalinks
- Keyboard shortcuts
- Thread collapse/expand
- Participant avatars
- Thread archiving

## Feedback

If you encounter issues or have suggestions:
1. Check server logs: `tail -f /tmp/chat-server.log`
2. Check browser console
3. Create an issue in the repository

Enjoy threaded conversations! 💬

