#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
STATE_BIN="${TMUX_AGENT_STATE_BIN:-${TMUX_AGENT_BIN:-$ROOT_DIR/bin/tmux-agent}}"
RUNTIME_KEY="${TMUX_AGENT_RUNTIME_KEY:-claude-$(date +%s)-$$}"

"$STATE_BIN" start \
  --source claude \
  --runtime-key "$RUNTIME_KEY" \
  --pane-id "${TMUX_PANE:-}" \
  --cwd "${PWD}" \
  --repo "${PWD}" \
  --title "${CLAUDE_TASK_TITLE:-claude session}" \
  --status running \
  --model "${CLAUDE_MODEL:-}"

echo "$RUNTIME_KEY" > "${TMPDIR:-/tmp}/tmux-agent-runtime-key"
