# Begin Blocker

At the start of every block the BeginBlocker of the cdp module:

- updates total CDP fees
- update fees for individual "risky" CDPs
- liquidates CDPs under the collateral ratio
- nets out system debt and, if necessary, starts auctions to re-balance it
- records the last block time

## Update Fees

- The total fees accumulated since the last block across all CDPs are calculated.
- An equal amount of debt coins are minted and sent to the system's CDP module account.
- An equal amount of stable asset coins are minted and sent to the system's liquidator module account

## Update risky cdps

- UpdateFeesForRiskyCdps calculates fees for risky CDPs
- Select the CDPs with 10% of the liquidation ratio - the risky CDPs
- Calculate additional accumulated fees on each of those CDPs
- Update the fees updated time for the CDP to the current block time

## Liquidate CDP

- Get every cdp that is under the liquidation ratio for its collateral type.
- For each cdp:
  - Calculate and update fees since last update.
  - Remove all collateral and internal debt coins from cdp and deposits and delete it. Send the coins to the liquidator module account.
  - Start auctions of a fixed size from this collateral (with any remainder in a smaller sized auction), sending collateral and debt coins to the auction module account.
  - Decrement total principal.

## Net Out System Debt, Re-Balance

- Burn the maximum possible equal amount of debt and stable asset from the liquidator module account.
- If there is enough debt remaining for an auction, start one.
- If there is enough surplus stable asset, minus surplus reserved for the savings rate, remaining for an auction, start one.
- Otherwise do nothing, leave debt/surplus to accumulate over subsequent blocks.

## Distribute Surplus Stable Asset According to the Savings Rate

- If `SavingsDistributionFrequency` seconds have elapsed since the previous distribution, the savings rate is applied to all accounts that hold stable asset.
- Each account that holds stable asset is distributed a ratable portion of the surplus that is apportioned to the savings rate.
- If distribution occurred, the time of the distribution is recorded.

## Update Previous Block Time

The current block time is recorded.
