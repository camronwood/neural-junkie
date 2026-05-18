#!/usr/bin/env bash
# Smoke test MCP tool servers and export storage (no running hub required for unit checks).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "== MCP package tests =="
go test ./internal/mcp/... ./internal/agent/... ./internal/ai/... ./internal/mcp_export/...

echo "== Build =="
go build ./...

echo "== MCP env config (dry) =="
export ENABLE_MCP=true
export ENABLE_BACKEND_MCP=true
export MCP_BACKEND_PORT=8081
go test ./internal/mcp/ -run TestGetMCPServerConfigEnabled -v

echo "OK: MCP smoke checks passed"
