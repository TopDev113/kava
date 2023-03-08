<!--
order: 3
-->

# Messages

Users can submit various messages to the cdp module which trigger state changes detailed below.

## CreateCDP

CreateCDP sets up and stores a new CDP, adding collateral from the sender, and drawing `Principle` debt.

```go
type MsgCreateCDP struct {
    Sender     sdk.AccAddress
    Collateral sdk.Coin
    Principal  sdk.Coin
}
```

State changes:

- a new CDP is created, `Sender` becomes CDP owner
- collateral taken from `Sender` and sent to cdp module account, new `Deposit` created
- `Principal` stable coins are minted and sent to `Sender`
- equal amount of internal debt coins created and stored in cdp module account

## Deposit

Deposit adds collateral to a CDP in the form of a deposit. Collateral is taken from `Depositor`.

```go
type MsgDeposit struct {
    Owner      sdk.AccAddress
    Depositor  sdk.AccAddress
    Collateral sdk.Coin
}
```

State Changes:

- `Collateral` taken from depositor and sent to cdp module account
- the depositor's `Deposit` struct is updated or a new one created
- cdp fees are updated (see below)

## Withdraw

Withdraw removes collateral from a CDP, provided it would not put the CDP under the liquidation ratio. Collateral is removed from one deposit only.

```go
type MsgWithdraw struct {
    Owner      sdk.AccAddress
    Depositor  sdk.AccAddress
    Collateral sdk.Coin
}
```

State Changes:

- `Collateral` coins are sent from the cdp module account to `Depositor`
- `Collateral` amount of coins subtracted from the `Deposit` struct. If the amount is now zero, the struct is deleted

## DrawDebt

DrawDebt creates debt in a CDP, minting new stable asset which is sent to the sender.

```go
type MsgDrawDebt struct {
    Sender    sdk.AccAddress
    CdpDenom  string
    Principal sdk.Coin
}
```

State Changes:

- mint `Principal` coins and send them to `Sender`, updating the CDP's `Principal` field
- mint equal amount of internal debt coins and store in the module account
- increment total principal for principal denom

## RepayDebt

RepayDebt removes some debt from a CDP and burns the corresponding amount of stable asset from the sender. If all debt is repaid, the collateral is returned to depositors and the cdp is removed from the store

```go
type MsgRepayDebt struct {
    Sender   sdk.AccAddress
    CdpDenom string
    Payment  sdk.Coin
}
```

State Changes:

- burn `Payment` coins taken from `Sender`, updating the CDP by reducing `Principal` field by `Paymment`
- burn an equal amount of internal debt coins
- decrement total principal for payment denom
- if fees and principal are zero, return collateral to depositors and delete the CDP struct:
  - For each deposit, send coins from the cdp module account to the depositor, and delete the deposit struct from store.

## Liquidate

Liquidate enables Keepers to liquidate a Borrower's CDP. If the CDP is below its Loan-to-Value obligations, the CDP's deposits are seized: a small percentage of the seized funds are sent to the Keeper with the rest auctioned off to recover the CDP's outstanding borrowed amount. Any deposited funds leftover that weren't needed to cover the Borrower's debts are returned to the Borrower.

Note: In kava v0.21.x and below, CDP's that have a collateral ratio exactly equal to the liquidation ratio can be liquidated through this method.

```go
// MsgLiquidate attempts to liquidate a borrower's cdp
type MsgLiquidate struct {
	Keeper         sdk.AccAddress `json:"keeper" yaml:"keeper"`
	Borrower       sdk.AccAddress `json:"borrower" yaml:"borrower"`
	CollateralType string         `json:"collateral_type" yaml:"collateral_type"`
}
```

State Changes:

- the CDP's outstanding interest is synchronized so that the deposit and borrow amount are accurate
- the liquidation attempt is validated by comparing the CDP's current collateralization ratio to its liquidation ratio
- the `Keeper` is paid out a percentage of the liquidated position; the exact percentage is specified in the module's params
- the CDP's deposits are seized and used to start an `Auction` to recover the CDP's outstanding borrowed funds
- the module's `TotalPrincipal` for the CDP's collateral type is decremented by the CDP's `Principal`
- the CDP is deleted from the store and removed from the liquidation index

## Fees

At the beginning of each block, fees accumulated since the last update are calculated and added on.

```
feesAccumulated = (outstandingDebt * (feeRate^periods)) - outstandingDebt
```

where:

- `outstandingDebt` is the CDP's `Principal` plus `AccumulatedFees`
- `periods` is the number of seconds since last fee update
- `feeRate` is the per second debt interest rate

Fees are divided between surplus and savings rate. For example, if the savings rate is 0.95, 95% of all fees go towards the savings rate and 5% go to surplus.

In the event that the rounded value of `feesAccumulated` is zero, fees are not updated, and the `FeesUpdated` value on the CDP struct is not updated. When a sufficient number of periods have passed such that the rounded value is no longer zero, fees will be updated.

## Database Indexes

When CDPs are update by the above messages the database indexes are also updated.
