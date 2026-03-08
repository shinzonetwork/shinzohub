#!/usr/bin/env bash
set -euo pipefail

REST_URL="${REST_URL:-http://localhost:1317}"

# curl -s "$REST_URL/shinzonetwork/view/v1/views" | jq .

curl -s "$REST_URL/shinzonetwork/view/v1/views?include_data=true" | jq .
