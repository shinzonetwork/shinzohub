package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	srvflags "github.com/cosmos/evm/server/flags"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/spf13/cast"

	"github.com/shinzonetwork/shinzohub/app"
)

// evmChainIDFromArgs resolves the chain id before cobra has parsed anything.
//
// The EVM config is global and the first app built in the process seals it.
// That app is the throwaway one in NewRootCmd, so its id is what eth_chainId
// reports and what the ante handler enforces from then on. Commands with no
// genesis, like init and keys, fall back to the default.
func evmChainIDFromArgs() uint64 {
	if chainID := argValue(os.Args, flags.FlagChainID); chainID != "" {
		if id, err := app.ParseChainID(chainID); err == nil {
			return id
		}
	}

	home := argValue(os.Args, flags.FlagHome)
	if home == "" {
		home = app.DefaultNodeHome
	}
	if chainID, err := genesisChainID(home); err == nil {
		if id, err := app.ParseChainID(chainID); err == nil {
			return id
		}
	}

	return app.DefaultEVMChainID
}

// argValue reads --name value and --name=value out of raw arguments.
func argValue(args []string, name string) string {
	flag := "--" + name
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, flag+"=") {
			return strings.TrimPrefix(arg, flag+"=")
		}
	}
	return ""
}

func genesisChainID(home string) (string, error) {
	path := filepath.Join(home, "config", "genesis.json")
	genesis, err := genutiltypes.AppGenesisFromFile(path)
	if err != nil {
		return "", fmt.Errorf("read chain id from %s, %w", path, err)
	}
	return genesis.ChainID, nil
}

// evmChainID resolves the chain id for a node that is about to serve, from the
// --chain-id flag or else from genesis. It rejects an app.toml that disagrees,
// since evm-chain-id gates the JSON RPC send path while eth_chainId and the
// ante handler read the sealed config.
func evmChainID(appOpts servertypes.AppOptions) (uint64, error) {
	chainID := cast.ToString(appOpts.Get(flags.FlagChainID))
	if chainID == "" {
		home := cast.ToString(appOpts.Get(flags.FlagHome))
		if home == "" {
			return 0, errors.New("no --chain-id and no home directory, cannot resolve the evm chain id")
		}
		fromGenesis, err := genesisChainID(home)
		if err != nil {
			return 0, err
		}
		chainID = fromGenesis
	}

	id, err := app.ParseChainID(chainID)
	if err != nil {
		return 0, err
	}

	if configured := cast.ToUint64(appOpts.Get(srvflags.EVMChainID)); configured != 0 && configured != id {
		return 0, fmt.Errorf(
			"evm-chain-id in app.toml is %d but chain id %q gives %d, set it to %d or remove it",
			configured, chainID, id, id,
		)
	}

	return id, nil
}

// checkSealedChainID confirms the global EVM config landed on id. It is set
// once per process, so an app built earlier with a different id wins and the
// node would report one id while enforcing another.
func checkSealedChainID(id uint64) error {
	sealed := evmtypes.GetEthChainConfig().ChainID
	if sealed == nil {
		return errors.New("the evm chain config was never set")
	}
	if !sealed.IsUint64() || sealed.Uint64() != id {
		return fmt.Errorf("the evm chain config is sealed on %s but this node runs %d", sealed, id)
	}
	return nil
}
