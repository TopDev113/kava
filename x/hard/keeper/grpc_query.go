package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kava-labs/kava/x/hard/types"
)

type queryServer struct {
	keeper        Keeper
	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
}

// NewQueryServerImpl creates a new server for handling gRPC queries.
func NewQueryServerImpl(keeper Keeper, ak types.AccountKeeper, bk types.BankKeeper) types.QueryServer {
	return &queryServer{
		keeper:        keeper,
		accountKeeper: ak,
		bankKeeper:    bk,
	}
}

var _ types.QueryServer = queryServer{}

func (s queryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get params
	params := s.keeper.GetParams(sdkCtx)

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

func (s queryServer) Accounts(ctx context.Context, req *types.QueryAccountsRequest) (*types.QueryAccountsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	macc := s.accountKeeper.GetModuleAccount(sdkCtx, types.ModuleAccountName)

	accounts := []authtypes.ModuleAccount{
		*macc.(*authtypes.ModuleAccount),
	}

	return &types.QueryAccountsResponse{
		Accounts: accounts,
	}, nil
}

func (s queryServer) Deposits(ctx context.Context, req *types.QueryDepositsRequest) (*types.QueryDepositsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	hasDenom := len(req.Denom) > 0
	hasOwner := len(req.Owner) > 0

	var owner sdk.AccAddress
	var err error
	if hasOwner {
		owner, err = sdk.AccAddressFromBech32(req.Owner)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
		}
	}

	var deposits types.Deposits
	switch {
	case hasOwner && hasDenom:
		deposit, found := s.keeper.GetSyncedDeposit(sdkCtx, owner)
		if found {
			for _, coin := range deposit.Amount {
				if coin.Denom == req.Denom {
					deposits = append(deposits, deposit)
				}
			}
		}
	case hasOwner:
		deposit, found := s.keeper.GetSyncedDeposit(sdkCtx, owner)
		if found {
			deposits = append(deposits, deposit)
		}
	case hasDenom:
		s.keeper.IterateDeposits(sdkCtx, func(deposit types.Deposit) (stop bool) {
			if deposit.Amount.AmountOf(req.Denom).IsPositive() {
				deposits = append(deposits, deposit)
			}
			return false
		})
	default:
		s.keeper.IterateDeposits(sdkCtx, func(deposit types.Deposit) (stop bool) {
			deposits = append(deposits, deposit)
			return false
		})
	}

	// If owner param was specified then deposits array already contains the user's synced deposit
	if hasOwner {
		return &types.QueryDepositsResponse{
			Deposits:   deposits.ToResponse(),
			Pagination: nil,
		}, nil
	}

	// Otherwise we need to simulate syncing of each deposit
	var syncedDeposits types.Deposits
	for _, deposit := range deposits {
		syncedDeposit, _ := s.keeper.GetSyncedDeposit(sdkCtx, deposit.Depositor)
		syncedDeposits = append(syncedDeposits, syncedDeposit)
	}

	// TODO: Use more optimal FilteredPaginate to directly iterate over the store
	// and not fetch everything. This currently also ignores certain fields in
	// the pagination request like Key, CountTotal, Reverse.
	page, limit, err := query.ParsePagination(req.Pagination)
	if err != nil {
		return nil, err
	}

	start, end := client.Paginate(len(syncedDeposits), page, limit, 100)
	if start < 0 || end < 0 {
		syncedDeposits = types.Deposits{}
	} else {
		syncedDeposits = syncedDeposits[start:end]
	}

	return &types.QueryDepositsResponse{
		Deposits:   syncedDeposits.ToResponse(),
		Pagination: nil,
	}, nil
}

