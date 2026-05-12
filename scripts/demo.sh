#!/bin/bash

# Neural Junkie Demo Script
# Unified demo script with multiple modes

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Default mode
MODE="agents"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --agents)
            MODE="agents"
            shift
            ;;
        --messages)
            MODE="messages"
            shift
            ;;
        --interactive)
            MODE="interactive"
            shift
            ;;
        --help|-h)
            echo "Neural Junkie Demo Script"
            echo ""
            echo "Usage: $0 [MODE]"
            echo ""
            echo "Modes:"
            echo "  --agents       Start agents and run demo scenarios (default)"
            echo "  --messages     Send test messages to existing agents"
            echo "  --interactive  Start interactive chat client"
            echo "  --help         Show this help"
            echo ""
            echo "Examples:"
            echo "  $0                    # Run agent demo"
            echo "  $0 --messages         # Send test messages"
            echo "  $0 --interactive      # Start interactive chat"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Check if server is running
if ! curl -s http://localhost:18765/api/channels > /dev/null 2>&1; then
    echo -e "${RED}❌ Server not running!${NC}"
    echo ""
    echo "Please start the server first:"
    echo -e "${BLUE}  make server${NC}"
    echo ""
    echo "Or in a separate terminal:"
    echo -e "${BLUE}  go run cmd/server/main.go${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Server is running${NC}"
echo ""

# Mode: Send test messages
if [ "$MODE" = "messages" ]; then
    echo "🎬 Neural Junkie Demo - Message Sending"
    echo "======================================"
    echo ""
    
    sleep 1
    
    echo "📤 Sending React question..."
    curl -s -X POST http://localhost:18765/api/send \
      -H "Content-Type: application/json" \
      -d '{"channel": "general", "content": "What are the best practices for using React hooks?", "type": "question"}'
    echo " ✓"
    sleep 3
    
    echo "📤 Sending database question..."
    curl -s -X POST http://localhost:18765/api/send \
      -H "Content-Type: application/json" \
      -d '{"channel": "general", "content": "How can I optimize my PostgreSQL queries?", "type": "question"}'
    echo " ✓"
    sleep 3
    
    echo "📤 Sending DevOps question..."
    curl -s -X POST http://localhost:18765/api/send \
      -H "Content-Type: application/json" \
      -d '{"channel": "general", "content": "What is the best way to set up Kubernetes monitoring?", "type": "question"}'
    echo " ✓"
    sleep 3
    
    echo ""
    echo -e "${GREEN}✅ Demo messages sent!${NC}"
    echo "Check the desktop app or chat client to see agent responses!"
    echo ""
    exit 0
fi

# Mode: Interactive chat
if [ "$MODE" = "interactive" ]; then
    echo "🎬 Neural Junkie - Interactive Demo Setup"
    echo "========================================="
    echo ""
    
    # Check for agents
    AGENT_COUNT=$(curl -s http://localhost:18765/api/agents | jq '. | length' 2>/dev/null || echo "0")
    
    if [ "$AGENT_COUNT" -eq "0" ]; then
        echo -e "${YELLOW}⚠️  No agents detected. Starting agents...${NC}"
        echo ""
        
        # Start agents in background
        go run cmd/agent/main.go --type frontend --name "React Expert" > /dev/null 2>&1 &
        echo -e "  ${GREEN}✓${NC} Frontend Agent started"
        sleep 1
        
        go run cmd/agent/main.go --type backend --name "Go Master" > /dev/null 2>&1 &
        echo -e "  ${GREEN}✓${NC} Backend Agent started"
        sleep 1
        
        go run cmd/agent/main.go --type security --name "Security Pro" > /dev/null 2>&1 &
        echo -e "  ${GREEN}✓${NC} Security Agent started"
        sleep 1
        
        echo ""
        echo -e "${GREEN}✅ Agents ready!${NC}"
    else
        echo -e "${GREEN}✅ Found $AGENT_COUNT active agent(s)${NC}"
    fi
    
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}  Ready to chat with your AI agent team! 🤖  ${BLUE}║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Starting interactive chat in 2 seconds..."
    echo ""
    echo -e "${YELLOW}Tips:${NC}"
    echo "  • Type naturally - agents will respond based on their expertise"
    echo "  • Use /agents to see who's online"
    echo "  • Use /help for more commands"
    echo "  • Press Ctrl+C to exit"
    echo ""
    
    sleep 2
    
    # Start the interactive chat
    go run cmd/chat/main.go
    exit 0
fi

# Mode: Agents (default)
echo "🎬 Neural Junkie Demo"
echo "==================="
echo ""

# Start agents in background
echo -e "${BLUE}🤖 Starting agents...${NC}"
echo ""

go run cmd/agent/main.go --type frontend --name "React Expert" --channel general > /dev/null 2>&1 &
FRONTEND_PID=$!
echo "  ✓ Frontend Agent (PID: $FRONTEND_PID)"
sleep 2

go run cmd/agent/main.go --type backend --name "Go Master" --channel general > /dev/null 2>&1 &
BACKEND_PID=$!
echo "  ✓ Backend Agent (PID: $BACKEND_PID)"
sleep 2

go run cmd/agent/main.go --type devops --name "Cloud Architect" --channel general > /dev/null 2>&1 &
DEVOPS_PID=$!
echo "  ✓ DevOps Agent (PID: $DEVOPS_PID)"
sleep 2

go run cmd/agent/main.go --type database --name "Database Expert" --channel general > /dev/null 2>&1 &
DATABASE_PID=$!
echo "  ✓ Database Agent (PID: $DATABASE_PID)"
sleep 2

go run cmd/agent/main.go --type security --name "Security Expert" --channel general > /dev/null 2>&1 &
SECURITY_PID=$!
echo "  ✓ Security Agent (PID: $SECURITY_PID)"
sleep 2

echo ""
echo -e "${GREEN}✅ All agents are active!${NC}"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}🛑 Stopping agents...${NC}"
    kill $FRONTEND_PID $BACKEND_PID $DEVOPS_PID $DATABASE_PID $SECURITY_PID 2>/dev/null || true
    echo -e "${GREEN}✅ Demo complete!${NC}"
    exit 0
}

trap cleanup EXIT INT TERM

# Demo scenarios
echo -e "${BLUE}📝 Scenario 1: Performance Issue${NC}"
echo "   Question: 'Our API is slow when fetching user data. What could be the issue?'"
echo ""
go run cmd/cli/main.go --channel general --message "Our API is slow when fetching user data. What could be the issue?"
sleep 8
echo ""

echo -e "${BLUE}📝 Scenario 2: Authentication Architecture${NC}"
echo "   Question: 'How should we implement authentication for our new app?'"
echo ""
go run cmd/cli/main.go --channel general --message "How should we implement authentication for our new app?"
sleep 8
echo ""

echo -e "${BLUE}📝 Scenario 3: Deployment Strategy${NC}"
echo "   Question: 'What's the best way to deploy our microservices?'"
echo ""
go run cmd/cli/main.go --channel general --message "What's the best way to deploy our microservices?"
sleep 8
echo ""

# Show final conversation
echo -e "${BLUE}💬 Recent conversation:${NC}"
echo ""
go run cmd/cli/main.go --list messages --channel general
echo ""

echo -e "${GREEN}🎉 Demo complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. Open the desktop app: make desktop"
echo "  2. Try sending your own messages with the CLI"
echo "  3. Start agents in specific channels for focused discussions"
echo ""
echo "Press Ctrl+C to stop all agents"
echo ""

# Keep running until interrupted
wait
