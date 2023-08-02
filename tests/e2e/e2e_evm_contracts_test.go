package e2e_test

import (
	"context"
	"math/big"
	"time"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/kava-labs/kava/app"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	evmutiltypes "github.com/kava-labs/kava/x/evmutil/types"

	"github.com/kava-labs/kava/tests/e2e/contracts/greeter"
	"github.com/kava-labs/kava/tests/util"
)

func (suite *IntegrationTestSuite) TestEthCallToGreeterContract() {
	// this test manipulates state of the Greeter contract which means other tests shouldn't use it.

	// setup funded account to interact with contract
	user := suite.Kava.NewFundedAccount("greeter-contract-user", sdk.NewCoins(ukava(1e6)))

	greeterAddr := suite.Kava.ContractAddrs["greeter"]
	contract, err := greeter.NewGreeter(greeterAddr, suite.Kava.EvmClient)
	suite.NoError(err)

	beforeGreeting, err := contract.Greet(nil)
	suite.NoError(err)

	updatedGreeting := "look at me, using the evm"
	tx, err := contract.SetGreeting(user.EvmAuth, updatedGreeting)
	suite.NoError(err)

	_, err = util.WaitForEvmTxReceipt(suite.Kava.EvmClient, tx.Hash(), 10*time.Second)
	suite.NoError(err)

	afterGreeting, err := contract.Greet(nil)
	suite.NoError(err)

	suite.Equal("what's up!", beforeGreeting)
	suite.Equal(updatedGreeting, afterGreeting)
}

func (suite *IntegrationTestSuite) TestEthCallToErc20() {
	randoReceiver := util.SdkToEvmAddress(app.RandomAddress())
	amount := big.NewInt(1)

	// make unauthenticated eth_call query to check balance
	beforeBalance := suite.Kava.GetErc20Balance(suite.DeployedErc20.Address, randoReceiver)

	// make authenticate eth_call to transfer tokens
	res := suite.FundKavaErc20Balance(randoReceiver, amount)
	suite.NoError(res.Err)

	// make another unauthenticated eth_call query to check new balance
	afterBalance := suite.Kava.GetErc20Balance(suite.DeployedErc20.Address, randoReceiver)

	suite.BigIntsEqual(big.NewInt(0), beforeBalance, "expected before balance to be zero")
	suite.BigIntsEqual(amount, afterBalance, "unexpected post-transfer balance")
}

func (suite *IntegrationTestSuite) TestEip712BasicMessageAuthorization() {
	// create new funded account
	sender := suite.Kava.NewFundedAccount("eip712-msgSend", sdk.NewCoins(ukava(2e4)))
	receiver := app.RandomAddress()

	// setup message for sending some kava to random receiver
	msgs := []sdk.Msg{
		banktypes.NewMsgSend(sender.SdkAddress, receiver, sdk.NewCoins(ukava(1e3))),
	}

	// create tx
	tx := suite.NewEip712TxBuilder(
		sender,
		suite.Kava,
		1e6,
		sdk.NewCoins(ukava(1e4)),
		msgs,
		"this is a memo",
	).GetTx()

	txBytes, err := suite.Kava.EncodingConfig.TxConfig.TxEncoder()(tx)
	suite.NoError(err)

	// broadcast tx
	res, err := suite.Kava.Tx.BroadcastTx(context.Background(), &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	})
	suite.NoError(err)
	suite.Equal(sdkerrors.SuccessABCICode, res.TxResponse.Code)

	_, err = util.WaitForSdkTxCommit(suite.Kava.Tx, res.TxResponse.TxHash, 6*time.Second)
	suite.NoError(err)

	// check that the message was processed & the kava is transferred.
	balRes, err := suite.Kava.Bank.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: receiver.String(),
		Denom:   "ukava",
	})
	suite.NoError(err)
	suite.Equal(sdk.NewInt(1e3), balRes.Balance.Amount)
}