func (s queryServer) UnsyncedDeposits(ctx context.Context, req *types.QueryUnsyncedDepositsRequest) (*types.QueryUnsyncedDepositsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	hasDenom := len(req.Denom) > 0
	hasOwner := len(req.Owner) > 0

	var owner sdk.AccAddress
	var err error
	if hasOwner {
		owner, err = sdk.AccAddressFromBech32(req.Owner)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
		}
	}

	var deposits types.Deposits
	switch {
	case hasOwner && hasDenom:
		deposit, found := s.keeper.GetDeposit(sdkCtx, owner)
		if found {
			for _, coin := range deposit.Amount {
				if coin.Denom == req.Denom {
					deposits = append(deposits, deposit)
				}
			}
		}
	case hasOwner:
		deposit, found := s.keeper.GetDeposit(sdkCtx, owner)
		if found {
			deposits = append(deposits, deposit)
		}
	case hasDenom:
		s.keeper.IterateDeposits(sdkCtx, func(deposit types.Deposit) (stop bool) {
			if deposit.Amount.AmountOf(req.Denom).IsPositive() {
				deposits = append(deposits, deposit)
			}
			return false
		})
	default:
		s.keeper.IterateDeposits(sdkCtx, func(deposit types.Deposit) (stop bool) {
			deposits = append(deposits, deposit)
			return false
		})
	}

	page, limit, err := query.ParsePagination(req.Pagination)
	if err != nil {
		return nil, err
	}

	start, end := client.Paginate(len(deposits), page, limit, 100)
	if start < 0 || end < 0 {
		deposits = types.Deposits{}
	} else {
		deposits = deposits[start:end]
	}

	return &types.QueryUnsyncedDepositsResponse{
		Deposits:   deposits.ToResponse(),
		Pagination: nil,
	}, nil
}

func (s queryServer) Borrows(ctx context.Context, req *types.QueryBorrowsRequest) (*types.QueryBorrowsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	hasDenom := len(req.Denom) > 0
	hasOwner := len(req.Owner) > 0

	var owner sdk.AccAddress
	var err error
	if hasOwner {
		owner, err = sdk.AccAddressFromBech32(req.Owner)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
		}
	}

	var borrows types.Borrows
	switch {
	case hasOwner && hasDenom:
		borrow, found := s.keeper.GetSyncedBorrow(sdkCtx, owner)
		if found {
			for _, coin := range borrow.Amount {
				if coin.Denom == req.Denom {
					borrows = append(borrows, borrow)
				}
			}
		}
	case hasOwner:
		borrow, found := s.keeper.GetSyncedBorrow(sdkCtx, owner)
		if found {
			borrows = append(borrows, borrow)
		}
	case hasDenom:
		s.keeper.IterateBorrows(sdkCtx, func(borrow types.Borrow) (stop bool) {
			if borrow.Amount.AmountOf(req.Denom).IsPositive() {
				borrows = append(borrows, borrow)
			}
			return false
		})
	default:
		s.keeper.IterateBorrows(sdkCtx, func(borrow types.Borrow) (stop bool) {
			borrows = append(borrows, borrow)
			return false
		})
	}

	// If owner param was specified then borrows array already contains the user's synced borrow
	if hasOwner {
		return &types.QueryBorrowsResponse{
			Borrows:    borrows.ToResponse(),
			Pagination: nil,
		}, nil
	}

	// Otherwise we need to simulate syncing of each borrow
	var syncedBorrows types.Borrows
	for _, borrow := range borrows {
		syncedBorrow, _ := s.keeper.GetSyncedBorrow(sdkCtx, borrow.Borrower)
		syncedBorrows = append(syncedBorrows, syncedBorrow)
	}

	page, limit, err := query.ParsePagination(req.Pagination)
	if err != nil {
		return nil, err
	}

	start, end := client.Paginate(len(syncedBorrows), page, limit, 100)
	if start < 0 || end < 0 {
		syncedBorrows = types.Borrows{}
	} else {
		syncedBorrows = syncedBorrows[start:end]
	}

	return &types.QueryBorrowsResponse{
		Borrows: syncedBorrows.ToResponse(),
	}, nil
}

