package api

import (
	"context"
	"shinzohub/pkg/sourcehub"
	"shinzohub/pkg/validators"
)

type ShinzoRegistrar struct {
	Validator validators.Validator
	Acp       sourcehub.ShinzoAcpClient
}

const IndexerGroup string = "Indexer"
const HostGroup string = "Host"

func (registrar *ShinzoRegistrar) RequestIndexerRole(ctx context.Context, did string) error {
	err := registrar.Validator.ValidateDid(did)
	if err != nil {
		return err
	}

	err = registrar.Acp.AddToGroup(ctx, IndexerGroup, did)
	if err != nil {
		return err
	}

	return nil
}

func (registrar *ShinzoRegistrar) RequestHostRole(ctx context.Context, did string) error {
	err := registrar.Validator.ValidateDid(did)
	if err != nil {
		return err
	}

	err = registrar.Acp.AddToGroup(ctx, HostGroup, did)
	if err != nil {
		return err
	}

	return nil
}

func (registrar *ShinzoRegistrar) BlockIndexer(ctx context.Context, did string) error {
	err := registrar.Validator.ValidateDid(did)
	if err != nil {
		return err
	}

	err = registrar.Acp.BlockFromGroup(ctx, IndexerGroup, did)
	if err != nil {
		return err
	}

	return nil
}

func (registrar *ShinzoRegistrar) BlockHost(ctx context.Context, did string) error {
	err := registrar.Validator.ValidateDid(did)
	if err != nil {
		return err
	}

	err = registrar.Acp.BlockFromGroup(ctx, HostGroup, did)
	if err != nil {
		return err
	}

	return nil
}

func (registrar *ShinzoRegistrar) SubscribeToDataFeed(ctx context.Context, did string, dataFeedId string) error {
	err := registrar.Validator.ValidateDid(did)
	if err != nil {
		return err
	}
	err = registrar.Validator.ValidateDataFeedId(dataFeedId)
	if err != nil {
		return err
	}

	err = registrar.Acp.GiveQueryAccess(ctx, dataFeedId, did)
	if err != nil {
		return err
	}

	return nil
}

func (registrar *ShinzoRegistrar) BanUserFromView(ctx context.Context, did string, dataFeedId string) error {
	err := registrar.Validator.ValidateDid(did)
	if err != nil {
		return err
	}
	err = registrar.Validator.ValidateDataFeedId(dataFeedId)
	if err != nil {
		return err
	}

	err = registrar.Acp.BanUserFromView(ctx, dataFeedId, did)
	if err != nil {
		return err
	}

	return nil
}

func (registrar *ShinzoRegistrar) CreateDataFeed(ctx context.Context, did string, dataFeedId string) error {
	err := registrar.Validator.ValidateDid(did)
	if err != nil {
		return err
	}
	err = registrar.Validator.ValidateDataFeedId(dataFeedId)
	if err != nil {
		return err
	}

	err = registrar.Acp.CreateDataFeed(ctx, dataFeedId, did)
	if err != nil {
		return err
	}

	return nil
}
