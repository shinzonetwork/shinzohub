package keeper

import (
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

func (k Keeper) SetPolicyId(ctx sdk.Context, txType string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set([]byte(types.PolicyId), []byte(txType))
}

func (k Keeper) GetPolicyId(ctx sdk.Context) string {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get([]byte(types.PolicyId))
	if bz == nil {
		return ""
	}
	return string(bz)
}

var policy = `
name: shinzo
resources:
  primitive:
    relations:
      admin:
        manages:
        - writer
        - syncer
        - subscriber
        - banned
        types:
        - actor
        - group->administrator
      writer:
        types:
        - actor
        - group->member
      syncer:
        types:
        - actor
        - group->member
      subscriber:
        types:
        - actor
      banned:
        types:
        - actor
    permissions:
      sync:
        expr: (writer + syncer + subscriber) - banned
      update:
        expr: writer - banned
      delete:
        expr: owner
      read:
        expr: (subscriber) - banned
  view:
    relations:
      admin:
        manages:
        - creator
        - writer
        - syncer
        - subscriber
        - parent
        - banned
        types:
        - actor
        - group->administrator
      creator:
        types:
        - actor
      writer:
        types:
        - actor
        - group->member
      syncer:
        types:
        - actor
        - group->member
      banned:
        types:
        - actor
      subscriber:
        types:
        - actor
      parent:
        types:
        - primitive
        - view
    permissions:
      sync:
        expr: writer + syncer + parent->sync + subscriber - banned
      update:
        expr: ((writer + parent->update)) - banned
      read:
        expr: (subscriber) - banned
      delete:
        expr: owner
      created:
        expr: creator - banned
  group:
    relations:
      owner:
        manages:
        - admin
        - guest
        - blocked
        types:
        - actor
      admin:
        manages:
        - guest
        - blocked
        types:
        - actor
        - group->administrator
      guest:
        types:
        - actor
      blocked:
        types:
        - actor
    permissions:
      member:
        expr: guest - blocked
      administrator:
        expr: (owner + admin) - blocked
`
