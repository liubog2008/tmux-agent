#!/usr/bin/env bash
set -euo pipefail

pane_id="$1"
tmux kill-pane -t "$pane_id"
