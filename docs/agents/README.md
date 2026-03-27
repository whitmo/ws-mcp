# Agent Integration Prompts

Each file in this directory is a prompt/guide for a specific AI agent or tool explaining how to use the ws-mcp bridge to communicate with other agents.

These files serve dual purpose:
1. **Documentation** for humans setting up the integration
2. **Prompts** that can be injected into agent context so they know how to use the bridge

## Files

| Agent | File | MCP | Hooks | Notes |
|---|---|---|---|---|
| [Claude Code](claude-code.md) | Terminal CLI | ✅ via .mcp.json | ✅ SessionStart/Stop | Primary dev agent |
| [Codex CLI](codex.md) | Terminal CLI | ✅ via `codex mcp` | — | OpenAI's coding agent |
| [Gemini CLI](gemini-cli.md) | Terminal CLI | ✅ via `gemini mcp` | — | Google's coding agent |
| [Cowork / Claude Desktop](cowork.md) | Desktop app | ✅ via config JSON | — | User-facing agent |
| [Ralph](ralph.md) | Orchestrator | ✅ via .mcp.json | ✅ ralph.yml hooks | Loop orchestration |
| [Multiclaude](multiclaude.md) | Worker swarm | ✅ inherited | ✅ inherited | Parallel workers |

## How it all connects

```
                    ┌─────────────────┐
                    │   ws-mcp bridge │
                    │  localhost:8080  │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
    HTTP POST           WebSocket            MCP stdio
    /event               /ws               --stdio
         │                   │                   │
    ┌────┴────┐      ┌──────┴──────┐    ┌───────┴───────┐
    │  hooks  │      │  observer   │    │  MCP clients  │
    │ scripts │      │  reactor    │    │               │
    └────┬────┘      └─────────────┘    └───────┬───────┘
         │                                      │
    ┌────┴────────────────────────────────────────┐
    │  ralph  │ multiclaude │ claude │ codex │ gemini │ cowork
    └─────────────────────────────────────────────────┘
```

## Installation

See [INSTALL.md](../INSTALL.md) for setup instructions.