func (s queryServer) UnsyncedBorrows(ctx context.Context, req *types.QueryUnsyncedBorrowsRequest) (*types.QueryUnsyncedBorrowsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	hasDenom := len(req.Denom) > 0
	hasOwner := len(req.Owner) > 0

	var owner sdk.AccAddress
	var err error
	if hasOwner {
		owner, err = sdk.AccAddressFromBech32(req.Owner)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
		}
	}

	var borrows types.Borrows
	switch {
	case hasOwner && hasDenom:
		borrow, found := s.keeper.GetBorrow(sdkCtx, owner)
		if found {
			for _, coin := range borrow.Amount {
				if coin.Denom == req.Denom {
					borrows = append(borrows, borrow)
				}
			}
		}
	case hasOwner:
		borrow, found := s.keeper.GetBorrow(sdkCtx, owner)
		if found {
			borrows = append(borrows, borrow)
		}
	case hasDenom:
		s.keeper.IterateBorrows(sdkCtx, func(borrow types.Borrow) (stop bool) {
			if borrow.Amount.AmountOf(req.Denom).IsPositive() {
				borrows = append(borrows, borrow)
			}
			return false
		})
	default:
		s.keeper.IterateBorrows(sdkCtx, func(borrow types.Borrow) (stop bool) {
			borrows = append(borrows, borrow)
			return false
		})
	}

	page, limit, err := query.ParsePagination(req.Pagination)
	if err != nil {
		return nil, err
	}

	start, end := client.Paginate(len(borrows), page, limit, 100)
	if start < 0 || end < 0 {
		borrows = types.Borrows{}
	} else {
		borrows = borrows[start:end]
	}

	return &types.QueryUnsyncedBorrowsResponse{
		Borrows:    borrows.ToResponse(),
		Pagination: nil,
	}, nil
}

func (s queryServer) TotalBorrowed(ctx context.Context, req *types.QueryTotalBorrowedRequest) (*types.QueryTotalBorrowedResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	borrowedCoins, found := s.keeper.GetBorrowedCoins(sdkCtx)
	if !found {
		// Use empty coins instead of returning an error
		borrowedCoins = sdk.NewCoins()
	}

	// If user specified a denom only return coins of that denom type
	if len(req.Denom) > 0 {
		borrowedCoins = sdk.NewCoins(sdk.NewCoin(req.Denom, borrowedCoins.AmountOf(req.Denom)))
	}

	return &types.QueryTotalBorrowedResponse{
		BorrowedCoins: borrowedCoins,
	}, nil
}

func (s queryServer) TotalDeposited(ctx context.Context, req *types.QueryTotalDepositedRequest) (*types.QueryTotalDepositedResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	suppliedCoins, found := s.keeper.GetSuppliedCoins(sdkCtx)
	if !found {
		// Use empty coins instead of returning an error
		suppliedCoins = sdk.NewCoins()
	}

	// If user specified a denom only return coins of that denom type
	if len(req.Denom) > 0 {
		suppliedCoins = sdk.NewCoins(sdk.NewCoin(req.Denom, suppliedCoins.AmountOf(req.Denom)))
	}

	return &types.QueryTotalDepositedResponse{
		SuppliedCoins: suppliedCoins,
	}, nil
}

