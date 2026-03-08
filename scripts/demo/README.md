# ShinzoHub Walkthrough

A practical, step-by-step guide to running ShinzoHub locally and interacting with its precompiles.

## Prerequisites

- Go 1.22+
- [Foundry](https://book.getfoundry.sh/getting-started/installation) (`cast` CLI)
- `openssl`, `jq`, `python3`, `xxd`

SourceHub will be cloned and built automatically if not already present.

## 1. Start Everything

This builds ShinzoHub, starts SourceHub, starts the Hermes relayer, registers the ICA, and sets up the Shinzo policy.

```bash
sh scripts/demo/start_all.sh
```

Wait until you see `All services are running!`. This means both chains are live, IBC is connected, and the ICA is registered.

To stop everything later:

```bash
sh scripts/demo/stop_all.sh
```

## 2. Register a Host

Create a fresh wallet, fund it, and register it as a host with auto-generated peer and node identity keys.

```bash
sh scripts/demo/hosts/register.sh
```

List all registered hosts:

```bash
sh scripts/demo/hosts/list.sh
```

## 3. Register an Indexer

This script does two things: submits an indexer assertion (via the admin), then registers the indexer on the precompile with peer/node keys.

```bash
sh scripts/demo/indexers/register.sh
```

List all registered indexers:

```bash
sh scripts/demo/indexers/list.sh
```

## 4. Create a View

Generates a fresh wallet, builds a random viewbundle, and deploys it via the ViewRegistry precompile. The script prints the deployed view contract address.

```bash
sh scripts/demo/views/create.sh
```

Save the view address from the output for the next steps:

```
View:     0xABC123...
```

List all registered views:

```bash
sh scripts/demo/views/list.sh
```

## 5. Query View Info

Check all properties of the deployed view contract:

```bash
VIEW_ADDR=0xABC123... sh scripts/demo/views/info.sh
```

You'll see the name, creator, viewbundle data, pricing info (all zeros initially since no hosts have reported), popularity (zero stake), and consumers.

## 6. Report on the View (as a Host)

Register a new host and have it report a random complexity coefficient and price per view on your view contract:

```bash
VIEW_ADDR=0xABC123... sh scripts/demo/views/report.sh
```

Run it multiple times to add more hosts. Each host reports independently; the view averages their values.

## 7. Stake on the View

Create a fresh wallet and stake a random amount (0.001-1 SHNZ) on the view:

```bash
VIEW_ADDR=0xABC123... sh scripts/demo/views/stake.sh
```

Run it multiple times to add more stakers. Staking increases the view's popularity, which increases its price (up to 4x the base price).

## 8. Check Updated Info

After reporting and staking, query the view again to see the updated pricing:

```bash
VIEW_ADDR=0xABC123... sh scripts/demo/views/info.sh
```

You should now see:

- **pricePerView** and **complexityCoefficient** reflect the host reports
- **popularity** shows the total staked amount
- **price** is non-zero: `basePrice * (1 + popularityPremium) * 0.95`

## Configuration

All scripts accept environment variable overrides:

| Variable | Default | Description |
|----------|---------|-------------|
| `BINARY` | `./build/shinzohubd` | Path to the ShinzoHub binary |
| `RPC_URL` | `http://localhost:8545` | EVM JSON-RPC endpoint |
| `NODE` | `tcp://127.0.0.1:26657` | Tendermint RPC endpoint |
| `CHAIN_ID` | `91273002` | Chain ID |
| `HOME_DIR` | `~/.shinzohub` | ShinzoHub home directory |
| `FUNDER` | `acc0` | Keyring key name used to fund fresh wallets |
| `SOURCEHUB_PATH` | `~/sourcehub` | Path to SourceHub repo (for start_all.sh) |

Example with overrides:

```bash
FUNDER=acc1 RPC_URL=http://localhost:8546 sh scripts/demo/views/create.sh
```

## Script Reference

| Script | Description |
|--------|-------------|
| `scripts/demo/start_all.sh` | Start ShinzoHub + SourceHub + Hermes + ICA + policy |
| `scripts/demo/stop_all.sh` | Stop all background processes |
| `scripts/demo/hosts/register.sh` | Register a new host |
| `scripts/demo/hosts/list.sh` | List all hosts |
| `scripts/demo/indexers/register.sh` | Attest and register a new indexer |
| `scripts/demo/indexers/list.sh` | List all indexers |
| `scripts/demo/views/create.sh` | Create a new view |
| `scripts/demo/views/list.sh` | List all views |
| `scripts/demo/views/info.sh` | Query a view's properties |
| `scripts/demo/views/report.sh` | Register host + report on a view |
| `scripts/demo/views/stake.sh` | Stake on a view |
