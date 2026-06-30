#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────
# Master script: start everything for ShinzoHub + SourceHub demo
# using Docker for SourceHub and Hermes.
#
# This script:
#   1. Builds ShinzoHub
#   2. Starts SourceHub in a container (ghcr.io image)
#   3. Starts ShinzoHub on the host (fresh state each run)
#   4. Starts Hermes in a container, which creates the IBC connection
#   5. Registers ICA + Shinzo policy from the host
#
# Usage:
#   sh scripts/demo/start_all_docker.sh
#
# To stop everything:
#   sh scripts/demo/stop_all_docker.sh
# ──────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
ICA_DIR="$SCRIPT_DIR/../ica"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yaml"
cd "$PROJECT_DIR"

LOG_DIR="$PROJECT_DIR/.logs"
mkdir -p "$LOG_DIR"

# ── Cleanup from previous runs ──
echo "==> Stopping previous processes and containers..."
killall shinzohubd 2>/dev/null || true
docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
sleep 1

# 1. Build ShinzoHub
echo "==> Building ShinzoHub..."
make build

# 2. Start SourceHub container 
echo "==> Starting SourceHub container..."
docker compose -f "$COMPOSE_FILE" up -d sourcehub

echo "==> Waiting for SourceHub to be healthy..."
for i in $(seq 1 60); do
  STATUS=$(docker compose -f "$COMPOSE_FILE" ps --format json sourcehub 2>/dev/null | grep -o '"Health":"[^"]*"' | cut -d'"' -f4 || true)
  if [ "$STATUS" = "healthy" ]; then
    echo "    SourceHub is healthy."
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "ERROR: SourceHub did not become healthy. Check: docker compose -f $COMPOSE_FILE logs sourcehub"
    exit 1
  fi
  sleep 1
done

# 3. Fund hermes account on SourceHub
# Hermes derives source1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme from the shared
# mnemonic. The standalone sourcehub container funds only validator + faucet,
# so we top up the hermes account from the faucet here.
HERMES_SOURCEHUB_ADDR="source1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme"
echo "==> Funding hermes account on SourceHub..."
docker compose -f "$COMPOSE_FILE" exec -T sourcehub \
  sourcehubd tx bank send faucet "$HERMES_SOURCEHUB_ADDR" 100000000uopen \
    --keyring-backend test \
    --chain-id sourcehub \
    --fees 1000uopen \
    --yes > /dev/null
sleep 3

# 4. Start ShinzoHub on host
echo "==> Starting ShinzoHub on host..."
sh "$ICA_DIR/start_shinzohub_node.sh" > "$LOG_DIR/shinzohub.log" 2>&1 &
SHINZOHUB_PID=$!
echo "    PID: $SHINZOHUB_PID (log: .logs/shinzohub.log)"

echo "==> Waiting for ShinzoHub RPC (port 26657)..."
for i in $(seq 1 60); do
  if curl -s http://localhost:26657/status > /dev/null 2>&1; then
    echo "    ShinzoHub is ready."
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "ERROR: ShinzoHub did not start within 60s. Check .logs/shinzohub.log"
    exit 1
  fi
  sleep 1
done

# 5. Start Hermes container
echo "==> Starting Hermes container..."
docker compose -f "$COMPOSE_FILE" up -d hermes

echo "==> Waiting for IBC connection to be established..."
# Hermes creates the connection during its entrypoint; allow time.
sleep 20

# 5. Register ICA + Shinzo policy from host
echo "==> Registering ICA..."
sh "$ICA_DIR/register_ica.sh"

echo "==> Waiting for ICA channel handshake to complete..."
BINARY="${BINARY:-./build/shinzohubd}"
for i in $(seq 1 60); do
  STATE=$($BINARY q ibc channel channels --node tcp://127.0.0.1:26657 -o json 2>/dev/null \
    | grep -o '"state":"STATE_OPEN"' | head -1 || true)
  if [ -n "$STATE" ]; then
    echo "    ICA channel is OPEN."
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "ERROR: ICA channel did not open within 120s. Check hermes logs."
    exit 1
  fi
  sleep 2
done

echo "==> Registering Shinzo policy..."
sh "$ICA_DIR/register_shinzo_policy.sh"
sleep 5

echo "==> Registering Shinzo objects (group/host, group/indexer, primitives)..."
sh "$ICA_DIR/register_shinzo_objects.sh"
sleep 5

echo ""
echo "==> All services are running!"
echo "    ShinzoHub:  tail -f .logs/shinzohub.log"
echo "    SourceHub:  docker compose -f $COMPOSE_FILE logs -f sourcehub"
echo "    Hermes:     docker compose -f $COMPOSE_FILE logs -f hermes"
echo ""
echo "    Stop:       sh scripts/demo/stop_all_docker.sh"
