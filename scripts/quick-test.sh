#!/bin/bash

# Quick Test Script for Neural Junkie GUI
# This script sets up everything you need to test the GUI

set -e

cd "$(dirname "$0")/.."

echo "🚀 Neural Junkie - Quick Test Setup"
echo "===================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"
    jobs -p | xargs kill 2>/dev/null || true
    echo -e "${GREEN}Done!${NC}"
}

trap cleanup EXIT INT TERM

# Step 1: Start Server
echo -e "${BLUE}Step 1: Starting Chat Hub Server...${NC}"
go run ./cmd/server > /tmp/ai-chat-server.log 2>&1 &
SERVER_PID=$!
echo -e "  ${GREEN}✓${NC} Server started (PID: $SERVER_PID)"
sleep 2

# Check if server is running
if ! curl -s http://localhost:18765/api/channels > /dev/null 2>&1; then
    echo -e "${RED}✗ Server failed to start!${NC}"
    echo "Check logs: tail /tmp/ai-chat-server.log"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Server is responding"
echo ""

# Step 2: Start Agents
echo -e "${BLUE}Step 2: Starting AI Agents...${NC}"

go run cmd/agent/main.go --type backend --name "Go Expert" > /dev/null 2>&1 &
echo -e "  ${GREEN}✓${NC} Backend Agent (Go Expert)"
sleep 2

go run cmd/agent/main.go --type frontend --name "React Pro" > /dev/null 2>&1 &
echo -e "  ${GREEN}✓${NC} Frontend Agent (React Pro)"
sleep 2

go run cmd/agent/main.go --type security --name "Security Expert" > /dev/null 2>&1 &
echo -e "  ${GREEN}✓${NC} Security Agent (Security Expert)"
sleep 2

go run cmd/agent/main.go --type database --name "SQL Master" > /dev/null 2>&1 &
echo -e "  ${GREEN}✓${NC} Database Agent (SQL Master)"
sleep 2

echo ""

# Check agents
AGENT_COUNT=$(curl -s http://localhost:18765/api/agents | jq '. | length' 2>/dev/null || echo "0")
echo -e "  ${GREEN}✓${NC} Active agents: $AGENT_COUNT"
echo ""

# Step 3: Instructions
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ Setup Complete!${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}Server:${NC}  http://localhost:18765"
echo -e "${BLUE}Agents:${NC}  $AGENT_COUNT active"
echo ""
echo -e "${YELLOW}Now you can:${NC}"
echo ""
echo -e "  ${BLUE}1. Test the GUI:${NC}"
echo "     make gui"
echo "     (or: go run cmd/gui/main.go)"
echo ""
echo -e "  ${BLUE}2. Test the Terminal Chat:${NC}"
echo "     make chat"
echo "     (or: go run cmd/chat/main.go)"
echo ""
echo -e "  ${BLUE}3. Test the Web UI:${NC}"
echo "     open http://localhost:18765"
echo ""
echo -e "${YELLOW}Example questions to try:${NC}"
echo "  • How do I prevent SQL injection?"
echo "  • Our API is slow. What could be wrong?"
echo "  • How should we implement authentication?"
echo "  • What's the best way to deploy microservices?"
echo ""
echo -e "${RED}Press Ctrl+C when done to stop all services${NC}"
echo ""

# Keep script running
wait

