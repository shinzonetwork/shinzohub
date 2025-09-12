#!/bin/sh

set -e

ICA_PACKET_JSON="scripts/ibc/ica_packet.json"
POLICY_CONTENT="name: ica test policy"

HERMES_HOME="$HOME/.hermes"

SOURCEHUB_PATH="$HOME/sourcehub"
SOURCEHUB_BIN="$HOME/sourcehub/build/sourcehubd"
SOURCEHUB_CHAIN_ID="sourcehub"
SOURCEHUB_HOME_DIR="$HOME/.sourcehub"
SOURCEHUB_P2P=tcp://0.0.0.0:27684
SOURCEHUB_ADDR=tcp://0.0.0.0:27685
SOURCEHUB_RPC=tcp://127.0.0.1:27686
SOURCEHUB_GRPC=localhost:9095
SOURCEHUB_PPROF=localhost:6065
SOURCEHUB_NAME="validator-node"

# Kill running processes
killall sourcehubd 2>/dev/null || true
killall shinzohubd 2>/dev/null || true

# Cleanup directories
rm -rf $SOURCEHUB_HOME_DIR
rm -rf sourcehub.log
rm -rf shinzohub.log
rm -rf $HERMES_HOME
rm -rf hermes.log

echo "==> Initializing sourcehub..."
$SOURCEHUB_BIN init $SOURCEHUB_NAME --chain-id $SOURCEHUB_CHAIN_ID --default-denom="uopen" --home="$SOURCEHUB_HOME_DIR"
$SOURCEHUB_BIN keys add $SOURCEHUB_NAME --keyring-backend=test --home="$SOURCEHUB_HOME_DIR"
VALIDATOR_ADDR=$($SOURCEHUB_BIN keys show $SOURCEHUB_NAME -a --keyring-backend=test --home="$SOURCEHUB_HOME_DIR")
$SOURCEHUB_BIN genesis add-genesis-account $VALIDATOR_ADDR 1000000000000uopen --home="$SOURCEHUB_HOME_DIR"
echo "divert tenant reveal hire thing jar carry lonely magic oak audit fiber earth catalog cheap merry print clown portion speak daring giant weird slight" | $SOURCEHUB_BIN keys add source --recover --keyring-backend=test --home="$SOURCEHUB_HOME_DIR"
SOURCEHUB_SOURCE_ADDR=$($SOURCEHUB_BIN keys show source -a --keyring-backend=test --home="$SOURCEHUB_HOME_DIR")
$SOURCEHUB_BIN genesis add-genesis-account $SOURCEHUB_SOURCE_ADDR 1000000000000uopen --home="$SOURCEHUB_HOME_DIR"
$SOURCEHUB_BIN genesis gentx $SOURCEHUB_NAME 100000000uopen --chain-id $SOURCEHUB_CHAIN_ID --keyring-backend=test --home="$SOURCEHUB_HOME_DIR"
$SOURCEHUB_BIN genesis collect-gentxs --home "$SOURCEHUB_HOME_DIR"
$SOURCEHUB_BIN genesis validate-genesis --home "$SOURCEHUB_HOME_DIR"

jq '.app_state.transfer.port_id = "transfer"' "$SOURCEHUB_HOME_DIR/config/genesis.json" > tmp.json && mv tmp.json "$SOURCEHUB_HOME_DIR/config/genesis.json"
jq '.app_state.transfer += {"params": {"send_enabled": true, "receive_enabled": true}}' "$SOURCEHUB_HOME_DIR/config/genesis.json" > tmp.json && mv tmp.json "$SOURCEHUB_HOME_DIR/config/genesis.json"

sed -i '' 's/minimum-gas-prices = ""/minimum-gas-prices = "0.001uopen,0.001ucredit"/' "$SOURCEHUB_HOME_DIR/config/app.toml"
sed -i '' 's/^timeout_propose = .*/timeout_propose = "500ms"/' "$SOURCEHUB_HOME_DIR/config/config.toml"
sed -i '' 's/^timeout_prevote = .*/timeout_prevote = "500ms"/' "$SOURCEHUB_HOME_DIR/config/config.toml"
sed -i '' 's/^timeout_precommit = .*/timeout_precommit = "500ms"/' "$SOURCEHUB_HOME_DIR/config/config.toml"
sed -i '' 's/^timeout_commit = .*/timeout_commit = "1s"/' "$SOURCEHUB_HOME_DIR/config/config.toml"


