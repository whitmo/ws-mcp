# Code Review: silly-fox + brave-raccoon Workers

## Status: BOTH WORKERS COMPLETED — PRs Open

### silly-fox (Phase 2: MCP JSON-RPC Server) — PR #4
- **Branch**: multiclaude/silly-fox (commit 19d52d9)
- **Files changed**: main.go, handlers.go, server.go (new), server_test.go (new)
- **Tests**: 17 tests in server_test.go, all passing

### brave-raccoon (Phase 3: MCP Config/Docs) — PR #5
- **Branch**: multiclaude/brave-raccoon (commit 4e7d2e1)
- **Files changed**: .mcp.json (new), scripts/mcp-serve.sh (new), scripts/smoke-test.sh (new), README.md, .gitignore

## Files Reviewed (Deep Analysis Pass)
- [x] silly-fox: src/cmd/bridge/main.go — stdio transport flag implementation
- [x] silly-fox: src/internal/mcp/server.go — JSON-RPC dispatch + ServeStdio/ServeIO
- [x] silly-fox: src/internal/mcp/handlers.go — HandleLatest, HandleFilter, HandleAck, HandleSummary
- [x] silly-fox: src/internal/mcp/server_test.go — 17 tests covering dispatch, HTTP, stdio
- [x] brave-raccoon: .mcp.json — MCP server config
- [x] brave-raccoon: scripts/mcp-serve.sh — stdio launch script
- [x] brave-raccoon: scripts/smoke-test.sh — round-trip test
- [x] brave-raccoon: README.md — MCP config documentation
- [x] git merge-tree analysis — no textual conflicts between branches

---

## CRITICAL: Interface Contract Mismatch (CONFIRMED)

**silly-fox main.go** (line 36):
```go
if len(os.Args) > 1 && os.Args[1] == "--stdio" {
```

**brave-raccoon .mcp.json**:
```json
"args": ["run", "./src/cmd/bridge", "--mode", "mcp"]
```

**brave-raccoon scripts/mcp-serve.sh**:
```bash
exec go run ./src/cmd/bridge --mode mcp "$@"
```

**brave-raccoon README.md** (all 3 config examples — Claude Code, Gemini, Codex):
```json
"args": ["run", "./src/cmd/bridge", "--mode", "mcp"]
```

**Impact**: When merged, `go run ./src/cmd/bridge --mode mcp` will start the HTTP server on :8080 (the default path in main.go), NOT the stdio transport. Every MCP client (Claude Code, Gemini CLI, Codex) that uses the documented config or `.mcp.json` will fail to communicate — they'll get HTTP output on stdout instead of JSON-RPC line protocol.

**Fix options** (pick one):
1. Change brave-raccoon: replace `--mode mcp` with `--stdio` in `.mcp.json`, `mcp-serve.sh`, and all README examples
2. Change silly-fox: replace `--stdio` with `--mode mcp` flag parsing in main.go
3. Change silly-fox: accept both `--stdio` and `--mode mcp` as aliases

Option 1 is simplest — one branch, text-only changes, no recompilation needed.

---

## Previous Finding: handlers.go time import — RESOLVED

The committed version of handlers.go (commit 19d52d9) correctly imports `"time"`. The earlier finding was from an uncommitted working state. No longer an issue.

## Additional Findings (Deep Analysis)

### Medium: HandleFilter hard-cap at 100 events
`handlers.go:HandleFilter` fetches `h.store.Latest(100)` and filters in memory. If >100 events exist, filter results will be incomplete. Documented as MVP limitation but should be noted for future work.

### Medium: HandleSummary same 100-event cap
`handlers.go:HandleSummary` also uses `h.store.Latest(100)` as its universe. A 60-minute window with >100 events will produce an inaccurate summary. Same MVP limitation.

### Low: smoke-test.sh tests HTTP mode, not MCP stdio
`smoke-test.sh` starts the bridge without `--stdio` (or `--mode mcp`), so it exercises the HTTP path only. It doesn't validate the MCP stdio transport that `.mcp.json` configures. Not a bug, but means the MCP interface contract has no automated test coverage beyond unit tests.

### Low: Arg parsing is fragile
`os.Args[1] == "--stdio"` only matches exact position 1. If any other flag precedes it (e.g., `--verbose --stdio`), stdio mode won't activate. Fine for MVP but worth noting.

### Positive Notes
- No merge conflicts between branches (verified via git merge-tree)
- server.go is well-structured: clean JSON-RPC 2.0 dispatch, proper error codes
- 17 tests with good coverage of dispatch, HTTP, stdio, and edge cases
- smoke-test.sh has proper cleanup, process management, graceful fallback
- Both workers stayed on task and on correct branches