// Note that this test works because the deployed erc20 is configured in evmutil & cdp params.
// This test matches the webapp's "USDT Earn" workflow
func (suite *IntegrationTestSuite) TestEip712ConvertToCoinAndDepositToLend() {
	// cdp requires minimum of $11 collateral
	amount := sdk.NewInt(11e6) // 11 USDT
	principal := sdk.NewCoin("usdx", sdk.NewInt(10e6))
	sdkDenom := suite.DeployedErc20.CosmosDenom

	// create new funded account
	depositor := suite.Kava.NewFundedAccount("eip712-lend-depositor", sdk.NewCoins(ukava(1e5)))
	// give them erc20 balance to deposit
	fundRes := suite.FundKavaErc20Balance(depositor.EvmAddress, amount.BigInt())
	suite.NoError(fundRes.Err)

	// setup messages for convert to coin & deposit into earn
	convertMsg := evmutiltypes.NewMsgConvertERC20ToCoin(
		evmutiltypes.NewInternalEVMAddress(depositor.EvmAddress),
		depositor.SdkAddress,
		evmutiltypes.NewInternalEVMAddress(suite.DeployedErc20.Address),
		amount,
	)
	depositMsg := cdptypes.NewMsgCreateCDP(
		depositor.SdkAddress,
		sdk.NewCoin(sdkDenom, amount),
		principal,
		suite.DeployedErc20.CdpCollateralType,
	)
	msgs := []sdk.Msg{
		// convert to coin
		&convertMsg,
		// deposit into cdp (Mint), take out USDX
		&depositMsg,
	}

	// create tx
	tx := suite.NewEip712TxBuilder(
		depositor,
		suite.Kava,
		1e6,
		sdk.NewCoins(ukava(1e4)),
		msgs,
		"doing the USDT Earn workflow! erc20 -> sdk.Coin -> USDX hard deposit",
	).GetTx()

	txBytes, err := suite.Kava.EncodingConfig.TxConfig.TxEncoder()(tx)
	suite.NoError(err)

	// broadcast tx
	res, err := suite.Kava.Tx.BroadcastTx(context.Background(), &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	})
	suite.NoError(err)
	suite.Equal(sdkerrors.SuccessABCICode, res.TxResponse.Code)

	_, err = util.WaitForSdkTxCommit(suite.Kava.Tx, res.TxResponse.TxHash, 6*time.Second)
	suite.Require().NoError(err)

	// check that depositor no longer has erc20 balance
	balance := suite.Kava.GetErc20Balance(suite.DeployedErc20.Address, depositor.EvmAddress)
	suite.BigIntsEqual(big.NewInt(0), balance, "expected no erc20 balance")

	// check that account has cdp
	cdpRes, err := suite.Kava.Cdp.Cdp(context.Background(), &cdptypes.QueryCdpRequest{
		CollateralType: suite.DeployedErc20.CdpCollateralType,
		Owner:          depositor.SdkAddress.String(),
	})
	suite.NoError(err)
	suite.True(cdpRes.Cdp.Collateral.Amount.Equal(amount))
	suite.True(cdpRes.Cdp.Principal.Equal(principal))

	// withdraw deposit & convert back to erc20 (this allows refund to recover erc20s used in test)
	withdraw := cdptypes.NewMsgRepayDebt(
		depositor.SdkAddress,
		suite.DeployedErc20.CdpCollateralType,
		principal,
	)
	convertBack := evmutiltypes.NewMsgConvertCoinToERC20(
		depositor.SdkAddress.String(),
		depositor.EvmAddress.Hex(),
		sdk.NewCoin(sdkDenom, amount),
	)
	withdrawAndConvertBack := util.KavaMsgRequest{
		Msgs:      []sdk.Msg{&withdraw, &convertBack},
		GasLimit:  1e6,
		FeeAmount: sdk.NewCoins(ukava(1000)),
		Data:      "withdrawing from mint & converting back to erc20",
	}
	lastRes := depositor.SignAndBroadcastKavaTx(withdrawAndConvertBack)
	suite.NoError(lastRes.Err)

	balance = suite.Kava.GetErc20Balance(suite.DeployedErc20.Address, depositor.EvmAddress)
	suite.BigIntsEqual(amount.BigInt(), balance, "expected returned erc20 balance")
}
