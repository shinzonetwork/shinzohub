#!/usr/bin/env bash
set -uo pipefail

# Stop everything started by start_all_docker.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yaml"

echo "==> Stopping ShinzoHub on host..."
killall shinzohubd 2>/dev/null && echo "    Stopped shinzohubd" || echo "    shinzohubd not running"

echo "==> Stopping SourceHub + Hermes containers..."
docker compose -f "$COMPOSE_FILE" down -v

echo "==> Done."
