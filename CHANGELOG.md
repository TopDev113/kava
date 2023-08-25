<!--
Guiding Principles:

Changelogs are for humans, not machines.
There should be an entry for every single version.
The same types of changes should be grouped.
Versions and sections should be linkable.
The latest version comes first.
The release date of each version is displayed.
Mention whether you follow Semantic Versioning.

Usage:

Change log entries are to be added to the Unreleased section under the
appropriate stanza (see below). Each entry should ideally include a tag and
the Github issue reference in the following format:

* (<tag>) \#<issue-number> message

The issue numbers will later be link-ified during the release process so you do
not have to worry about including a link manually, but you can if you wish.

Types of changes (Stanzas):

"Features" for new features.
"Improvements" for changes in existing functionality.
"Deprecated" for soon-to-be removed features.
"Bug Fixes" for any bug fixes.
"Client Breaking" for breaking CLI commands and REST routes.
"State Machine Breaking" for breaking the AppState

Ref: https://keepachangelog.com/en/1.0.0/
-->

# Changelog

## [Unreleased]

### State Machine Breaking
[\#1158](https://github.com/Kava-Labs/kava/pull/1158) Split existing auction `bid_duration` parameter into `forward_bid_duration` and `reverse_bid_duration`

## [v0.16.1](https://github.com/Kava-Labs/kava/releases/tag/v0.16.1)

### State Machine Breaking

[\#1152](https://github.com/Kava-Labs/kava/pull/1152) Fix MultiSpend Proposal With Async Upgrade Time 

## [v0.16.0](https://github.com/Kava-Labs/kava/releases/tag/v0.16.0)

### State Machine Breaking
[\#1106](https://github.com/Kava-Labs/kava/pull/1106) Upgrades app to cosmos-sdk v0.44.x and adds IBC and ICS-20 modules. 

## [v0.13.0]

* Hard Protocol - Introduces borrowing functionality to HARD protocol. See full [spec](https://github.com/Kava-Labs/kava/tree/master/x/hard/spec)

### Breaking changes

[\#750](https://github.com/Kava-Labs/kava/pull/750) Update CDP liquidations to allow for liquidation by external keeper.

[\#751](https://github.com/Kava-Labs/kava/pull/751) Use accumulators for CDP interest accumulation.

[\#780](https://github.com/Kava-Labs/kava/pull/780) Moves HARD token distribution from `harvest` module to `incentive` module. All HARD supply, borrow, and delegator reward objects and claims are moved to the `incentive` module.



## [v0.12.0]

[\#701](https://github.com/Kava-Labs/kava/pull/701) Patch issue that prevented atomic swaps from completing successfully
## [v0.11.0]

[\#591](https://github.com/Kava-Labs/kava/pull/591) Add a `raw-params` cli method to query raw parameter values for use in manual verification of gov proposals.

[\#596](https://github.com/Kava-Labs/kava/pull/596) Add REST client and CLI query to get module account information for the CDP module

[\#590](https://github.com/Kava-Labs/kava/pull/590) Add CLI query to return kavadist module account balance

[\#584](https://github.com/Kava-Labs/kava/pulls/584) Add REST client and CLI queries for `kavadist` module

[\#578](https://github.com/Kava-Labs/kava/pulls/578) Add v0.3 compatible REST client that supports

[\#629](https://github.com/Kava-Labs/kava/pulls/629) Add CDP collateral type as a field for CDPs and collateral parameters.

[\#658](https://github.com/Kava-Labs/kava/pulls/658) Add harvest v1 and HARD token distribution schedule

### Breaking changes

* CDPs have an additional field, Type, which is a string that represents the unique collateral type that this CDP holds. This enables, for example, a single denom such as 'bnb' to have two CDP types, 'bnb-a' and 'bnb-b'.
* CollateralParam has an additional field, Type, which is a string that represents the collateral type of CDPs that this collateral parameter governs. It must be non-empty at genesis or when altering CDP fields. It is UNSAFE to alter the type of an existing collateral param using unchain governance.
* CDP messages must specify the collateral type 'bnb-a', rather than the denom of the cdp.
* In the incentive module, fields previously named `Denom` have been changed to `CollateralType`. Previously, 'Denom' was validated to check that it satisfied `sdk.ValidateDenom`, now, the validation checks that the `CollateralType` is not blank.
* Incentive module messages now require the user to specify the collateral type ('bnb-a'), rather than the denom of the cdp ('bnb')

```plaintext
/v0_3/node_info
/v0_3/auth/accounts/<address>
/v0_3/<hash>
/v0_3/txs
/v0_3/staking/delegators/<address>/delegations
/v0_3/staking/delegators/<address>/unbonding_delegations
/v0_3/distribution/delegators/<address>/rewards
```

[\#598](https://github.com/Kava-Labs/kava/pulls/598) CLI and REST queries for committee proposals (ie `kvcli q committee proposal 1`) now query the historical state to return the proposal object before it was deleted from state
[\#625](https://github.com/Kava-Labs/kava/pull/625) The Cosmos SDK has been updated to v0.39.1. This brings with it several breaking changes detailed [in their changelog](https://github.com/cosmos/cosmos-sdk/blob/v0.39.1/CHANGELOG.md). Notably account JSON serialization has been modified to use amino instead of the Go stdlib, so numbers are serialized to strings, and public keys are no longer encoded into bech32 strings. Also pruning config has changed: `pruning=everything` and `pruning=nothing` still work but there are different flags for custom pruning configuration.

## [v0.8.1](https://github.com/Kava-Labs/kava/releases/tag/v0.8.1) kava-3 Patch Release

This version mitigates a memory leak in tendermint that was found prior to launching kava-3. It is fully compatible with v0.8.0 and is intended to replace that version as the canonical software version for upgrading the Kava mainnet from kava-2 to kava-3. Note that there are no breaking changes between the versions, but a safety check was added to this version to prevent starting the node with an unsafe configuration.

### Bugfix

The default tendermint pruning strategy, `pruning="syncable"` is currently unsafe due to a [memory leak](https://github.com/tendermint/iavl/issues/256) that can cause irrecoverable data loss. This patch release prevents `kvd` from being started with the `pruning="syncable"` configuration. Until a patch for tendermint is released, the ONLY pruning strategies that are safe to run are `everything` (an archival node) or `nothing` (only the most recent state is kept). It is strongly recommended that validators use `pruning="nothing"` for kava-3. It is expected that a patch to tendermint will be released in a non-breaking manner and that nodes will be able to update seamlessly after the launch of kava-3.

The steps for upgrading to kava-3 can be found [here](https://github.com/Kava-Labs/kava/blob/v0.10.0/contrib/kava-3/migration.md). Please note the additional section on [pruning](https://github.com/Kava-Labs/kava/blob/v0.10.0/contrib/kava-3/migration.md#Pruning).

## [v0.8.0](https://github.com/Kava-Labs/kava/releases/tag/v0.8.0) kava-3 Release

This version is intended to be the canonical software version for upgrading the Kava mainnet from kava-2 to kava-3. As a result, no subsequent versions of Kava will be released until kava-3 launches unless necessary due to critical state-machine faults that require a new version to launch successfully.

### Migration

The steps for upgrading to kava-3 can be found [here](https://github.com/Kava-Labs/kava/blob/v0.10.0/contrib/kava-3/migration.md)

### Features

This is the first release that includes all the modules which comprise the [CDP system](https://docs.kava.io/).

### State Machine Breaking Changes

(sdk) Update Cosmos-SDK version to v0.38.4. To review cosmos-sdk changes, see the [changelog](https://github.com/cosmos/cosmos-sdk/blob/v0.38.4/CHANGELOG.md).


## [v0.3.5](https://github.com/Kava-Labs/kava/releases/tag/v0.3.5)

Bump tendermint version to 0.32.10 to address [cosmos security advisory Lavender](https://forum.cosmos.network/t/cosmos-mainnet-security-advisory-lavender/3511)

## [v0.3.2](https://github.com/Kava-Labs/kava/releases/tag/v0.3.2)

[\#364](https://github.com/Kava-Labs/kava/pulls/364)  Use new BIP44 coin type in the CLI, retain support for the old one through a flag.

## [v0.3.1](https://github.com/Kava-Labs/kava/releases/tag/v0.3.1)

[\#266](https://github.com/Kava-Labs/kava/pulls/266) Bump tendermint version to 0.32.7 to address cosmos security advisory [Periwinkle](https://forum.cosmos.network/t/cosmos-mainnet-security-advisory-periwinkle/2911)

## [v0.3.0](https://github.com/Kava-Labs/kava/releases/tag/v0.3.0)

### Features

[\#253](https://github.com/Kava-Labs/kava/pulls/253) Add a new validator vesting account type, which releases coins on a periodic vesting schedule based on if a specific validator signed sufficient pre-commits. If the validator didn't sign enough pre-commits, the vesting coins are burned or sent to a return address.

[\#260](https://github.com/Kava-Labs/kava/pulls/260) Pin to cosmos-sdk commit #18de630 (tendermint 0.32.6)

### Improvements

[\#257](https://github.com/Kava-Labs/kava/pulls/257) Include scripts to run large-scale simulations remotely using aws-batch
