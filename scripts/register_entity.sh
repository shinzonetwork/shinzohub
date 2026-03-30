#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
FROM_ADDR="${FROM_ADDR:-0xabd39bcd18199976acf5379450c52f06edbcf4f3}"
PRECOMPILE_ADDR="0x0000000000000000000000000000000000000211"
GAS_HEX="0x100000"

NODE_PUB="0x041871f34ea7a26aa3dfa831b1e03681ec1bc99a0dcf9e8b4fd3f450c46462285db9f5f07bb582ff21239ed724397896f2fc8c6f1c86871132786491f616828056"
NODE_SIG="0x3045022100bca215bd97cc3f27573e7cda7a0a05e452d397643b4962581a5512bd7453e17e022067a01b68663b68533b544959ea3966feda2ae69478345cc4822cdfedd971cdb0"

# "entity-registration-test-nonce" in hex, with 0x
MESSAGE="0x656e746974792d726567697374726174696f6e2d746573742d6e6f6e6365"
CONNECTION_STRING="${CONNECTION_STRING:-192.168.1.1:8080}"

DATA=$(cast calldata \
  "register(bytes,bytes,bytes,string)" \
  "$NODE_PUB" \
  "$NODE_SIG" \
  "$MESSAGE" \
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
