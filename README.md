# tmux-agent MVP

This minimal MVP provides three main pieces:

- `agent-sidebar.tmux`
- `tmux-agent`

## Build

```bash
./scripts/build.sh
```

This produces:

```bash
./bin/tmux-agent
```

## Load in tmux

Add this to `~/.tmux.conf`:

```tmux
source-file /path/to/tmux-agent/agent-sidebar.tmux
```

Then reload your tmux config:

```bash
tmux source-file ~/.tmux.conf
```

Default key binding:

- `prefix + A`: open or close the right sidebar

The tmux plugin defaults `@agent-sidebar-bin` to `tmux-agent`, so either add `./bin` to your `PATH` or override it in `~/.tmux.conf`:

```tmux
set -g @agent-sidebar-bin "/absolute/path/to/bin/tmux-agent"
```

## Manually write test state

Run this inside a tmux pane:

```bash
export TMUX_AGENT_RUNTIME_KEY="codex-$(date +%s)-$$"
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
