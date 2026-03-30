#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
FROM_ADDR="${FROM_ADDR:-0xabd39bcd18199976acf5379450c52f06edbcf4f3}"
PRECOMPILE_ADDR="0x0000000000000000000000000000000000000211"
GAS_HEX="0x100000"

CONNECTION_STRING="${CONNECTION_STRING:-192.168.1.1:8080}"

DATA=$(cast calldata \
  "register(string)" \
  "$CONNECTION_STRING")

curl -s -X POST "$RPC_URL" \
  -H "Content-Type: application/json" \
  --data-raw "{
    \"jsonrpc\":\"2.0\",
    \"method\":\"eth_sendTransaction\",
    \"params\":[{
      \"from\":\"$FROM_ADDR\",
      \"to\":\"$PRECOMPILE_ADDR\",
      \"gas\":\"$GAS_HEX\",
      \"value\":\"0x0\",
      \"data\":\"$DATA\"
    }],
    \"id\":1
  }" | jq .
