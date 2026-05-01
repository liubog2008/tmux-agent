#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$(tmux show-option -gv @agent-sidebar-bin)"
WIDTH="$(tmux show-option -gv @agent-sidebar-width)"
SIDE="$(tmux show-option -gv @agent-sidebar-side)"

if [[ ! -x "$BIN" ]]; then
  "$ROOT_DIR/scripts/build.sh"
fi

split_args=(-d -l "$WIDTH" -P -F "#{pane_id}")
if [[ "$SIDE" == "left" ]]; then
  split_args=(-hb "${split_args[@]}")
else
  split_args=(-h "${split_args[@]}")
fi

pane_id="$(tmux split-window "${split_args[@]}" "$BIN")"
tmux set-option -p -t "$pane_id" @agent_sidebar_role sidebar
tmux select-pane -t "$pane_id" -T "tmux-agent-sidebar"
