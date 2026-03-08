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

# Ed25519 peer key
openssl genpkey -algorithm Ed25519 -out "$TMP/peer.pem" 2>/dev/null
PEER_PUB="0x$(openssl pkey -in "$TMP/peer.pem" -pubout -outform DER 2>/dev/null | tail -c 32 | xxd -p | tr -d '\n')"
PEER_SIG="0x$(openssl pkeyutl -sign -inkey "$TMP/peer.pem" -rawin -in "$TMP/msg.bin" | xxd -p | tr -d '\n')"

# secp256k1 node identity key
openssl ecparam -name secp256k1 -genkey -noout -out "$TMP/node.pem" 2>/dev/null
NODE_PUB="0x$(openssl ec -in "$TMP/node.pem" -pubout -outform DER 2>/dev/null | tail -c 65 | xxd -p | tr -d '\n')"
NODE_SIG="0x$(openssl dgst -sha256 -sign "$TMP/node.pem" "$TMP/msg.bin" | xxd -p | tr -d '\n')"

MESSAGE="0x$(echo -n "$MESSAGE_RAW" | xxd -p | tr -d '\n')"

echo "=== Register Host ==="
echo "Address: $FROM_ADDR"
echo ""

cast send "$HOST_REGISTRY" \
  "register(bytes,bytes,bytes,bytes,bytes)" \
  "$PEER_PUB" "$PEER_SIG" "$NODE_PUB" "$NODE_SIG" "$MESSAGE" \
  --private-key "$PRIVATE_KEY" \
  --rpc-url "$RPC_URL" \
  --gas-limit 1000000

echo ""
echo "=== Done ==="
