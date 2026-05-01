#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$ROOT_DIR/bin"

cd "$ROOT_DIR"
go build -o "$ROOT_DIR/bin/tmux-agent" ./cmd/tmux-agent
