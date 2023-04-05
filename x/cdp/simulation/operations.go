package simulation

// import (
// 	"fmt"
// 	"math/rand"

// 	"github.com/cosmos/cosmos-sdk/baseapp"
// 	"github.com/cosmos/cosmos-sdk/codec"
// 	"github.com/cosmos/cosmos-sdk/simapp/helpers"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/cosmos/cosmos-sdk/x/simulation"

// 	appparams "github.com/kava-labs/kava/app/params"
// 	"github.com/kava-labs/kava/x/cdp/keeper"
// 	"github.com/kava-labs/kava/x/cdp/types"
// )

// // Simulation operation weights constants
// const (
// 	OpWeightMsgCdp = "op_weight_msg_cdp"
// )

// // WeightedOperations returns all the operations from the module with their respective weights
// func WeightedOperations(
// 	appParams simulation.AppParams, cdc *codec.Codec, ak types.AccountKeeper,
// 	k keeper.Keeper, pfk types.PricefeedKeeper,
// ) simulation.WeightedOperations {
// 	var weightMsgCdp int

// 	appParams.GetOrGenerate(cdc, OpWeightMsgCdp, &weightMsgCdp, nil,
// 		func(_ *rand.Rand) {
// 			weightMsgCdp = appparams.DefaultWeightMsgCdp
// 		},
// 	)

// 	return simulation.WeightedOperations{
// 		simulation.NewWeightedOperation(
// 			weightMsgCdp,
// 			SimulateMsgCdp(ak, k, pfk),
// 		),
// 	}
// }

// // SimulateMsgCdp generates a MsgCreateCdp or MsgDepositCdp with random values.
// func SimulateMsgCdp(ak types.AccountKeeper, k keeper.Keeper, pfk types.PricefeedKeeper) simulation.Operation {
// 	return func(
// 		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simulation.Account, chainID string,
// 	) (simulation.OperationMsg, []simulation.FutureOperation, error) {

// 		simAccount, _ := simulation.RandomAcc(r, accs)
// 		acc := ak.GetAccount(ctx, simAccount.Address)
// 		if acc == nil {
// 			return simulation.NoOpMsg(types.ModuleName), nil, nil
// 		}

// 		coins := acc.GetCoins()
// 		collateralParams := k.GetParams(ctx).CollateralParams
// 		if len(collateralParams) == 0 {
// 			return simulation.NoOpMsg(types.ModuleName), nil, nil
// 		}

// 		randCollateralParam := collateralParams[r.Intn(len(collateralParams))]
// 		debtParam, _ := k.GetDebtParam(ctx, randCollateralParam.DebtLimit.Denom)
// 		if coins.AmountOf(randCollateralParam.Denom).IsZero() {
// 			return simulation.NoOpMsg(types.ModuleName), nil, nil
// 		}

// 		price, err := pfk.GetCurrentPrice(ctx, randCollateralParam.SpotMarketID)
// 		if err != nil {
// 			return simulation.NoOpMsg(types.ModuleName), nil, nil // pricefeed going down is an expected event
// 		}
// 		// convert the price to the same units as the debt param
// 		priceShifted := ShiftDec(price.Price, debtParam.ConversionFactor)

// 		spendableCoins := acc.SpendableCoins(ctx.BlockTime())
// 		fees, err := simulation.RandomFees(r, ctx, spendableCoins)
// 		if err != nil {
// 			return simulation.NoOpMsg(types.ModuleName), nil, err
// 		}
// 		spendableCoins = spendableCoins.Sub(fees)

