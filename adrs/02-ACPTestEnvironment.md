# 02 - Access Control Policies (ACPs) Testing Environment - A Follow Up to ADR 01

## Status
Written by Quinn Purdy on August 7, 2025

## Context
As we introduce ACPs to Shinzo, we a way to test and confirm that the access control guarantees described in our ACP are being enforced. This requires setting up a test environment that mirrors, as closely as possible, that of what our production (and testnet) environment will eventually look like.

We have some dependencies, owned by other teams, that impact our decision:
1) SourceHub is not yet live with a testnet or mainnet. With this, we will need to test against the development branches (ideally, finding a stable commit or pre-release to work off of - input from SourceHub team will be required to determine which commit(s) we should work off of) and run the service in either a local or containerized environment
2) In a production or testnet environment, ShinzoHub and SourceHub will need communicate via IBC. Source does not yet have an IBC integration with SourceHub that will serve our purposes. The SourceHub team is investigating and is responsible for building this IBC module that we will consume.

In ADR 01, we've actually already written the "tests". Specifically, we've written tests that work in the context of [the playground](http://acp-playground.stage.infra.source.network?share=b66d71b7). They are useful to allow SourceHub integrators to visualize/validate the guarantees enforced by the policies and relationships they've setup in the playground during the design stage.

## Decision
First, let's address how we will handle the missing dependencies. 
1) Similar to what we've done to test the Indexer, the team will put together a shell script that will automate the setup of the test environment. We will run the services locally. We've opted not to use a containerized approach as we feel, with our current skillsets, that a containerized approach is more work than the value we'd gain from doing so, especially given the size of our team (3 people) and the frequency with which we expect to need to run the integration test suite.
2) For now, the team will use the SourceHub Go SDK to communicate with the local SourceHub instance. This will be done through an interface dependency as opposed to a concrete dependency. This allows us to, once IBC is available, create an adapter such that we can use IBC with the same (or similar) interface, allowing us to slot it in in-place of the SDK integration while preserving our core logic. With this, we can test all but our IBC integration/adapter, minimizing the impact from the missing dependency.

### Understanding the Test Environment
For our local test environment, we require the following services:

- DefraDB, 2(+) instances -> we need to confirm that the appropriate data is propogating from one node to another and that each node can write to the documents as specified by their permissions
- SourceHub
- Indexer/block_poster
- ShinzoHub's registrar

These should all be started with one command via a script and gracefully torn down when finished.

Once setup, the system would resemble [this Excalidraw diagram](https://excalidraw.com/#json=_7wTTJQY_huxkOzpsoTks,FjZMfkX63cMtMy0xQMr9GQ).

### Understanding the Tests
Our tests aim to confirm the access control policies are being managed properly in our integration. We aim to confirm that the guarantees, described in `acp/tests.yaml`, are being adhered to by our system.

For the sake of accuracy, convenience, and future developer experience, the team would like to explore using the playground tests (found in `acp/tests.yaml` for this project) to generate the core logic for our integration tests. This has the added benefit of allowing us to instantly update our tests if we need to tweak our policy design - we can do so in the playground where visualization and experimentation is easiest (and more white-labelled) and then essentially "import" the new policy, relationship, and test yaml files and test them out in our test environment.

Our tests will mostly communicate with our various services via HTTP connections with the local host. This mode of communication is familiar to most developers and is a safe choice that is close-enough to how our production environment will work for the sake of testing this behaviour.

## Consequences
1) As a consequence of opting for a non-containerized approach, setup on each dev machine requires a bit more legwork at first. However, once setup, this shouldn't be much of an issue. The more impactful consequence of this approach is that hooking our integration tests into our CI system becomes significantly more difficult, certainly more difficult than the effort is worth at this stage in the Shinzo project. As a result, the team will need to be a bit more vigilant and make sure to actually run these integration tests on occasion, especially if a change has been made to the Registrar module of ShinzoHub or the ACPs themselves. Fortunately, we don't expect the ACPs or Registrar system to change too much, especially once we have gone live into production.
Note: that our Indexer/Block_Poster integration tests are also not hooked into our CI system and need to be run manually whenever relevant changes are made.
2) There is very little long-term impact of choosing to use the Go SDK for our tests. The team must simply recognize that the work isn't fully done until we can migrate our Registrar implementation over to IBC and that there might be some refactoring work required and tests may need to be updated accordingly. In general, these changes, particularly to our tests, should be rather minimal.


