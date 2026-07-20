#!/bin/bash

# Load Neural Junkie environment variables from env.local

if [ -f "env.local" ]; then
    export $(cat env.local | grep -v '^#' | xargs)
    echo "✅ Environment variables loaded from env.local"
    # Use :- defaults so callers with `set -u` (overnight, preflight) do not abort
    # when env.local omits optional AI hub keys.
    echo "   USE_AI_HUB: ${USE_AI_HUB:-}"
    echo "   AI_HUB_ENDPOINT: ${AI_HUB_ENDPOINT:-}"
    echo "   AI_HUB_MODEL: ${AI_HUB_MODEL:-}"
elif [ ! -f ".gemini-api-key" ]; then
    echo "❌ env.local file not found"
    echo "   Copy env.example to env.local and configure your settings"
    exit 1
fi

if [ -z "${GEMINI_API_KEY:-}" ] && [ -f ".gemini-api-key" ]; then
    export GEMINI_API_KEY="$(
        grep -v '^[[:space:]]*#' .gemini-api-key \
        | grep -v '^[[:space:]]*$' \
        | head -1 \
        | tr -d '[:space:]'
    )"
    echo "✅ GEMINI_API_KEY loaded from .gemini-api-key (first key)"
fi

if [ -z "${CURSOR_API_KEY:-}" ] && [ -f ".cursor-api-key" ]; then
    export CURSOR_API_KEY="$(tr -d '[:space:]' < .cursor-api-key)"
    echo "✅ CURSOR_API_KEY loaded from .cursor-api-key"
fi

# Drop stale LiteLLM/Claude proxy overrides unless explicitly opted in.
if [ "${NEURAL_JUNKIE_CLAUDE_CUSTOM_ROUTING:-}" != "1" ]; then
    unset ANTHROPIC_BASE_URL ANTHROPIC_AUTH_TOKEN ANTHROPIC_MODEL ANTHROPIC_SMALL_FAST_MODEL
fi
