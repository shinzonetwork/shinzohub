package keeper

import (
	"context"
	"errors"

	"github.com/shinzonetwork/shinzohub/x/indexer/types"
)

func errUnimplemented(name string) error { return errors.New(name + " not implemented yet") }

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

// TODO: real implementations land in the next commit.

func (m msgServer) AddIndexerAssertion(
	_ context.Context,
	_ *types.MsgIndexerAssertion,
) (*types.MsgIndexerAssertionResponse, error) {
	return nil, errUnimplemented("AddIndexerAssertion")
}

func (m msgServer) SetPayout(
	_ context.Context,
	_ *types.MsgSetPayout,
) (*types.MsgSetPayoutResponse, error) {
	return nil, errUnimplemented("SetPayout")
}

func (m msgServer) RevokeIndexer(
	_ context.Context,
	_ *types.MsgRevokeIndexer,
) (*types.MsgRevokeIndexerResponse, error) {
	return nil, errUnimplemented("RevokeIndexer")
}
