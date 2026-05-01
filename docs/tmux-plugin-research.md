# tmux Agent Sidebar Research

Date: 2026-05-01

## Goal

Build a tmux plugin installable through TPM that shows running agent sessions in a right-hand sidebar.

Constraints:

- Use Go as the implementation language
- Use Bubble Tea for the TUI
- The sidebar should be persistent, not a one-off popup
- The data source should cover multiple tmux sessions, windows, and panes
- Interaction should stay inside tmux as much as possible

## Conclusion

The recommended approach is a two-layer architecture:

1. tmux plugin layer
   - Handles installation, key bindings, and creating or closing the sidebar pane
   - Passes pane, session, and window context into the Go binary
   - Stores minimal tmux-side metadata such as `@agent_sidebar_*`
2. Go/Bubble Tea sidebar process
   - Renders the agent list, status, filters, and details
   - Polls tmux pane data and agent state
   - Maps user actions to `tmux select-pane`, `switch-client`, `display-popup`, or `run-shell`

This is a better fit than a pure shell plugin for non-trivial state and interaction, and easier to maintain than storing everything in tmux options.

## Why Use a Pane Instead of a Popup

If the requirement is a real sidebar, the main implementation should be based on `split-window`, not `display-popup`.

Reasons:

- A pane is a first-class tmux layout element and works well for a persistent UI
- A pane can stay visible alongside the main work area
- A pane naturally supports `select-pane`, `resize-pane`, and `kill-pane`
- A popup is better for details, filters, confirmations, or help

Recommendation:

- Main view: fixed right sidebar pane
- Secondary interaction: popup for details, help, or confirmation when needed

## Recommended Minimum Feature Set

The MVP should only do these six things:

1. Toggle the right sidebar with `prefix + A`
2. Show a global agent list
3. Show status such as `running / waiting / idle / error`
4. Jump to the target pane from a selected item
5. Filter by session or repo
6. Refresh with a 1-second polling loop

Do not put these into the first version:

- Complex live log tailing
- Multi-level subagent trees
- Automatic worktree creation and cleanup
- Desktop notifications
- Too many tmux hooks

These can be added later, but should not block the first usable version.

## Architecture Recommendation

### 1. Plugin directory layout

Suggested layout:

```text
tmux-agent-sidebar/
  agent-sidebar.tmux
  scripts/
    toggle.sh
    open.sh
    close.sh
    focus.sh
    env.sh
  cmd/
    sidebar/
      main.go
  internal/
    app/
    tmux/
    state/
    agent/
    ui/
  bin/
    tmux-agent-sidebar
```

Notes:

- `agent-sidebar.tmux` is the TPM entry point
- `scripts/*.sh` should only contain tmux glue logic
- `cmd/sidebar` is the executable entry point
- `internal/tmux` owns tmux command execution and format parsing
- `internal/agent` reads agent state
- `internal/ui` contains the Bubble Tea model and view

### 2. tmux plugin layer responsibilities

`agent-sidebar.tmux` should:

- Define key bindings
- Read user configuration
- Call `scripts/toggle.sh`
- Tag the sidebar pane, for example with `@agent_sidebar_pane=1`

Suggested user options:

```tmux
set -g @agent-sidebar-width '42'
set -g @agent-sidebar-side 'right'
set -g @agent-sidebar-refresh-ms '1000'
set -g @agent-sidebar-key 'A'
set -g @agent-sidebar-bin '~/.tmux/plugins/tmux-agent-sidebar/bin/tmux-agent-sidebar'
```

Core `toggle.sh` logic:

1. Check whether the current window already has a sidebar pane
2. If it exists, close it or focus it
3. If it does not exist, launch the Go program with `split-window -h -d -l "$width"`
4. Write pane options to the new pane so it can be identified later

Recommended identification:

- Pane option: `@agent_sidebar_role=sidebar`
- Pane title: `tmux-agent-sidebar`

Do not rely only on pane width; that is too fragile.

