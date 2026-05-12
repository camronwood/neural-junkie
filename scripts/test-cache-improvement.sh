#!/bin/bash

# Script to demonstrate the improved repository agent caching
# This shows that multiple agents pointing to the same repo share the cache

set -e

echo "🧪 Testing Improved Repository Agent Caching"
echo "=============================================="
echo ""

# Function to check cache status
check_cache() {
    local repo_path="$1"
    echo "📂 Checking cache for: $repo_path"
    
    # Use Go to generate cache key and check existence
    go run -C /Users/camronwood/development/sandbox/neural-junkie <<EOF
package main

import (
    "fmt"
    "github.com/camron/neural-junkie/internal/repo"
)

func main() {
    storage, err := repo.NewStorage()
    if err != nil {
        fmt.Printf("❌ Error: %v\n", err)
        return
    }
    
    cacheKey, err := storage.GetCacheKeyForPath("$repo_path")
    if err != nil {
        fmt.Printf("❌ Error generating cache key: %v\n", err)
        return
    }
    
    fmt.Printf("🔑 Cache Key: %s\n", cacheKey)
    
    if storage.IndexExists(cacheKey) {
        fmt.Println("✅ Cache EXISTS")
        
        // Try to load metadata
        metadata, err := storage.LoadMetadata(cacheKey)
        if err == nil {
            fmt.Printf("📋 Agents using this cache: %v\n", metadata.AgentNames)
        }
    } else {
        fmt.Println("❌ Cache DOES NOT EXIST")
    }
}
EOF
    echo ""
}

# Test with the current repo
CURRENT_REPO="/Users/camronwood/development/sandbox/neural-junkie"

echo "1️⃣  Check current cache status for neural-junkie repo:"
check_cache "$CURRENT_REPO"

echo ""
echo "2️⃣  Verify cache key consistency:"
echo "   Multiple calls to same path should produce identical cache keys..."
check_cache "$CURRENT_REPO"

echo ""
echo "3️⃣  Different paths should produce different cache keys:"
check_cache "/tmp/test-repo"

echo ""
echo "✅ Cache Improvement Summary:"
echo "   - Cache keys are now based on repository PATH (SHA256 hash)"
echo "   - Multiple agents pointing to the same repo SHARE the cache"
echo "   - Cache is persistent across agent deletions/recreations"
echo "   - Users get visual feedback about cache status"
echo ""
echo "🎯 To test in real usage:"
echo "   1. Start server: make server"
echo "   2. Start desktop: make desktop"
echo "   3. Create agent: /create-repo-agent $CURRENT_REPO Agent1"
echo "   4. Watch for cache messages in chat"
echo "   5. Delete agent: /delete-agent Agent1"
echo "   6. Create new agent: /create-repo-agent $CURRENT_REPO Agent2"
echo "   7. Should load instantly from cache!"
echo ""

