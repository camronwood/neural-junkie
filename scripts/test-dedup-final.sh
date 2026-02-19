#!/bin/bash

echo "🧪 FINAL DEDUPLICATION TEST"
echo "============================="
echo ""

# Clean everything
pkill -f "bin/agent" 2>/dev/null
pkill -f "bin/server" 2>/dev/null
sleep 2

# Start fresh server
cd /Users/camron.wood.ext/development/sandbox/neural-junkie
./bin/server > /dev/null 2>&1 &
SERVER_PID=$!
sleep 2

echo "✅ Server started (PID: $SERVER_PID)"

# Start ONE agent with full logging
./bin/agent --type security --name "Security Expert" > /tmp/security-full.log 2>&1 &
AGENT_PID=$!
sleep 3

echo "✅ Agent started (PID: $AGENT_PID)"
echo ""

# Check agent is registered
echo "📋 Registered agents:"
curl -s http://localhost:8080/api/agents | jq '.[] | {name: .name, id: .id}'
echo ""

# Send a test message
echo "📤 Sending test message..."
curl -s -X POST http://localhost:8080/api/send \
  -H "Content-Type: application/json" \
  -d '{"channel": "general", "content": "TEST: What are the best security practices?", "type": "question", "from": {"name": "TestUser", "type": "human"}}' 
echo ""

# Wait for responses
echo "⏳ Waiting 5 seconds for agent to respond..."
sleep 5

# Count responses
echo ""
echo "📊 Messages from Security Expert:"
curl -s 'http://localhost:8080/api/messages?channel=general&limit=50' | \
  jq '.[] | select(.from.name == "Security Expert") | {id: .id[0:8], content: .content[0:50]}'

echo ""
echo "🔍 Expected: 1-2 messages (join + response)"
echo "🔍 Checking logs..."
echo ""

# Show relevant log lines
echo "=== AGENT LOGS ==="
grep -E "RECEIVED|IGNORING|SKIPPING|MARKED|WILL RESPOND" /tmp/security-full.log | tail -20

# Cleanup
kill $AGENT_PID 2>/dev/null
kill $SERVER_PID 2>/dev/null

echo ""
echo "✅ Test complete"

