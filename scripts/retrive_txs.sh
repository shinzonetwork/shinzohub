#!/usr/bin/env bash
set -euo pipefail

RPC="${RPC:-http://localhost:27686}"
PER_PAGE="${PER_PAGE:-100}"
OUT="${OUT:-txhashes.txt}"

> "$OUT"
page=1
total=0
seen=0

while :; do
  resp=$(curl -s "$RPC" -H "Content-Type: application/json" --data "{
    \"jsonrpc\":\"2.0\",
    \"method\":\"tx_search\",
    \"params\":{
      \"query\":\"tx.height > 0\",
      \"prove\": false,
      \"page\": \"${page}\",
      \"per_page\": \"${PER_PAGE}\",
      \"order_by\":\"asc\"
    },
    \"id\":\"1\"
  }")

  # stop on error
  err=$(echo "$resp" | jq -r '.error // empty')
  if [ -n "$err" ] && [ "$err" != "null" ]; then
    echo "$resp" | jq
    exit 1
  fi

  # pull total_count (string) and current page size
  total=$(echo "$resp" | jq -r '.result.total_count | tonumber')
  count=$(echo "$resp" | jq -r '.result.txs | length')

  # append tx hashes (note: Tendermint returns base64 tx + result; no eth-style hash)
  # Here we compute the SHA256 hash of the raw tx bytes (uppercased hex), which matches
  # CometBFT’s tx hash format.
  echo "$resp" | jq -r '.result.txs[].tx' | while read -r b64; do
    [ -z "$b64" ] && continue
    # macOS: use `base64 -D`; GNU: `base64 -d`
    raw=$(echo "$b64" | base64 -D 2>/dev/null || echo "$b64" | base64 -d)
    echo -n "$raw" | shasum -a 256 | awk '{print toupper($1)}'
  done >> "$OUT"

  seen=$(( seen + count ))
  echo "Page $page: +$count (seen=$seen / total=$total)"

  # done?
  if [ "$seen" -ge "$total" ] || [ "$count" -eq 0 ]; then
    break
  fi

  page=$(( page + 1 ))
done

echo "Wrote $(wc -l < \"$OUT\") tx hashes to $OUT"
