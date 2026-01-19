#!/usr/bin/env bash
set -euo pipefail

export KEY="acc0"
export CHAIN_ID=${CHAIN_ID:-"shinzo"}
export KEYRING=${KEYRING:-"test"}
export HOME_DIR=$(eval echo "${HOME_DIR:-"~/.shinzohub"}")
export BINARY="./build/shinzohubd"
export RPC=${RPC:-"26657"}

export RESOURCE="view"
export STREAM_ID="FilteredAndDecodedLogs_0xc5d55f9a4e8788abaaf74d4772c2a4afe"
export DID="testuserdid"

$BINARY tx sourcehub request-stream $RESOURCE $STREAM_ID $DID \
  --from $KEY \
  --keyring-backend $KEYRING \
  --chain-id $CHAIN_ID \
  --home $HOME_DIR \
  --node "tcp://127.0.0.1:$RPC" \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 9000ushinzo \
  --yes
