#!/usr/bin/env bash
set -euo pipefail

export SHINZOHUB_CONNECTION_ID="connection-0"
export SOURCEHUB_CONNECTION_ID="connection-0"

export KEY="acc0"
export CHAIN_ID=${CHAIN_ID:-"9001"}
export KEYRING=${KEYRING:-"test"}
export HOME_DIR=$(eval echo "${HOME_DIR:-"~/.shinzohub"}")
export BINARY="./build/shinzohubd"
export RPC=${RPC:-"26657"}

$BINARY tx sourcehub register-ica $SHINZOHUB_CONNECTION_ID $SOURCEHUB_CONNECTION_ID \
  --from $KEY \
  --keyring-backend $KEYRING \
  --chain-id $CHAIN_ID \
  --home $HOME_DIR \
  --node "tcp://127.0.0.1:$RPC" \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 9000ushinzo \
  --yes
