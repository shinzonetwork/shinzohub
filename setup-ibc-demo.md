# ShinzoHub ↔ SourceHub IBC + ICA Demo

This demo shows how to establish an **IBC connection** between **ShinzoHub** and **SourceHub** using **Hermes** and then register an **Interchain Account (ICA)** for the `sourcehub` module.

It demonstrates:

- Pulling and building the **SourceHub** and **ShinzoHub** binaries
- Spinning up two local chains (ShinzoHub + SourceHub)
- Starting Hermes relayer and waiting for it to connect
- Registering the ICA from ShinzoHub → SourceHub
- Debugging and verifying IBC channels, connections, and ICA addresses
- Testing the ViewRegistry precompile to create a view + policy

---

## 0. Prerequisites: Build SourceHub + ShinzoHub

Before running this demo, you need to have both binaries built and available locally.

### Build SourceHub

Clone the **SourceHub** repository and build the binary:

```bash
git clone https://github.com/sourcenetwork/sourcehub.git
cd sourcehub
make build
```

This produces the `sourcehubd` binary inside the project’s `build/` directory.  
Make sure it is in your home dir or referenced directly in your scripts.

### Build ShinzoHub

From your **ShinzoHub** repository:

```bash
just build
```

This produces the `shinzohubd` binary inside the `build/` directory.  
Again, ensure it’s accessible in your shell or scripts.

---

## 1. Start the blockchains

Start both local single-node testnets:

```bash
./scripts/ica/start_shinzohub_node.sh
./scripts/ica/start_sourcehub_node.sh
```

By default these scripts expose:

- **ShinzoHub JSON-RPC**: `http://localhost:26657`
- **SourceHub JSON-RPC**: `http://localhost:27657`

---

## 2. Start Hermes

Run the Hermes setup script:

```bash
./scripts/ica/start_hermes.sh
```

This does the following:

- Initializes Hermes home dir
- Installs keys for both chains (using a shared test mnemonic)
- Creates the IBC connection between `shinzo` and `sourcehub`
- Starts Hermes in the foreground, relaying packets

👉 **Wait until you see `Hermes started` before continuing.**  
This ensures the IBC handshake is complete and the connection is `STATE_OPEN`.

---

## 3. Register the Interchain Account (ICA)

With Hermes running, open a new terminal and run:

```bash
./scripts/ica/register_ica.sh
```

This executes:

```bash
build/shinzohubd tx sourcehub register-ica connection-0 connection-0 \
  --from acc0 \
  --chain-id 9001 \
  --keyring-backend test \
  --home ~/.shinzo \
  --node tcp://127.0.0.1:26657 \
  --gas auto --gas-adjustment 1.5 --fees 9000ushinzo \
  --yes
```

If successful, it opens a new ICA channel from **ShinzoHub → SourceHub**.

---

## 4. Debug and Verify

### Check channels

```bash
build/shinzohubd q ibc channel channels -o json | jq
```

Example:

```json
{
  "channels": [
    {
      "state": "STATE_OPEN",
      "ordering": "ORDER_ORDERED",
      "counterparty": {
        "port_id": "icahost",
        "channel_id": "channel-0"
      },
      "connection_hops": ["connection-0"],
      "version": "{\"version\":\"ics27-1\",\"controller_connection_id\":\"connection-0\",\"host_connection_id\":\"connection-0\",\"address\":\"source1jcg...\",\"encoding\":\"proto3\",\"tx_type\":\"sdk_multi_msg\"}",
      "port_id": "icacontroller-shinzo15rrya9m8arep2p0kn9seyg8k9ly27vwzhrvjs3",
      "channel_id": "channel-0"
    }
  ]
}
```

---

### Query the ICA address

```bash
build/shinzohubd q interchain-accounts controller interchain-account shinzo15rrya9m8arep2p0kn9seyg8k9ly27vwzhrvjs3 connection-0
```

Example:

```
address: source1257tmnghmrxgg5pjpeeu6zyljfh9cj63d3r225j40jq70qc7ln8q5evwxh
```

This is the **SourceHub account** controlled by the ShinzoHub module via ICA.

---

### Check IBC connections

```bash
build/shinzohubd q ibc connection connections -o json | jq
```

Example:

```json
{
  "connections": [
    {
      "id": "connection-0",
      "client_id": "07-tendermint-0",
      "versions": [{"identifier":"1","features":["ORDER_ORDERED","ORDER_UNORDERED"]}],
      "state": "STATE_OPEN",
      "counterparty": {
        "client_id": "07-tendermint-0",
        "connection_id": "connection-0",
        "prefix": {"key_prefix":"aWJj"}
      },
      "delay_period":"0"
    }
  ]
}
```

---

## 5. What you get

- **Open IBC Connection** (`connection-0`) between ShinzoHub ↔ SourceHub
- **Open ICA Channel** (`channel-0`) bound to the sourcehub module
- **Interchain Account Address** on SourceHub (controlled by ShinzoHub)

This proves the **IBC + ICA workflow** is working end-to-end, and you can now send Cosmos SDK messages from ShinzoHub to execute on SourceHub.

---

## 6. Test the Precompile (View + Policy Creation)

Once the ICA channel is open, you can also test the connection by calling the **ViewRegistry precompile** on ShinzoHub.  

This creates a **view**, which also creates a corresponding **policy** on SourceHub.

### Send the transaction

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

Example result:

```json
{"jsonrpc":"2.0","id":1,"result":"0xc5d55f9a4e8788abaaf74d4772c2a4afe2d1c30a1384d6dcb1c748e8ddeeb48c"}
```

This transaction registers the view `"hello"` and triggers policy creation.

---

### Query the created policy

On **SourceHub**, run:

```bash
sourcehubd q acp policy-ids --node "tcp://127.0.0.1:27686"
```

Example output:

```
ids:
- df5dea5c508a6eadd3f8a1312c6f33d04a08c67e1ea7c90332a7a61a46d7ad51
```

This confirms that a new **policy** was created as a result of the precompile call.  

👉 Remember to adjust the `--node` flag (and the binary path) if your SourceHub build directory differs.
