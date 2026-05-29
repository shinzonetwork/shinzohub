#!/usr/bin/env bash
set -euo pipefail

# Reports a view using the Cosmos LCD REST endpoint (no JSON-RPC / port 8545 needed).
# Uses the ethermint EVM query to call the view contract.

LCD_URL="${LCD_URL:-http://localhost:1317}"
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

# Helper: hex calldata -> base64
hex_to_b64() {
  echo -n "${1#0x}" | xxd -r -p | base64
}

# Helper: base64 ABI result -> hex
b64_to_hex() {
  echo -n "$1" | base64 -d | xxd -p | tr -d '\n'
}

# Helper: call a view contract method via LCD
lcd_call() {
  local to="$1"
  local calldata_hex="$2"
  local args_b64
  args_b64=$(hex_to_b64 "$calldata_hex")

  local resp
  resp=$(curl -s -X POST "${LCD_URL}/ethermint/evm/v1/eth_call" \
    -H "Content-Type: application/json" \
    -d "{\"args\":\"${args_b64}\",\"to\":\"${to}\"}")

  echo "$resp" | jq -r '.ret // empty'
}

# ── 1. Fresh throwaway wallet ──
PRIVATE_KEY=$(cast wallet new | grep 'Private key' | awk '{print $3}')
FROM_ADDR=$(cast wallet address --private-key "$PRIVATE_KEY")
BECH32_ADDR=$($BINARY debug addr "${FROM_ADDR#0x}" 2>&1 | grep 'Bech32 Acc' | awk '{print $3}')

# ── 2. Random rate and complexity ──
RAND1=$(od -A n -t u4 -N 4 /dev/urandom | tr -d ' ')
RAND2=$(od -A n -t u4 -N 4 /dev/urandom | tr -d ' ')
RATE=$(python3 -c "print(1000000000000 + ($RAND1 % (10000000000000000 - 1000000000000 + 1)))")
COMPLEXITY=$(( (RAND2 % 100) + 1 ))

echo "=== Register Host & Report View (LCD) ==="
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

openssl ecparam -name secp256k1 -genkey -noout -out "$TMP/node.pem" 2>/dev/null
NODE_PUB="0x$(openssl ec -in "$TMP/node.pem" -pubout -outform DER 2>/dev/null | tail -c 65 | xxd -p | tr -d '\n')"
NODE_SIG="0x$(openssl dgst -sha256 -sign "$TMP/node.pem" "$TMP/msg.bin" | xxd -p | tr -d '\n')"

MESSAGE="0x$(echo -n "$MESSAGE_RAW" | xxd -p | tr -d '\n')"
CONNECTION_STRING="${CONNECTION_STRING:-192.168.1.1:8080}"
ENDPOINT_ADDRESS="${ENDPOINT_ADDRESS:-https://192.168.1.1/api/v0/graphql}"

echo "==> Registering host..."
TX_RESULT=$(cast send "$HOST_REGISTRY" \
  "register(bytes,bytes,bytes,string,string)" \
  "$NODE_PUB" "$NODE_SIG" "$MESSAGE" "$CONNECTION_STRING" "$ENDPOINT_ADDRESS" \
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

# ── 5. Query results via LCD (no JSON-RPC needed from here) ──
echo "=== Querying via Cosmos LCD ==="

# hosts()
HOSTS_CALLDATA=$(cast calldata "hosts()")
HOSTS_RET=$(lcd_call "$VIEW_ADDR" "$HOSTS_CALLDATA")
if [ -n "$HOSTS_RET" ]; then
  HOSTS_HEX=$(b64_to_hex "$HOSTS_RET")
  echo "Hosts (raw):    0x$HOSTS_HEX"
  echo "Hosts (decoded): $(cast --abi-decode "hosts()(address[])" "0x$HOSTS_HEX")"
fi

# rate()
RATE_CALLDATA=$(cast calldata "rate()")
RATE_RET=$(lcd_call "$VIEW_ADDR" "$RATE_CALLDATA")
if [ -n "$RATE_RET" ]; then
  RATE_HEX=$(b64_to_hex "$RATE_RET")
  echo "Avg rate:       $(cast --to-dec "0x$RATE_HEX")"
fi

# complexity()
COMP_CALLDATA=$(cast calldata "complexity()")
COMP_RET=$(lcd_call "$VIEW_ADDR" "$COMP_CALLDATA")
if [ -n "$COMP_RET" ]; then
  COMP_HEX=$(b64_to_hex "$COMP_RET")
  echo "Avg complexity: $(cast --to-dec "0x$COMP_HEX")"
fi

# price()
PRICE_CALLDATA=$(cast calldata "price()")
PRICE_RET=$(lcd_call "$VIEW_ADDR" "$PRICE_CALLDATA")
if [ -n "$PRICE_RET" ]; then
  PRICE_HEX=$(b64_to_hex "$PRICE_RET")
  echo "Price:          $(cast --to-dec "0x$PRICE_HEX")"
fi

echo ""
echo "=== Done ==="
