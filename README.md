<!--
  This README covers local setup, Docker, and deployment only.
  Do not add: architecture explanations, API reference, configuration 
  deep-dives, or troubleshooting guides. Those belong in the Shinzo 
  documentation site. If you're tempted to add a section, link to the docs 
  instead.
-->

# ShinzoHub

[![Docker](https://img.shields.io/github/actions/workflow/status/shinzonetwork/shinzohub/.github/workflows/docker.yml?label=docker)](https://github.com/shinzonetwork/shinzohub/actions)
[![License](https://img.shields.io/github/license/shinzonetwork/shinzohub)](./LICENSE)

Cosmos SDK hub chain for the Shinzo network with EVM compatibility and on-chain access control policy management.

> ![WARNING]
> If you're looking to run an Indexer, Host, or deploy a View, you don't need this repo. See the [Shinzo documentation site](https://docs.shinzo.network) for the right starting point. This repo is for Shinzo core developers and integration testers working on the chain itself.

## Getting started

```shell
git clone git@github.com:shinzonetwork/shinzohub.git
cd shinzohub
make build
make sh-testnet
```

> [!TIP]
> See [BUILD.md](./BUILD.md) for full build-from-source instructions.

## Deployment

See the [Shinzo documentation site](https://docs.shinzo.network) for production deployment instructions.

## Contributing

Open an issue before submitting a PR. See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## License

[MIT](./LICENSE)
