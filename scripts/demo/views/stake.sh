#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
VIEW_ADDR="${VIEW_ADDR:?Set VIEW_ADDR to the view contract address}"
BINARY="${BINARY:-./build/shinzohubd}"
CHAIN_ID="${CHAIN_ID:-91273002}"
HOME_DIR="${HOME_DIR:-$HOME/.shinzohub}"
NODE="${NODE:-tcp://127.0.0.1:26657}"
FUNDER="${FUNDER:-acc0}"

# ── 1. Fresh throwaway wallet ──
PRIVATE_KEY=$(cast wallet new | grep 'Private key' | awk '{print $3}')
FROM_ADDR=$(cast wallet address --private-key "$PRIVATE_KEY")
BECH32_ADDR=$($BINARY debug addr "${FROM_ADDR#0x}" 2>&1 | grep 'Bech32 Acc' | awk '{print $3}')

# ── 2. Random stake amount: 1e15 to 1e18 wei (0.001 to 1 SHNZ) ──
RAND_RAW=$(od -A n -t u4 -N 4 /dev/urandom | tr -d ' ')
# Map to range: 1000000000000000 (1e15) to 1000000000000000000 (1e18)
MIN_STAKE=1000000000000000
MAX_STAKE=1000000000000000000
STAKE_WEI=$(python3 -c "print($MIN_STAKE + ($RAND_RAW % ($MAX_STAKE - $MIN_STAKE + 1)))")

# Fund wallet with slightly more than stake amount (extra for gas)
FUND_AMOUNT=$((STAKE_WEI + 100000000000000000))
FUND_USHINZO="${FUND_AMOUNT}ushinzo"

echo "=== Stake on View ==="
echo "View:    $VIEW_ADDR"
echo "Staker:  $FROM_ADDR"
echo "Amount:  ${STAKE_WEI} wei ($(python3 -c "print(f'{$STAKE_WEI/1e18:.4f}')") SHNZ)"
echo ""

echo "==> Funding $BECH32_ADDR..."
$BINARY tx bank send "$FUNDER" "$BECH32_ADDR" "$FUND_USHINZO" \
  --keyring-backend test \
  --chain-id "$CHAIN_ID" \
  --home "$HOME_DIR" \
  --node "$NODE" \
  --fees 5000ushinzo \
  --yes > /dev/null 2>&1
sleep 3

# ── 3. Stake on the view ──
echo "==> Staking..."
TX_RESULT=$(cast send "$VIEW_ADDR" \
  "stake()" \
  --value "$STAKE_WEI" \
  --private-key "$PRIVATE_KEY" \
  --rpc-url "$RPC_URL" \
  --gas-limit 200000 \
  --json)

TX_HASH=$(echo "$TX_RESULT" | jq -r '.transactionHash')
echo "TX Hash: $TX_HASH"

STATUS=$(cast receipt "$TX_HASH" --rpc-url "$RPC_URL" --json | jq -r '.status')
if [ "$STATUS" = "0x0" ]; then
  echo "TX reverted!"
  echo "  cast receipt $TX_HASH --rpc-url $RPC_URL"
else
  echo "Staked successfully!"
  echo ""
  TOTAL_STAKE=$(cast call "$VIEW_ADDR" "totalStake()(uint256)" --rpc-url "$RPC_URL")
  STAKE_OF=$(cast call "$VIEW_ADDR" "stakeOf(address)(uint256)" "$FROM_ADDR" --rpc-url "$RPC_URL")
  echo "Staker balance: $STAKE_OF wei"
  echo "Total stake:    $TOTAL_STAKE wei"
fi

echo ""
echo "=== Done ==="
