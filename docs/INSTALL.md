# ws-mcp Bridge Installation

## Quick Start

```bash
# Clone and build
git clone https://github.com/whitmo/ws-mcp.git
cd ws-mcp
go build -o ~/bin/ws-mcp-bridge ./src/cmd/bridge
go build -o ~/bin/ws-mcp-observer ./src/cmd/observer
go build -o ~/bin/ws-mcp-reactor ./src/cmd/reactor

# Start the bridge
bash scripts/bridge-start.sh

# Verify
curl http://localhost:8080/healthz  # → OK
```

## Prerequisites

- Go 1.24+
- `~/bin` on your PATH

## Agent Setup

### Claude Code
Already configured via `.mcp.json` in the repo. For global access:
```bash
# Adds to ~/.claude/settings.json
python3 -c "
import json
with open('$HOME/.claude/settings.json') as f: d=json.load(f)
d.setdefault('mcpServers',{})['ws-mcp']={'command':'ws-mcp-bridge','args':['--stdio']}
with open('$HOME/.claude/settings.json','w') as f: json.dump(d,f,indent=2)
"
```

### Codex CLI
```bash
codex mcp add ws-mcp -- ws-mcp-bridge --stdio
```

### Gemini CLI
```bash
gemini mcp add ws-mcp ws-mcp-bridge -- --stdio
```

### Claude Desktop (Cowork)
Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "ws-mcp": {
      "command": "/Users/YOUR_USERNAME/bin/ws-mcp-bridge",
      "args": ["--stdio"]
    }
  }
}
```
Restart Claude Desktop after editing.

### Ralph
Already configured via `ralph.yml` hooks. No additional setup needed if using this repo.

### Multiclaude
Workers inherit Claude Code's `.mcp.json` automatically. No additional setup.

## Persistent Service (macOS)

To keep the bridge running across reboots, install the launchd agent:
```bash
bash scripts/bridge-install.sh
```

To remove:
```bash
bash scripts/bridge-uninstall.sh
```

## Verify Everything Works

```bash
# Run the full test suite
go test ./...

# Run the smoke test
bash scripts/smoke-test.sh

# Test MCP stdio protocol
bash scripts/test-mcp-stdio.sh

# Post a test event and query it back
bash scripts/post-event.sh system.healthcheck system '{"test":true}'
bash scripts/query-events.sh events.latest '{"limit":1}'
```
