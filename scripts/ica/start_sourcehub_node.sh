#!/bin/sh

set -e

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

# Cleanup directories
rm -rf $SOURCEHUB_HOME_DIR

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

echo "==> Starting Sourcehub..."
$SOURCEHUB_BIN start --home $SOURCEHUB_HOME_DIR --rpc.laddr $SOURCEHUB_RPC --rpc.pprof_laddr $SOURCEHUB_PPROF --p2p.laddr $SOURCEHUB_P2P --grpc.address $SOURCEHUB_GRPC --address $SOURCEHUB_ADDR
