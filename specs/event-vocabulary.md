# Event Vocabulary

Standard event types for the ws-mcp bridge. Events use dotted namespace convention: `<domain>.<action>`.

## Task Events

### `task.started`
Emitted when a worker agent begins executing a task.

| Field | Type | Description |
|-------|------|-------------|
| `task_id` | string | Unique task identifier |
| `agent` | string | Name of the agent executing the task |
| `description` | string | Human-readable task description |

**Typical source:** `multiclaude`, `ralph`

### `task.completed`
Emitted when a task finishes successfully.

| Field | Type | Description |
|-------|------|-------------|
| `task_id` | string | Unique task identifier |
| `agent` | string | Name of the agent that completed the task |
| `summary` | string | Completion summary |

**Typical source:** `multiclaude`, `ralph`

### `task.failed`
Emitted when a task fails.

| Field | Type | Description |
|-------|------|-------------|
| `task_id` | string | Unique task identifier |
| `agent` | string | Name of the agent |
| `reason` | string | Failure reason |

**Typical source:** `multiclaude`, `ralph`

## Git Events

### `commit.pushed`
Emitted when commits are pushed to a remote.

| Field | Type | Description |
|-------|------|-------------|
| `branch` | string | Branch name |
| `sha` | string | Commit SHA |
| `message` | string | Commit message |
| `author` | string | Commit author |

**Typical source:** `multiclaude`, `ralph`

## Pull Request Events

### `pr.opened`
Emitted when a pull request is created.

| Field | Type | Description |
|-------|------|-------------|
| `pr_number` | int | PR number |
| `title` | string | PR title |
| `branch` | string | Source branch |
| `url` | string | PR URL |

**Typical source:** `multiclaude`

### `pr.merged`
Emitted when a pull request is merged.

| Field | Type | Description |
|-------|------|-------------|
| `pr_number` | int | PR number |
| `merge_sha` | string | Merge commit SHA |
| `merged_by` | string | Who/what merged it |

**Typical source:** `multiclaude`

### `pr.reviewed`
Emitted when a pull request review is submitted.

| Field | Type | Description |
|-------|------|-------------|
| `pr_number` | int | PR number |
| `reviewer` | string | Reviewer name/agent |
| `verdict` | string | `approved`, `changes_requested`, or `commented` |

**Typical source:** `multiclaude`

## Review Events

### `review.requested`
Emitted when a review is requested for a PR or artifact.

| Field | Type | Description |
|-------|------|-------------|
| `pr_number` | int | PR number |
| `reviewer` | string | Requested reviewer |

**Typical source:** `multiclaude`

### `review.completed`
Emitted when a review process finishes.

| Field | Type | Description |
|-------|------|-------------|
| `pr_number` | int | PR number |
| `reviewer` | string | Reviewer |
| `result` | string | Review outcome |

**Typical source:** `multiclaude`

## Agent Lifecycle Events

### `agent.started`
Emitted when an agent process starts.

| Field | Type | Description |
|-------|------|-------------|
| `agent` | string | Agent name |
| `worktree` | string | Worktree path (if applicable) |

**Typical source:** `multiclaude`, `ralph`

### `agent.stopped`
Emitted when an agent process stops.

| Field | Type | Description |
|-------|------|-------------|
| `agent` | string | Agent name |
| `reason` | string | `completed`, `failed`, `killed` |

**Typical source:** `multiclaude`, `ralph`

## System Events

### `system.healthcheck`
Periodic heartbeat indicating a component is alive.

| Field | Type | Description |
|-------|------|-------------|
| `component` | string | Component name |
| `status` | string | `ok` or `degraded` |

**Typical source:** `system`

### `system.error`
Emitted when a system-level error occurs.

| Field | Type | Description |
|-------|------|-------------|
| `component` | string | Component that errored |
| `error` | string | Error message |
| `severity` | string | `warning`, `error`, `critical` |

**Typical source:** `system`

## Extensibility

Unknown event types are accepted by the ingest endpoint but logged as warnings. This allows forward-compatible evolution — new event types can be introduced by agents without requiring bridge updates.
