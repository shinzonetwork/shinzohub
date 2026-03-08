#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
VIEW_ADDR="${VIEW_ADDR:?Set VIEW_ADDR to a deployed view contract address}"

echo "=== View Info ==="
echo "Address: $VIEW_ADDR"
echo ""

echo "--- name ---"
cast call "$VIEW_ADDR" "name()(string)" --rpc-url "$RPC_URL"
echo ""

echo "--- creator ---"
cast call "$VIEW_ADDR" "creator()(address)" --rpc-url "$RPC_URL"
echo ""

echo "--- price ---"
cast call "$VIEW_ADDR" "price()(uint256)" --rpc-url "$RPC_URL"
echo ""

echo "--- rate ---"
cast call "$VIEW_ADDR" "rate()(uint256)" --rpc-url "$RPC_URL"
echo ""

echo "--- complexity ---"
cast call "$VIEW_ADDR" "complexity()(uint256)" --rpc-url "$RPC_URL"
echo ""

echo "--- hosts ---"
cast call "$VIEW_ADDR" "hosts()(address[])" --rpc-url "$RPC_URL"
echo ""

echo "--- totalStake ---"
cast call "$VIEW_ADDR" "totalStake()(uint256)" --rpc-url "$RPC_URL"
echo ""

echo "--- earnings ---"
cast call "$VIEW_ADDR" "earnings()(uint256)" --rpc-url "$RPC_URL"
echo ""

echo "--- pricingContract ---"
cast call "$VIEW_ADDR" "pricingContract()(address)" --rpc-url "$RPC_URL"
echo ""

echo "=== Done ==="
