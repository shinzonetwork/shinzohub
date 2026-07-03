package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	settlementkeeper "github.com/shinzonetwork/shinzohub/x/settlement/keeper"
	"github.com/shinzonetwork/shinzohub/x/settlement/types"
)

func TestMsgClaim_HappyPath(t *testing.T) {
	f := newFixture(t)
	claimer := addr(3)
	require.NoError(t, f.keeper.Credit(f.ctx, claimer, math.NewInt(1_000_000)))

	srv := settlementkeeper.NewMsgServerImpl(f.keeper)
	resp, err := srv.Claim(sdk.WrapSDKContext(f.ctx), &types.MsgClaim{
		Claimer: claimer.String(),
		Amount:  "750000",
	})

	require.NoError(t, err)
	require.Equal(t, "250000", resp.Remaining)
	require.Equal(t, math.NewInt(250_000), f.keeper.GetBalance(f.ctx, claimer))

	require.Len(t, f.bank.moves, 2, "msg server claim must mint then transfer via keeper")
	require.Equal(t, "mint", f.bank.moves[0].kind)
	require.Equal(t, "out", f.bank.moves[1].kind)
	require.Equal(t, claimer.String(), f.bank.moves[1].to)
}

func TestMsgClaim_RejectsInvalidBech32(t *testing.T) {
	f := newFixture(t)
	srv := settlementkeeper.NewMsgServerImpl(f.keeper)

	_, err := srv.Claim(sdk.WrapSDKContext(f.ctx), &types.MsgClaim{
		Claimer: "not-a-bech32-address",
		Amount:  "100",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "claimer")
}

func TestMsgClaim_RejectsNonIntegerAmount(t *testing.T) {
	f := newFixture(t)
	claimer := addr(3)
	srv := settlementkeeper.NewMsgServerImpl(f.keeper)

	_, err := srv.Claim(sdk.WrapSDKContext(f.ctx), &types.MsgClaim{
		Claimer: claimer.String(),
		Amount:  "abc",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "not an integer")
}

func TestMsgClaim_BubblesInsufficientFromKeeper(t *testing.T) {
	f := newFixture(t)
	claimer := addr(3)
	require.NoError(t, f.keeper.Credit(f.ctx, claimer, math.NewInt(50)))

	srv := settlementkeeper.NewMsgServerImpl(f.keeper)
	_, err := srv.Claim(sdk.WrapSDKContext(f.ctx), &types.MsgClaim{
		Claimer: claimer.String(),
		Amount:  "100",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "insufficient")
	require.Equal(t, math.NewInt(50), f.keeper.GetBalance(f.ctx, claimer),
		"failed claim must leave the pending balance untouched")
}

func TestMsgClaim_BubblesUnknownAddressFromKeeper(t *testing.T) {
	f := newFixture(t)
	claimer := addr(7)
	srv := settlementkeeper.NewMsgServerImpl(f.keeper)

	_, err := srv.Claim(sdk.WrapSDKContext(f.ctx), &types.MsgClaim{
		Claimer: claimer.String(),
		Amount:  "1",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "no settlement balance")
}

func TestMsgClaim_BubblesNonPositiveFromKeeper(t *testing.T) {
	f := newFixture(t)
	claimer := addr(3)
	require.NoError(t, f.keeper.Credit(f.ctx, claimer, math.NewInt(100)))

	srv := settlementkeeper.NewMsgServerImpl(f.keeper)

	_, errZero := srv.Claim(sdk.WrapSDKContext(f.ctx), &types.MsgClaim{
		Claimer: claimer.String(),
		Amount:  "0",
	})
	require.ErrorContains(t, errZero, "positive")

	_, errNeg := srv.Claim(sdk.WrapSDKContext(f.ctx), &types.MsgClaim{
		Claimer: claimer.String(),
		Amount:  "-1",
	})
	require.ErrorContains(t, errNeg, "positive")
}

func TestMsgClaim_RemainingMatchesKeeperBalance(t *testing.T) {
	f := newFixture(t)
	claimer := addr(3)
	require.NoError(t, f.keeper.Credit(f.ctx, claimer, math.NewInt(500)))

	srv := settlementkeeper.NewMsgServerImpl(f.keeper)
	resp, err := srv.Claim(sdk.WrapSDKContext(f.ctx), &types.MsgClaim{
		Claimer: claimer.String(),
		Amount:  "200",
	})

	require.NoError(t, err)
	require.Equal(t, f.keeper.GetBalance(f.ctx, claimer).String(), resp.Remaining)
}
