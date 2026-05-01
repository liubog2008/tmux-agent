# Agent State Protocol

Date: 2026-05-01

## Goal

Define a shared protocol that allows the tmux sidebar to consume runtime state from both `Codex` and `Claude`.

This protocol solves three things:

1. Identify which tmux pane is an agent pane
2. Let different agents report state through one shared format
3. Let the sidebar read state reliably instead of parsing terminal output

This protocol does not define:

- The agent's internal task model
- Log stream formats
- Synchronization with remote services

## Design Principles

- Separate pane discovery from detail retrieval
- Keep the sidebar read-only
- Treat tmux pane existence as the source of truth for liveness
- Let `Codex hooks` and `Claude hooks` share the same write path
- Start with an MVP and leave room for richer metadata later

## Protocol Layers

The protocol has two layers:

1. `tmux pane option`
   - Lightweight index layer
   - Used for quick discovery of agent panes and basic status
2. `runtime json`
   - Detail layer
   - Used for sidebar rendering and future expansion

## tmux Pane Option Protocol

Each agent pane must write the following tmux user options:

```tmux
@agent_role=agent
@agent_source=codex|claude
@agent_runtime_key=<string>
@agent_status=starting|running|waiting_input|idle|success|error|stale
@agent_updated_at=<rfc3339>
```

Recommended fields:

```tmux
@agent_title=<short text>
@agent_session_id=<string>
@agent_parent_id=<string>
@agent_repo=<abs path>
```

Constraints:

- `@agent_role=agent` is the single hard marker used by the sidebar to identify agent panes
- `@agent_runtime_key` must be globally unique for one runtime instance
- `@agent_status` must use the fixed enum values
- `@agent_updated_at` must use RFC3339 format

## Runtime JSON Protocol

Default path:

```text
$XDG_RUNTIME_DIR/tmux-agent-sidebar/<runtime_key>.json
```

If `XDG_RUNTIME_DIR` is not set, fall back to:

```text
/tmp/tmux-agent-sidebar/<runtime_key>.json
```

JSON shape:

```json
{
  "version": 1,
  "runtime_key": "agent-7f3c",
  "source": "codex",
  "pane_id": "%12",
  "tmux_session": "$0",
  "tmux_window": "@8",
  "tmux_window_name": "editor",
  "repo": "/home/no68/data/src/project",
  "cwd": "/home/no68/data/src/project",
  "status": "running",
  "title": "fix auth middleware",
  "task_summary": "editing auth retry flow",
  "session_id": "sess_123",
  "parent_agent_id": "",
  "model": "gpt-5.5",
  "started_at": "2026-05-01T22:00:00+08:00",
  "updated_at": "2026-05-01T22:10:00+08:00",
  "ended_at": "",
  "last_error": "",
  "metadata": {
    "tool_name": "Edit",
    "branch": "feature/auth-retry"
  }
}
```

Minimum required fields:

- `version`
- `runtime_key`
- `source`
- `pane_id`
- `status`
- `started_at`
- `updated_at`

## Status Enum

The protocol defines these fixed status values:

- `starting`
- `running`
- `waiting_input`
- `idle`
- `success`
- `error`
- `stale`

Meaning:

- `starting`: the agent has just started and is not yet in a stable working state
- `running`: the agent is actively working, or there has been recent activity
- `waiting_input`: the agent is explicitly waiting for user input, confirmation, or authorization
- `idle`: the process is still alive, but no activity has been observed for a while
- `success`: the agent finished normally
- `error`: the agent finished abnormally, or a hook explicitly reported an error
- `stale`: the tmux pane no longer exists, but state has not been cleaned up yet

## State Writer Command Protocol

Define a shared local command:

```bash
tmux-agent-state start ...
tmux-agent-state update ...
tmux-agent-state finish ...
tmux-agent-state cleanup ...
```

This command is the only allowed writer for:

- tmux pane options
- runtime json

The sidebar does not write state.

### start

Example:

```bash
tmux-agent-state start \
  --source codex \
  --pane-id %12 \
  --runtime-key codex-1746117600123-48291-a91c \
  --repo /abs/repo \
  --cwd /abs/repo \
  --title "fix auth bug" \
  --session-id sess_123 \
  --model gpt-5.5
```

Behavior:

- Create the runtime JSON
- Write the required tmux pane options
- Set the initial status to `starting`
- Write `started_at` and `updated_at`

### update

