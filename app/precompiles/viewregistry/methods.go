package viewregistry

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	ViewRegistryRegisterMethod = "register"
	ViewRegistryGetMethod      = "get"
)

func (p Precompile) ViewRegistryRegister(method *abi.Method, args []interface{}) ([]byte, error) {
	return nil, fmt.Errorf("method not implemented")
}

func (p Precompile) ViewRegistryGet(method *abi.Method, args []interface{}) ([]byte, error) {
	return nil, fmt.Errorf("method not implemented")
}
