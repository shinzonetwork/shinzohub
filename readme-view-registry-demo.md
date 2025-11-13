# ShinzoHub ViewRegistry Precompile Demo

This demo shows how to use the custom **ViewRegistry precompile** on the ShinzoHub chain.
The precompile lives at **EVM address `0x000...0210`** and lets you register simple “views” on-chain.

It demonstrates:

* Sending an Ethereum tx to the precompile (`eth_sendTransaction`)
* Seeing the tx receipt with the emitted EVM log
* Subscribing to Cosmos events via Tendermint WebSocket
* Understanding the ABI encoding of the calldata and event logs

---

## 1. Start the blockchain

Spin up a local single-node testnet:

```bash
make sh-testnet
```

By default this starts:

* JSON-RPC (Ethereum API) at `http://localhost:8545`
* WebSocket (Tendermint events) at `ws://localhost:26657/websocket`

---

## 2. Send a transaction

Here’s an example `curl` that calls the `register(string)` method of the precompile at `0x210`, registering the view `"hello"`:

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  --data-raw '{
    "jsonrpc":"2.0",
    "method":"eth_sendTransaction",
    "params":[{
      "from":"0xabd39bcd18199976acf5379450c52f06edbcf4f3",
      "to":"0x0000000000000000000000000000000000000210",
      "gas":"0x100000",
      "value":"0x0",
      "data":"0x82fbdc9c0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000568656c6c6f000000000000000000000000000000000000000000000000000000"
    }],
    "id":1
  }'
