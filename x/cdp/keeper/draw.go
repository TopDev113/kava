package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/cdp/types"
)

// AddPrincipal adds debt to a cdp if the additional debt does not put the cdp below the liquidation ratio
func (k Keeper) AddPrincipal(ctx sdk.Context, owner sdk.AccAddress, collateralType string, principal sdk.Coin) error {
	// validation
	cdp, found := k.GetCdpByOwnerAndCollateralType(ctx, owner, collateralType)
	if !found {
		return errorsmod.Wrapf(types.ErrCdpNotFound, "owner %s, denom %s", owner, collateralType)
	}
	err := k.ValidatePrincipalDraw(ctx, principal, cdp.Principal.Denom)
	if err != nil {
		return err
	}

	err = k.ValidateDebtLimit(ctx, cdp.Type, principal)
	if err != nil {
		return err
	}
	k.hooks.BeforeCDPModified(ctx, cdp)
	cdp = k.SynchronizeInterest(ctx, cdp)

	err = k.ValidateCollateralizationRatio(ctx, cdp.Collateral, cdp.Type, cdp.Principal.Add(principal), cdp.AccumulatedFees)
	if err != nil {
		return err
	}

	// mint the principal and send it to the cdp owner
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(principal))
	if err != nil {
		panic(err)
	}
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, owner, sdk.NewCoins(principal))
	if err != nil {
		panic(err)
	}

	// mint the corresponding amount of debt coins in the cdp module account
	err = k.MintDebtCoins(ctx, types.ModuleName, k.GetDebtDenom(ctx), principal)
	if err != nil {
		panic(err)
	}

	// emit cdp draw event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCdpDraw,
			sdk.NewAttribute(sdk.AttributeKeyAmount, principal.String()),
			sdk.NewAttribute(types.AttributeKeyCdpID, fmt.Sprintf("%d", cdp.ID)),
		),
	)

	// update cdp state
	cdp.Principal = cdp.Principal.Add(principal)

	// increment total principal for the input collateral type
	k.IncrementTotalPrincipal(ctx, cdp.Type, principal)

	// set cdp state and indexes in the store
	collateralToDebtRatio := k.CalculateCollateralToDebtRatio(ctx, cdp.Collateral, cdp.Type, cdp.GetTotalPrincipal())
	return k.UpdateCdpAndCollateralRatioIndex(ctx, cdp, collateralToDebtRatio)
}

// RepayPrincipal removes debt from the cdp
// If all debt is repaid, the collateral is returned to depositors and the cdp is removed from the store
func (k Keeper) RepayPrincipal(ctx sdk.Context, owner sdk.AccAddress, collateralType string, payment sdk.Coin) error {
	// validation
	cdp, found := k.GetCdpByOwnerAndCollateralType(ctx, owner, collateralType)
	if !found {
		return errorsmod.Wrapf(types.ErrCdpNotFound, "owner %s, denom %s", owner, collateralType)
	}

	err := k.ValidatePaymentCoins(ctx, cdp, payment)
	if err != nil {
		return err
	}

	err = k.ValidateBalance(ctx, payment, owner)
	if err != nil {
		return err
	}
	k.hooks.BeforeCDPModified(ctx, cdp)
	cdp = k.SynchronizeInterest(ctx, cdp)

	// Note: assumes cdp.Principal and cdp.AccumulatedFees don't change during calculations
	totalPrincipal := cdp.GetTotalPrincipal()

	// calculate fee and principal payment
	feePayment, principalPayment := k.calculatePayment(ctx, totalPrincipal, cdp.AccumulatedFees, payment)

	err = k.validatePrincipalPayment(ctx, cdp, principalPayment)
	if err != nil {
		return err
	}
	// send the payment from the sender to the cpd module
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, owner, types.ModuleName, sdk.NewCoins(feePayment.Add(principalPayment)))
	if err != nil {
		return err
	}

	// burn the payment coins
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(feePayment.Add(principalPayment)))
	if err != nil {
		panic(err)
	}

	// burn the corresponding amount of debt coins
	cdpDebt := k.getModAccountDebt(ctx, types.ModuleName)
	paymentAmount := feePayment.Add(principalPayment).Amount

	debtDenom := k.GetDebtDenom(ctx)
	coinsToBurn := sdk.NewCoin(debtDenom, paymentAmount)

	if paymentAmount.GT(cdpDebt) {
		coinsToBurn = sdk.NewCoin(debtDenom, cdpDebt)
	}

	err = k.BurnDebtCoins(ctx, types.ModuleName, debtDenom, coinsToBurn)

	if err != nil {
		panic(err)
	}

	// emit repayment event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCdpRepay,
			sdk.NewAttribute(sdk.AttributeKeyAmount, feePayment.Add(principalPayment).String()),
			sdk.NewAttribute(types.AttributeKeyCdpID, fmt.Sprintf("%d", cdp.ID)),
		),
	)

	// remove the old collateral:debt ratio index

	// update cdp state
	if !principalPayment.IsZero() {
		cdp.Principal = cdp.Principal.Sub(principalPayment)
	}
	cdp.AccumulatedFees = cdp.AccumulatedFees.Sub(feePayment)

	// decrement the total principal for the input collateral type
	k.DecrementTotalPrincipal(ctx, cdp.Type, feePayment.Add(principalPayment))

	// if the debt is fully paid, return collateral to depositors,
	// and remove the cdp and indexes from the store
	if cdp.Principal.IsZero() && cdp.AccumulatedFees.IsZero() {
		k.ReturnCollateral(ctx, cdp)
		k.RemoveCdpOwnerIndex(ctx, cdp)
		err := k.DeleteCdpAndCollateralRatioIndex(ctx, cdp)
		if err != nil {
			return err
		}

		// emit cdp close event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeCdpClose,
				sdk.NewAttribute(types.AttributeKeyCdpID, fmt.Sprintf("%d", cdp.ID)),
			),
		)
		return nil
	}

	// set cdp state and update indexes
	collateralToDebtRatio := k.CalculateCollateralToDebtRatio(ctx, cdp.Collateral, cdp.Type, cdp.GetTotalPrincipal())
	return k.UpdateCdpAndCollateralRatioIndex(ctx, cdp, collateralToDebtRatio)
}