// 		existingCDP, found := k.GetCdpByOwnerAndCollateralType(ctx, acc.GetAddress(), randCollateralParam.Type)
// 		if !found {
// 			// calculate the minimum amount of collateral that is needed to create a cdp with the debt floor amount of debt and the minimum liquidation ratio
// 			// (debtFloor * liquidationRatio)/priceShifted
// 			minCollateralDeposit := (sdk.NewDecFromInt(debtParam.DebtFloor).Mul(randCollateralParam.LiquidationRatio)).Quo(priceShifted)
// 			// convert to proper collateral units
// 			minCollateralDeposit = ShiftDec(minCollateralDeposit, randCollateralParam.ConversionFactor)
// 			// convert to integer and always round up
// 			minCollateralDepositRounded := minCollateralDeposit.TruncateInt().Add(sdk.OneInt())
// 			if spendableCoins.AmountOf(randCollateralParam.Denom).LT(minCollateralDepositRounded) {
// 				// account doesn't have enough funds to open a cdp for the min debt amount
// 				return simulation.NewOperationMsgBasic(types.ModuleName, "no-operation", "insufficient funds to open cdp", false, nil), nil, nil
// 			}
// 			// set the max collateral deposit to the amount of coins in the account
// 			maxCollateralDeposit := spendableCoins.AmountOf(randCollateralParam.Denom)

// 			// randomly select a collateral deposit amount
// 			collateralDeposit := sdkmath.NewInt(int64(simulation.RandIntBetween(r, int(minCollateralDepositRounded.Int64()), int(maxCollateralDeposit.Int64()))))
// 			// calculate how much the randomly selected deposit is worth
// 			collateralDepositValue := ShiftDec(sdk.NewDecFromInt(collateralDeposit), randCollateralParam.ConversionFactor.Neg()).Mul(priceShifted)
// 			// calculate the max amount of debt that could be drawn for the chosen deposit
// 			maxDebtDraw := collateralDepositValue.Quo(randCollateralParam.LiquidationRatio).TruncateInt()
// 			// check that the debt limit hasn't been reached
// 			availableAssetDebt := randCollateralParam.DebtLimit.Amount.Sub(k.GetTotalPrincipal(ctx, randCollateralParam.Type, debtParam.Denom))
// 			if availableAssetDebt.LTE(debtParam.DebtFloor) {
// 				// debt limit has been reached
// 				return simulation.NewOperationMsgBasic(types.ModuleName, "no-operation", "debt limit reached, cannot open cdp", false, nil), nil, nil
// 			}
// 			// ensure that the debt draw does not exceed the debt limit
// 			maxDebtDraw = sdk.MinInt(maxDebtDraw, availableAssetDebt)
// 			// randomly select a debt draw amount
// 			debtDraw := sdkmath.NewInt(int64(simulation.RandIntBetween(r, int(debtParam.DebtFloor.Int64()), int(maxDebtDraw.Int64()))))

// 			msg := types.NewMsgCreateCDP(acc.GetAddress(), sdk.NewCoin(randCollateralParam.Denom, collateralDeposit), sdk.NewCoin(debtParam.Denom, debtDraw), randCollateralParam.Type)

// 			tx := helpers.GenTx(
// 				[]sdk.Msg{msg},
// 				fees,
// 				helpers.DefaultGenTxGas,
// 				chainID,
// 				[]uint64{acc.GetAccountNumber()},
// 				[]uint64{acc.GetSequence()},
// 				simAccount.PrivKey,
// 			)

// 			_, _, err := app.Deliver(tx)
// 			if err != nil {
// 				// to aid debugging, add the stack trace to the comment field of the returned opMsg
// 				return simulation.NewOperationMsg(msg, false, fmt.Sprintf("%+v", err)), nil, err
// 			}

// 			return simulation.NewOperationMsg(msg, true, ""), nil, nil
// 		}

