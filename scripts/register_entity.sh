#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
FROM_ADDR="${FROM_ADDR:-0xabd39bcd18199976acf5379450c52f06edbcf4f3}"
PRECOMPILE_ADDR="0x0000000000000000000000000000000000000211"
GAS_HEX="0x100000"
ENTITY=1 

PEER_PUB="0x703896c8fc429d0af204513a76a067b170ba71bf0be5ca8184e16ffce5b9732b"
PEER_SIG="0xf365e86878959ab3de294f92ad90644726b1be4978b31250a1a01da5c50c87fecafd0c63a051e81544bfcd69a99301f9c48ef79808a93dd645f61c251533880f"

NODE_PUB="0x041871f34ea7a26aa3dfa831b1e03681ec1bc99a0dcf9e8b4fd3f450c46462285db9f5f07bb582ff21239ed724397896f2fc8c6f1c86871132786491f616828056"
NODE_SIG="0x3045022100bca215bd97cc3f27573e7cda7a0a05e452d397643b4962581a5512bd7453e17e022067a01b68663b68533b544959ea3966feda2ae69478345cc4822cdfedd971cdb0"

# "entity-registration-test-nonce" in hex, with 0x
MESSAGE="0x656e746974792d726567697374726174696f6e2d746573742d6e6f6e6365"

DATA=$(cast calldata \
  "register(bytes,bytes,bytes,bytes,bytes,uint8)" \
  "$PEER_PUB" \
  "$PEER_SIG" \
  "$NODE_PUB" \
  "$NODE_SIG" \
  "$MESSAGE" \
  $ENTITY)

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
