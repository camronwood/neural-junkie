#!/bin/bash

# Load Neural Junkie environment variables from env.local

if [ -f "env.local" ]; then
    export $(cat env.local | grep -v '^#' | xargs)
    echo "✅ Environment variables loaded from env.local"
    echo "   USE_AI_HUB: $USE_AI_HUB"
    echo "   AI_HUB_ENDPOINT: $AI_HUB_ENDPOINT"
    echo "   AI_HUB_MODEL: $AI_HUB_MODEL"
elif [ ! -f ".gemini-api-key" ]; then
    echo "❌ env.local file not found"
    echo "   Copy env.example to env.local and configure your settings"
    exit 1
fi

if [ -z "${GEMINI_API_KEY:-}" ] && [ -f ".gemini-api-key" ]; then
    export GEMINI_API_KEY="$(tr -d '[:space:]' < .gemini-api-key)"
    echo "✅ GEMINI_API_KEY loaded from .gemini-api-key"
fi
