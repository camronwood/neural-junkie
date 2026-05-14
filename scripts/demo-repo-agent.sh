#!/bin/bash

# Demo script for Repository Expert Agents

set -e

echo "================================================"
echo "🤖 Repository Expert Agent Demo"
echo "================================================"
echo ""

# Check if repository path is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <repository-path> [agent-name]"
    echo ""
    echo "Example:"
    echo "  $0 /Users/john/projects/my-app \"MyApp Expert\""
    echo ""
    echo "This will:"
    echo "  1. Start the chat server (if not running)"
    echo "  2. Create a repository expert agent"
    echo "  3. Index the repository"
    echo "  4. Make the agent available in the chat"
    echo ""
    exit 1
fi

REPO_PATH="$1"
AGENT_NAME="${2:-$(basename $REPO_PATH) Expert}"

# Verify repository exists
if [ ! -d "$REPO_PATH" ]; then
    echo "❌ Error: Repository path does not exist: $REPO_PATH"
    exit 1
fi

echo "📁 Repository: $REPO_PATH"
echo "🤖 Agent Name: $AGENT_NAME"
echo ""

# Load environment variables
if [ -f "env.local" ]; then
    export $(cat env.local | grep -v '^#' | xargs)
    echo "✅ Environment variables loaded from env.local"
else
    echo "⚠️  Warning: env.local not found, using default settings"
fi

# Check if server is running
echo ""
echo "📡 Checking server status..."
if curl -s http://localhost:18765/api/channels > /dev/null 2>&1; then
    echo "✅ Server is running"
else
    echo "❌ Server is not running"
    echo "   Start it with: go run ./cmd/server"
    echo ""
    read -p "Start server now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "🚀 Starting server in background..."
        go run ./cmd/server > /tmp/ai-chat-server.log 2>&1 &
        SERVER_PID=$!
        echo "   Server PID: $SERVER_PID"
        echo "   Waiting for server to be ready..."
        sleep 3
        
        # Verify server started
        if curl -s http://localhost:18765/api/channels > /dev/null 2>&1; then
            echo "✅ Server is ready"
        else
            echo "❌ Failed to start server. Check /tmp/ai-chat-server.log"
            exit 1
        fi
    else
        echo "Please start the server manually and run this script again."
        exit 1
    fi
fi

# Start the repository agent
echo ""
echo "🔍 Creating repository expert agent..."
echo "   This may take a minute for large repositories..."
echo ""

# Run the agent with output
go run cmd/agent/main.go \
    --type repo \
    --repo-path "$REPO_PATH" \
    --name "$AGENT_NAME" \
    --mock=false \
    --channel general \
    --server http://localhost:18765

echo ""
echo "================================================"
echo "✅ Repository Agent Demo Complete!"
echo "================================================"
echo ""
echo "The agent '$AGENT_NAME' is now active and ready to answer questions!"
echo ""
echo "Try asking questions like:"
echo "  - What's the main entry point?"
echo "  - How is authentication handled?"
echo "  - What frameworks are used?"
echo "  - Where should I add a new feature?"
echo ""
echo "Open the GUI to chat: make gui"
echo ""


