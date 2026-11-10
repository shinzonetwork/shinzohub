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

// The chain id is resolved twice per process. NewRootCmd needs it before cobra
// has parsed anything, because the throwaway app built there seals the global
// EVM config. newApp and appExport need it again from the server's viper. If
// the two disagree, checkSealedChainID stops the node, so both go through
// resolveChainID and only differ in where they read their inputs.

// chainIDInputs is what the chain id is resolved from: an explicit chain id,
// or else the genesis file under home.
type chainIDInputs struct {
	chainID string
	home    string
}

// resolveChainID returns the explicit chain id when there is one, and the
// chain id in home's genesis file otherwise.
func resolveChainID(in chainIDInputs) (string, error) {
	if in.chainID != "" {
		return in.chainID, nil
	}
	if in.home == "" {
		return "", errors.New("no --chain-id and no home directory, cannot resolve the evm chain id")
	}
	return genesisChainID(in.home)
}

// inputsFromArgs reads the inputs the way the server's viper will once cobra
// runs: the flag first, then the environment variable, then the default home.
// The SDK names those variables after the binary, like SHINZOHUBD_HOME and
// SHINZOHUBD_CHAIN_ID.
func inputsFromArgs(args []string, getenv func(string) string, envPrefix string) chainIDInputs {
	lookup := func(name string) string {
		if v := argValue(args, name); v != "" {
			return v
		}
		return envValue(getenv, envPrefix, name)
	}

	in := chainIDInputs{
		chainID: lookup(flags.FlagChainID),
		home:    lookup(flags.FlagHome),
	}
	if in.home == "" {
		in.home = app.DefaultNodeHome
	}
	return in
}

// inputsFromAppOpts reads the inputs from the server's viper, which has
// already merged flags, environment variables and config files.
func inputsFromAppOpts(appOpts servertypes.AppOptions) chainIDInputs {
	return chainIDInputs{
		chainID: cast.ToString(appOpts.Get(flags.FlagChainID)),
		home:    cast.ToString(appOpts.Get(flags.FlagHome)),
	}
}

// evmChainIDFromArgs resolves the chain id for the throwaway app in
// NewRootCmd, whose id is what eth_chainId reports and what the ante handler
// enforces from then on. Commands with no genesis, like init and keys, fall
// back to the default. A node that falls back here fails checkSealedChainID
// with the real reason from evmChainID.
func evmChainIDFromArgs() uint64 {
	in := inputsFromArgs(os.Args, os.Getenv, binaryName())
	chainID, err := resolveChainID(in)
	if err != nil {
		return app.DefaultEVMChainID
	}
	id, err := app.ParseChainID(chainID)
	if err != nil {
		return app.DefaultEVMChainID
	}
	return id
}

// evmChainID resolves the chain id for a node that is about to serve or
// export. It rejects an app.toml that disagrees, since evm-chain-id gates the
// JSON RPC send path while eth_chainId and the ante handler read the sealed
// config.
func evmChainID(appOpts servertypes.AppOptions) (uint64, error) {
	chainID, err := resolveChainID(inputsFromAppOpts(appOpts))
	if err != nil {
		return 0, err
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

// envValue reads a flag's environment variable the way the server's viper
// does. AutomaticEnv looks up PREFIX_NAME in upper case, and the SDK also
// binds prefix_NAME with the binary name as is, checked in that order.
func envValue(getenv func(string) string, prefix, name string) string {
	if prefix == "" {
		return ""
	}
	suffix := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	if v := getenv(strings.ToUpper(prefix) + "_" + suffix); v != "" {
		return v
	}
	return getenv(prefix + "_" + suffix)
}

// binaryName is the prefix the SDK gives environment variables, the base name
// of the running executable.
func binaryName() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Base(exe)
}

func genesisChainID(home string) (string, error) {
	path := filepath.Join(home, "config", "genesis.json")
	genesis, err := genutiltypes.AppGenesisFromFile(path)
	if err != nil {
		return "", fmt.Errorf("read chain id from %s: %w", path, err)
	}
	return genesis.ChainID, nil
}
