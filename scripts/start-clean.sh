#!/bin/bash

# Kill all existing processes
echo "🧹 Cleaning up old processes..."
pkill -9 -f "go run" 2>/dev/null
sleep 2

cd "$(dirname "$0")/.."

# Source environment
source load-env.sh

echo "🚀 Starting server..."
go run ./cmd/server &
SERVER_PID=$!
sleep 3

echo "🤖 Starting Go Expert..."
go run cmd/agent/main.go --type backend --name "Go Expert" &
sleep 2

echo "🤖 Starting SQL Master..."
go run cmd/agent/main.go --type database --name "SQL Master" &
sleep 1

echo "🤖 Starting Security Expert..."
go run cmd/agent/main.go --type security --name "Security Expert" &
sleep 1

echo "✅ System started!"
echo ""
echo "Agents running:"
curl -s http://localhost:18765/api/agents | python3 -c "import sys, json; agents = json.load(sys.stdin); print(f'  Total: {len(agents)} agents'); [print(f'    - {a[\"name\"]} ({a[\"type\"]})') for a in agents]"
echo ""
echo "Starting GUI in 2 seconds..."
sleep 2
go run cmd/gui/main.go

