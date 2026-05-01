# tmux-agent MVP

This minimal MVP provides three main pieces:

- `agent-sidebar.tmux`
- `tmux-agent-state`
- `sidebar`

## Build

```bash
./scripts/build.sh
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

## Manually write test state

Run this inside a tmux pane:

```bash
export TMUX_AGENT_RUNTIME_KEY="codex-$(date +%s)-$$"
./bin/tmux-agent-state start \
  --source codex \
  --runtime-key "$TMUX_AGENT_RUNTIME_KEY" \
  --title "demo codex task" \
  --status running
```

Then run:

```bash
./bin/tmux-agent-state update \
  --runtime-key "$TMUX_AGENT_RUNTIME_KEY" \
  --status waiting_input \
  --title "waiting for review"
```

The sidebar will show the state in the `Agents` section.

## Hook examples

Example scripts live in:

- `examples/hooks/codex/`
- `examples/hooks/claude/`

These scripts call the shared `tmux-agent-state` command and write state into tmux pane options and runtime JSON.

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
