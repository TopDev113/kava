package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DONTCOVER

// Errors used by the issuance module
var (
	ErrAssetNotFound           = sdkerrors.Register(ModuleName, 2, "no asset with input denom found")
	ErrNotAuthorized           = sdkerrors.Register(ModuleName, 3, "account not authorized")
	ErrAssetPaused             = sdkerrors.Register(ModuleName, 4, "asset is paused")
	ErrAccountBlocked          = sdkerrors.Register(ModuleName, 5, "account is blocked")
	ErrAccountAlreadyBlocked   = sdkerrors.Register(ModuleName, 6, "account is already blocked")
	ErrAccountAlreadyUnblocked = sdkerrors.Register(ModuleName, 7, "account is already unblocked")
	ErrIssueToModuleAccount    = sdkerrors.Register(ModuleName, 8, "cannot issue tokens to module account")
	ErrExceedsSupplyLimit      = sdkerrors.Register(ModuleName, 9, "asset supply over limit")
	ErrAssetUnblockable        = sdkerrors.Register(ModuleName, 10, "asset does not support block/unblock functionality")
	ErrAccountNotFound         = sdkerrors.Register(ModuleName, 11, "cannot block account that does not exist in state")
)
