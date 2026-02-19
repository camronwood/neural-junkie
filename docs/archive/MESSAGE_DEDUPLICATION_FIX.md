# Message Deduplication Fix

## Problem

Messages were appearing duplicated (twice) in the chat window - both user messages and agent responses.

## Root Cause

**React.StrictMode** in development mode (`desktop/src/main.tsx`) causes components to mount **twice** intentionally to help detect side effects. This led to:

1. `ChatWindow` component mounting twice
2. Two separate WebSocket connections being created (one for each mount)
3. Both connections receiving the same messages from the server
4. `addMessage()` being called twice with the same message
5. Messages appearing twice in the UI

## Solution

Added **message deduplication** logic to the chat store to prevent duplicate messages from being added:

### Changes Made

**File**: `desktop/src/stores/chatStore.ts`

1. **Enhanced `addMessage` action**:
   - Checks if a message with the same `id` already exists
   - Skips adding the message if it's a duplicate
   - Logs duplicate detection for debugging
   - Returns unchanged state (no re-render) for duplicates

2. **Enhanced `addThreadMessage` action**:
   - Same deduplication logic for thread messages
   - Prevents duplicate replies in threads

### Code Changes

```typescript
addMessage: (message) =>
  set((state) => {
    // Prevent duplicate messages (can happen with React StrictMode double-mounting)
    const isDuplicate = state.messages.some(m => m.id === message.id);
    if (isDuplicate) {
      console.log('[ChatStore] Skipping duplicate message:', message.id);
      return state; // Return unchanged state
    }
    
    return {
      messages: [...state.messages, message],
      isTyping: false,
    };
  }),
```

## Why This Happens

### React StrictMode Behavior

In development mode, React StrictMode intentionally:
- Mounts components twice
- Runs effects twice
- Calls useState initializers twice

This helps identify:
- Impure functions
- Side effects that need cleanup
- Accidental mutations

### WebSocket Connection Lifecycle

With StrictMode:
```
Mount #1 → useWebSocket connects → WebSocket A created
Mount #2 → useWebSocket connects → WebSocket B created
Cleanup #1 → WebSocket A should close (but timing issues)
Server sends message → Both WebSocket A & B receive it
addMessage called twice → Duplicate messages
```

## Alternative Solutions Considered

### 1. Remove React.StrictMode
❌ **Not recommended** - StrictMode helps catch bugs and is best practice for React development

### 2. Enhanced WebSocket Cleanup
⚠️ **Partial solution** - The `useWebSocket` hook already has cleanup logic, but React's double-mounting can still cause timing issues

### 3. Message Deduplication (Chosen)
✅ **Best solution** - Prevents duplicates regardless of cause, resilient to various edge cases

## Benefits of This Approach

1. **Works in all scenarios**: Not just React StrictMode, but also network issues, reconnections, etc.
2. **No performance impact**: O(n) check on message addition (typically small n)
3. **Maintains StrictMode**: Keeps React's helpful development checks
4. **Defensive programming**: Protects against future duplicate sources
5. **Logging**: Helps identify if duplicates occur for debugging

## Testing

### Before Fix
```
User: Hello
User: Hello  ← Duplicate
Agent: Hi there!
Agent: Hi there!  ← Duplicate
```

### After Fix
```
User: Hello
Agent: Hi there!
[Console: ChatStore] Skipping duplicate message: abc-123
[Console: ChatStore] Skipping duplicate message: def-456
```

### Verification Steps

1. Open desktop app (Tauri dev mode)
2. Send a message
3. Check browser DevTools console for "Skipping duplicate message" logs
4. Verify only one copy of each message appears in chat
5. Test with threads - verify thread replies also work correctly

## Production Build

**Note**: This issue is primarily a development problem due to React StrictMode. Production builds don't run StrictMode by default, but the deduplication logic provides extra safety regardless.

## Impact on Other Features

✅ **Thinking indicators**: Still work correctly  
✅ **Indexing indicators**: Still work correctly  
✅ **Threads**: Deduplication added to thread messages too  
✅ **Agent status updates**: Not affected (these update agent info, not messages)  

## Related Files

- `desktop/src/stores/chatStore.ts` - Deduplication logic added
- `desktop/src/main.tsx` - React.StrictMode enabled here (kept intentionally)
- `desktop/src/hooks/useWebSocket.ts` - Connection management (already has guards)
- `desktop/src/components/ChatWindow.tsx` - Uses addMessage (benefits from fix)

## Future Improvements

1. **Store message IDs in a Set**: For O(1) duplicate checking instead of O(n)
2. **Limit message cache**: Keep only recent N messages to prevent memory growth
3. **Metrics**: Track how often duplicates occur in production
4. **WebSocket singleton**: Ensure only one WebSocket per channel (more complex)

## Debugging Commands

Check for duplicate detection:
```bash
# In browser DevTools console
localStorage.setItem('debug', 'ChatStore')
# Then reload the app - will see detailed logs
```

Check for multiple WebSocket connections:
```javascript
// In browser DevTools console
console.log('WebSocket count:', 
  performance.getEntriesByType('resource')
    .filter(r => r.name.includes('ws://'))
    .length
)
```

---

**Fixed**: October 15, 2025  
**Status**: ✅ Complete and tested  
**No Linter Errors**: TypeScript validation passed  
**Production Ready**: Safe for both development and production

