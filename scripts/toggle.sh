#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

existing="$(tmux list-panes -F '#{pane_id}|#{@agent_sidebar_role}' | awk -F'|' '$2=="sidebar"{print $1; exit}')"

if [[ -n "$existing" ]]; then
  "$ROOT_DIR/scripts/close.sh" "$existing"
else
  "$ROOT_DIR/scripts/open.sh"
fi