// 		// a cdp already exists, deposit to it, draw debt from it, or repay debt to it
// 		// close 25% of the time
// 		if canClose(spendableCoins, existingCDP, debtParam.Denom) && shouldClose(r) {
// 			repaymentAmount := spendableCoins.AmountOf(debtParam.Denom)
// 			// check that repayment isn't below the debt floor
// 			if existingCDP.Principal.Amount.Add(existingCDP.AccumulatedFees.Amount).Sub(repaymentAmount).LT(debtParam.DebtFloor) {
// 				repaymentAmount = existingCDP.Principal.Amount.Add(existingCDP.AccumulatedFees.Amount).Sub(debtParam.DebtFloor)
// 			}
// 			msg := types.NewMsgRepayDebt(acc.GetAddress(), randCollateralParam.Type, sdk.NewCoin(debtParam.Denom, repaymentAmount))

// 			tx := helpers.GenTx(
// 				[]sdk.Msg{msg},
// 				fees,
// 				helpers.DefaultGenTxGas,
// 				chainID,
// 				[]uint64{acc.GetAccountNumber()},
// 				[]uint64{acc.GetSequence()},
// 				simAccount.PrivKey,
// 			)

// 			_, _, err := app.Deliver(tx)
// 			if err != nil {
// 				return simulation.NewOperationMsg(msg, false, fmt.Sprintf("%+v", err)), nil, err
// 			}

// 			return simulation.NewOperationMsg(msg, true, ""), nil, nil
// 		}

// 		// deposit 25% of the time
// 		if hasCoins(spendableCoins, randCollateralParam.Denom) && shouldDeposit(r) {
// 			randDepositAmount := sdkmath.NewInt(int64(simulation.RandIntBetween(r, 1, int(spendableCoins.AmountOf(randCollateralParam.Denom).Int64()))))
// 			msg := types.NewMsgDeposit(acc.GetAddress(), acc.GetAddress(), sdk.NewCoin(randCollateralParam.Denom, randDepositAmount), randCollateralParam.Type)

// 			tx := helpers.GenTx(
// 				[]sdk.Msg{msg},
// 				fees,
// 				helpers.DefaultGenTxGas,
// 				chainID,
// 				[]uint64{acc.GetAccountNumber()},
// 				[]uint64{acc.GetSequence()},
// 				simAccount.PrivKey,
// 			)

// 			_, _, err := app.Deliver(tx)
// 			if err != nil {
// 				return simulation.NewOperationMsg(msg, false, fmt.Sprintf("%+v", err)), nil, err
// 			}

// 			return simulation.NewOperationMsg(msg, true, ""), nil, nil
// 		}

// 		// draw debt 25% of the time
// 		if shouldDraw(r) {
// 			collateralShifted := ShiftDec(sdk.NewDecFromInt(existingCDP.Collateral.Amount), randCollateralParam.ConversionFactor.Neg())
// 			collateralValue := collateralShifted.Mul(priceShifted)
// 			newFeesAccumulated := k.CalculateNewInterest(ctx, existingCDP)
// 			totalFees := existingCDP.AccumulatedFees.Add(newFeesAccumulated)
// 			// given the current collateral value, calculate how much debt we could add while maintaining a valid liquidation ratio
// 			debt := existingCDP.Principal.Add(totalFees)
// 			maxTotalDebt := collateralValue.Quo(randCollateralParam.LiquidationRatio)
// 			maxDebt := (maxTotalDebt.Sub(sdk.NewDecFromInt(debt.Amount))).Mul(sdk.MustNewDecFromStr("0.95")).TruncateInt()
// 			if maxDebt.LTE(sdk.OneInt()) {
// 				// debt in cdp is maxed out
// 				return simulation.NewOperationMsgBasic(types.ModuleName, "no-operation", "cdp debt maxed out, cannot draw more debt", false, nil), nil, nil
// 			}
// 			// check if the debt limit has been reached
// 			availableAssetDebt := randCollateralParam.DebtLimit.Amount.Sub(k.GetTotalPrincipal(ctx, randCollateralParam.Type, debtParam.Denom))
// 			if availableAssetDebt.LTE(sdk.OneInt()) {
// 				// debt limit has been reached
// 				return simulation.NewOperationMsgBasic(types.ModuleName, "no-operation", "debt limit reached, cannot draw more debt", false, nil), nil, nil
// 			}
// 			maxDraw := sdk.MinInt(maxDebt, availableAssetDebt)

