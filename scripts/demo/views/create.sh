#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
VIEW_REGISTRY="0x0000000000000000000000000000000000000210"
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
sleep 3

RAND_SUFFIX=$(head -c 4 /dev/urandom | xxd -p)
VIEW_NAME="View${RAND_SUFFIX}"

QUERY="Log {address topics data transactionHash blockNumber}"
SDL="type ${VIEW_NAME} @materialized(if: false) {transactionHash: String}"

BUNDLE_HEX=$(QUERY="$QUERY" SDL="$SDL" python3 -c "
import struct, os
q = os.environ['QUERY'].encode()
s = os.environ['SDL'].encode()
out = b'VWL' + struct.pack('<B', 1)
out += struct.pack('<I', len(q)) + q
out += struct.pack('<I', len(s)) + s
out += struct.pack('<H', 0)      # lens count
out += struct.pack('<B', 0)      # codec (none)
out += struct.pack('<I', 0)      # lens blob len
print(out.hex())
")

echo ""
echo "=== Create View ==="
echo "Address: $FROM_ADDR"
echo "Name:    $VIEW_NAME"
echo ""

TX_RESULT=$(cast send "$VIEW_REGISTRY" \
  "register(bytes)(address,string)" \
  "0x${BUNDLE_HEX}" \
  --private-key "$PRIVATE_KEY" \
  --rpc-url "$RPC_URL" \
  --gas-limit 5000000 \
  --json)

TX_HASH=$(echo "$TX_RESULT" | jq -r '.transactionHash')
echo "TX Hash:  $TX_HASH"

STATUS=$(cast receipt "$TX_HASH" --rpc-url "$RPC_URL" --json | jq -r '.status')
if [ "$STATUS" = "0x0" ]; then
  echo "TX reverted. Inspect:"
  echo "  cast receipt $TX_HASH --rpc-url $RPC_URL"
else
  VIEW_ADDR=$(cast receipt "$TX_HASH" --rpc-url "$RPC_URL" --json \
    | jq -r '.logs[0].topics[1]' \
    | xargs cast --to-address 2>/dev/null || echo "")
  if [ -n "$VIEW_ADDR" ] && [ "$VIEW_ADDR" != "null" ]; then
    echo "View:     $VIEW_ADDR"
    echo ""
    echo "Next:"
    echo "  VIEW_ADDR=$VIEW_ADDR sh scripts/demo/views/info.sh"
  else
    echo "Check receipt: cast receipt $TX_HASH --rpc-url $RPC_URL"
  fi
fi

echo ""
echo "=== Done ==="
