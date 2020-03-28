package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/cosmos/cosmos-sdk/client/keys"
	crkeys "github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkrest "github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authrest "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client/utils"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/kava-labs/kava/app"
	"github.com/tendermint/go-amino"
)

func init() {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	app.SetBip44CoinType(config)
	config.Seal()
}

func main() {
	sendProposal()
	sendDeposit()
	sendVote()
	sendDelegation()
	sendUndelegation()
	sendCoins()

	sendProposal()
	sendDeposit()
	sendVote()
	sendDelegation()
	sendUndelegation()

	sendCoins()

}

func sendProposal() {
	// get the address
	address := getTestAddress()
	// get the keyname and password
	keyname, password := getKeynameAndPassword()

	proposalContent := gov.ContentFromProposalType("A Test Title", "A test description on this proposal.", gov.ProposalTypeText)
	addr, err := sdk.AccAddressFromBech32(address) // validator address
	if err != nil {
		panic(err)
	}

	// create a message to send to the blockchain
	msg := gov.NewMsgSubmitProposal(
		proposalContent,
		sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)),
		addr,
	)

	// helper methods for transactions
	cdc := app.MakeCodec() // make codec for the app

	// get the keybase
	keybase := getKeybase()

	// SEND THE PROPOSAL
	// cast to the generic msg type
	msgToSend := []sdk.Msg{msg}

	// send the PROPOSAL message to the blockchain
	sendMsgToBlockchain(cdc, address, keyname, password, msgToSend, keybase)

}

func sendDeposit() {
	// get the address
	address := getTestAddress()
	// get the keyname and password
	keyname, password := getKeynameAndPassword()

	addr, err := sdk.AccAddressFromBech32(address) // validator
	if err != nil {
		panic(err)
	}

	// helper methods for transactions
	cdc := app.MakeCodec() // make codec for the app

	// get the keybase
	keybase := getKeybase()

	// NOW SEND THE DEPOSIT

	// create a deposit transaction to send to the proposal
	amount := sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10000000))
	deposit := gov.NewMsgDeposit(addr, 2, amount) // TODO IMPORTANT '2' must match 'x-example' in swagger.yaml
	depositToSend := []sdk.Msg{deposit}

	sendMsgToBlockchain(cdc, address, keyname, password, depositToSend, keybase)

}

func sendVote() {
	// get the address
	address := getTestAddress()
	// get the keyname and password
	keyname, password := getKeynameAndPassword()

	addr, err := sdk.AccAddressFromBech32(address) // validator
	if err != nil {
		panic(err)
	}

	// helper methods for transactions
	cdc := app.MakeCodec() // make codec for the app

	// get the keybase
	keybase := getKeybase()

	// NOW SEND THE VOTE

	// create a vote on a proposal to send to the blockchain
	vote := gov.NewMsgVote(addr, uint64(2), types.OptionYes) // TODO IMPORTANT '2' must match 'x-example' in swagger.yaml

	// send a vote to the blockchain
	voteToSend := []sdk.Msg{vote}
	sendMsgToBlockchain(cdc, address, keyname, password, voteToSend, keybase)

}

// this should send coins from one address to another
func sendCoins() {
	// get the address
	address := getTestAddress()
	// get the keyname and password
	keyname, password := getKeynameAndPassword()

	addrFrom, err := sdk.AccAddressFromBech32(address) // validator
	if err != nil {
		panic(err)
	}

	addrTo, err := sdk.AccAddressFromBech32("kava1ls82zzghsx0exkpr52m8vht5jqs3un0ceysshz") // TODO IMPORTANT this is the faucet address
	if err != nil {
		panic(err)
	}

	// helper methods for transactions
	cdc := app.MakeCodec() // make codec for the app

	// get the keybase
	keybase := getKeybase()

	// create coins
	amount := sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 2000000))

	coins := bank.NewMsgSend(addrFrom, addrTo, amount) // TODO IMPORTANT '2' must match 'x-example' in swagger.yaml
	coinsToSend := []sdk.Msg{coins}

	// NOW SEND THE COINS

	// send the coin message to the blockchain
	sendMsgToBlockchain(cdc, address, keyname, password, coinsToSend, keybase)

}

func getTestAddress() (address string) {
	// the test address - TODO IMPORTANT make sure this lines up with startchain.sh
	address = "kava1ffv7nhd3z6sych2qpqkk03ec6hzkmufy0r2s4c"
	return address
}

func getKeynameAndPassword() (keyname string, password string) {
	keyname = "vlad"      // TODO - IMPORTANT this must match the keys in the startchain.sh script
	password = "password" // TODO - IMPORTANT this must match the keys in the startchain.sh script
	return keyname, password
}

