# Shinzo ACP Testing

This directory contains integration tests for the Access Control Policy (ACP) system in Shinzo.

## Overview

The testing system verifies relies on the tests, relationships, and policy defined in the [Sourcehub playground](http://acp-playground.stage.infra.source.network/) during the design stage. You can find them in the `/acp` directory located at the root of this project's file path.

The biggest advantage of defining our tests in this manner is that, should our design need to change to accomodate new features (or similar), we can simply re-open the playground as we return to the design phase. Then, after we've gotten some quick feedback on our new design from working in the playground, we can simply re-export the yaml files from the playground and replace what we have here. Next time we re-run the integration test suite, it will assemble the testing environment and tests according to how we defined it in the playground while designing - this helps us keep our code and design up to date at all times.

## Usage

### Prerequisites

1. Clone all required dependency repositories -> [Indexer](https://github.com/shinzonetwork/indexer), [SourceHub](https://github.com/sourcenetwork/sourcehub), and [DefraDB](https://github.com/sourcenetwork/defradb)
2. Complete setup steps for each dependency resource; most importantly, make sure you have installed their dependencies
3. Set environment variables in your system, adding them to your `~/.zshrc` or similar. Example environment variables to set:
   ```
   DEFRA_PATH=../defradb
   SOURCEHUB_PATH=../sourcehub
   INDEXER_PATH=../indexer
   SHINZOHUB_PRIVATE_KEY=some_random_private_key
   ```
4. If you'd like to use the debugger, place these in a `.env` file as well in the root of this project

### Running Tests

#### Option 1: Makefile
```bash
make integration-test
```

This will:
1. Bootstrap the entire system
2. Wait for all services to be ready
3. Run the ACP integration tests
4. Report results and save them in `<project root directory>/logs/integration_test_output.txt`

#### Option 2: Running with debugger
1. `make bootstrap`
2. `./scripts/wait_for_services.sh` in another command line window and wait for the script to exit - this means all dependencies are setup and ready for you
3. Setup breakpoints
4. In your IDE, navigate to `tests/integration_test.go` and click to debug the `TestAccessControl` test entrypoint

### Troubleshooting

If you're encountering issues running the tests, there are a few things to try.

First, a quick `make stop` is worth running - this will double check that all of your services have been cleaned up properly (closing any it finds).

Next, you'll want to check the logs of the various services to determine what is going on. These can be found in the `logs` directory. You'll also find that since `make bootstrap` is also calling `make bootstrap` on the `Indexer` dependency that some of its dependencies' logs can be found in its corresponding `logs` directory.

## How It Works

### Setup

1. Real DIDs are generated using the SourceHub SDK and mapped to their alias DID -> e.g. `did:user:addo` might get mapped to something like `did:key:z6MkqkFaZazbSsVVmRzTRXhx236bLrXeM6GJK3uP1xUsJvhD`
2. Test objects are registered. This includes `primitives:blocks`, `view:datafeedA`, `view:datafeedB`, `group:shinzoteam`, `group:indexer`, and `group:host` currently. These are registered by the SourceHub instance's validator node, making the SourceHub instance, itself, the owner of these resources.
3. Admin permissions are given out to the appropriate users as described in the relationships file. These permissions are granted by the resource owner, the SourceHub instance.
4. Group and resource to resource relations are defined. The SourceHub instance grants all the parent relations and group access rights.
5. Users are added to appropriate groups via the registrar API client - the registrar API client, aliased as `shinzohub` in our test yaml files from the playground, is granted its relations during steps 3 and 4.
6. Users are granted subscriber and banned relations via the registrar API as appropriate.
7. Test cases are parsed and generated. For each test case, check with SourceHub instance to see if the specified user meets the specified permission check and compare against our expected result. 
   a) For delegation tests, i.e. tests where we check if an actor has permission to add/remove a relation from an actor on a specific resource, we will fund the account and attempt to modify that relation.

## Future Enhancements

1. **End to End**: Instead of creating dummy objects in SourceHub and making queries using SourceHub SDK to check for access, instead, create real objects with Defra and actually attempt to modify/read/action them.