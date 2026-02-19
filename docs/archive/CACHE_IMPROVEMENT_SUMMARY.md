# Repository Agent Cache Improvement Summary

## Problem Solved

Repository agents were showing full indexing progress (0-100%) every time, even when a cache should have been available. The root causes were:

1. **Cache was keyed by agent name** instead of repository path
   - Creating multiple agents for the same repo would create separate caches
   - Deleting and recreating an agent with a different name wouldn't find the cache
   
2. **No user feedback about cache status**
   - Users couldn't tell if cache was being used or not
   - Silent cache loading made it seem like caching wasn't working

## Changes Implemented

### 1. Storage Layer (`internal/repo/storage.go`)

**New cache key generation:**
```go
// Old: cache key = sanitized agent name
// New: cache key = SHA256(absolute_repo_path)
```

**New methods added:**
- `GetCacheKeyForPath()` - Generates SHA256 hash of repository path
- `SaveMetadata()` - Stores repository metadata alongside index
- `LoadMetadata()` - Loads repository metadata

**Updated methods:**
- `SaveIndex(cacheKey, index)` - Now uses cache key instead of agent name
- `LoadIndex(cacheKey)` - Now uses cache key instead of agent name
- `DeleteIndex(cacheKey)` - Now uses cache key instead of agent name
- `IndexExists(cacheKey)` - Now uses cache key instead of agent name

**New metadata structure:**
```go
type RepoMetadata struct {
    Path       string   // Absolute path to repository
    CacheKey   string   // SHA256 hash of path
    AgentNames []string // Agents using this cache
}
```

### 2. Repo Agent (`internal/agent/repo_agent.go`)

**Updated indexRepository():**
- Generates cache key from repository path (not agent name)
- Adds user-visible status messages for cache operations
- Saves metadata alongside index

**New user feedback messages:**
- ✅ "Loaded from cache (instant) - repository already indexed!"
- 🔄 "Cache stale (reason), performing incremental update..."
- 📊 "No cache found, performing full analysis (this may take 30-60 seconds)..."
- ⚠️  Error messages when cache operations fail

**New helper method:**
- `sendStatusMessage()` - Sends MessageTypeSystemInfo to channel for user visibility

**Updated Cleanup():**
- Now uses cache key to delete stored data

### 3. Command Handler (`internal/hub/commands.go`)

**Updated /create-repo-agent:**
- Checks if cache exists for the repository path
- Shows different messages based on cache availability:
  - 💾 "Cache found! Loading will be instant if cache is fresh."
  - 📊 "No cache found - first indexing may take 30-60 seconds."
  - "Future agents for this repository will load instantly from cache!"

## Benefits

### 1. Shared Caches Across Agents
Multiple agents pointing to the same repository now share the same cache:

```bash
# First agent creates cache
/create-repo-agent /path/to/repo Agent1  # Takes 30-60 seconds

# Second agent uses existing cache
/create-repo-agent /path/to/repo Agent2  # Loads instantly!
```

### 2. Persistent Cache After Agent Deletion
Cache survives agent deletion and recreation:

```bash
/create-repo-agent /path/to/repo MyAgent    # Creates cache
/delete-agent MyAgent                        # Deletes agent
/create-repo-agent /path/to/repo NewAgent   # Uses cached index!
```

### 3. User Visibility
Users now see clear feedback about cache operations in the chat:

- Cache found and fresh → instant load message
- Cache stale → incremental update message with reason
- No cache → full indexing message with time estimate

### 4. Cache Key Consistency
Same repository always produces the same cache key:

```bash
# These all use the same cache:
/create-repo-agent /path/to/repo Expert1
/create-repo-agent /path/to/repo Expert2
/create-repo-agent /path/to/repo MyCustomName
```

## Cache Location

Caches are stored at: `~/.neural-junkie/repos/<cache-key>/`

Where `<cache-key>` is the SHA256 hash of the absolute repository path.

### Example Cache Keys

```
Repository: /Users/camron.wood.ext/development/sandbox/neural-junkie
Cache Key:  20c9c3f75f65c72c117bf7e8c139f6abfa8f7ff056ebe9baa519eed962c53fc9
Location:   ~/.neural-junkie/repos/20c9c3f75f65c72c117bf7e8c139f6abfa8f7ff056ebe9baa519eed962c53fc9/
```

