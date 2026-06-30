#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
HOST_REGISTRY="0x0000000000000000000000000000000000000211"
BINARY="${BINARY:-./build/shinzohubd}"
CHAIN_ID="${CHAIN_ID:-91273002}"
HOME_DIR="${HOME_DIR:-$HOME/.shinzohub}"
NODE="${NODE:-tcp://127.0.0.1:26657}"
FUNDER="${FUNDER:-acc0}"
TMP=$(mktemp -d)
trap "rm -rf $TMP" EXIT

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

MESSAGE_RAW="entity-registration-test-nonce"
echo -n "$MESSAGE_RAW" > "$TMP/msg.bin"

# secp256k1 node identity key
openssl ecparam -name secp256k1 -genkey -noout -out "$TMP/node.pem" 2>/dev/null
NODE_PUB="0x$(openssl ec -in "$TMP/node.pem" -pubout -outform DER 2>/dev/null | tail -c 65 | xxd -p | tr -d '\n')"
NODE_SIG="0x$(openssl dgst -sha256 -sign "$TMP/node.pem" "$TMP/msg.bin" | xxd -p | tr -d '\n')"

MESSAGE="0x$(echo -n "$MESSAGE_RAW" | xxd -p | tr -d '\n')"
CONNECTION_STRING="${CONNECTION_STRING:-192.168.1.1:8080}"
ENDPOINT_ADDRESS="${ENDPOINT_ADDRESS:-https://192.168.1.1/api/v0/graphql}"

echo "=== Register Host ==="
echo "Address:           $FROM_ADDR"
echo "Connection String: $CONNECTION_STRING"
echo "Endpoint Address:  $ENDPOINT_ADDRESS"
echo ""

cast send "$HOST_REGISTRY" \
  "register(bytes,bytes,bytes,string,string)" \
  "$NODE_PUB" "$NODE_SIG" "$MESSAGE" "$CONNECTION_STRING" "$ENDPOINT_ADDRESS" \
  --private-key "$PRIVATE_KEY" \
  --rpc-url "$RPC_URL" \
  --gas-limit 1000000

echo ""
echo "=== Done ==="
