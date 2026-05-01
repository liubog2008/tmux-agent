#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
STATE_BIN="${TMUX_AGENT_STATE_BIN:-${TMUX_AGENT_BIN:-$ROOT_DIR/bin/tmux-agent}}"
RUNTIME_KEY="${TMUX_AGENT_RUNTIME_KEY:-$(cat "${TMPDIR:-/tmp}/tmux-agent-runtime-key" 2>/dev/null || true)}"

if [[ -z "$RUNTIME_KEY" ]]; then
  exit 0
fi

"$STATE_BIN" update \
  --runtime-key "$RUNTIME_KEY" \
  --status running \
  --metadata tool_name=unknown
