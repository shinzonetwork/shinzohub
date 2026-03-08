# ShinzoHub View System

## What is a View?

A **View** is an on-chain data product. It is a smart contract that wraps a **viewbundle** (a query, a schema, and optional transforms) and manages its own economy — pricing, staking, host reporting, consumer access, and creator earnings.

Every View is deployed as its own EVM contract via the **ViewRegistry precompile**. When a View is created, it is also registered on **SourceHub** via ICA (Interchain Accounts) so that access control policies (ACP) can govern who interacts with it.

Views are the core economic unit of ShinzoHub. They represent curated, queryable data products that hosts serve, consumers pay for, and stakers signal demand on.

## View Lifecycle

1. **Creation** — A creator submits a viewbundle to the ViewRegistry precompile. A new View contract is deployed on-chain containing the bundle data, the creator's address, and an optional custom pricing contract.

2. **Registration** — The view is stored in the Cosmos module and an ACP object is created on SourceHub via ICA. This makes the view subject to the network's access control policies.

3. **Host Discovery** — Registered hosts discover the view, begin serving its data, and report their pricing parameters (complexity coefficient and price per view).

4. **Consumer Access** — Consumers pay SHNZ to access the view's data. Payments flow to the creator's balance.

5. **Staking** — Anyone can stake SHNZ on a view to signal demand. Higher stake increases the view's price, attracting more hosts and increasing creator revenue.

## Participants

### Creator

The account that deploys the view. Creators earn 100% of consumer access payments and can withdraw their accumulated balance at any time. The creator address is set at deployment and cannot be changed.

### Hosts

Registered network nodes that serve view data to consumers. Hosts register on-chain with cryptographic identity proofs (peer key + node identity key) and then report pricing parameters on the views they serve. Each host independently reports a **complexity coefficient** and a **price per view**. The view contract averages all host reports to determine the base price.

### Stakers

Any account can stake SHNZ on a view to signal demand. Staking increases the view's **popularity score**, which drives up the price via a bounded premium. Stakers can unstake at any time and receive their SHNZ back immediately.

### Consumers

End users who pay SHNZ to access view data. Consumers are identified by a **DID** (Decentralized Identifier). They can access views directly or delegate access to another address. Consumers accumulate a balance within each view they access and can transfer that balance to other views.

## Pricing Model

The View price is determined by three factors: **host reports**, **staking popularity**, and the **Shinzo protocol cut**.

### Base Price

The base price comes from host reports:

**Base Price = Average Price Per View x Average Complexity Coefficient**

Each host independently reports these two values. The view averages them across all reporting hosts, creating a decentralized pricing oracle where no single entity controls the price.

### Popularity Premium

Staking increases the price via a bounded premium that approaches a maximum of **4x** the base price:

| Total Stake | Multiplier | Effect |
|-------------|-----------|--------|
| 0 SHNZ | 1.0x | Base price only |
| 0.33 SHNZ | ~1.5x | Moderate premium |
| 1 SHNZ | ~2.5x | Strong premium |
| 10 SHNZ | ~3.7x | Near maximum |
| 100+ SHNZ | ~4.0x | Maximum premium |

The premium follows the formula: **premium = 30,000 x totalStake / (totalStake + 1 SHNZ)** basis points, capped at 30,000 bps (3x additional multiplier on top of base).

### Shinzo Protocol Cut

A flat **5%** cut is applied to the final price regardless of how it was calculated. This applies even when a custom pricing contract is used.

**Final Price = (Base Price x Popularity Multiplier) x 0.95**

### Custom Pricing

Creators can deploy a view with a custom pricing contract that overrides the default base price calculation. The custom contract can implement any pricing logic. The 5% Shinzo cut is always applied after the custom price is returned.

## Staking & Popularity

Staking is the mechanism by which the network discovers which views are valuable.

- **Stake**: Send SHNZ to a view's stake function. The SHNZ is held by the view contract.
- **Unstake**: Withdraw any amount up to your staked balance. SHNZ is returned immediately.
- **Popularity**: The total amount of SHNZ staked across all stakers on a view.

