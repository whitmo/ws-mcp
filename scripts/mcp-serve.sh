#!/usr/bin/env bash
# Start the ws-mcp bridge in MCP stdio mode for agent consumption.
# Usage: ./scripts/mcp-serve.sh [--port 8080]
set -euo pipefail

cd "$(dirname "$0")/.."
exec go run ./src/cmd/bridge --mode mcp "$@"
