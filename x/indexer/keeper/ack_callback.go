package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	indexertypes "github.com/shinzonetwork/shinzohub/x/indexer/types"
	sourcehubtypes "github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

type AckCallback struct {
	keeper Keeper
}

func NewAckCallback(k Keeper) AckCallback {
	return AckCallback{keeper: k}
}

func (c AckCallback) OnPacketAck(ctx sdk.Context, req sourcehubtypes.PendingICARequest) error {
	if req.Kind != sourcehubtypes.RequestKind_REQUEST_KIND_SET_RELATIONSHIP {
		return nil
	}

	var meta sourcehubtypes.SetRelationshipMeta
	if err := c.keeper.cdc.Unmarshal(req.Meta, &meta); err != nil {
		return fmt.Errorf("decode SetRelationshipMeta: %w", err)
	}
	if meta.Group != indexertypes.GroupIndexerName {
		return nil
	}

	claim, found, err := c.keeper.GetPendingClaim(ctx, meta.Did)
	if err != nil {
		return fmt.Errorf("read pending claim %s: %w", meta.Did, err)
	}
	if !found {
		return nil
	}

	switch req.Status {
	case sourcehubtypes.RequestStatus_REQUEST_STATUS_SUCCESS:
		if err := c.keeper.ApplyRegistration(ctx, claim.OperatorAddress, meta.Did, claim.ConnectionString); err != nil {
			return err
		}
		c.keeper.DeletePendingClaim(ctx, meta.Did)

	case sourcehubtypes.RequestStatus_REQUEST_STATUS_FAILURE,
		sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT:
		reason := req.Error
		if req.Status == sourcehubtypes.RequestStatus_REQUEST_STATUS_TIMEOUT {
			reason = "ica timeout"
		}
		c.keeper.DeletePendingClaim(ctx, meta.Did)
		emitRegistrationFailed(ctx, claim.OperatorAddress, meta.Did, reason)
	}
	return nil
}
