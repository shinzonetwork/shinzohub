# shinzohub
**shinzohub** is a blockchain built using Cosmos SDK and Tendermint and created with [Ignite CLI](https://ignite.com/cli).

## Get started

```
ignite chain serve
```

`serve` command installs dependencies, builds, initializes, and starts your blockchain in development.

### Configure

Your blockchain in development can be configured with `config.yml`. To learn more, see the [Ignite CLI docs](https://docs.ignite.com).

### Local SourceHub Setup

When working locally with ShinzoHub, it is useful to also work with a local instance of SourceHub. This allows you to mess around in a sandbox environment during development.

First, you'll need to clone [the SourceHub repo](https://github.com/sourcenetwork/sourcehub) onto your machine and perform the setup steps. 

Most importantly, you'll want to make sure to build it with `make build` from the root directory of the SourceHub repo on your machine and then run the genesis setup script `./scripts/genesis-setup.sh`.

From here, it is recommended to try and run sourcehub to confirm it has been setup correctly. `build/sourcehubd start` from the root directory of the SourceHub repo. You should see a bunch of logs getting continuously posted in your command line. End it with `Control + C`.

### Bootstrapping a Local Development Environment

First, open up a command line to the root directory for ShinzoHub. It is recommended to set the path (either relative or absolute) to your cloned instance of the SourceHub repo on your machine as an environment variable `SOURCEHUB_PATH`.

Then, run `make bootstrap` - this will start up a local instance of SourceHub and ShinzoHub on your machine - allowing you to experiment in a sandbox environment during development.

### Web Frontend

Additionally, Ignite CLI offers a frontend scaffolding feature (based on Vue) to help you quickly build a web frontend for your blockchain:

Use: `ignite scaffold vue`
This command can be run within your scaffolded blockchain project.

For more information see the [monorepo for Ignite front-end development](https://github.com/ignite/web).

## Release
To release a new version of your blockchain, create and push a new tag with `v` prefix. A new draft release with the configured targets will be created.

```
git tag v0.1
git push origin v0.1
```

After a draft release is created, make your final changes from the release page and publish it.

### Install
To install the latest version of your blockchain node's binary, execute the following command on your machine:

```
curl https://get.ignite.com/username/shinzohub@latest! | sudo bash
```
`username/shinzohub` should match the `username` and `repo_name` of the Github repository to which the source code was pushed. Learn more about [the install process](https://github.com/allinbits/starport-installer).

## Learn more

- [Ignite CLI](https://ignite.com/cli)
- [Tutorials](https://docs.ignite.com/guide)
- [Ignite CLI docs](https://docs.ignite.com)
- [Cosmos SDK docs](https://docs.cosmos.network)
- [Developer Chat](https://discord.gg/ignite)
