#!/usr/bin/env bash
set -euo pipefail

REST_URL="${REST_URL:-http://localhost:1317}"

curl -s "$REST_URL/shinzonetwork/host/v1/hosts" | jq .
