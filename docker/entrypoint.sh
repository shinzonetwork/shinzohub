#!/bin/bash

set -e 

DEFAULT_CHAIN_ID="shinzohub"
DEFAULT_MONIKER="node"

if [ ! -d /shinzohub/.initialized ]; then
    echo "Initializing ShinzoHub"

    if [ -z "$CHAIN_ID" ]; then 
        echo "CHAIN_ID not set: using default"
        CHAIN_ID=$DEFAULT_CHAIN_ID
    fi

    if [ -z "$MONIKER" ]; then 
        echo "MONIKER not set: using default"
        MONIKER=$DEFAULT_MONIKER
    fi

    shinzohubd init "$MONIKER" --chain-id $CHAIN_ID --default-denom="ushinzo" 2>/dev/null

    # recover account mnemonic
    if [ -n "$MNEMONIC_PATH" ]; then
        echo "MNEMONIC_PATH set: '$MNEMONIC_PATH': recovering key"
        test -s $MNEMONIC_PATH || (echo "error: mnemonic file is empty" && exit 1)
        echo $(cat $MNEMONIC_PATH) | shinzohubd keys add validator --recover --keyring-backend test
    fi

    # if consensus key is set, we recover the full
    # node, including p2p and consensus key
    if [ -n "$CONSENSUS_KEY_PATH" ]; then
        echo "CONSENSUS_KEY_PATH set: '$CONSENSUS_KEY_PATH': recovering validator"
        test -s $CONSENSUS_KEY_PATH || (echo "error: consensus key file is empty" && exit 1)
        test -s $COMET_NODE_KEY_PATH || (echo "error: comet node key file is empty" && exit 1)

        cp $CONSENSUS_KEY_PATH /shinzohub/config/priv_validator_key.json
        cp $COMET_NODE_KEY_PATH /shinzohub/config/node_key.json
    fi

    # initialize chain in standalone
    if [ "$STANDALONE" = "1" ]; then 
        echo "Standalone mode: generating new genesis"
        # initialize chain / create genesis
        shinzohubd keys add validator --keyring-backend test
        VALIDATOR_ADDR=$(shinzohubd keys show validator -a --keyring-backend test)
        shinzohubd genesis add-genesis-account $VALIDATOR_ADDR 1000000000000000000000ushinzo
        shinzohubd genesis gentx validator 1000000000000000000000ushinzo --chain-id $CHAIN_ID --keyring-backend test 
        shinzohubd genesis collect-gentxs
        echo "initialized shinzohub genesis"
        # TODO copy config files
    else 
        if [ -z "$GENESIS_PATH" ]; then
            echo "GENESIS_PATH not set and standalone is false: provide a genesis file or set env STANDALONE=1"
            exit 1
        fi
        cp $GENESIS_PATH /shinzohub/config/genesis.json
        echo "Loaded Genesis from $GENESIS_PATH"
    fi

    touch /shinzohub/.initialized
else
    echo "Skipping initialization: container previously initialized"
fi

if [ -n "$COMET_CONFIG_PATH" ]; then 
    echo "COMET_CONFIG_PATH set: updating comet config with $COMET_CONFIG_PATH"
    cp $COMET_CONFIG_PATH /shinzohub/config/config.toml
fi

if [ -n "$APP_CONFIG_PATH" ]; then 
    echo "APP_CONFIG_PATH set: updating app config with $APP_CONFIG_PATH"
    cp $APP_CONFIG_PATH /shinzohub/config/app.toml
fi

exec $@