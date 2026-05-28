# Build from source

## Prerequisites

- Go ≥ 1.24.
- Make.
- Git.
- [Buf](https://buf.build/docs/installation) (only needed for Protobuf regeneration).

## Steps

```shell
git clone git@github.com:shinzonetwork/shinzohub.git
cd shinzohub
make build
```

The compiled binary goes into `./build/shinzohubd`.

## Useful commands

| Command | What it does |
| --- | --- |
| `make build` | Build `shinzohubd` into `./build`. |
| `make build-linux-amd64` | Cross-compile for Linux amd64. |
| `make build-linux-arm64` | Cross-compile for Linux arm64. |
| `make install` | Build and copy the binary to `~/.local/bin`. |
| `make install-gopath` | Build and copy the binary to `$(GOPATH)/bin`. |
| `make sh-testnet` | Start a local single-node testnet with a fresh chain state. |
| `make doctor` | Check whether the binary was built and is on `PATH`. |
| `make verify-deps` | Run `go mod verify` and `go mod tidy`. |
| `make clean` | Remove the `./build` directory. |
| `make proto-all` | Format, lint, and regenerate Go code from `.proto` files. |
| `make proto-deps` | Install the required `protoc` plugins. |

## Environment overrides

| Variable | Default | Description |
| --- | --- | --- |
| `BUILD_DIR` | `./build` | Output directory for compiled binaries. |
| `LEDGER_ENABLED` | `true` | Enable or disable Ledger hardware wallet support. |
