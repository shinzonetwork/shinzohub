# Contributing

## Before you start

Open an issue to discuss your proposed change before submitting a PR. This avoids wasted effort if the change isn't a good fit. PRs without an attached issue will be closed.

## Making changes

The repository is structured as follows:

| Directory | Purpose |
| --- | --- |
| `app/` | Cosmos SDK application wiring, EVM precompile registration, ante handlers. |
| `app/precompiles/` | Custom EVM precompiles: `ViewRegistry` and `EntityRegistry`. |
| `cmd/shinzohubd/` | Entry point for the chain daemon binary. |
| `cmd/registrar/` | Entry point for the standalone registrar service. |
| `pkg/` | Shared packages: SourceHub IBC/ICA client adapters, validators, and utilities. |
| `proto/` | Protobuf definitions. Run `make proto-all` after making changes here. |
| `sdk/` | Go SDK for interacting with ShinzoHub from external clients. |
| `scripts/` | Shell scripts for building, running a local testnet, and IBC/ICA demos. |
| `x/sourcehub/` | The custom Cosmos SDK module that handles ICA message dispatch. |
| `acp/` | Access control policy YAML definitions and test cases. |
| `adrs/` | Architecture Decision Records. Add a new ADR before making significant design changes. |
| `tests/` | ACP integration tests. |

## Submitting a PR

- Keep PRs focused. One change per PR.
- Describe what you changed and why in the PR description.
- Make sure `make build` and `make verify-deps` pass before requesting review.
- If your change touches Protobuf definitions, run `make proto-all` and commit the generated files.
- Leave plenty of comments. Assume reviewers are unfamiliar with the specific change you're making.
