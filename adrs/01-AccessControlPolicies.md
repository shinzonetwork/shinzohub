# 01 - Access Control Policies (ACPs)

## Status
Ammended by Quinn Purdy on August 8, 2025
Ammended by Quinn Purdy on July 24, 2025
Written by Quinn Purdy on July 18, 2025

## Context
As we are designing and prototyping Shinzo, it has become clear that ACPs are a necessity for our system - we need to gate access to Data Feeds/Views to subscribers, we need to limit who can write to where to preserve data integrity, and we want to limit the data pushed to a user's Defra instance to be the data they need, no more or less.

With respect to choosing the specific technologies to build our ACPs, the Shinzo team, by design, has a pre-chosen decision. The Shinzo team will use SourceHub for ACPs; this is a core technology developed in-house at Source; the Shinzo team, in addition to building Shinzo, is responsible for dogfooding core Source technologies and providing the appropriate teams with useful feedback. Not to mention, SourceHub and DefraDB have been designed to work together, simplifying interactions.

SourceHub has yet to go live with a testnet or mainnet, but this is expected in the near-medium term.

## Decision
Shinzo will enfore ACPs using a shard of SourceHub. Independant blockspace on SourceHub is chosen because Shinzo is expected to generate a fair amount of transactions and can benefit from the parallelized execution by segmenting itself from the rest of the SourceHub. In addition, the Shinzo team feels that having the flexibility to modify our SourceHub instance is valuable - we may, for example, introduce an additional layer of middleware that confirms that a user not only has access (via ACPs) to a Data Feed/View, but also has the funds to pay for them; the middleware would perform this check and pay for the resource access if possible atomically.

We will be introducing the ACPs as described in 01-ChosenACPs.json - this can be imported into [the playground](http://acp-playground.stage.infra.source.network/) for visualization. As a companion to the ACPs, we have [this diagram in Excalidraw](https://excalidraw.com/#json=VBQFY9nF_aMvAZG1gV-3Z,uR6gLzxrm1YpPEHJZQC3Ew) to help illustrate the flow of information.

At a high level, the ACPs introduce a 4 different types of access permissions.

1) `sync` (renamed from "read") - allows you to sync a document (data) onto your DefraDB instance on your device. The data remains encrypted but can be used for Lens transformations
2) `update` - allows you to write to a document
3) `read` (renamed from "query") - allows you to unencrypt the document (data) in your DefraDB instance on your device -> enabling you to perform GraphQL queries against it
4) `delete` - allows you to delete a document for all users. This permission remains with the Owner. In the case of Shinzo, ShinzoHub will be the Owner of every document; no user or administrator of Shinzo is permitted to delete a document.

The ACPs also introduce 3 types of resources.

### Groups

Groups are a useful way of bundling groups of `did`s together. Groups have two levels of access, `member` and `administrator`, represented as `permissions` in the ACPs. A group `member` is a contributor - they are typically given read and/or write permissions for a resource. A group `administrator` isn't typically given read or write permissions, instead, they are responsible for adding and removing `member`s from a group.

Currently, we have defined 3 separate groups.

1. Indexers - these `member`s are given read and write permissions on `Primitives`
2. Hosts - these `member`s are given read permissions on `Primitives`. In addition, the `Host` group is granted write permissions on Views.
3. ShinzoTeam - `Administrator`s of the Shinzo team are also `Administrators` for the other groups and our other policies - they are trusted to perform admin overrides on access right requests when the relevant services are not behaving.

### Primitives

The Primitives policy applies to our primary data source documents - Blocks, Transactions, and Logs.

Primitives are contributed to exclusively by `member`s of the Indexer group; they alone have write access.

Any member of the `Host` group is given read access automatically. In addition, any other user can request to have read access; this request is fulfilled by `ShinzoHub`.

### Views

The Views policy applies to all n-ary data sources: Data Feeds/Views.