// 			randDrawAmount := sdkmath.NewInt(int64(simulation.RandIntBetween(r, 1, int(maxDraw.Int64()))))
// 			msg := types.NewMsgDrawDebt(acc.GetAddress(), randCollateralParam.Type, sdk.NewCoin(debtParam.Denom, randDrawAmount))

// 			tx := helpers.GenTx(
// 				[]sdk.Msg{msg},
// 				fees,
// 				helpers.DefaultGenTxGas,
// 				chainID,
// 				[]uint64{acc.GetAccountNumber()},
// 				[]uint64{acc.GetSequence()},
// 				simAccount.PrivKey,
// 			)

// 			_, _, err := app.Deliver(tx)
// 			if err != nil {
// 				return simulation.NewOperationMsg(msg, false, fmt.Sprintf("%+v", err)), nil, err
// 			}

// 			return simulation.NewOperationMsg(msg, true, ""), nil, nil
// 		}

// 		// repay debt 25% of the time
// 		if hasCoins(spendableCoins, debtParam.Denom) {
// 			debt := existingCDP.Principal.Amount.Add(existingCDP.AccumulatedFees.Amount)
// 			payableDebt := debt.Sub(debtParam.DebtFloor)
// 			if payableDebt.IsZero() {
// 				return simulation.NewOperationMsgBasic(types.ModuleName, "no-operation", "cannot make partial repayment, cdp at debt floor", false, nil), nil, nil
// 			}
// 			maxRepay := sdk.MinInt(
// 				spendableCoins.AmountOf(debtParam.Denom),
// 				payableDebt,
// 			)
// 			var randRepayAmount sdkmath.Int
// 			if maxRepay.Equal(sdk.OneInt()) {
// 				randRepayAmount = sdk.OneInt()
// 			} else {
// 				randRepayAmount = sdkmath.NewInt(int64(simulation.RandIntBetween(r, 1, int(maxRepay.Int64()))))
// 			}

// 			msg := types.NewMsgRepayDebt(acc.GetAddress(), randCollateralParam.Type, sdk.NewCoin(debtParam.Denom, randRepayAmount))

// 			tx := helpers.GenTx(
// 				[]sdk.Msg{msg},
// 				fees,
// 				helpers.DefaultGenTxGas,
// 				chainID,
// 				[]uint64{acc.GetAccountNumber()},
// 				[]uint64{acc.GetSequence()},
// 				simAccount.PrivKey,
// 			)

// 			_, _, err := app.Deliver(tx)
// 			if err != nil {
// 				return simulation.NewOperationMsg(msg, false, fmt.Sprintf("%+v", err)), nil, err
// 			}

// 			return simulation.NewOperationMsg(msg, true, ""), nil, nil
// 		}

// 		return simulation.NoOpMsg(types.ModuleName), nil, nil
// 	}
// }

// func shouldDraw(r *rand.Rand) bool {
// 	threshold := 50
// 	value := simulation.RandIntBetween(r, 1, 100)
// 	return value > threshold
// }

// func shouldDeposit(r *rand.Rand) bool {
// 	threshold := 66
// 	value := simulation.RandIntBetween(r, 1, 100)
// 	return value > threshold
// }

// func hasCoins(spendableCoins sdk.Coins, denom string) bool {
// 	return spendableCoins.AmountOf(denom).IsPositive()
// }

// func shouldClose(r *rand.Rand) bool {
// 	threshold := 75
// 	value := simulation.RandIntBetween(r, 1, 100)
// 	return value > threshold
// }

// func canClose(spendableCoins sdk.Coins, c types.CDP, denom string) bool {
// 	repaymentAmount := c.Principal.Add(c.AccumulatedFees).Amount
// 	return spendableCoins.AmountOf(denom).GTE(repaymentAmount)
// }