```

This returns a tx hash, e.g.:

```json
{"jsonrpc":"2.0","id":1,"result":"0xd677c55107c43600bf4e5f3913bc9e3111b3cf7f55dadadf9a529eede6e955a1"}
```

---

## 3. Subscribe to Cosmos events

Open a WebSocket connection to Tendermint:

```bash
wscat -c ws://localhost:26657/websocket
```

Then subscribe to `Registered` events:

```json
{"jsonrpc":"2.0","method":"subscribe","id":1,"params":{"query":"tm.event='Tx' AND Registered.key EXISTS"}}
```

When the tx is processed, you’ll see a live event like:

```json
{
  "jsonrpc":"2.0",
  "method":"event",
  "params":{
    "result":{
      "data":{
        "type":"tendermint/event/Tx",
        "value":{
          "TxResult":{
            "events":[
              {
                "type":"Registered",
                "attributes":[
                  {"key":"key","value":"0xc26d2ef9f0e108c9..."},
                  {"key":"creator","value":"shinzo140fehngcrxvhdt84x729p3f0qmkmea8nq3rk92"},
                  {"key":"view","value":"hello"}
                ]
              }
            ]
          }
        }
      }
    }
  }
}
```

### Breakdown

* **type** → `Registered` (custom event type from `ctx.EventManager()` in Go)
* **attributes**:

  * `key`: derived from `keccak256(msg.sender, value)`
  * `creator`: the Cosmos-formatted address of the sender
  * `view`: the raw string you registered (`hello`)

This is the **Cosmos SDK view of the event**, delivered in real-time via the Tendermint WebSocket. It complements the **EVM log** that’s visible in Ethereum clients.

---

## 4. Check the EVM receipt

You can also confirm via the Ethereum JSON-RPC:

```bash
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_getTransactionReceipt","params":["0xd677c55107c43600bf4e5f3913bc9e3111b3cf7f55dadadf9a529eede6e955a1"],"id":1}' | jq
```

Example output:

```json
{
  "status":"0x1",
  "to":"0x0000000000000000000000000000000000000210",
  "logs":[
    {
      "address":"0x0000000000000000000000000000000000000210",
      "topics":[
        "0x7d917fcb...307c",
        "0xc26d2ef9...97858",
        "0x000000000000000000000000abd39bcd18199976acf5379450c52f06edbcf4f3"
      ],
      "data":"0x68656c6c6f"
    }
  ]
}
```

The `data` field (`0x68656c6c6f`) is the ASCII encoding of `"hello"`.

---

## 5. Understanding the Calldata

The transaction’s `data` field was:

```
0x82fbdc9c
0000000000000000000000000000000000000000000000000000000000000020
0000000000000000000000000000000000000000000000000000000000000005
68656c6c6f000000000000000000000000000000000000000000000000000000
```

### Breakdown

1. **Function selector**

   * `0x82fbdc9c` → First 4 bytes of `keccak256("register(bytes)")`

2. **Offset to parameter data**

   * `0x20` = 32 bytes → where the dynamic `bytes` value begins

3. **Length of bytes array**

   * `0x05` = 5 → the string length

4. **Actual bytes, padded**

   * `68656c6c6f` = `"hello"` in ASCII (padded to 32 bytes)

So the call is exactly:

```solidity
ViewRegistry.register("hello");
```

---

## 6. Understanding the Event

The precompile emits:

```solidity
event Registered(bytes32 indexed key, address indexed sender, bytes value);
```

In the EVM log:

| Field       | Location  | Value Example                                                         |
| ----------- | --------- | --------------------------------------------------------------------- |
| `topics[0]` | Event sig | `keccak256("Registered(bytes32,address,bytes)")`                      |
| `topics[1]` | key       | `0xc26d2ef9f0e108c9...` (derived from `keccak256(msg.sender, value)`) |
| `topics[2]` | sender    | `0xabd39bcd18199976acf5379450c52f06edbcf4f3`                          |
| `data`      | value     | `0x68656c6c6f` → `"hello"`                                            |

At the same time, the Cosmos layer emitted the Tendermint `Registered` event with attributes `{key, creator, view}`.

---

## 7. What you get

* **Cosmos Event (Tendermint)** → human-friendly, attribute-based events (`view = "hello"`)
* **EVM Log (Ethereum)** → raw ABI-encoded log visible to Ethereum clients and tools

Both event systems fire from the same transaction. This makes your precompile usable in both the Cosmos and Ethereum ecosystems.

---

## 8. Advanced Query Example

The precompile also accepts **arbitrary JSON payloads** as views, not just plain strings. For example:

```json
{
  "query": "Log {address topics data transactionHash blockNumber}",
  "sdl": "type FilteredAndDecodedLogs @materialized(if: false) {transactionHash: String}",
  "transform": {"lenses": []}
}
```

This describes a view with:
- **query**: which fields to capture
- **sdl**: schema definition
- **transform**: optional transforms

### Sending it

```bash
curl -X POST http://localhost:8545   -H "Content-Type: application/json"   --data-raw '{
    "jsonrpc":"2.0",
    "method":"eth_sendTransaction",
    "params":[{
      "from":"0xabd39bcd18199976acf5379450c52f06edbcf4f3",
      "to":"0x0000000000000000000000000000000000000210",
      "gas":"0x100000",
      "value":"0x0",
      "data":"0x82fbdc9c000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000b27b227175657279223a224c6f67207b6164647265737320746f706963732064617461207472616e73616374696f6e4861736820626c6f636b4e756d6265727d222c2273646c223a22747970652046696c7465726564416e644465636f6465644c6f677320406d6174657269616c697a65642869663a2066616c736529207b7472616e73616374696f6e486173683a20537472696e677d222c227472616e73666f726d223a7b226c656e736573223a5b5d7d7d0000000000000000000000000000"
    }],
    "id":1
  }'
```

This transaction carries the full JSON view definition ABI-encoded as `bytes`.

### Events

When processed:
- **Cosmos event** → includes `view` attribute containing the JSON
- **EVM log** → raw ABI log with the same data

This shows how you can register structured views directly on-chain.

---

## 9. What you get

* **Cosmos Event (Tendermint)** → human-readable attributes like `view = { ... }`
* **EVM Log (Ethereum)** → ABI-encoded logs for Ethereum tools

Both systems emit from the same transaction, making the precompile usable in **both Cosmos and Ethereum ecosystems**.


---

✨ That’s it — you now have live end-to-end proof that the **ViewRegistry precompile works on ShinzoHub**, with events accessible through **both Cosmos and Ethereum APIs**.
