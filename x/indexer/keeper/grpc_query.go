package keeper

import (
	"context"
	"errors"

	"github.com/shinzonetwork/shinzohub/x/indexer/types"
)

func errUnimplemented(name string) error { return errors.New(name + " not implemented yet") }

type queryServer struct {
	Keeper
}

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

// TODO: real implementations land in the queries commit.

func (q queryServer) Indexers(
	_ context.Context,
	_ *types.QueryIndexersRequest,
) (*types.QueryIndexersResponse, error) {
	return nil, errUnimplemented("Indexers")
}

func (q queryServer) Indexer(
	_ context.Context,
	_ *types.QueryIndexerRequest,
) (*types.QueryIndexerResponse, error) {
	return nil, errUnimplemented("Indexer")
}

func (q queryServer) IndexerByAddress(
	_ context.Context,
	_ *types.QueryIndexerByAddressRequest,
) (*types.QueryIndexerByAddressResponse, error) {
	return nil, errUnimplemented("IndexerByAddress")
}

func (q queryServer) IndexerCount(
	_ context.Context,
	_ *types.QueryIndexerCountRequest,
) (*types.QueryIndexerCountResponse, error) {
	return nil, errUnimplemented("IndexerCount")
}
