#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
VIEW_ADDR="${VIEW_ADDR:?Set VIEW_ADDR to a deployed view contract address}"
HOST_REGISTRY="0x0000000000000000000000000000000000000211"
BINARY="${BINARY:-./build/shinzohubd}"
CHAIN_ID="${CHAIN_ID:-91273002}"
HOME_DIR="${HOME_DIR:-$HOME/.shinzohub}"
NODE="${NODE:-tcp://127.0.0.1:26657}"
FUNDER="${FUNDER:-acc0}"

TMP=$(mktemp -d)
trap "rm -rf $TMP" EXIT

# ── 1. Fresh throwaway wallet ──
PRIVATE_KEY=$(cast wallet new | grep 'Private key' | awk '{print $3}')
FROM_ADDR=$(cast wallet address --private-key "$PRIVATE_KEY")
BECH32_ADDR=$($BINARY debug addr "${FROM_ADDR#0x}" 2>&1 | grep 'Bech32 Acc' | awk '{print $3}')

# ── 2. Random price and complexity ──
# rate: 1e12 to 1e16 wei (0.000001 to 0.01 SHNZ)
# complexity: 1 to 100
RAND1=$(od -A n -t u4 -N 4 /dev/urandom | tr -d ' ')
RAND2=$(od -A n -t u4 -N 4 /dev/urandom | tr -d ' ')
RATE=$(python3 -c "print(1000000000000 + ($RAND1 % (10000000000000000 - 1000000000000 + 1)))")
COMPLEXITY=$(( (RAND2 % 100) + 1 ))

echo "=== Register Host & Report View ==="
echo "View:        $VIEW_ADDR"
echo "Host:        $FROM_ADDR"
echo "Rate:        $RATE wei"
echo "Complexity:  $COMPLEXITY"
echo ""

echo "==> Funding $BECH32_ADDR..."
$BINARY tx bank send "$FUNDER" "$BECH32_ADDR" 1000000000000000000ushinzo \
  --keyring-backend test \
  --chain-id "$CHAIN_ID" \
  --home "$HOME_DIR" \
  --node "$NODE" \
  --fees 5000ushinzo \
  --yes > /dev/null 2>&1
sleep 3

# ── 3. Register as host ──
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

echo "==> Registering host..."
TX_RESULT=$(cast send "$HOST_REGISTRY" \
  "register(bytes,bytes,bytes,bytes,bytes)" \
  "$PEER_PUB" "$PEER_SIG" "$NODE_PUB" "$NODE_SIG" "$MESSAGE" \
  --private-key "$PRIVATE_KEY" \
  --rpc-url "$RPC_URL" \
  --gas-limit 1000000 \
  --json)

TX_HASH=$(echo "$TX_RESULT" | jq -r '.transactionHash')
STATUS=$(cast receipt "$TX_HASH" --rpc-url "$RPC_URL" --json | jq -r '.status')
if [ "$STATUS" = "0x0" ]; then
  echo "Host registration reverted!"
  echo "  cast receipt $TX_HASH --rpc-url $RPC_URL"
  exit 1
fi
echo "Host registered."

# ── 4. Report on the view ──
echo "==> Reporting..."
TX_RESULT=$(cast send "$VIEW_ADDR" \
  "report(uint256,uint256)" \
  "$COMPLEXITY" "$RATE" \
  --private-key "$PRIVATE_KEY" \
  --rpc-url "$RPC_URL" \
  --gas-limit 200000 \
  --json)

TX_HASH=$(echo "$TX_RESULT" | jq -r '.transactionHash')
STATUS=$(cast receipt "$TX_HASH" --rpc-url "$RPC_URL" --json | jq -r '.status')
if [ "$STATUS" = "0x0" ]; then
  echo "Report reverted!"
  echo "  cast receipt $TX_HASH --rpc-url $RPC_URL"
  exit 1
fi

echo "Reported successfully!"
echo ""
HOSTS=$(cast call "$VIEW_ADDR" "hosts()(address[])" --rpc-url "$RPC_URL")
AVG_RATE=$(cast call "$VIEW_ADDR" "rate()(uint256)" --rpc-url "$RPC_URL")
AVG_COMPLEXITY=$(cast call "$VIEW_ADDR" "complexity()(uint256)" --rpc-url "$RPC_URL")
echo "Hosts:          $HOSTS"
echo "Avg rate:       $AVG_RATE"
echo "Avg complexity: $AVG_COMPLEXITY"

echo ""
echo "=== Done ==="
