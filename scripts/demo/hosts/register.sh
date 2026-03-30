#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
HOST_REGISTRY="0x0000000000000000000000000000000000000211"
BINARY="${BINARY:-./build/shinzohubd}"
CHAIN_ID="${CHAIN_ID:-91273002}"
HOME_DIR="${HOME_DIR:-$HOME/.shinzohub}"
NODE="${NODE:-tcp://127.0.0.1:26657}"
FUNDER="${FUNDER:-acc0}"

PRIVATE_KEY=$(cast wallet new | grep 'Private key' | awk '{print $3}')
FROM_ADDR=$(cast wallet address --private-key "$PRIVATE_KEY")
BECH32_ADDR=$($BINARY debug addr "${FROM_ADDR#0x}" 2>&1 | grep 'Bech32 Acc' | awk '{print $3}')

echo "==> Funding $BECH32_ADDR..."
$BINARY tx bank send "$FUNDER" "$BECH32_ADDR" 1000000000000000000ushinzo \
  --keyring-backend test \
  --chain-id "$CHAIN_ID" \
  --home "$HOME_DIR" \
  --node "$NODE" \
  --fees 5000ushinzo \
  --yes > /dev/null 2>&1
sleep 2

CONNECTION_STRING="${CONNECTION_STRING:-192.168.1.1:8080}"

echo "=== Register Host ==="
echo "Address:           $FROM_ADDR"
echo "Connection String: $CONNECTION_STRING"
echo ""

cast send "$HOST_REGISTRY" \
  "register(string)" \
  "$CONNECTION_STRING" \
  --private-key "$PRIVATE_KEY" \
  --rpc-url "$RPC_URL" \
  --gas-limit 1000000

echo ""
echo "=== Done ==="
