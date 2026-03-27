# Deep Analysis: bright-hawk Worker Setup Failure

## Root Cause: Missing base-branch configuration

The bright-hawk worker's `work/bright-hawk` branch is based on `cd99eba` (origin/main), not `1ce0aba` (001-websocket-mcp-bridge). The multiclaude state.json shows:

- **No `branch` field** for bright-hawk (other workers in the same state file do have one)
- **No project-level `base_branch`** configured for ws-mcp
- The multiclaude repo's default checked-out branch is `001-websocket-mcp-bridge`, but `git worktree add` without an explicit start-point defaults to HEAD of the repo, which was apparently `origin/main` at worktree creation time

**Conclusion:** multiclaude created the worktree without specifying `--branch 001-websocket-mcp-bridge` or equivalent. The worker started on a bare initial commit with no Go source code, making all 4 assigned tasks impossible to complete.

## Evidence

1. `git log work/bright-hawk` shows only `cd99eba Initial commit from Specify template`
2. `git merge-base work/bright-hawk 001-websocket-mcp-bridge` = `cd99eba` (diverged at root)
3. Worktree contents: only `.gemini/` and `.specify/` directories, zero Go source
4. state.json bright-hawk entry has no `branch` field (contrast with other workers that do)

## Recurrence Risk: HIGH

This will recur on any retry unless the worker is spawned with an explicit `--branch 001-websocket-mcp-bridge` or `--push-to` flag. The default remote HEAD is `001-websocket-mcp-bridge`, but the worktree creation did not honor it.

## Feature Branch Status (all 4 tasks still broken)

Verified on `001-websocket-mcp-bridge` (1ce0aba):

| Task | Status | Detail |
|------|--------|--------|
| go.mod: add gorilla/websocket | BROKEN | `go.mod` has no websocket dep; hub.go fails to compile |
| HandleFilter syntax error | BROKEN | handlers.go:37 — missing return + closing brace before HandleAck |
| main.go wiring | BROKEN | `src/cmd/bridge/` exists but no main.go wiring RingBuffer/Hub/Router |
| go test ./... green | BROKEN | 3 packages fail (hub, api, mcp); only store passes |

## Recommendation

1. Delete bright-hawk worktree and respawn with explicit base branch: `multiclaude worker add bright-hawk --branch 001-websocket-mcp-bridge`
2. Or fix inline: `cd worktree && git reset --hard 001-websocket-mcp-bridge`