### Why Stake?

Staking creates a positive feedback loop:

1. **Higher stake** increases the view's price
2. **Higher price** means more revenue for hosts serving that view
3. **More hosts** means better availability and reliability for consumers
4. **Better service** attracts more consumers, generating more revenue for the creator

Staking is an economic signal. A view with high stake is a view the community values, and the pricing model rewards that signal by directing more revenue to the hosts and creators that serve popular data.

## Host Reporting

Hosts are the infrastructure providers that serve view data. Their pricing reports directly determine the base cost of accessing a view.

### How It Works

1. A host registers on the **HostRegistry precompile** with cryptographic proofs of its peer key and node identity key.
2. The host's registration is also recorded on SourceHub via ICA, establishing an ACP relationship.
3. Once registered, the host can call report on any View contract to submit two values:
    - **Complexity Coefficient** — Reflects how computationally expensive the view is to serve
    - **Price Per View** — The host's desired price for serving a single query
4. Multiple hosts can report on the same view. The view averages all reports.

### Decentralized Pricing

No single host controls the price. The averaging mechanism means:

- If one host reports very high prices, it gets diluted by others
- New hosts joining a view naturally adjust the average
- The market converges toward a fair price based on actual serving costs

## Consumer Access & Payment

Consumers pay SHNZ to access view data. Every payment is recorded on-chain and credits both the consumer's balance and the creator's withdrawable earnings.

### Direct Access

A consumer calls access with their DID and sends SHNZ. The view records:

- The consumer's DID (for tracking unique consumers)
- The consumer's balance (cumulative SHNZ spent on this view)
- The creator's balance (cumulative earnings)

### Delegated Access

A consumer can pay on behalf of another address using accessFrom. The delegate receives the balance credit while the payer sends the SHNZ. This enables third-party payment models.

### Consumer Balances

Each consumer accumulates a balance within each view they access. This balance represents their spending history and can be transferred to other views.

## Value Transfer Between Views

Consumers can move their balance from one view to another. This enables **composable data products** where value flows across a graph of interconnected views.

### How It Works

1. A consumer has balance in **View A** from previous access payments
2. The consumer calls transferBalance on View A, targeting View B
3. View A verifies the consumer has sufficient balance and sends SHNZ to View B
4. View B verifies the caller is a registered view (via ViewRegistry) and credits the consumer's balance

### Why This Matters

Value transfer creates a data product ecosystem:

- Views can form economic relationships with each other
- Consumers aren't locked into a single view — their spending has portability
- Creators benefit from network effects as value flows through connected views

## Creator Economics

Creators are the builders of views. They define what data is available and earn from every consumer access.

### Revenue

- **100% of consumer access payments** go to the creator's balance
- Revenue accumulates in the view contract until the creator withdraws
- The creator can withdraw their full balance at any time

### Incentive Alignment

The pricing model aligns creator incentives with network health:

- **More hosts reporting** increases pricing accuracy, building consumer trust
- **More stakers** increases the price, directly increasing creator revenue per access
- **More consumers** generates more total revenue
- **Higher quality views** attract all three, creating a virtuous cycle

## SHNZ Flow Overview

### Consumer Access Flow

Consumer pays SHNZ via access → View Contract credits consumer balance and creator balance → Creator withdraws via withdrawCreatorBalance → Creator Wallet

### Staking Flow

Staker sends SHNZ via stake → View Contract popularity pool (increases popularity score, increases price via premium) → Staker calls unstake → SHNZ returned to Staker

### Cross-View Transfer

Consumer has balance in View A → Consumer calls transferBalance targeting View B → View A sends SHNZ and calls receiveBalance on View B → View B verifies View A is registered → View B credits consumer balance

### Full Economic Cycle

- **Stakers** stake SHNZ on the View Contract, increasing the price
- **Hosts** report pricing parameters to the View Contract
- **Consumers** pay SHNZ to the View Contract for access
- **Creator** withdraws accumulated earnings from the View Contract
