#!/bin/bash
set -euo pipefail

LABEL="com.ws-mcp.bridge"
PLIST_SRC="$(cd "$(dirname "$0")/../configs/launchd" && pwd)/${LABEL}.plist"
PLIST_DST="$HOME/Library/LaunchAgents/${LABEL}.plist"
LOG_DIR="$HOME/.bridge/logs"
PORT=8080

# Verify source plist exists
if [ ! -f "$PLIST_SRC" ]; then
    echo "Error: source plist not found at $PLIST_SRC" >&2
    exit 1
fi

# Create log directory
mkdir -p "$LOG_DIR"

# Unload existing agent if loaded
if launchctl list "$LABEL" &>/dev/null; then
    echo "Unloading existing $LABEL..."
    launchctl unload "$PLIST_DST" 2>/dev/null || true
fi

# Copy plist with $HOME substituted
sed "s|__HOME__|$HOME|g" "$PLIST_SRC" > "$PLIST_DST"

echo "Installed plist to $PLIST_DST"

# Load the agent
launchctl load "$PLIST_DST"
echo "Loaded $LABEL via launchctl"

# Wait briefly for the service to start
sleep 2

# Health check
echo "Checking health at localhost:${PORT}..."
if curl -sf --max-time 5 "http://localhost:${PORT}/health" >/dev/null 2>&1; then
    echo "Health check passed."
elif curl -sf --max-time 5 "http://localhost:${PORT}/" >/dev/null 2>&1; then
    echo "Service is responding on port ${PORT}."
else
    echo "Warning: service not yet responding on port ${PORT}." >&2
    echo "Check logs at $LOG_DIR for details." >&2
    exit 1
fi

echo "ws-mcp-bridge installed and running."
