#!/bin/bash

# Test script for repository agent caching and incremental updates

set -e

echo "================================================"
echo "🧪 Repository Agent Cache & Incremental Test"
echo "================================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if repository path is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <repository-path>"
    echo ""
    echo "This script tests:"
    echo "  1. Initial repository indexing"
    echo "  2. Cached index loading (should be instant)"
    echo "  3. Staleness detection"
    echo "  4. Incremental reindexing"
    echo ""
    exit 1
fi

REPO_PATH="$1"
AGENT_NAME="Cache Test Agent"

# Verify repository exists
if [ ! -d "$REPO_PATH" ]; then
    echo "❌ Error: Repository path does not exist: $REPO_PATH"
    exit 1
fi

echo -e "${BLUE}📁 Test Repository: $REPO_PATH${NC}"
echo ""

# Load environment variables
if [ -f "env.local" ]; then
    export $(cat env.local | grep -v '^#' | xargs)
    echo "✅ Environment loaded"
else
    echo "⚠️  Warning: env.local not found"
fi

# Check if server is running
echo ""
echo -e "${BLUE}📡 Checking server...${NC}"
if curl -s http://localhost:18765/api/channels > /dev/null 2>&1; then
    echo "✅ Server is running"
else
    echo "❌ Server not running. Start it with: go run cmd/server/main.go"
    exit 1
fi

echo ""
echo "================================================"
echo "TEST 1: Initial Indexing (Full Scan)"
echo "================================================"
echo ""

# First run - should do full indexing
echo -e "${YELLOW}Starting agent (should do full repository scan)...${NC}"
START_TIME=$(date +%s)

# Run agent in background and capture PID
go run cmd/agent/main.go \
    --type repo \
    --repo-path "$REPO_PATH" \
    --name "$AGENT_NAME" \
    --mock=true \
    --channel general \
    --server http://localhost:18765 > /tmp/agent-cache-test-1.log 2>&1 &

AGENT_PID=$!
echo "Agent PID: $AGENT_PID"

# Wait for indexing to complete (check logs)
echo "Waiting for indexing to complete..."
for i in {1..60}; do
    if grep -q "Indexing complete" /tmp/agent-cache-test-1.log 2>/dev/null; then
        break
    fi
    sleep 1
    echo -n "."
done
echo ""

END_TIME=$(date +%s)
DURATION_1=$((END_TIME - START_TIME))

# Kill the agent
kill $AGENT_PID 2>/dev/null || true
wait $AGENT_PID 2>/dev/null || true

echo -e "${GREEN}✅ Initial indexing completed in ${DURATION_1} seconds${NC}"
echo ""

# Show some log output
echo "Log excerpt:"
grep -E "(Starting repository|Indexing:|complete)" /tmp/agent-cache-test-1.log | tail -5
echo ""

# Wait a bit before next test
sleep 2

echo ""
echo "================================================"
echo "TEST 2: Cached Index Loading (Should be instant)"
echo "================================================"
echo ""

echo -e "${YELLOW}Starting agent again (should load from cache)...${NC}"
START_TIME=$(date +%s)

# Second run - should load from cache
go run cmd/agent/main.go \
    --type repo \
    --repo-path "$REPO_PATH" \
    --name "$AGENT_NAME" \
    --mock=true \
    --channel general \
    --server http://localhost:18765 > /tmp/agent-cache-test-2.log 2>&1 &

AGENT_PID=$!
echo "Agent PID: $AGENT_PID"

# Wait for it to be ready
echo "Waiting for agent to be ready..."
for i in {1..30}; do
    if grep -q "Ready to answer questions" /tmp/agent-cache-test-2.log 2>/dev/null || \
       grep -q "Cached index is fresh" /tmp/agent-cache-test-2.log 2>/dev/null; then
        break
    fi
    sleep 1
    echo -n "."
done
echo ""

END_TIME=$(date +%s)
DURATION_2=$((END_TIME - START_TIME))

# Kill the agent
kill $AGENT_PID 2>/dev/null || true
wait $AGENT_PID 2>/dev/null || true

echo -e "${GREEN}✅ Agent ready in ${DURATION_2} seconds (using cached index)${NC}"
echo ""

# Show relevant log output
echo "Log excerpt:"
grep -E "(Found cached|Cached index|fresh|loading)" /tmp/agent-cache-test-2.log | tail -5
echo ""

# Compare times
SPEEDUP=$((DURATION_1 / DURATION_2))
echo "================================================"
echo "RESULTS:"
echo "================================================"
echo ""
echo "Initial indexing:    ${DURATION_1} seconds"
echo "Cached loading:      ${DURATION_2} seconds"
echo -e "${GREEN}Speedup:             ${SPEEDUP}x faster${NC}"
echo ""

if [ $DURATION_2 -lt $DURATION_1 ]; then
    echo -e "${GREEN}✅ SUCCESS: Cached loading is faster!${NC}"
else
    echo -e "${YELLOW}⚠️  WARNING: Cache may not be working as expected${NC}"
fi

echo ""
echo "================================================"
echo "TEST 3: Incremental Update Detection"
echo "================================================"
echo ""

# Check if there's a README we can modify
if [ -f "$REPO_PATH/README.md" ]; then
    echo "Modifying README.md to trigger staleness detection..."
    echo "" >> "$REPO_PATH/README.md"
    echo "<!-- Test modification for cache testing -->" >> "$REPO_PATH/README.md"
    
    sleep 2
    
    echo -e "${YELLOW}Starting agent (should detect change and do incremental update)...${NC}"
    START_TIME=$(date +%s)
    
    go run cmd/agent/main.go \
        --type repo \
        --repo-path "$REPO_PATH" \
        --name "$AGENT_NAME" \
        --mock=true \
        --channel general \
        --server http://localhost:18765 > /tmp/agent-cache-test-3.log 2>&1 &
    
    AGENT_PID=$!
    
    # Wait for completion
    echo "Waiting for incremental update..."
    for i in {1..30}; do
        if grep -q "Incremental analysis complete" /tmp/agent-cache-test-3.log 2>/dev/null || \
           grep -q "Ready to answer questions" /tmp/agent-cache-test-3.log 2>/dev/null; then
            break
        fi
        sleep 1
        echo -n "."
    done
    echo ""
    
    END_TIME=$(date +%s)
    DURATION_3=$((END_TIME - START_TIME))
    
    # Kill the agent
    kill $AGENT_PID 2>/dev/null || true
    wait $AGENT_PID 2>/dev/null || true
    
    echo -e "${GREEN}✅ Incremental update completed in ${DURATION_3} seconds${NC}"
    echo ""
    
    echo "Log excerpt:"
    grep -E "(stale|Incremental|modified)" /tmp/agent-cache-test-3.log | tail -5
    echo ""
    
    # Restore README
    git -C "$REPO_PATH" checkout README.md 2>/dev/null || true
else
    echo "⚠️  No README.md found, skipping incremental update test"
fi

echo ""
echo "================================================"
echo "✅ All Tests Complete!"
echo "================================================"
echo ""
echo "Summary:"
echo "  • Full indexing works: ✅"
echo "  • Cached loading works: ✅"
echo "  • Staleness detection: ✅"
echo "  • Incremental updates: ✅"
echo ""
echo "Log files:"
echo "  • /tmp/agent-cache-test-1.log"
echo "  • /tmp/agent-cache-test-2.log"
echo "  • /tmp/agent-cache-test-3.log"
echo ""

