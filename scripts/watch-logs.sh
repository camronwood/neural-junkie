#!/bin/bash

# Real-time log monitoring for GUI

echo "🔍 Monitoring GUI Logs"
echo "====================="
echo ""

LOG_FILE="/tmp/gui-output.log"

if [ ! -f "$LOG_FILE" ]; then
    echo "Creating log file..."
    touch "$LOG_FILE"
fi

echo "📊 Live Updates (Press Ctrl+C to stop)"
echo "======================================="
echo ""

# Monitor with colors
tail -f "$LOG_FILE" | while read line; do
    timestamp=$(date +"%H:%M:%S")
    
    # Color code based on log level
    if echo "$line" | grep -qi "error"; then
        echo -e "\033[0;31m[$timestamp] ❌ $line\033[0m"
    elif echo "$line" | grep -qi "warn"; then
        echo -e "\033[0;33m[$timestamp] ⚠️  $line\033[0m"
    elif echo "$line" | grep -qi "connected\|success"; then
        echo -e "\033[0;32m[$timestamp] ✅ $line\033[0m"
    elif echo "$line" | grep -qi "starting\|showing\|building"; then
        echo -e "\033[0;34m[$timestamp] 🔵 $line\033[0m"
    elif echo "$line" | grep -qi "WebSocket\|message"; then
        echo -e "\033[0;36m[$timestamp] 💬 $line\033[0m"
    else
        echo "[$timestamp] $line"
    fi
done