### Cache Contents

Each cache directory contains:
- `index.json` - Repository index with file structure, dependencies, git info
- `metadata.json` - Repository metadata with path, cache key, agent names

## Testing

### Unit Tests

Run cache tests:
```bash
go test -v ./test/cache_test.go
```

Tests verify:
- Cache key generation is consistent
- Metadata save/load works correctly
- Different paths produce different keys

### Check Cache Status

Use the provided utility script:
```bash
go run scripts/check-cache.go /path/to/your/repo
```

Output shows:
- Cache key for the repository
- Whether cache exists
- Cache details if available (file count, size, last indexed)

### Manual Testing

1. Start the system:
   ```bash
   make server
   make desktop
   ```

2. Create first agent (no cache):
   ```bash
   /create-repo-agent /path/to/repo Agent1
   ```
   - Should show "📊 No cache found"
   - Shows indexing progress 0-100%
   - Takes 30-60 seconds
   - Shows "✅ Indexing complete! Repository cached for future use."

3. Create second agent (with cache):
   ```bash
   /create-repo-agent /path/to/repo Agent2
   ```
   - Should show "💾 Cache found!"
   - Shows "✅ Loaded from cache (instant)"
   - Takes <2 seconds

4. Test cache after deletion:
   ```bash
   /delete-agent Agent1
   /delete-agent Agent2
   /create-repo-agent /path/to/repo Agent3
   ```
   - Should still use cache and load instantly

## Migration Notes

### Existing Caches

Old caches (keyed by agent name) remain in `~/.neural-junkie/repos/`:
- `ArchitectureExpert/`
- `Code_Navigator/`
- `MSNotificationsExpert/`

These will not be automatically used by new agents. The system will create new caches with hash-based keys.

### Migration Strategy

To avoid re-indexing existing repositories:

1. Note the repository paths for existing agents
2. Delete old agent name-based cache directories (optional)
3. First new agent will rebuild cache with new key (one-time cost)
4. Subsequent agents will use the new cache

Alternatively, leave old caches in place - they don't cause harm and will eventually be replaced.

## Performance Characteristics

### With Cache (Fresh)
- Loading time: <2 seconds
- User sees: "✅ Loaded from cache (instant)"
- No progress bar shown

### With Cache (Stale)
- Loading time: 5-15 seconds (incremental update)
- User sees: "🔄 Cache stale (reason), performing incremental update..."
- Progress shown for update

### Without Cache
- Loading time: 30-60 seconds (full analysis)
- User sees: "📊 No cache found, performing full analysis..."
- Progress shown 0-100%
- Cache saved for future use

## Troubleshooting

### Cache not being used

Check if cache exists:
```bash
go run scripts/check-cache.go /path/to/your/repo
```

If cache doesn't exist:
- First agent creation will build it
- Watch for status messages in chat

If cache exists but not loading:
- Check server logs for errors
- Verify repository path is correct (absolute path)
- Try manually triggering reindex: `/reindex-agent <name>`

### Cache seems stale

Staleness is detected by:
1. Git commit hash changed
2. Key files modified (README, package.json, go.mod, etc.)
3. File count changed by >10%
4. Cache older than 7 days

This is expected behavior - incremental update will be fast.

## Files Modified

- `internal/repo/storage.go` - Cache key generation and metadata handling
- `internal/agent/repo_agent.go` - Cache loading and user feedback
- `internal/hub/commands.go` - Cache status in creation message
- `test/cache_test.go` - Unit tests for cache functionality
- `scripts/check-cache.go` - Utility to check cache status

## Backward Compatibility

The new system is backward compatible in that:
- Old caches won't break anything
- New caches use a different naming scheme
- Both can coexist in `~/.neural-junkie/repos/`
- No migration required (new caches created as needed)

## Future Enhancements

Potential improvements:
1. Cache cleanup command to remove old/unused caches
2. Cache statistics dashboard
3. Manual cache invalidation per repository
4. Cache warming (pre-index repositories)
5. Cache sharing across machines (export/import)

---

**Last Updated:** October 15, 2025  
**Status:** ✅ Implemented and Tested

