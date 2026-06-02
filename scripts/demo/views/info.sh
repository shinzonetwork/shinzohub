#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
VIEW_REGISTRY="0x0000000000000000000000000000000000000210"
VIEW_ADDR="${VIEW_ADDR:?Set VIEW_ADDR to a registered view address}"

echo "=== View Info ==="
echo "Address: $VIEW_ADDR"
echo ""

echo "--- getView ---"
cast call "$VIEW_REGISTRY" \
  "getView(address)((address,string,string,uint64))" \
  "$VIEW_ADDR" \
  --rpc-url "$RPC_URL"
echo ""

echo "--- viewCount ---"
cast call "$VIEW_REGISTRY" "viewCount()(uint256)" --rpc-url "$RPC_URL"
echo ""

echo "=== Done ==="