func (s queryServer) InterestRate(ctx context.Context, req *types.QueryInterestRateRequest) (*types.QueryInterestRateResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var moneyMarketInterestRates types.MoneyMarketInterestRates
	var moneyMarkets types.MoneyMarkets
	if len(req.Denom) > 0 {
		moneyMarket, found := s.keeper.GetMoneyMarket(sdkCtx, req.Denom)
		if !found {
			return nil, types.ErrMoneyMarketNotFound
		}
		moneyMarkets = append(moneyMarkets, moneyMarket)
	} else {
		moneyMarkets = s.keeper.GetAllMoneyMarkets(sdkCtx)
	}

	// Calculate the borrow and supply APY interest rates for each money market
	for _, moneyMarket := range moneyMarkets {
		denom := moneyMarket.Denom
		macc := s.accountKeeper.GetModuleAccount(sdkCtx, types.ModuleName)
		cash := s.bankKeeper.GetBalance(sdkCtx, macc.GetAddress(), denom).Amount

		borrowed := sdk.NewCoin(denom, sdk.ZeroInt())
		borrowedCoins, foundBorrowedCoins := s.keeper.GetBorrowedCoins(sdkCtx)
		if foundBorrowedCoins {
			borrowed = sdk.NewCoin(denom, borrowedCoins.AmountOf(denom))
		}

		reserves, foundReserves := s.keeper.GetTotalReserves(sdkCtx)
		if !foundReserves {
			reserves = sdk.NewCoins()
		}

		// CalculateBorrowRate calculates the current interest rate based on utilization (the fraction of supply that has ien borrowed)
		borrowAPY, err := CalculateBorrowRate(moneyMarket.InterestRateModel, sdk.NewDecFromInt(cash), sdk.NewDecFromInt(borrowed.Amount), sdk.NewDecFromInt(reserves.AmountOf(denom)))
		if err != nil {
			return nil, err
		}

		utilRatio := CalculateUtilizationRatio(sdk.NewDecFromInt(cash), sdk.NewDecFromInt(borrowed.Amount), sdk.NewDecFromInt(reserves.AmountOf(denom)))
		fullSupplyAPY := borrowAPY.Mul(utilRatio)
		realSupplyAPY := fullSupplyAPY.Mul(sdk.OneDec().Sub(moneyMarket.ReserveFactor))

		moneyMarketInterestRate := types.MoneyMarketInterestRate{
			Denom:              denom,
			SupplyInterestRate: realSupplyAPY.String(),
			BorrowInterestRate: borrowAPY.String(),
		}

		moneyMarketInterestRates = append(moneyMarketInterestRates, moneyMarketInterestRate)
	}

	return &types.QueryInterestRateResponse{
		InterestRates: moneyMarketInterestRates,
	}, nil
}

func (s queryServer) Reserves(ctx context.Context, req *types.QueryReservesRequest) (*types.QueryReservesResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	reserveCoins, found := s.keeper.GetTotalReserves(sdkCtx)
	if !found {
		reserveCoins = sdk.Coins{}
	}

	// If user specified a denom only return coins of that denom type
	if len(req.Denom) > 0 {
		reserveCoins = sdk.NewCoins(sdk.NewCoin(req.Denom, reserveCoins.AmountOf(req.Denom)))
	}

	return &types.QueryReservesResponse{
		Amount: reserveCoins,
	}, nil
}

func (s queryServer) InterestFactors(ctx context.Context, req *types.QueryInterestFactorsRequest) (*types.QueryInterestFactorsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	var interestFactors types.InterestFactors
	if len(req.Denom) > 0 {
		// Fetch supply/borrow interest factors for a single denom
		interestFactor := types.InterestFactor{}
		interestFactor.Denom = req.Denom
		supplyInterestFactor, found := s.keeper.GetSupplyInterestFactor(sdkCtx, req.Denom)
		if found {
			interestFactor.SupplyInterestFactor = supplyInterestFactor.String()
		}
		borrowInterestFactor, found := s.keeper.GetBorrowInterestFactor(sdkCtx, req.Denom)
		if found {
			interestFactor.BorrowInterestFactor = borrowInterestFactor.String()
		}
		interestFactors = append(interestFactors, interestFactor)
	} else {
		interestFactorMap := make(map[string]types.InterestFactor)
		// Populate mapping with supply interest factors
		s.keeper.IterateSupplyInterestFactors(sdkCtx, func(denom string, factor sdk.Dec) (stop bool) {
			interestFactor := types.InterestFactor{Denom: denom, SupplyInterestFactor: factor.String()}
			interestFactorMap[denom] = interestFactor
			return false
		})
		// Populate mapping with borrow interest factors
		s.keeper.IterateBorrowInterestFactors(sdkCtx, func(denom string, factor sdk.Dec) (stop bool) {
			interestFactor, ok := interestFactorMap[denom]
			if !ok {
				newInterestFactor := types.InterestFactor{Denom: denom, BorrowInterestFactor: factor.String()}
				interestFactorMap[denom] = newInterestFactor
			} else {
				interestFactor.BorrowInterestFactor = factor.String()
				interestFactorMap[denom] = interestFactor
			}
			return false
		})
		// Translate mapping to slice
		for _, val := range interestFactorMap {
			interestFactors = append(interestFactors, val)
		}
	}

	return &types.QueryInterestFactorsResponse{
		InterestFactors: interestFactors,
	}, nil
}
