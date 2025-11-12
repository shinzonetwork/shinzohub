#!/usr/bin/env bash
set -euo pipefail

# Your RPC endpoint
RPC_URL="${RPC_URL:-http://localhost:8545}"

# EntityRegistry precompile address
PRECOMPILE_ADDR="0x0000000000000000000000000000000000000211"

# Optional: filter by owner (indexed address)
OWNER="${OWNER:-}"

# Event: EntityRegistered(bytes32,address,bytes,bytes)
TOPIC0=$(cast keccak "EntityRegistered(bytes32,address,bytes,bytes)")

FROM_BLOCK="0x1"   # important: CometBFT complains if 0
TO_BLOCK="latest"

echo "RPC_URL      = $RPC_URL"
echo "Precompile   = $PRECOMPILE_ADDR"
echo "Event topic0 = $TOPIC0"
if [ -n "$OWNER" ]; then
  echo "Filter owner = $OWNER"
fi
echo

if [ -n "$OWNER" ]; then
  OWNER_TOPIC="0x$(printf '%064s' "${OWNER#0x}" | tr ' ' '0')"

  read -r -d '' FILTER_PAYLOAD <<EOF || true
{
  "fromBlock": "$FROM_BLOCK",
  "toBlock": "$TO_BLOCK",
  "address": "$PRECOMPILE_ADDR",
  "topics": [
    "$TOPIC0",
    null,
    "$OWNER_TOPIC"
  ]
}
EOF

else

  read -r -d '' FILTER_PAYLOAD <<EOF || true
{
  "fromBlock": "$FROM_BLOCK",
  "toBlock": "$TO_BLOCK",
  "address": "$PRECOMPILE_ADDR",
  "topics": [
    "$TOPIC0"
  ]
}
EOF

fi

curl -s -X POST "$RPC_URL" \
  -H "Content-Type: application/json" \
  --data-raw "{
    \"jsonrpc\":\"2.0\",
    \"method\":\"eth_getLogs\",
    \"params\":[
      $FILTER_PAYLOAD
    ],
    \"id\":1
  }" | jq .
