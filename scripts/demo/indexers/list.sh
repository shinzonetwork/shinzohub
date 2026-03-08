#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────
# List all registered indexers
# ──────────────────────────────────────────────
# Just run: sh scripts/demo/indexers/list.sh

REST_URL="${REST_URL:-http://localhost:1317}"

echo "=== All Indexers ==="
curl -s "$REST_URL/shinzonetwork/indexer/v1/indexers" | jq .

echo ""
echo "=== Indexer Count ==="
curl -s "$REST_URL/shinzonetwork/indexer/v1/indexer_count" | jq .
