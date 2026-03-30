#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
INDEXER_REGISTRY="0x0000000000000000000000000000000000000212"
BINARY="${BINARY:-./build/shinzohubd}"
CHAIN_ID="${CHAIN_ID:-91273002}"
HOME_DIR="${HOME_DIR:-$HOME/.shinzohub}"
NODE="${NODE:-tcp://127.0.0.1:26657}"
FUNDER="${FUNDER:-acc0}"
ADMIN_KEY="${ADMIN_KEY:-acc0}"

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

DIGEST_HEX=$(cast keccak "shinzo-indexer-assertion-delegate")
DIGEST_RAW="${DIGEST_HEX#0x}"

SIG_HEX=$(cast wallet sign --private-key "$PRIVATE_KEY" --no-hash "$DIGEST_HEX")
SIG_RAW="${SIG_HEX#0x}"

V_BYTE="${SIG_RAW: -2}"
if [ "$V_BYTE" = "1b" ]; then
  SIG_RAW="${SIG_RAW:0:128}00"
elif [ "$V_BYTE" = "1c" ]; then
  SIG_RAW="${SIG_RAW:0:128}01"
fi

SOURCE_CHAIN="ethereum"
SOURCE_CHAIN_ID=1
CONNECTION_STRING="${CONNECTION_STRING:-192.168.1.1:8080}"

echo ""
echo "=== Step 1: IndexerAssertion ==="
echo "Admin:    $ADMIN_KEY"
echo "Delegate: $BECH32_ADDR ($FROM_ADDR)"
echo ""

$BINARY tx indexer add-indexer-assertion \
  --delegate-address "$BECH32_ADDR" \
  --consensus-pub-key "consensus-$(echo "$FROM_ADDR" | cut -c3-12)" \
  --source-chain "$SOURCE_CHAIN" \
  --source-chain-id "$SOURCE_CHAIN_ID" \
  --assertion-id "assertion-$(date +%s)" \
  --delegate-digest "$DIGEST_RAW" \
  --delegate-signature "$SIG_RAW" \
  --from $ADMIN_KEY \
  --keyring-backend test \
  --chain-id "$CHAIN_ID" \
  --home "$HOME_DIR" \
  --node "$NODE" \
  --gas auto --gas-adjustment 1.5 --fees 9000ushinzo \
  --yes
sleep 2

MESSAGE_RAW="entity-registration-test-nonce"
echo -n "$MESSAGE_RAW" > "$TMP/msg.bin"

# secp256k1 node identity key
openssl ecparam -name secp256k1 -genkey -noout -out "$TMP/node.pem" 2>/dev/null
NODE_PUB="0x$(openssl ec -in "$TMP/node.pem" -pubout -outform DER 2>/dev/null | tail -c 65 | xxd -p | tr -d '\n')"
NODE_SIG="0x$(openssl dgst -sha256 -sign "$TMP/node.pem" "$TMP/msg.bin" | xxd -p | tr -d '\n')"

MESSAGE="0x$(echo -n "$MESSAGE_RAW" | xxd -p | tr -d '\n')"

echo ""
echo "=== Step 2: Register Indexer ==="
echo "Address:           $FROM_ADDR"
echo "Connection String: $CONNECTION_STRING"
echo "Source chain:      $SOURCE_CHAIN"
echo "Source chain ID:   $SOURCE_CHAIN_ID"
echo ""

cast send "$INDEXER_REGISTRY" \
  "register(bytes,bytes,bytes,string,string,uint64)" \
  "$NODE_PUB" "$NODE_SIG" "$MESSAGE" \
  "$CONNECTION_STRING" "$SOURCE_CHAIN" "$SOURCE_CHAIN_ID" \
  --private-key "$PRIVATE_KEY" \
  --rpc-url "$RPC_URL" \
  --gas-limit 1000000

echo ""
echo "=== Done ==="
