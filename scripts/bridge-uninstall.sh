#!/bin/bash
set -euo pipefail

LABEL="com.ws-mcp.bridge"
PLIST_DST="$HOME/Library/LaunchAgents/${LABEL}.plist"

# Unload the agent if loaded
if launchctl list "$LABEL" &>/dev/null; then
    echo "Unloading $LABEL..."
    launchctl unload "$PLIST_DST"
    echo "Unloaded."
else
    echo "$LABEL is not currently loaded."
fi

# Remove the plist
if [ -f "$PLIST_DST" ]; then
    rm "$PLIST_DST"
    echo "Removed $PLIST_DST"
else
    echo "No plist found at $PLIST_DST"
fi

echo "ws-mcp-bridge uninstalled."
