package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func NewParams(admin string) Params {
	if admin == "" {
		admin = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	}
	if _, err := sdk.AccAddressFromBech32(admin); err != nil {
		panic(err)
	}
	return Params{Admin: admin}
}

func DefaultParams() Params {
	return NewParams(authtypes.NewModuleAddress(govtypes.ModuleName).String())
}