### 3. Go sidebar process responsibilities

The sidebar binary should read from three input classes:

1. tmux pane metadata
   - `tmux list-panes -a`
   - session, window, pane id, title, path, active state
2. agent state metadata
   - Prefer a shared state file or Unix socket
   - Fall back to tmux pane options
3. local runtime state
   - Current filter
   - Current selection
   - Error state

Suggested Bubble Tea model:

```go
type Model struct {
    panes        []AgentPane
    filtered     []AgentPane
    selected     int
    filter       Filter
    width        int
    height       int
    loading      bool
    err          error
}
```

Recommended libraries:

- `bubbletea`
- `bubbles/list` or a custom list
- `bubbles/viewport` for the detail area
- `lipgloss` for status colors and layout

## State Source Design

This is the most important part of the whole solution.

### Option A: Read from tmux pane options

For example, each agent pane writes:

- `@agent_type=codex`
- `@agent_status=running`
- `@agent_task=fix auth bug`
- `@agent_updated_at=...`

Pros:

- Naturally aligned with tmux
- Easy to query across sessions
- No extra daemon needed

Cons:

- Too many fields become brittle
- Large text or logs do not belong in tmux options
- High-frequency updates create extra tmux command overhead

### Option B: Read from external state files

For example:

- `/tmp/tmux-agent-sidebar/<pane_id>.json`
- or `$XDG_RUNTIME_DIR/tmux-agent-sidebar/<pane_id>.json`

Pros:

- Structured and easy to extend
- Can hold more fields
- Better fit for integration with agent hooks

Cons:

- Needs stale file cleanup
- Must keep pane lifecycle and state files in sync

### Recommended approach

Use a hybrid model:

- Store index fields in tmux pane options
  - `@agent_type`
  - `@agent_status`
  - `@agent_runtime_key`
- Store detailed state in JSON files

This lets the sidebar quickly discover agent panes from a single `list-panes -a` call and then load detail by `runtime_key`.

This layering is more robust than “all tmux” and easier to reason about than a separate detached index.

## Refresh Strategy

Default to a full refresh every second:

1. Run `tmux list-panes -a -F ...`
2. Build snapshots for all agent panes
3. Read the matching JSON state files in batch
4. Update the Bubble Tea model

Reasons:

- It is the simplest correct implementation
- The overhead is acceptable for dozens of panes
- It is easier to make correct than a fully event-driven design

Possible optimizations later:

- Use tmux hooks to trigger immediate refresh
- Use `fsnotify` to watch state file changes
- Move repo-derived information into background workers

## tmux Query Guidance

The key rule is: do not call tmux once per pane.

Use one bulk query such as:

```bash
tmux list-panes -a -F '#{session_id}|#{session_name}|#{window_id}|#{window_index}|#{pane_id}|#{pane_title}|#{pane_current_path}|#{pane_active}|#{@agent_type}|#{@agent_status}|#{@agent_runtime_key}|#{@agent_sidebar_role}'
```

This keeps Go-side parsing simple and avoids per-pane overhead.

Avoid:

- One `show-options` call per pane
- One `display-message` call per field

That will become slow as the pane count grows.

## Interaction Recommendation

Keep Bubble Tea key bindings restrained:

- `j/k` or `up/down`: move
- `enter`: jump to the selected pane
- `f`: switch filter
- `r`: force refresh
- `q`: close the sidebar
- `o`: open detail in a popup

Action mapping:

- Jump to pane: `tmux select-pane -t <pane_id>`
- Jump across windows: `tmux select-window -t <window_id>` then `select-pane`
- Close sidebar: `tmux kill-pane -t $TMUX_PANE`

Notes:

- If the sidebar runs in its own pane, `enter` should hand focus back to the target work pane
- The sidebar should not take over tmux's full key table

## Bubble Tea Integration Notes

Bubble Tea can run directly inside a tmux pane without structural issues.

Practical recommendations:

- Do not use the alt screen by default for a persistent pane
- Start simple with `tea.NewProgram(model)`
- Only consider `ReleaseTerminal` and `RestoreTerminal` later if the UI needs to temporarily give up terminal control

A sidebar is an embedded TUI, not a fullscreen application. Simplicity matters more than visual effects.

## Version Recommendation

Suggested minimum versions:

- Go 1.23+
- tmux 3.2+

Reasons:

- tmux 3.2+ gives better popup support for future expansion
- tmux 3.x is common enough now to avoid excessive compatibility work

If the implementation uses only split panes, it could support older versions in theory, but the extra compatibility cost is usually not worth it.

## Relation to Existing Ecosystem

Existing examples show the direction is valid:

- `tmux-plugins/tmux-sidebar` proves a persistent pane-based sidebar fits tmux workflows
- `hiroppy/tmux-agent-sidebar` shows that aggregating agent state across sessions is a real use case

This proposal differs in that:

- It is an agent monitoring panel, not a directory tree
- It is implemented in Go + Bubble Tea, not pure shell
- The data model is built around panes and agent runtime state

## Risk Areas

### 1. State consistency

The pane may be gone while JSON remains, or JSON may be newer than pane metadata.

Mitigation:

- Treat pane existence in tmux as the truth
- Clean up state files for panes that no longer exist
- Always write `updated_at`

### 2. Performance degradation

Too many tmux calls or too much per-frame file reading will make the sidebar feel slow.

Mitigation:

- Query tmux in bulk
- Load details on demand where possible
- Move expensive derived fields off the hot path

### 3. Focus and layout disruption

Users care more about the sidebar not breaking their layout than about extra features.

Mitigation:

- Default to `split-window -h -d`
- Do not steal focus by default
- Reuse an existing sidebar pane when possible

### 4. Installation complexity

The plugin entry point is shell, but the main logic is in Go, so binary distribution must be handled.

Recommended rollout:

1. Development phase: local `go build`, plugin scripts call the repo-local binary
2. Release phase:
   - Publish binaries through GitHub Releases
   - Download the correct binary on first plugin run
   - Fall back to local `go build` if download fails

## Recommended Development Order

### Phase 1

- Create the TPM plugin skeleton
- Implement `toggle/open/close`
- Run a Bubble Tea sidebar with mock data

### Phase 2

- Implement batch `list-panes -a`
- Define the `AgentPane` data model
- Support pane jumping

### Phase 3

- Integrate runtime state files
- Implement status filtering and refresh
- Handle stale data cleanup

### Phase 4

- Add popup detail
- Add repo grouping
- Add install and release scripts

## Final Recommendation

Choose this combination and stop debating popup versus pure shell:

- UI container: tmux right pane
- Plugin distribution: TPM
- Interaction rendering: Go + Bubble Tea
- Pane discovery: bulk `tmux list-panes -a`
- State storage: hybrid `tmux pane option + JSON runtime file`
- Refresh strategy: start with 1-second polling, add hooks or file watching later

Advantages:

- Fits tmux naturally
- Keeps the MVP cost under control
- Scales toward popup detail, notifications, worktrees, and logs

Trade-offs:

- Binary distribution must be handled
- The agent state protocol must be defined
- This is not a trivial “few lines in tmux.conf” plugin

If the goal is to continuously show and control running agent sessions, this is the most balanced approach between cost and maintainability.

## References

- tmux Plugin Manager: https://github.com/tmux-plugins/tpm
- tmux Sidebar: https://github.com/tmux-plugins/tmux-sidebar
- tmux Wiki Recipes: https://github.com/tmux/tmux/wiki/Recipes
- tmux CHANGES: https://github.com/tmux/tmux/blob/master/CHANGES?plain=1
- Bubble Tea package docs: https://pkg.go.dev/github.com/charmbracelet/bubbletea
- Bubbles components: https://github.com/charmbracelet/bubbles
- Reference competitor: https://hiroppy.github.io/tmux-agent-sidebar/