// this should send a delegation
func sendDelegation() {
	// get the address
	address := getTestAddress()
	// get the keyname and password
	keyname, password := getKeynameAndPassword()

	addrFrom, err := sdk.AccAddressFromBech32(address) // validator
	if err != nil {
		panic(err)
	}

	// helper methods for transactions
	cdc := app.MakeCodec() // make codec for the app

	// get the keybase
	keybase := getKeybase()

	// get the validator address for delegation
	valAddr, err := sdk.ValAddressFromBech32("kavavaloper1ffv7nhd3z6sych2qpqkk03ec6hzkmufyz4scd0") // **FAUCET**
	if err != nil {
		panic(err)
	}

	// create delegation amount
	delAmount := sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000000)
	delegation := staking.NewMsgDelegate(addrFrom, valAddr, delAmount)
	delegationToSend := []sdk.Msg{delegation}

	// send the delegation to the blockchain
	sendMsgToBlockchain(cdc, address, keyname, password, delegationToSend, keybase)
}

// this should send a MsgUndelegate
func sendUndelegation() {
	// get the address
	address := getTestAddress()
	// get the keyname and password
	keyname, password := getKeynameAndPassword()

	addrFrom, err := sdk.AccAddressFromBech32(address) // validator
	if err != nil {
		panic(err)
	}

	// helper methods for transactions
	cdc := app.MakeCodec() // make codec for the app

	// get the keybase
	keybase := getKeybase()

	// get the validator address for delegation
	valAddr, err := sdk.ValAddressFromBech32("kavavaloper1ffv7nhd3z6sych2qpqkk03ec6hzkmufyz4scd0") // **FAUCET**
	if err != nil {
		panic(err)
	}

	// create delegation amount
	undelAmount := sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000000)
	undelegation := staking.NewMsgUndelegate(addrFrom, valAddr, undelAmount)
	delegationToSend := []sdk.Msg{undelegation}

	// send the delegation to the blockchain
	sendMsgToBlockchain(cdc, address, keyname, password, delegationToSend, keybase)

}

func getKeybase() crkeys.Keybase {
	// create a keybase
	// IMPORTANT - TAKE THIS FROM COMMAND LINE PARAMETER and does NOT work with tilde i.e. ~/ does NOT work
	keybase, err := keys.NewKeyBaseFromDir(os.Args[1])
	if err != nil {
		panic(err)
	}

	return keybase
}

// sendMsgToBlockchain sends a message to the blockchain via the rest api
func sendMsgToBlockchain(cdc *amino.Codec, address string, keyname string,
	password string, msg []sdk.Msg, keybase crkeys.Keybase) {

	// get the account number and sequence number
	accountNumber, sequenceNumber := getAccountNumberAndSequenceNumber(cdc, address)

	txBldr := auth.NewTxBuilderFromCLI().
		WithTxEncoder(authclient.GetTxEncoder(cdc)).WithChainID("testing").
		WithKeybase(keybase).WithAccountNumber(accountNumber).
		WithSequence(sequenceNumber)

		// build and sign the transaction
	// this is the *Amino* encoded version of the transaction
	// fmt.Printf("%+v", txBldr.Keybase())
	txBytes, err := txBldr.BuildAndSign("vlad", "password", msg)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("txBytes: %s", txBytes)

	// need to convert the Amino encoded version back to an actual go struct
	var tx auth.StdTx
	cdc.UnmarshalBinaryLengthPrefixed(txBytes, &tx) // might be unmarshal binary bare

	// now we re-marshall it again into json
	jsonBytes, err := cdc.MarshalJSON(
		authrest.BroadcastReq{
			Tx:   tx,
			Mode: "block",
		},
	)
	if err != nil {
		panic(err)
	}
	// fmt.Println("post body: ", string(jsonBytes))

	resp, err := http.Post("http://localhost:1317/txs", "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n\nBody:\n\n")
	fmt.Println(string(body))

}

// getAccountNumberAndSequenceNumber gets an account number and sequence number from the blockchain
func getAccountNumberAndSequenceNumber(cdc *amino.Codec, address string) (accountNumber uint64, sequenceNumber uint64) {

	// we need to setup the account number and sequence in order to have a valid transaction
	resp, err := http.Get("http://localhost:1317/auth/accounts/" + address)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var bodyUnmarshalled sdkrest.ResponseWithHeight
	err = cdc.UnmarshalJSON(body, &bodyUnmarshalled)
	if err != nil {
		panic(err)
	}

	var account authexported.Account
	err = cdc.UnmarshalJSON(bodyUnmarshalled.Result, &account)
	if err != nil {
		panic(err)
	}

	return account.GetAccountNumber(), account.GetSequence()

}
