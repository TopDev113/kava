# Withdraw

Withdraw coins from a swap liquidity pool

## Command
```
kava tx swap withdraw <shares> <minCoinA> <minCoinB> <deadline> <flags>
```

Using ```kava``` call the ```tx``` subcommand followed by the module name which is```swap```, then define the action which is ```withdraw``` and finally follow up with required arguments or flags.

### Arguments
position|name|expects
|--|--|--|
1|shares| number of shares
2|minCoinA| amount & name (no spaces)
3|minCoinB| amount & name (no spaces)
4|deadline| timestamp 




### Example
```
kava tx swap withdraw 153000 10000000ukava 20000000usdx 176293740 --from <key>
```
 
### Options
```

  -a, --account-number uint      The account number of the signing account (offline mode only)
  -b, --broadcast-mode string    Transaction broadcasting mode (sync|async|block) (default "sync")
      --dry-run                  ignore the --gas flag and perform a simulation of a transaction, but don't broadcast it
      --fees string              Fees to pay along with transaction; eg: 10uatom
      --from string              Name or address of private key with which to sign
      --gas string               gas limit to set per-transaction; set to "auto" to calculate required gas automatically (default 200000) (default "200000")
      --gas-adjustment float     adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored  (default 1)
      --gas-prices string        Gas prices to determine the transaction fee (e.g. 10uatom)
      --generate-only            Build an unsigned transaction and write it to STDOUT (when enabled, the local Keybase is not accessible and the node operates offline)
  -h, --help                     help for withdraw
      --indent                   Add indent to JSON response
      --keyring-backend string   Select keyring's backend (os|file|test) (default "os")
      --ledger                   Use a connected Ledger device
      --memo string              Memo to send along with transaction
      --node string              <host>:<port> to tendermint rpc interface for this chain (default "tcp://localhost:26657")
  -s, --sequence uint            The sequence number of the signing account (offline mode only)
      --trust-node               Trust connected full node (don't verify proofs for responses) (default true)
  -y, --yes                      Skip tx broadcasting prompt confirmation

```

### Options inherited from parent commands
```

      --chain-id string   Chain ID of tendermint node

```