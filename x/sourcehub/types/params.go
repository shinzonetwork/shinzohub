package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(admin string) Params {
	if admin == "" {
		admin = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	}

	if _, err := sdk.AccAddressFromBech32(admin); err != nil {
		panic(err)
	}

	return Params{
		Admin: admin,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	// yes, this sets gov 2 times if we use defaults, rather than setting empty
	return NewParams(authtypes.NewModuleAddress(govtypes.ModuleName).String())
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params
func (p *Params) Validate() error {
	return nil
}
