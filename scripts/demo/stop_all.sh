#!/usr/bin/env bash

# ──────────────────────────────────────────────
# Stop all ShinzoHub + SourceHub + Hermes
# ──────────────────────────────────────────────

echo "==> Stopping all processes..."

killall shinzohubd 2>/dev/null && echo "    Stopped shinzohubd" || echo "    shinzohubd not running"
killall sourcehubd 2>/dev/null && echo "    Stopped sourcehubd" || echo "    sourcehubd not running"
killall hermes 2>/dev/null && echo "    Stopped hermes" || echo "    hermes not running"

rm -f .logs/*.pid 2>/dev/null

echo "==> Done."
