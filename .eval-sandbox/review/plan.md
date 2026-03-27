# Review Plan: silly-fox + brave-raccoon Workers

## Step 1: Primary Pass (current)
- Identify scope, review in-progress code from both workers
- Flag cross-worker dependency risks
- Check spec conformance

## Step 2: Deep Analysis — Cross-Worker Interface Contract
- Once both workers have committed, verify --mode mcp / stdio transport interface matches
- Run go test ./... in both worktrees after commits
- Check JSON-RPC dispatch wiring matches spec methods
- Verify smoke-test.sh can exercise the bridge end-to-end

## Final: Synthesis and Completion
- Consolidate findings, assess merge readiness
- Emit REVIEW_COMPLETE or REQUEST_CHANGES
