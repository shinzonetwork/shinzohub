package osmoutils

import (
	context "context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"

	"cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// This function lets you run the function f, but if there's an error or panic
// drop the state machine change and log the error.
// If there is no error, proceeds as normal (but with some slowdown due to SDK store weirdness)
// Try to avoid usage of iterators in f.
//
// If its an out of gas panic, this function will also panic like in normal tx execution flow.
// This is still safe for beginblock / endblock code though, as they do not have out of gas panics.
func ApplyFuncIfNoError(ctx context.Context, f func(ctx context.Context) error) (err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return applyFunc(ctx, f, sdkCtx.Logger().Error)
}

// ApplyFuncIfNoErrorLogToDebug is the same as ApplyFuncIfNoError, but sends logs to debug instead of error if there is an error.
func ApplyFuncIfNoErrorLogToDebug(ctx context.Context, f func(ctx context.Context) error) (err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return applyFunc(ctx, f, sdkCtx.Logger().Debug)
}

func applyFunc(ctx context.Context, f func(ctx context.Context) error, logFunc func(string, ...interface{})) (err error) {
	// Add a panic safeguard
	defer func() {
		if recoveryError := recover(); recoveryError != nil {
			if isErr, _ := IsOutOfGasError(recoveryError); isErr {
				// We panic with the same error, to replicate the normal tx execution flow.
				panic(recoveryError)
			} else {
				PrintPanicRecoveryError(ctx, recoveryError)
				err = errors.New("panic occurred during execution")
			}
		}
	}()
	// makes a new cache context, which all state changes get wrapped inside of.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cacheCtx, write := sdkCtx.CacheContext()
	err = f(cacheCtx)
	if err != nil {
		logFunc(err.Error())
	} else {
		// no error, write the output of f
		write()
	}
	return err
}

// Frustratingly, this has to return the error descriptor, not an actual error itself
// because the SDK errors here are not actually errors. (They don't implement error interface)
func IsOutOfGasError(err any) (bool, string) {
	switch e := err.(type) {
	case types.ErrorOutOfGas:
		return true, e.Descriptor
	case types.ErrorGasOverflow:
		return true, e.Descriptor
	default:
		return false, ""
	}
}

// PrintPanicRecoveryError error logs the recoveryError, along with the stacktrace, if it can be parsed.
// If not emits them to stdout.
func PrintPanicRecoveryError(ctx context.Context, recoveryError interface{}) {
	logger := sdk.UnwrapSDKContext(ctx).Logger()
	errStackTrace := string(debug.Stack())
	switch e := recoveryError.(type) {
	case types.ErrorOutOfGas:
		logger.Debug("out of gas error inside panic recovery block: " + e.Descriptor)
		return
	case string:
		logger.Error("Recovering from (string) panic: " + e)
	case runtime.Error:
		logger.Error("recovered (runtime.Error) panic: " + e.Error())
	case error:
		logger.Error("recovered (error) panic: " + e.Error())
	default:
		logger.Error("recovered (default) panic. Could not capture logs in ctx, see stdout")
		fmt.Println("Recovering from panic ", recoveryError)
		debug.PrintStack()
		return
	}
	logger.Error("stack trace: " + errStackTrace)
}
