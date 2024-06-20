package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMain(m *testing.M) {
	accountPubKeyPrefix := "source" + "pub"
	validatorAddressPrefix := "source" + "valoper"
	validatorPubKeyPrefix := "source" + "valoperpub"
	consNodeAddressPrefix := "source" + "valcons"
	consNodePubKeyPrefix := "source" + "valconspub"

	// Set and seal config
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("souce", accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	config.Seal()
}
