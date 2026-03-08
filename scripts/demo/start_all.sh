#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────
# Master script: start everything needed for ShinzoHub + SourceHub
# ──────────────────────────────────────────────────────────────
#
# This script:
#   1. Builds ShinzoHub
#   2. Starts SourceHub (background)
#   3. Starts ShinzoHub (background)
#   4. Waits for both chains to be ready
#   5. Starts Hermes relayer (background)
#   6. Waits for IBC connection
#   7. Registers ICA
#   8. Registers Shinzo policy
#
# Usage:
#   sh scripts/demo/start_all.sh
#
# To stop everything:
#   sh scripts/demo/stop_all.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
ICA_DIR="$SCRIPT_DIR/../ica"
cd "$PROJECT_DIR"

SOURCEHUB_PATH="${SOURCEHUB_PATH:-$HOME/sourcehub}"
LOG_DIR="$PROJECT_DIR/.logs"
mkdir -p "$LOG_DIR"

# ── Cleanup from previous runs ──
echo "==> Stopping previous processes..."
killall shinzohubd 2>/dev/null || true
killall sourcehubd 2>/dev/null || true
killall hermes 2>/dev/null || true
sleep 1

# ── 1. Build ShinzoHub ──
echo "==> Building ShinzoHub..."
make build

# ── 2. Start SourceHub ──
if [ ! -f "$SOURCEHUB_PATH/build/sourcehubd" ]; then
  echo "==> SourceHub not found at $SOURCEHUB_PATH, cloning and building..."
  git clone https://github.com/sourcenetwork/sourcehub.git "$SOURCEHUB_PATH"
  (cd "$SOURCEHUB_PATH" && make build)
  if [ ! -f "$SOURCEHUB_PATH/build/sourcehubd" ]; then
    echo "ERROR: Failed to build SourceHub. Check output above."
    exit 1
  fi
fi

echo "==> Starting SourceHub..."
sh "$ICA_DIR/start_sourcehub_node.sh" > "$LOG_DIR/sourcehub.log" 2>&1 &
SOURCEHUB_PID=$!
echo "    PID: $SOURCEHUB_PID (log: .logs/sourcehub.log)"

# ── 3. Start ShinzoHub ──
echo "==> Starting ShinzoHub..."
sh "$ICA_DIR/start_shinzohub_node.sh" > "$LOG_DIR/shinzohub.log" 2>&1 &
SHINZOHUB_PID=$!
echo "    PID: $SHINZOHUB_PID (log: .logs/shinzohub.log)"

# ── 4. Wait for both chains ──
echo "==> Waiting for ShinzoHub RPC (port 26657)..."
for i in $(seq 1 60); do
  if curl -s http://localhost:26657/status > /dev/null 2>&1; then
    echo "    ShinzoHub is ready."
    break
  fi
  if [ $i -eq 60 ]; then
    echo "ERROR: ShinzoHub did not start within 60s. Check .logs/shinzohub.log"
    exit 1
  fi
  sleep 1
done

echo "==> Waiting for SourceHub RPC (port 27686)..."
for i in $(seq 1 60); do
  if curl -s http://localhost:27686/status > /dev/null 2>&1; then
    echo "    SourceHub is ready."
    break
  fi
  if [ $i -eq 60 ]; then
    echo "ERROR: SourceHub did not start within 60s. Check .logs/sourcehub.log"
    exit 1
  fi
  sleep 1
done

# ── 5. Start Hermes ──
echo "==> Starting Hermes relayer..."
sh "$ICA_DIR/start_hermes.sh" > "$LOG_DIR/hermes.log" 2>&1 &
HERMES_PID=$!
echo "    PID: $HERMES_PID (log: .logs/hermes.log)"

# ── 6. Wait for Hermes + IBC connection ──
echo "==> Waiting for Hermes to establish IBC connection (up to 120s)..."
for i in $(seq 1 120); do
  if grep -q "Hermes has started" "$LOG_DIR/hermes.log" 2>/dev/null; then
    echo "    Hermes started and IBC connection is OPEN."
    break
  fi
  if [ $i -eq 120 ]; then
    echo "ERROR: Hermes did not start within 120s. Check .logs/hermes.log"
    exit 1
  fi
  sleep 1
done

# ── 7. Register ICA ──
echo "==> Registering ICA..."
sh "$ICA_DIR/register_ica.sh"
sleep 5  # wait for ICA channel to open via Hermes

# ── 8. Register Shinzo policy ──
echo "==> Registering Shinzo policy..."
sh "$ICA_DIR/register_shinzo_policy.sh"
sleep 5  # wait for IBC relay

# ── Done ──
echo ""
echo "=========================================="
echo "  All services are running!"
echo "=========================================="
echo ""
echo "  ShinzoHub RPC:    http://localhost:26657"
echo "  ShinzoHub EVM:    http://localhost:8545"
echo "  ShinzoHub REST:   http://localhost:1317"
echo "  SourceHub RPC:    http://localhost:27686"
echo ""
echo "  Logs:             .logs/"
echo ""
echo "  PIDs:"
echo "    ShinzoHub:  $SHINZOHUB_PID"
echo "    SourceHub:  $SOURCEHUB_PID"
echo "    Hermes:     $HERMES_PID"
echo ""
echo "  Stop everything:  sh scripts/demo/stop_all.sh"
echo "=========================================="

# Save PIDs for stop script
echo "$SHINZOHUB_PID" > "$LOG_DIR/shinzohub.pid"
echo "$SOURCEHUB_PID" > "$LOG_DIR/sourcehub.pid"
echo "$HERMES_PID" > "$LOG_DIR/hermes.pid"

# Wait so the script doesn't exit (keeps background processes alive)
wait
