# tmux-agent MVP

This minimal MVP provides three main pieces:

- `agent-sidebar.tmux`
- `tmux-agent`

## Build

```bash
make
```

This produces:

```bash
./bin/tmux-agent
```

The existing build script still works:

```bash
./scripts/build.sh
```

## Install

```bash
make install
```

By default this installs:

```bash
/usr/local/bin/tmux-agent
/usr/local/share/tmux-agent/agent-sidebar.tmux
```

You can override the install prefix if needed:

```bash
make install PREFIX="$HOME/.local"
```

## Load in tmux

Add this to `~/.tmux.conf`:

```tmux
source-file /usr/local/share/tmux-agent/agent-sidebar.tmux
```

Then reload your tmux config:

```bash
tmux source-file ~/.tmux.conf
```

Default key binding:

- `prefix + A`: open or close the right sidebar
- `prefix + N`: create a new `agent` window and register it in the sidebar

The tmux plugin defaults `@agent-sidebar-bin` to `tmux-agent`, so make sure the installed binary is in your `PATH`, or override it in `~/.tmux.conf`:

```tmux
set -g @agent-sidebar-bin "/absolute/path/to/bin/tmux-agent"
```

The plugin also installs tmux hooks so that:

- opening the sidebar immediately focuses it
- moving focus away from the sidebar automatically closes it

This behavior is implemented with tmux `focus-events` and the `pane-focus-out` hook, which calls `tmux-agent close --pane-id #{hook_pane}`.

## Manually write test state

Run this inside a tmux pane:

```bash
export TMUX_AGENT_RUNTIME_KEY="codex-$(date +%s)-$$"
./bin/tmux-agent prepare \
  --source codex \
  --runtime-key "$TMUX_AGENT_RUNTIME_KEY" \
  --title "new codex window"

./bin/tmux-agent start \
  --source codex \
  --runtime-key "$TMUX_AGENT_RUNTIME_KEY" \
  --title "demo codex task" \
  --status running
```

Then run:

```bash
./bin/tmux-agent update \
  --runtime-key "$TMUX_AGENT_RUNTIME_KEY" \
  --status waiting_input \
  --title "waiting for review"
```

`prepare` can be called as soon as a new tmux window is created so the pane is immediately classified as an agent pane in the sidebar. `start` then promotes the same runtime to `running` when the real agent process begins.

The sidebar will show the state in the `Agents` section.

## Hook examples

Example scripts live in:

- `examples/hooks/codex/`
- `examples/hooks/claude/`

These scripts call the shared `tmux-agent` binary and write state into tmux pane options and runtime JSON.

## Notes

This is still a minimal MVP:

- It supports a right sidebar
- It shows both agent items and normal sessions
- It supports `Codex` and `Claude` as sources
- Hook config files still need to be wired into your local CLI setup

Not implemented yet:

- Rich detail views
- Fine-grained hook payload parsing
- TPM auto-install support