Example:

```bash
tmux-agent-state update \
  --runtime-key codex-1746117600123-48291-a91c \
  --status running \
  --title "fix auth bug" \
  --task-summary "editing middleware" \
  --metadata tool_name=Edit
```

Behavior:

- Update the runtime JSON
- Update `@agent_status` in pane options
- Update `@agent_updated_at` in pane options
- If `title` is provided, update `@agent_title`

### finish

Example:

```bash
tmux-agent-state finish \
  --runtime-key codex-1746117600123-48291-a91c \
  --status success
```

Or:

```bash
tmux-agent-state finish \
  --runtime-key codex-1746117600123-48291-a91c \
  --status error \
  --last-error "permission denied"
```

Behavior:

- Write the final status
- Write `ended_at`
- Update `updated_at`

### cleanup

Example:

```bash
tmux-agent-state cleanup --runtime-key codex-1746117600123-48291-a91c
```

Behavior:

- Delete the runtime JSON
- Clear pane options, or keep the final state briefly before cleanup depending on implementation

## Event Mapping

### Codex

Recommended mapping:

- `SessionStart` -> `start(status=starting)`
- `PreToolUse` -> `update(status=running)`
- `PostToolUse` -> `update(status=running)`
- `UserPromptSubmit` -> `update(status=running)`
- `Stop` -> `finish(status=success)`

If Codex exposes explicit events for confirmation, permissions, or user input, map them to:

- `waiting_input`

If `Stop` or another event includes failure details, abnormal exit should map to:

- `finish(status=error)`

### Claude

Recommended mapping:

- `SessionStart` -> `start(status=starting)`
- `PreToolUse` -> `update(status=running)`
- `PostToolUse` -> `update(status=running)`
- `Notification` -> `update(status=waiting_input)`
- `Stop` -> `finish(status=success)`
- `SessionEnd` with abnormal termination -> `finish(status=error)`

## Sidebar Read Protocol

Each sidebar refresh runs in two steps.

### Step 1: Bulk-read tmux pane indexes

Recommended command:

```bash
tmux list-panes -a -F '#{pane_id}|#{session_name}|#{window_name}|#{pane_current_path}|#{@agent_role}|#{@agent_source}|#{@agent_runtime_key}|#{@agent_status}|#{@agent_updated_at}|#{@agent_title}'
```

Only panes with `@agent_role=agent` should be treated as agent panes.

### Step 2: Read detail JSON by runtime key

For each agent pane:

- Read the matching runtime JSON
- Let JSON fields override the basic pane option values

Merge rules:

- Prefer runtime JSON
- Fall back to tmux pane options if JSON is missing
- If the pane no longer exists but JSON remains, mark it as `stale`

## Consistency Rules

These rules must hold:

- `runtime_key` must not be reused during a single agent lifecycle
- `runtime_key` is the primary key for the detail layer
- `pane_id` is used to target the tmux pane, but is not a cross-lifecycle unique key
- Every state change must update `updated_at`
- The sidebar treats tmux pane existence as the source of truth
- Runtime JSON provides detail, not liveness

## Runtime Key Format

Recommended format:

```text
<source>-<unixms>-<pid>-<rand>
```

Example:

```text
codex-1746117600123-48291-a91c
```

Requirements:

- Unique enough for concurrent local runs
- No dependency on an external generator
- Easy to identify the source from the prefix

## Cleanup Strategy

Recommended behavior:

- Keep final state briefly after normal exit for display purposes
- During sidebar refresh, if the pane no longer exists and the JSON has not been updated for a while, delete it
- Let `cleanup` be callable by hooks or a background cleanup task

The stale timeout should be configurable.

## MVP Minimum Protocol

For the first version, only these fields are required.

### tmux pane options

```tmux
@agent_role
@agent_source
@agent_runtime_key
@agent_status
@agent_updated_at
@agent_title
```

### runtime json

```json
{
  "version": 1,
  "runtime_key": "codex-1746117600123-48291-a91c",
  "source": "codex",
  "pane_id": "%12",
  "status": "running",
  "title": "fix auth bug",
  "task_summary": "editing middleware",
  "started_at": "2026-05-01T22:00:00+08:00",
  "updated_at": "2026-05-01T22:10:00+08:00"
}
```

This is already enough to support:

- The right sidebar list
- Status colors
- Pane jumping
- Filtering by source `codex/claude`
- Stale cleanup
