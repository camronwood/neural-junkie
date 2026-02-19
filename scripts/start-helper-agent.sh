#!/bin/bash

# Start a helper agent by name
# Usage: ./start-helper-agent.sh <helper-name> [channel]

HELPER_NAME="${1:-day-one}"
CHANNEL="${2:-general}"
SERVER="http://localhost:8080"

if [ -z "$HELPER_NAME" ]; then
    echo "Usage: $0 <helper-name> [channel]"
    echo "Example: $0 day-one general"
    exit 1
fi

# Check if helper exists
HELPER_DIR="$HOME/.neural-junkie/helpers/$HELPER_NAME"
if [ ! -d "$HELPER_DIR" ]; then
    echo "Error: Helper agent '$HELPER_NAME' not found at: $HELPER_DIR"
    echo ""
    echo "Available helpers:"
    ls -1 "$HOME/.neural-junkie/helpers/" 2>/dev/null || echo "  (none found)"
    exit 1
fi

if [ ! -f "$HELPER_DIR/config.json" ]; then
    echo "Error: config.json not found for helper '$HELPER_NAME'"
    exit 1
fi

echo "Starting helper agent: $HELPER_NAME"
echo "Channel: $CHANNEL"
echo "Server: $SERVER"
echo ""

# Build and run the helper agent
cd "$(dirname "$0")/.."
go run cmd/helper-agent/main.go \
    --name "$HELPER_NAME" \
    --channel "$CHANNEL" \
    --server "$SERVER"

