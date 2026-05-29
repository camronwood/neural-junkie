#!/usr/bin/env bash
# Smoke test MCP tool servers and export storage (no running hub required for unit checks).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "== MCP package tests =="
go test ./internal/mcp/... ./internal/agent/... ./internal/ai/... ./internal/mcp_export/... ./internal/delegation/... -count=1

echo "== Build =="
go build ./cmd/... ./internal/...

echo "== MCP env config (dry) =="
export ENABLE_MCP=true
export ENABLE_BACKEND_MCP=true
export MCP_BACKEND_PORT=8081
go test ./internal/mcp/ -run TestGetMCPServerConfig -v

echo "OK: MCP smoke checks passed"
