#!/bin/bash

# Ensure dependencies are installed before running scripts
npm install --silent 2>/dev/null || true

if [ "$NODE_ENV" = "production" ]; then
  npm run build
else
  npm run dev
fi

npx tauri dev