export KEY="acc0"
export KEY2="acc1"

export CHAIN_ID=${CHAIN_ID:-"shinzohub_9000-1"}
export MONIKER="validator"
export KEYALGO="eth_secp256k1"
export KEYRING=${KEYRING:-"test"}
export HOME_DIR=$(eval echo "${HOME_DIR:-"~/.shinzohub"}")
export BINARY="./build/shinzohubd"
export DENOM=${DENOM:-ushinzo}

export CLEAN=${CLEAN:-"true"}
export RPC=${RPC:-"26657"}
export REST=${REST:-"1317"}
export PROFF=${PROFF:-"6060"}
export P2P=${P2P:-"26656"}
export GRPC=${GRPC:-"9090"}
export GRPC_WEB=${GRPC_WEB:-"9091"}
export ROSETTA=${ROSETTA:-"8080"}
export BLOCK_TIME=${BLOCK_TIME:-"1s"}

# if which binary does not exist, install it
if [ -z `which $BINARY` ]; then
  just install

  if [ -z `which $BINARY` ]; then
    echo "Ensure $BINARY is installed and in your PATH"
    exit 1
  fi
fi

command -v $BINARY > /dev/null 2>&1 || { echo >&2 "$BINARY command not found. Ensure this is setup / properly installed in your GOPATH (just install)."; exit 1; }
command -v jq > /dev/null 2>&1 || { echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"; exit 1; }

set_config() {
  $BINARY config set client chain-id $CHAIN_ID
  $BINARY config set client keyring-backend $KEYRING
}
set_config


from_scratch () {
  # Fresh install on current branch
  just install

  # remove existing daemon files.
  if [ ${#HOME_DIR} -le 2 ]; then
      echo "HOME_DIR must be more than 2 characters long"
      return
  fi
  rm -rf $HOME_DIR && echo "Removed $HOME_DIR"

  # reset values if not set already after whipe
  set_config

  add_key() {
    key=$1
    mnemonic=$2
    echo $mnemonic | $BINARY keys add $key --keyring-backend $KEYRING --algo $KEYALGO --home $HOME_DIR --recover
  }

  # shinzo1g4zla6r5erlf364x5lcgvff6rmek4uwxwlfzs8
  add_key $KEY "divert tenant reveal hire thing jar carry lonely magic oak audit fiber earth catalog cheap merry print clown portion speak daring giant weird slight"
  # shinzo1r6yue0vuyj9m7xw78npspt9drq2tmtvgvv9hkp
  add_key $KEY2 "wealth flavor believe regret funny network recall kiss grape useless pepper cram hint member few certain unveil rather brick bargain curious require crowd raise"

  $BINARY init $MONIKER --chain-id $CHAIN_ID --default-denom $DENOM --home $HOME_DIR

  update_test_genesis () {
    cat $HOME_DIR/config/genesis.json | jq "$1" > $HOME_DIR/config/tmp_genesis.json && mv $HOME_DIR/config/tmp_genesis.json $HOME_DIR/config/genesis.json
  }

  # === CORE MODULES ===

  # Block
  update_test_genesis '.consensus_params["block"]["max_gas"]="100000000"'

  # Gov
  update_test_genesis `printf '.app_state["gov"]["params"]["min_deposit"]=[{"denom":"%s","amount":"1000000"}]' $DENOM`
  update_test_genesis '.app_state["gov"]["params"]["voting_period"]="30s"'
  update_test_genesis '.app_state["gov"]["params"]["expedited_voting_period"]="15s"'

  update_test_genesis `printf '.app_state["evm"]["params"]["evm_denom"]="%s"' $DENOM`
  update_test_genesis '.app_state["evm"]["params"]["active_static_precompiles"]=["0x0000000000000000000000000000000000000100","0x0000000000000000000000000000000000000210","0x0000000000000000000000000000000000000400","0x0000000000000000000000000000000000000800","0x0000000000000000000000000000000000000801","0x0000000000000000000000000000000000000802","0x0000000000000000000000000000000000000803","0x0000000000000000000000000000000000000804","0x0000000000000000000000000000000000000805"]'
  update_test_genesis '.app_state["erc20"]["native_precompiles"]=["0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"]' # https://eips.ethereum.org/EIPS/eip-7528
  update_test_genesis `printf '.app_state["erc20"]["token_pairs"]=[{contract_owner:1,erc20_address:"0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",denom:"%s",enabled:true}]' $DENOM`
  update_test_genesis '.app_state["feemarket"]["params"]["no_base_fee"]=true'
  update_test_genesis '.app_state["feemarket"]["params"]["base_fee"]="0.000000000000000000"'

  # staking
  update_test_genesis `printf '.app_state["staking"]["params"]["bond_denom"]="%s"' $DENOM`
  update_test_genesis '.app_state["staking"]["params"]["min_commission_rate"]="0.050000000000000000"'

  # mint
  update_test_genesis `printf '.app_state["mint"]["params"]["mint_denom"]="%s"' $DENOM`

  # crisis
  update_test_genesis `printf '.app_state["crisis"]["constant_fee"]={"denom":"%s","amount":"1000"}' $DENOM`

  ## abci
  update_test_genesis '.consensus["params"]["abci"]["vote_extensions_enable_height"]="1"'
  update_test_genesis '.consensus["params"]["block"]["max_bytes"]="104857600"'

  # === CUSTOM MODULES ===
  # tokenfactory
  update_test_genesis '.app_state["tokenfactory"]["params"]["denom_creation_fee"]=[]'
  update_test_genesis '.app_state["tokenfactory"]["params"]["denom_creation_gas_consume"]=100000'

  # --- IBC / ICA / Transfer params ---

  # enable ICA controller on ShinzoHub
#   update_test_genesis '.app_state["interchainaccounts"]["controller_genesis_state"]["params"]["controller_enabled"]=true'

  # explicitly disable ICA host on ShinzoHub (optional but clearer)
#   update_test_genesis '.app_state["interchainaccounts"]["host_genesis_state"]["params"]["host_enabled"]=false'

  BASE_GENESIS_ALLOCATIONS="100000000000000000000000000$DENOM,100000000test"

  # Allocate genesis accounts
  $BINARY genesis add-genesis-account $KEY $BASE_GENESIS_ALLOCATIONS --keyring-backend $KEYRING --home $HOME_DIR --append
  $BINARY genesis add-genesis-account $KEY2 $BASE_GENESIS_ALLOCATIONS --keyring-backend $KEYRING --home $HOME_DIR --append

  OWNER_ADDR=$($BINARY keys show $KEY -a --keyring-backend test --home "$HOME_DIR")

  # TODO
#   HERMES_ADDR="shinzo1kn8fcqzy7m9zwqmsq09yupm4frkukfszwhshk2"
#   $BINARY genesis add-genesis-account $HERMES_ADDR $BASE_GENESIS_ALLOCATIONS --keyring-backend $KEYRING --home $HOME_DIR --append
  
  # Sign genesis transaction
  $BINARY genesis gentx $KEY 1000000000000000000000$DENOM --gas-prices 0${DENOM} --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME_DIR

  $BINARY genesis collect-gentxs --home $HOME_DIR

  $BINARY genesis validate-genesis --home $HOME_DIR
  err=$?
  if [ $err -ne 0 ]; then
    echo "Failed to validate genesis"
    return
  fi
}

# check if CLEAN is not set to false
if [ "$CLEAN" != "false" ]; then
  echo "Starting from a clean state"
  from_scratch
fi

echo "Starting node..."

# Opens the RPC endpoint to outside connections
sed -i -e 's/laddr = "tcp:\/\/127.0.0.1:26657"/c\laddr = "tcp:\/\/0.0.0.0:'$RPC'"/g' $HOME_DIR/config/config.toml
sed -i -e 's/cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' $HOME_DIR/config/config.toml

# REST endpoint
sed -i -e 's/address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:'$REST'"/g' $HOME_DIR/config/app.toml
sed -i -e 's/enable = false/enable = true/g' $HOME_DIR/config/app.toml
sed -i -e 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' $HOME_DIR/config/app.toml

# peer exchange
sed -i -e 's/pprof_laddr = "localhost:6060"/pprof_laddr = "localhost:'$PROFF'"/g' $HOME_DIR/config/config.toml
sed -i -e 's/laddr = "tcp:\/\/0.0.0.0:26656"/laddr = "tcp:\/\/0.0.0.0:'$P2P'"/g' $HOME_DIR/config/config.toml

# GRPC
sed -i -e 's/address = "localhost:9090"/address = "0.0.0.0:'$GRPC'"/g' $HOME_DIR/config/app.toml
sed -i -e 's/address = "localhost:9091"/address = "0.0.0.0:'$GRPC_WEB'"/g' $HOME_DIR/config/app.toml

# Rosetta Api
sed -i -e 's/address = ":8080"/address = "0.0.0.0:'$ROSETTA'"/g' $HOME_DIR/config/app.toml

# Faster blocks
sed -i -e 's/timeout_commit = "1s"/timeout_commit = "'$BLOCK_TIME'"/g' $HOME_DIR/config/config.toml

echo "==> Starting sourcehub..."
$SOURCEHUB_BIN start \
  --home $SOURCEHUB_HOME_DIR \
  --rpc.laddr $SOURCEHUB_RPC \
  --rpc.pprof_laddr $SOURCEHUB_PPROF \
  --p2p.laddr $SOURCEHUB_P2P \
  --grpc.address $SOURCEHUB_GRPC \
  --address $SOURCEHUB_ADDR \
  > sourcehub.log 2>&1 &

SOURCEHUB_PID=$!

echo "==> Starting shinzohub..."
$BINARY start \
  --pruning=nothing \
  --minimum-gas-prices=0$DENOM \
  --rpc.laddr="tcp://0.0.0.0:$RPC" \
  --home $HOME_DIR \
  --json-rpc.api=eth,txpool,personal,net,debug,web3 \
  --chain-id="$CHAIN_ID" \
  > shinzohub.log 2>&1 &

SHINZOHUB_PID=$!

echo "Sourcehub PID: $SOURCEHUB_PID"
echo "Shinzohub PID: $SHINZOHUB_PID"

sleep 5

# Make hermes dir and copy the config
mkdir $HERMES_HOME
mkdir $HERMES_HOME/bin/
tar -C $HERMES_HOME/bin/ -vxzf $HOME/hermes-v1.13.3-aarch64-apple-darwin.tar.gz
cp scripts/ibc/hermes_config.toml "$HERMES_HOME/config.toml"

xattr -d com.apple.quarantine "$HERMES_HOME/bin/hermes" || true

# Add hermes key (same for both chains)
echo "divert tenant reveal hire thing jar carry lonely magic oak audit fiber earth catalog cheap merry print clown portion speak daring giant weird slight" > /tmp/mnemonic.txt
hermes keys add --chain $SOURCEHUB_CHAIN_ID --mnemonic-file /tmp/mnemonic.txt
hermes keys add --chain $CHAIN_ID --mnemonic-file /tmp/mnemonic.txt --hd-path "m/44'/60'/0'/0/0"
rm /tmp/mnemonic.txt

hermes keys list --chain $CHAIN_ID
hermes keys list --chain $SOURCEHUB_CHAIN_ID

echo "==> Creating IBC connection..."
hermes create connection \
  --a-chain $SOURCEHUB_CHAIN_ID \
  --b-chain $CHAIN_ID

sleep 1

echo "==> Starting Hermes..."
hermes start > hermes.log 2>&1 &

sleep 1

# Detect the actual connection IDs assigned by the chains
CONNECTION_ID=$(
  $BINARY q ibc connection connections --node "tcp://127.0.0.1:$RPC" -o json \
  | jq -r '.connections[] | select(.id != "connection-localhost") | .id' | head -n1
)

# CONNECTION_ID="connection-0"
HOST_CONNECTION_ID=$($SOURCEHUB_BIN q ibc connection connections --node "$SOURCEHUB_RPC" -o json | jq -r '.connections | last | .id')

echo "==> Detected connection IDs:"
echo "Controller connection: $CONNECTION_ID"
echo "Host connection: $HOST_CONNECTION_ID"

CTRL_HOME=$HOME_DIR
CTRL_RPC="tcp://127.0.0.1:$RPC"
CTRL_CHAIN_ID=$CHAIN_ID
ICA_VERSION=$(cat <<EOF
{
  "version": "ics27-1",
  "controller_connection_id": "$CONNECTION_ID",
  "host_connection_id": "$HOST_CONNECTION_ID",
  "address": "",
  "encoding": "proto3",
  "tx_type": "sdk_multi_msg"
}
EOF
)

echo "==> Registering interchain account on $CTRL_CHAIN_ID (controller) via $CONNECTION_ID"
echo "Using version metadata: $ICA_VERSION"
$BINARY tx interchain-accounts controller register "$CONNECTION_ID" \
  --from $KEY \
  --keyring-backend test \
  --chain-id $CHAIN_ID \
  --home "$CTRL_HOME" \
  --node "$CTRL_RPC" \
  --ordering ORDER_ORDERED \
  --version "$ICA_VERSION" \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 5000ushinzo \
  --yes

echo "==> Waiting for ICA handshake to complete..."
sleep 30

# echo "==> Checking Hermes relayer balances..."

# hermes keys list --chain $CHAIN_ID
# hermes keys list --chain $SOURCEHUB_CHAIN_ID

# echo "Hermes Shinzohub address: $HERMES_SHINZO_ADDR"
# echo "Hermes Sourcehub address: $HERMES_SOURCE_ADDR"

# echo "-- Balance on Shinzohub --"
# $BINARY q bank balances "shinzo1g4zla6r5erlf364x5lcgvff6rmek4uwxwlfzs8" --node "tcp://127.0.0.1:$RPC" -o json

# echo "-- Balance on Sourcehub --"
# $SOURCEHUB_BIN q bank balances "source1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme" --node "$SOURCEHUB_RPC" -o json

# sleep 10

# # On controller (shinzohub)

# echo "==> Querying IBC channels on controller (Shinzohub)..."
# $BINARY q ibc channel channels --node "tcp://127.0.0.1:$RPC" -o json

# echo "==> Querying IBC channels on host (Sourcehub)..."
# $SOURCEHUB_BIN q ibc channel channels --node "$SOURCEHUB_RPC" -o json

echo "==> Querying host ICA address..."
ICA_ADDR=$($BINARY q interchain-accounts controller interchain-account $OWNER_ADDR "$CONNECTION_ID" \
  --node "$CTRL_RPC" -o json | jq -r '.address')
echo "Host ICA address: $ICA_ADDR"

if [ -z "$ICA_ADDR" ] || [ "$ICA_ADDR" = "null" ]; then
  echo "Failed to resolve ICA address. Check Hermes and connection IDs."
  exit 1
fi

echo "==> Preparing ICA packet with ACP MsgCreatePolicy..."
go run ./scripts/ibc/ica_packet_gen --creator "$ICA_ADDR" --policy "$POLICY_CONTENT" > $ICA_PACKET_JSON

echo "Packet:
$(cat $ICA_PACKET_JSON)"

echo "==> Sending ICA tx packet from controller..."
$BINARY tx interchain-accounts controller send-tx $HOST_CONNECTION_ID $ICA_PACKET_JSON \
  --from $OWNER_ADDR \
  --keyring-backend test \
  --chain-id $CHAIN_ID \
  --home "$CTRL_HOME" \
  --node "$CTRL_RPC" \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 9000ushinzo \
  --yes

echo "Waiting for host execution..."
sleep 10

echo "==> Querying ACP policy IDs on host..."
$SOURCEHUB_BIN q acp policy-ids --node "$SOURCEHUB_RPC" -o json || true

echo "DONE"

