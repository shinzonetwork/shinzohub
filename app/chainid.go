package app

import (
	"fmt"
	"strconv"
)

// DefaultEVMChainID is a placeholder for commands that run without a chain id,
// like init and keys. A node always resolves its own with ParseChainID.
const DefaultEVMChainID uint64 = 91273001

// ParseChainID returns the EIP 155 chain id for a cosmos chain id. Every shinzo
// network uses the number itself as its chain id, so this is a plain parse.
func ParseChainID(chainID string) (uint64, error) {
	id, err := strconv.ParseUint(chainID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("chain id %q is not a number, %w", chainID, err)
	}
	if id == 0 {
		return 0, fmt.Errorf("chain id %q resolves to zero", chainID)
	}
	return id, nil
}
