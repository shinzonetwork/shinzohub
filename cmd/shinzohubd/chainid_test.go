package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/shinzonetwork/shinzohub/app"
)

const testPrefix = "shinzohubd"

// writeHome creates a node home with a genesis for chainID and, when
// evmChainID is not empty, an app.toml that sets evm-chain-id.
func writeHome(t *testing.T, chainID, evmChainID string) string {
	t.Helper()
	home := t.TempDir()
	config := filepath.Join(home, "config")
	require.NoError(t, os.MkdirAll(config, 0o755))

	genesis := genutiltypes.NewAppGenesisWithVersion(chainID, json.RawMessage(`{}`))
	require.NoError(t, genesis.SaveAs(filepath.Join(config, "genesis.json")))

	if evmChainID != "" {
		appToml := fmt.Sprintf("[evm]\nevm-chain-id = %s\n", evmChainID)
		require.NoError(t, os.WriteFile(filepath.Join(config, "app.toml"), []byte(appToml), 0o644))
	}
	return home
}

// serverViper builds a viper the way the SDK builds the server's: flags bound,
// environment variables under the binary's prefix, and app.toml from home.
func serverViper(t *testing.T, args []string) *viper.Viper {
	t.Helper()
	fs := pflag.NewFlagSet("start", pflag.ContinueOnError)
	fs.String(flags.FlagHome, app.DefaultNodeHome, "")
	fs.String(flags.FlagChainID, "", "")
	require.NoError(t, fs.Parse(args))

	v := viper.New()
	require.NoError(t, v.BindPFlags(fs))
	v.SetEnvPrefix(testPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	appToml := filepath.Join(v.GetString(flags.FlagHome), "config", "app.toml")
	if _, err := os.Stat(appToml); err == nil {
		v.SetConfigFile(appToml)
		require.NoError(t, v.ReadInConfig())
	}
	return v
}

func TestEVMChainID(t *testing.T) {
	cases := []struct {
		name       string
		genesis    string
		evmChainID string
		args       func(home string) []string
		want       uint64
		wantErr    string
	}{
		{
			name:    "genesis",
			genesis: "91273002",
			args:    func(home string) []string { return []string{"--home", home} },
			want:    91273002,
		},
		{
			name:       "app.toml agrees",
			genesis:    "91273002",
			evmChainID: "91273002",
			args:       func(home string) []string { return []string{"--home", home} },
			want:       91273002,
		},
		{
			name:       "app.toml disagrees",
			genesis:    "91273002",
			evmChainID: "262144",
			args:       func(home string) []string { return []string{"--home", home} },
			wantErr:    "evm-chain-id in app.toml is 262144",
		},
		{
			name:    "flag overrides genesis",
			genesis: "91273002",
			args:    func(home string) []string { return []string{"--home", home, "--chain-id", "91273001"} },
			want:    91273001,
		},
		{
			name:    "named genesis id",
			genesis: "shinzohub-dev",
			args:    func(home string) []string { return []string{"--home", home} },
			wantErr: `chain id "shinzohub-dev" is not a number`,
		},
		{
			name:    "missing genesis",
			args:    func(string) []string { return []string{"--home", t.TempDir()} },
			wantErr: "read chain id from",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if tc.genesis != "" {
				home = writeHome(t, tc.genesis, tc.evmChainID)
			}
			got, err := evmChainID(serverViper(t, tc.args(home)))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// The id that seals the EVM config before cobra runs must match the id the
// node resolves afterwards, whichever way home and the chain id are given.
func TestResolversAgree(t *testing.T) {
	flagHome := writeHome(t, "91273002", "")
	envHome := writeHome(t, "91273003", "")

	cases := []struct {
		name string
		args []string
		env  map[string]string
		want string
	}{
		{name: "home flag", args: []string{"--home", flagHome}, want: "91273002"},
		{name: "home flag with equals", args: []string{"--home=" + flagHome}, want: "91273002"},
		{name: "home env", env: map[string]string{"SHINZOHUBD_HOME": envHome}, want: "91273003"},
		{
			name: "home flag beats env",
			args: []string{"--home", flagHome},
			env:  map[string]string{"SHINZOHUBD_HOME": envHome},
			want: "91273002",
		},
		{
			name: "chain id env beats genesis",
			args: []string{"--home", flagHome},
			env:  map[string]string{"SHINZOHUBD_CHAIN_ID": "91273001"},
			want: "91273001",
		},
		{
			name: "chain id flag beats env",
			args: []string{"--home", flagHome, "--chain-id", "91273000"},
			env:  map[string]string{"SHINZOHUBD_CHAIN_ID": "91273001"},
			want: "91273000",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SHINZOHUBD_HOME", "")
			t.Setenv("SHINZOHUBD_CHAIN_ID", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			before, err := resolveChainID(inputsFromArgs(append([]string{"shinzohubd", "start"}, tc.args...), os.Getenv, testPrefix))
			require.NoError(t, err)
			after, err := resolveChainID(inputsFromAppOpts(serverViper(t, tc.args)))
			require.NoError(t, err)

			require.Equal(t, tc.want, before, "resolved before cobra")
			require.Equal(t, tc.want, after, "resolved from viper")
		})
	}
}

func TestInputsFromArgsDefaultsHome(t *testing.T) {
	in := inputsFromArgs([]string{"shinzohubd", "keys", "list"}, func(string) string { return "" }, testPrefix)
	require.Equal(t, app.DefaultNodeHome, in.home)
	require.Empty(t, in.chainID)
}

// A new app.toml must carry the id the process sealed, or evmChainID refuses
// to start the node it was written for.
func TestInitAppConfigWritesSealedID(t *testing.T) {
	_, cfg := initAppConfig(91273005)
	custom, ok := cfg.(CustomAppConfig)
	require.True(t, ok)
	require.Equal(t, uint64(91273005), custom.EVM.EVMChainID)
}
