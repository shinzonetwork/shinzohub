# Shinzohub

Shinzohub is a Cosmos SDK–based blockchain project with EVM compatibility and custom modules. This repo provides everything you need to build, install, and run the chain locally.

---

## ⚡️ Requirements

Before you start, ensure you have the following installed:

- **Go** ≥ 1.24
- **Make**  
- **Git**  
- **Protobuf compiler (`protoc`)**  
- [Buf](https://buf.build/docs/installation) (for linting/formatting Protobuf)  
- Optional: [asdf](https://asdf-vm.com/) (if you manage your Go version via asdf)  

---

## 🔨 Building

By default, binaries are built into `./build`.

### Build for your local system
```bash
make build
```

Result:
```
./build/shinzohubd
```

### Cross-compile for Linux
```bash
make build-linux-amd64
make build-linux-arm64
```

---

## 🚀 Installing

### To your local bin (`~/.local/bin`)
```bash
make install
```

Afterwards, confirm:
```bash
shinzohubd version
```

### To GOPATH/bin
```bash
make install-gopath
```

---

## 🧹 Cleaning

Remove all build artifacts:
```bash
make clean
```

---

## 🛠 Verifying Dependencies

To ensure Go modules are tidy and not corrupted:
```bash
make verify-deps
```

---

## 📦 Protobuf

Protobuf files live under `./proto`.

### Install Protobuf dependencies
```bash
make proto-deps
```

### Generate Protobuf code
```bash
make proto-gen
```

### Lint Protobuf definitions
```bash
make proto-lint
```

### Format Protobuf files
```bash
make proto-format
```

Or run all in one go:
```bash
make proto-all
```

---

## 🌐 Development

### Doctor check
Quick project health check:
```bash
make doctor
```

Output will confirm:
- If the build artifact exists
- If `shinzohubd` is on your PATH  


### Start a local testnet
```bash
make sh-testnet
```

This spins up a chain with:

- `CHAIN_ID=91273002`  
- `BLOCK_TIME=1000ms`  
- Fresh state each run (`CLEAN=true`)  

---

## 📝 Notes

- You can override build settings via environment variables:  
  - `BUILD_DIR` → change binary output directory  
  - `LEDGER_ENABLED` → enable/disable Ledger support (default: `true`)  

Example:
```bash
BUILD_DIR=/tmp/shinzohub LEDGER_ENABLED=false make build
```

---

## ✅ Quickstart

```bash
# 1. Verify dependencies
make verify-deps

# 2. Build the binary
make build

# 3. Install it
make install

# 4. Check installation
shinzohubd version

# 5. Run a local testnet
make sh-testnet
```
