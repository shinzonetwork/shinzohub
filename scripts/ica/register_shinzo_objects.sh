#!/usr/bin/env bash
set -euo pipefail

KEY="acc0"
CHAIN_ID="${CHAIN_ID:-"91273002"}"
KEYRING="${KEYRING:-test}"
HOME_DIR="$(eval echo "${HOME_DIR:-~/.shinzohub}")"
BINARY="./build/shinzohubd"
RPC="${RPC:-26657}"

# list your resources here (positional args to the command)
RESOURCES=(block logs event transaction)

"$BINARY" tx sourcehub register-objects "${RESOURCES[@]}" \
  --from "$KEY" \
  --keyring-backend "$KEYRING" \
  --chain-id "$CHAIN_ID" \
  --home "$HOME_DIR" \
  --node "tcp://127.0.0.1:$RPC" \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 9000ushinzo \
  --yes
