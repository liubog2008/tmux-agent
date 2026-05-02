#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
STATE_BIN="${TMUX_AGENT_STATE_BIN:-${TMUX_AGENT_BIN:-$ROOT_DIR/bin/tmux-agent}}"
RUNTIME_KEY="${TMUX_AGENT_RUNTIME_KEY:-codex-$(date +%s)-$$}"

"$STATE_BIN" start \
  --source codex \
  --runtime-key "$RUNTIME_KEY" \
  --pane-id "${TMUX_PANE:-}" \
  --cwd "${PWD}" \
  --repo "${PWD}" \
  --title "${CODEX_TASK_TITLE:-codex session}" \
  --status running \
  --model "${CODEX_MODEL:-}"

echo "$RUNTIME_KEY" > "${TMPDIR:-/tmp}/tmux-agent-runtime-key"
