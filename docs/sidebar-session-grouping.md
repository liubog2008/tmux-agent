# Sidebar Session Grouping

Date: 2026-05-01

## Goal

Define how sessions should be grouped and displayed inside the tmux agent sidebar.

Target behavior:

- The main area of the right sidebar shows agent-related sessions or panes
- The lower area shows normal tmux sessions
- Normal sessions should preserve default tmux interaction patterns as much as possible

This document is about logical grouping inside the sidebar, not changing tmux's underlying session model.

## Conclusion

The sidebar content can be split into two display groups:

1. `agent sessions`
2. `normal sessions`

This is a UI-level grouping, not a requirement to maintain two real session types inside tmux.

Recommended layout:

- Upper section: `agent sessions`
- Lower section: `normal sessions`

## Grouping Rules

Grouping should prefer explicit markers first, with automatic inference as fallback.

Recommended classification order:

1. If a session explicitly sets `@session_kind`
   - Classify by `@session_kind=agent|normal`
2. Otherwise, if the session contains any pane with `@agent_role=agent`
   - Classify it as `agent`
3. Otherwise
   - Classify it as `normal`

This allows:

- Automatic detection of agent sessions
- Manual override when needed
- Future extension to more session types

## Why Not Change the tmux Session Model

It is not recommended to build two separate session systems at the tmux layer.

Reasons:

- tmux does not provide a native session type model that needs to be extended
- Logical grouping is enough for the UI requirement
- Splitting the underlying behavior would add complexity
- It would interfere with existing tmux workflows

So the correct approach is:

- Keep one tmux server and the default session system
- Apply logical grouping inside the sidebar

## Data Sources

### Agent Sessions Section

Data source:

- `tmux list-panes -a`
- `tmux pane options`
- runtime json

Detection rule:

- The pane has `@agent_role=agent`

The UI can aggregate at two levels:

1. Per agent pane
2. Per session

Recommended MVP:

- Show agent panes first
- Also display the owning session in the UI

This is simpler and allows direct jumping to the exact pane.

### Normal Sessions Section

Data source:

- `tmux list-sessions`

Display object:

- Standard tmux sessions

If a session has already been classified as `agent`, it should not also appear in the `normal sessions` section by default.

## Recommended Layout

Use a vertical split layout:

```text
+--------------------------------------+
| Agents                               |
| codex  fix auth bug       running    |
| claude refactor tui       waiting    |
| codex  write tests        success    |
|                                      |
| Sessions                             |
| dev-main                  4 windows  |
| ops                       2 windows  |
| notes                     1 window   |
+--------------------------------------+
```

Layout requirements:

- `Agents` is the primary section
- `Sessions` is the secondary section
- `Sessions` must keep a minimum height

Recommendation:

- `Sessions` minimum height: `5` to `8` lines
- When there are many agent items, `Agents` should scroll
- `Sessions` should also support independent scrolling

## Interaction Rules

### Agents Section

Recommended actions:

- `enter`: jump to the target pane
- `o`: open detail view
- `r`: refresh status

Jump logic:

- If the target pane is in the current window, call `select-pane`
- If it is in another window, switch window first and then `select-pane`
- If it is in another session, call `switch-client` first and then locate the pane

### Sessions Section

Recommended actions:

- `enter`: switch to the target tmux session
- `o`: call tmux native `choose-tree -s` or `choose-session`

Design principle:

- Deeper browsing of normal sessions should stay close to tmux-native behavior
- The lower section should only act as a lightweight entry point

## Focus Model

The sidebar should keep two independent focus areas:

1. `agents`
2. `sessions`

Suggested keys:

- `tab`: switch focus between the two sections
- `j/k` or arrow keys: move inside the current section
- `/`: filter the current section

Optional keys:

- `a`: show only `agents`
- `s`: show only `sessions`
- `b`: return to the default grouped layout

## Classification Details

### Option A: Infer from panes

Rule:

- If a session contains any pane with `@agent_role=agent`
- The session is treated as an `agent session`

Pros:

- Highly automatic
- No extra user configuration required

Cons:

- A mixed session with both agent panes and normal panes will be classified as `agent`

### Option B: Explicit session option

Rule:

```tmux
set -t <session> @session_kind agent
set -t <session> @session_kind normal
```

Pros:

- Predictable behavior
- Easy to control manually

Cons:

- Requires explicit configuration

### Recommended Option: Hybrid mode

Priority:

1. Read `@session_kind` first
2. If it is not set, infer from panes

This is the recommended final behavior.

## Duplicate Display Policy

Default policy:

- A session classified as `agent` should not also appear in `normal sessions`

Reasons:

- Avoid showing the same session in both sections
- Reduce visual noise

Possible extension:

- Allow a config option to show all sessions in the normal section as well

That should not be the default.

## Refresh Strategy

The refresh logic for the two sections should be decoupled.

### Agents Section

- Default poll interval: 1 second
- State changes are frequent

### Sessions Section

- Default poll interval: 3 to 5 seconds
- Or refresh together with `Agents`, but at lower priority

This reduces total refresh overhead.

## Recommended tmux Queries

### Query agent panes

```bash
tmux list-panes -a -F '#{session_id}|#{session_name}|#{window_id}|#{window_name}|#{pane_id}|#{pane_current_path}|#{@agent_role}|#{@agent_source}|#{@agent_runtime_key}|#{@agent_status}|#{@agent_title}'
```

### Query sessions

```bash
tmux list-sessions -F '#{session_id}|#{session_name}|#{session_windows}|#{session_attached}|#{session_last_attached}|#{@session_kind}'
```

The sidebar should first collect sessions and then combine them with pane data to determine grouping.

## Suggested Data Model

On the Go side, keep two view models.

### AgentItem

```go
type AgentItem struct {
    RuntimeKey  string
    Source      string
    SessionID   string
    SessionName string
    WindowID    string
    WindowName  string
    PaneID      string
    Status      string
    Title       string
    Repo        string
}
```

### SessionItem

```go
type SessionItem struct {
    SessionID      string
    SessionName    string
    WindowCount    int
    Attached       bool
    SessionKind    string
    ContainsAgent  bool
}
```

## Edge Cases

### One session contains both agent panes and normal panes

Recommended handling:

- Classify the session as `agent`
- Do not separately show its normal panes in `normal sessions`

This keeps the UI simpler.

### No agent sessions exist

Recommended handling:

- Show an empty state in `Agents`
- Show all normal sessions in `Sessions`

### All sessions are agent sessions

Recommended handling:

- Show an empty state in `Sessions`
- Keep `Agents` as the primary functional section

## MVP Recommendation

For the first version, implement only:

1. Upper section shows agent panes
2. Lower section shows normal sessions
3. Use the hybrid classification rule
4. `enter` performs different actions in the two sections
5. `tab` switches focus

Do not implement in the MVP:

- Multi-level tree session structures
- Complex expand/collapse behavior between sessions and panes
- Drag-to-resize or dynamic section weighting

## Final Recommendation

The recommended solution is:

- Use a single tmux server
- Apply logical grouping inside one sidebar
- Show `agent sessions` on top
- Show `normal sessions` below
- Prefer `@session_kind` when present, otherwise infer from `@agent_role=agent`
- Keep `normal sessions` behavior close to native tmux behavior

This meets the goal of surfacing agent activity while preserving the normal tmux workflow.