// ValidatePaymentCoins validates that the input coins are valid for repaying debt
func (k Keeper) ValidatePaymentCoins(ctx sdk.Context, cdp types.CDP, payment sdk.Coin) error {
	debt := cdp.GetTotalPrincipal()
	if payment.Denom != debt.Denom {
		return errorsmod.Wrapf(types.ErrInvalidPayment, "cdp %d: expected %s, got %s", cdp.ID, debt.Denom, payment.Denom)
	}
	_, found := k.GetDebtParam(ctx, payment.Denom)
	if !found {
		return errorsmod.Wrapf(types.ErrInvalidPayment, "payment denom %s not found", payment.Denom)
	}
	return nil
}

// ReturnCollateral returns collateral to depositors on a cdp and removes deposits from the store
func (k Keeper) ReturnCollateral(ctx sdk.Context, cdp types.CDP) {
	deposits := k.GetDeposits(ctx, cdp.ID)
	for _, deposit := range deposits {
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, deposit.Depositor, sdk.NewCoins(deposit.Amount)); err != nil {
			panic(err)
		}
		k.DeleteDeposit(ctx, cdp.ID, deposit.Depositor)
	}
}

// calculatePayment divides the input payment into the portions that will be used to repay fees and principal
// owed - Principal + AccumulatedFees
// fees - AccumulatedFees
// CONTRACT: owned and payment denoms must be checked before calling this function.
func (k Keeper) calculatePayment(ctx sdk.Context, owed, fees, payment sdk.Coin) (sdk.Coin, sdk.Coin) {
	// divides repayment into principal and fee components, with fee payment applied first.

	feePayment := sdk.NewCoin(payment.Denom, sdk.ZeroInt())
	principalPayment := sdk.NewCoin(payment.Denom, sdk.ZeroInt())
	var overpayment sdk.Coin
	// return zero value coins if payment amount is invalid
	if !payment.Amount.IsPositive() {
		return feePayment, principalPayment
	}
	// check for over payment
	if payment.Amount.GT(owed.Amount) {
		overpayment = payment.Sub(owed)
		payment = payment.Sub(overpayment)
	}
	// if no fees, 100% of payment is principal payment
	if fees.IsZero() {
		return feePayment, payment
	}
	// pay fees before repaying principal
	if payment.Amount.GT(fees.Amount) {
		feePayment = fees
		principalPayment = payment.Sub(fees)
	} else {
		feePayment = payment
	}
	return feePayment, principalPayment
}

// validatePrincipalPayment checks that the payment is either full or does not put the cdp below the debt floor
// CONTRACT: payment denom must be checked before calling this function.
func (k Keeper) validatePrincipalPayment(ctx sdk.Context, cdp types.CDP, payment sdk.Coin) error {
	proposedBalance := cdp.Principal.Amount.Sub(payment.Amount)
	dp, _ := k.GetDebtParam(ctx, payment.Denom)
	if proposedBalance.GT(sdk.ZeroInt()) && proposedBalance.LT(dp.DebtFloor) {
		return errorsmod.Wrapf(types.ErrBelowDebtFloor, "proposed %s < minimum %s", sdk.NewCoin(payment.Denom, proposedBalance), dp.DebtFloor)
	}
	return nil
}
