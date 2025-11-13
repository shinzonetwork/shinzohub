# ShinzoHub Indexer & Host Registration Guide
## Indexer & Host Registration Guide

This guide explains how **Indexers** and **Hosts** register themselves with **ShinzoHub** using the `EntityRegistry` precompile at `0x0000000000000000000000000000000000000211`, and how that flows through **ICA** to create/update records on **SourceHub** (groups, objects, and memberships). It’s designed to sit alongside your existing IBC + ICA demo.

> TLDR: Each node (indexer or host) proves control of a **peer key** and a **node key** by submitting both public keys and ECDSA signatures over a shared message. The precompile verifies the cryptography, emits on‑chain events, and triggers an ICA batch that updates the ACP policy on SourceHub, adding the node to the right **Group** (`indexers` or `host`).

---

## 0) Prerequisites

- Follow the **setup-ibc-demo.md** Readme and set it up.
- Foundry `cast` installed for calldata encoding and quick keccak/decoding.

---

## 1) What is being registered?

Two roles:
- **Indexer** — participates in indexing and query execution.
- **Host** — provides hosting/serving for view materialization and API gateways.

On ShinzoHub the registration call goes to the **Entity precompile** at `0x0000000000000000000000000000000000000211`:
```solidity
// Example signature (adjust if your interface differs)
function register(
  bytes peerPub,  // secp256k1 uncompressed or 32/33‑byte variant as you define
  bytes peerSig,  // ECDSA signature over the shared msg
  bytes nodePub,  // secp256k1 uncompressed or 32/33‑byte variant
  bytes nodeSig,  // ECDSA signature over the shared msg
  bytes message,  // shared message/message all parties sign
  uint8 entity    // 1 = Indexer, 2 = Host  (example mapping)
) external;
```

If signatures check out, the contract:
1. Emits a **Registration** event with `did`, `pid`, and the entity type.
2. Sends an **ICA** packet to SourceHub with batched ACP commands to add the registering node to the corresponding group (`indexers` or `host`).

---

## 2) Environment variables

```bash
export RPC_URL=http://localhost:8545
export FROM_ADDR=0xabd39bcd18199976acf5379450c52f06edbcf4f3
export PRECOMPILE_ADDR=0x0000000000000000000000000000000000000211
export GAS_HEX=0x100000
# 0 = Indexer, 1 = Host (tweak to match your enum)
export ENTITY=1
```

---

## 3) Registration request example

This is your working example (kept verbatim, only annotated). It registers **ENTITY=1** (Indexer).

```bash
#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://localhost:8545}"
FROM_ADDR="${FROM_ADDR:-0xabd39bcd18199976acf5379450c52f06edbcf4f3}"
PRECOMPILE_ADDR="0x0000000000000000000000000000000000000211"
GAS_HEX="0x100000"
ENTITY=1 

# secp256k1 public key of the P2P identity (libp2p or peer key)
PEER_PUB="0x703896c8fc429d0af204513a76a067b170ba71bf0be5ca8184e16ffce5b9732b"
# ECDSA signature over MESSAGE with the peer private key
PEER_SIG="0xf365e86878959ab3de294f92ad90644726b1be4978b31250a1a01da5c50c87fecafd0c63a051e81544bfcd69a99301f9c48ef79808a93dd645f61c251533880f"

# secp256k1 public key of the node (runtime/executor) identity
NODE_PUB="0x041871f34ea7a26aa3dfa831b1e03681ec1bc99a0dcf9e8b4fd3f450c46462285db9f5f07bb582ff21239ed724397896f2fc8c6f1c86871132786491f616828056"
# ECDSA signature over MESSAGE with the node private key
NODE_SIG="0x3045022100bca215bd97cc3f27573e7cda7a0a05e452d397643b4962581a5512bd7453e17e022067a01b68663b68533b544959ea3966feda2ae69478345cc4822cdfedd971cdb0"

# Replay‑protection message; here literal ascii "entity-registration-test-message"
MESSAGE="0x656e746974792d726567697374726174696f6e2d746573742d6e6f6e6365"

DATA=$(cast calldata \
  "register(bytes,bytes,bytes,bytes,bytes,uint8)" \
  "$PEER_PUB" \
  "$PEER_SIG" \
  "$NODE_PUB" \
  "$NODE_SIG" \
  "$MESSAGE" \
  $ENTITY)

curl -s -X POST "$RPC_URL" \
  -H "Content-Type: application/json" \
  --data-raw "{
    \"jsonrpc\":\"2.0\",
    \"method\":\"eth_sendTransaction\",
    \"params\":[{
      \"from\":\"$FROM_ADDR\",
      \"to\":\"$PRECOMPILE_ADDR\",
      \"gas\":\"$GAS_HEX\",
      \"value\":\"0x0\",
      \"data\":\"$DATA\"
    }],
    \"id\":1
  }" | jq .
```

**Switching to Host:** set `ENTITY=1` (or whatever enum you use for Host) and re‑run.

---

## 4) Verifying success

### 4.1 EVM receipt (ShinzoHub)
After you get a tx hash, fetch the receipt:
```bash
cast receipt <TX_HASH> --rpc-url $RPC_URL --json | jq
# or:
curl -s -X POST "$RPC_URL" -H "Content-Type: application/json" \
 --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getTransactionReceipt\",\"params\":[\"<TX_HASH>\"],\"id\":1}"
```

You should see:
- `status: 0x1`
- `to: 0x…0211`
- at least one log from `address: 0x…0211`

Optionally decode the event:
```bash
cast decode-logs --abi <(cat <<'ABI'
[
  "event Registered(string did,string idHash,address indexed sender,uint8 entity)"
]
ABI
) <TX_HASH> --rpc-url $RPC_URL
```