Members of the `Host` group are given write access to Data Feeds/Views that are derived solely from Primitives (and not from any other Data Feeds/Views). Futher Data Feeds/Views that are derived from other Data Feeds/Views (and optionally Primitives) are assigned a "Parent" relationship to the Data Feeds/Views they are derived from. Through this Parent relationship, members of the `Host` group inherrit write permissions.

### Handling Bad Behaviour

As an open protocol, bad actors are to be expected. A `did` that is determined to be acting maliciously can have their access permissions revoked by giving them the `banned` role for `Primitives` and `Views` or the `blocked` role for `Groups`. It is preferred to give bad actors a new role as opposed to revoking the old role so that they can be easily identified. In the future, if a bad actor attempts to re-request read/write permissions, we can more easily identify that they were previously given those permissions and those permissions were revoked. You can essentially think of it as a denylist as opposed to removing them from the allowlists.

### Overview of ShinzoHub's role

To facilitate these ACPs, ShinzoHub will take a very important role.

First, Shinzo will leverage a niche document creation flow exposed by SourceHub where, when a document is created, ownership rights are immediately relinquised to the SourceHub protocol itself. In SourceHub, the owner of a document has complete control over that document; they may read, update, delete, etc. the document as they please. While this behaviour is desirable for most DefraDB applications which emphasize giving users (and other entities) control over their own data, Shinzo's purpose is instead to give the community better control of all blockchain data. For this reason, transferring control to the protocol itself aligns more closely with Shinzo's vision.

ShinzoHub will be given a special role on all documents and groups used by Shinzo, the "admin" role. This role gives ShinzoHub the permission to manage sync, write, and read permissions on each document on a did or group basis and also gives ShinzoHub the permission to add and remove people from the Indexer and Host groups. In practice, this gives ShinzoHub 3 main responsibilities:

1) ShinzoHub needs to assign the appropriate read/write permissions for Indexer and Host groups on each document created at the time of creation (once admin rights are assigned). This assignment of permissions is done either via a direct assignment of roles to the groups or via establishing a parent relationship with the collection the data has been derived from (see Views section).

2) ShinzoHub needs to expose APIs where prospective Indexers and Hosts can register to join their respective group. This registration process may require the collection, validation, and/or storage of some data.

3) `read` permissions on a Primitive or View will require payment. While the details are not yet finalized, payments are likely to be facilitated by a smart contract deployed on each Shinzo-supported chain; the Shinzo team has nicknamed this the "Outpost" contract for now. The Outpost contract will emit events whenever a successful payment is made. ShinzoHub will be required to listen for these events via IBC and then will need to give the paying did the appropriate role(s) such that they will have `read` permissions on the chosen Primitive or View.

### Node Access Control

By default, when working with SourceHub and Defra, any node has the ability to create new policies and attach relationships to them. This kind of flexible access control, while valuable for some use cases, is not desirable for Shinzo - Shinzo's goal is to make blockchain data highly available to paying users via a decentralized protocol and infrastructure, not to give user's ownership over data. In the case of Shinzo, the protocol, ShinzoHub, owns the data and it alone should maintain administrator priveledges on the data.

To facilitate this, the default Node Access Control policy supported by DefraDB will be applied. This effectively locks down the network such that only ShinzoHub will be permitted to create new policies and apply relationships.

## Consequences
Going forward, testing and interacting with the Shinzo system will become more difficult; you will need to have appropriate permissions to read and write to files and any test environments will likely require more setup to accomodate this. However, though this will initially be a bit of a hassle, it is worthwhile because it creates a local environment closer to what we will expect to see in production. Migrating early on in the development lifecycle is helpful because the amount and complexity of changes required to accomodate the newly introduced ACPs is much smaller early on than it will be in the future with more of Shinzo implemented.

Our integration test environment will need some updates - these should be included in the implementation PR(s).

Due to the first of ShinzoHub's responsibilities and the ACP design, identifying the Data Feeds/Views and Primitives that a Data Feed/View is derived from is important. View_creator may need updating to require creators to include the parent collection(s) in their request.
