package policy

import (
	"strconv"

	"cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	rcdb "github.com/sourcenetwork/raccoondb"
)

// FIXME make this concurrent

const prefix = "policy/"
const key = "id_counter"

func newPolicyCounter(kv store.KVStore) policyCounter {
	return policyCounter{
		kv: kv,
	}
}

type policyCounter struct {
	kv store.KVStore
}

func (r *policyCounter) getStore() rcdb.KVStore {
	adapted := runtime.KVStoreAdapter(r.kv)
	kv := rcdb.KvFromCosmosKv(adapted)
	kv = rcdb.NewWrapperKV(kv, []byte(prefix))
	return kv
}

// GetFree returns the next free number in the counter
func (r *policyCounter) GetNext(ctx sdk.Context) (uint64, error) {
	kv := r.getStore()

	var currID uint64 = 1
	counter, err := kv.Get([]byte(key))
	if err != nil {
		return 0, err
	}
	if counter != nil {
		counterStr := string(counter)
		currID, err = strconv.ParseUint(counterStr, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	freeID := currID + 1
	return freeID, nil
}

// Increment updates the counter to the next free number
func (r *policyCounter) setCounter(ctx sdk.Context, counter uint64) error {
	kv := r.getStore()

	err := kv.Set([]byte(key), []byte(strconv.FormatUint(counter, 10)))
	if err != nil {
		return err
	}

	return nil
}

// Increment increments the counter by 1
func (r *policyCounter) Increment(ctx sdk.Context) error {
	_, err := r.GetNextAndIncrement(ctx)
	return err
}

// GetNextAndIncrement atomically gets the next free counter and increments it
func (r *policyCounter) GetNextAndIncrement(ctx sdk.Context) (uint64, error) {
	free, err := r.GetNext(ctx)
	if err != nil {
		return 0, err
	}

	err = r.setCounter(ctx, free)
	if err != nil {
		return 0, err
	}

	return free, nil
}
