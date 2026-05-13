# x/indexer

Tracks Shinzo indexers. One row per validator (keyed by `(source_chain_id,
validator_pubkey)`), filled in over two phases.

## Phase A: assertion (admin / relayer)

`MsgIndexerAssertion` mirrors an outpost `IndexerDelegated` event from the
validator's source chain. The relayer (or admin acting as the relayer in v1)
submits it. The row is created or updated with the validator-side facts
(`validator_pubkey`, `assertion_authority`, `nonce`, `chain_specific`) and the
delegation facts (`operator_address`, `payout_address`). At this point the row
exists but `registered = false`. The operator hasn't claimed the slot yet.

Two companion messages from the same trust domain:

- `MsgSetPayout` — payout-only update. Operator-side fields untouched.
- `MsgRevokeIndexer` — tears the row down without a replacement.

All three share a single monotonic `nonce` per `(source_chain_id, validator_pubkey)`.

## Phase B: registration (operator)

`IndexerRegistry.register` (EVM precompile) is called by the operator from its
own wallet. The operator supplies a node identity key (separate from the
operator/delegate key — used only for DID derivation), proof of possession,
and a connection string. The row's operator-side fields flip:
`registered = true`, `did = derive(node_identity_pubkey)`,
`connection_string` set. The keeper fires an ICA `SetRelationshipCmd` to
sourcehub on first registration. The IBC ack is fire-and-forget — there is no
pending-then-promote state machine; the row is authoritative the moment the
precompile tx commits.

## Rotation, payout change, revocation

Rotation: a fresh `MsgIndexerAssertion` for the same `(source_chain_id,
validator_pubkey)` with a different `operator_address`. The old operator's
`addr_idx` entry is deleted, the row's operator-side fields reset to pending,
the new operator must call `register` again. There is no parallel "old
operator" row left behind — the assertion overwrites.

Payout-only: `MsgSetPayout` updates `payout_address` and bumps `nonce`. Does
not touch operator-side fields.

Revocation: `MsgRevokeIndexer` deletes the row and inverse index entries.

## Store layout

```
indexer/<source_chain_id>/<validator_pubkey>  →  Indexer proto bytes
addr_idx/<operator_address>                    →  (source_chain_id, validator_pubkey)
indexer_count                                   →  uint64
```

The only inverse index is by operator address, used by the precompile to
resolve "which validator does this caller serve?" and by the existing
address-keyed read views (`IsRegistered(addr)` etc.).

## Chain-agnostic by design

`validator_pubkey`, `assertion_authority`, and `chain_specific` are all
`bytes`. Shinzohub never inspects their shape. The relayer is the only thing
that knows how to encode them per source chain (Ethereum BLS, Solana ed25519,
Cosmos ed25519, etc.). `chain_specific` is a catch-all for per-chain audit
data (e.g. on Ethereum: a proto-encoded `{validator_index, proven_at_root,
assertion_id}` from the outpost event) — stored opaquely, used by off-chain
verifiers.

Trust model: admin signs the cosmos tx and is trusted to faithfully relay
outpost events. Cryptographic verification of validator existence happens on
the source chain's outpost contract, not on shinzohub.
