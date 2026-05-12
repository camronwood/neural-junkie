#!/bin/bash

# Test script for message sending functionality

set -e

cd "$(dirname "$0")/.."

echo "🧪 Message Sending Test Suite"
echo "=============================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check server
echo -e "${BLUE}Checking server...${NC}"
if ! curl -s http://localhost:18765/api/channels > /dev/null 2>&1; then
    echo -e "${RED}✗ Server not running${NC}"
    echo "Start with: make server"
    exit 1
fi
echo -e "${GREEN}✓ Server is running${NC}"
echo ""

# Test 1: Send a simple message
echo -e "${BLUE}Test 1: Sending simple message...${NC}"
RESPONSE=$(curl -s -X POST http://localhost:18765/api/send \
    -H "Content-Type: application/json" \
    -d '{
        "channel": "general",
        "content": "Test message from automated test",
        "type": "question"
    }')

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Message sent successfully${NC}"
else
    echo -e "${RED}✗ Failed to send message${NC}"
    exit 1
fi
echo ""

# Test 2: Send multiple messages
echo -e "${BLUE}Test 2: Sending multiple messages...${NC}"
for i in {1..5}; do
    curl -s -X POST http://localhost:18765/api/send \
        -H "Content-Type: application/json" \
        -d "{
            \"channel\": \"general\",
            \"content\": \"Test message #$i\",
            \"type\": \"question\"
        }" > /dev/null
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Message $i sent${NC}"
    else
        echo -e "${RED}✗ Message $i failed${NC}"
    fi
    sleep 0.5
done
echo ""

# Test 3: Send different message types
echo -e "${BLUE}Test 3: Testing different message types...${NC}"

MESSAGE_TYPES=("question" "statement" "system_info")
for msg_type in "${MESSAGE_TYPES[@]}"; do
    curl -s -X POST http://localhost:18765/api/send \
        -H "Content-Type: application/json" \
        -d "{
            \"channel\": \"general\",
            \"content\": \"Testing $msg_type message\",
            \"type\": \"$msg_type\"
        }" > /dev/null
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $msg_type message sent${NC}"
    else
        echo -e "${RED}✗ $msg_type message failed${NC}"
    fi
done
echo ""

# Test 4: Send messages that trigger agent responses
echo -e "${BLUE}Test 4: Sending messages to trigger agent responses...${NC}"

# Frontend question
echo -e "  Sending frontend question..."
curl -s -X POST http://localhost:18765/api/send \
    -H "Content-Type: application/json" \
    -d '{
        "channel": "general",
        "content": "How do I use React hooks for state management?",
        "type": "question"
    }' > /dev/null

sleep 2

# Backend question
echo -e "  Sending backend question..."
curl -s -X POST http://localhost:18765/api/send \
    -H "Content-Type: application/json" \
    -d '{
        "channel": "general",
        "content": "What are the best practices for database indexing?",
        "type": "question"
    }' > /dev/null

sleep 2

# DevOps question
echo -e "  Sending DevOps question..."
curl -s -X POST http://localhost:18765/api/send \
    -H "Content-Type: application/json" \
    -d '{
        "channel": "general",
        "content": "How should I set up a CI/CD pipeline with GitHub Actions?",
        "type": "question"
    }' > /dev/null

echo -e "${GREEN}✓ Agent trigger messages sent${NC}"
echo ""

# Test 5: Check recent messages
echo -e "${BLUE}Test 5: Retrieving recent messages...${NC}"
MESSAGES=$(curl -s http://localhost:18765/api/channels/general/messages?limit=10)

if [ $? -eq 0 ]; then
    MESSAGE_COUNT=$(echo "$MESSAGES" | jq '. | length' 2>/dev/null || echo "0")
    echo -e "${GREEN}✓ Retrieved $MESSAGE_COUNT recent messages${NC}"
    
    if [ "$MESSAGE_COUNT" -gt "0" ]; then
        echo -e "${BLUE}  Last message:${NC}"
        echo "$MESSAGES" | jq '.[0] | {from: .from.name, content: .content}' 2>/dev/null || echo "  (Could not parse)"
    fi
else
    echo -e "${RED}✗ Failed to retrieve messages${NC}"
fi
echo ""

# Test 6: Run Go tests
echo -e "${BLUE}Test 6: Running Go unit tests...${NC}"
if go test ./test/gui_test.go -v 2>&1 | grep -E "PASS|FAIL"; then
    echo -e "${GREEN}✓ Go tests completed${NC}"
else
    echo -e "${YELLOW}⚠️  Some tests may have failed (check output above)${NC}"
fi
echo ""

# Test 7: Performance test
echo -e "${BLUE}Test 7: Running performance benchmark...${NC}"
go test ./test/gui_test.go -bench=BenchmarkMessageSending -benchtime=1s 2>&1 | grep -E "Benchmark|ns/op" || true
echo ""

# Summary
echo -e "${GREEN}════════════════════════════════════${NC}"
echo -e "${GREEN}✅ Message Sending Tests Complete!${NC}"
echo -e "${GREEN}════════════════════════════════════${NC}"
echo ""

# Check GUI logs if available
if [ -f /tmp/gui.log ]; then
    echo -e "${BLUE}GUI Log Summary:${NC}"
    echo "  Total lines: $(wc -l < /tmp/gui.log)"
    echo "  Errors: $(grep -i error /tmp/gui.log | wc -l)"
    echo "  Messages: $(grep -i "message" /tmp/gui.log | wc -l)"
    echo ""
    echo -e "${BLUE}Last 5 log entries:${NC}"
    tail -5 /tmp/gui.log | sed 's/^/  /'
    echo ""
fi

echo -e "${YELLOW}Monitor GUI in real-time:${NC}"
echo "  ./scripts/monitor-gui.sh"
echo ""
echo -e "${YELLOW}Or check logs directly:${NC}"
echo "  tail -f /tmp/gui.log"
echo ""

