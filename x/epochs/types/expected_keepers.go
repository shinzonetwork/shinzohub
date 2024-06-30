package types

// EpochsHooksWrapper is a wrapper for modules to inject EpochsHooks using depinject.
type EpochsHooksWrapper struct{ EpochHooks }

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (EpochsHooksWrapper) IsOnePerModuleType() {}
