# Validator Guide

This is an updated guide on setting up a mainnet validator. Note that this is a minimal guide and does not cover more advanced topics like [sentry node architecture](https://github.com/stakefish/cosmos-validator-design) and [double signing protection](https://github.com/tendermint/tmkms). It is strongly recommended that any parties considering validating do additional research.  If you have questions, please join the active conversation in the #validators thread of our [__Discord Channel__](https://discord.com/invite/kQzh3Uv).
## Installing Kava

### Prerequisites
You should select an all-purpose server with at least 8GB of RAM, good connectivity, and a solid state drive with sufficient disk space. Storage requirements are discussed further in the section below. In addition, you’ll need to open **port 26656** to connect to the Kava peer-to-peer network. As the usage of the blockchain grows, the server requirements may increase as well, so you should have a plan for updating your server as well.

### Storage
The monthly storage requirements for a node are as follows. These are estimated values based on experience, but should serve as a good guide.

- An archival node (`pruning = "nothing"`) grows at a rate of ~100 GB per month
- A fully pruning node (`pruning = "everything"`) grows at a rate of ~5 GB per month
- A default pruning node (`pruning = “default”`) grows at a rate of ~25 GB per month

## Install Go
Kava is built using Go and requires Go version 1.17+. In this example, you will be installing Go on a fresh install of ubuntu 20.04.

```bash
# Update ubuntu
sudo apt update
sudo apt upgrade -y

# Install packages necessary to run go and jq for pretty formatting command line outputs
sudo apt install build-essential jq -y

# Install git
sudo apt install git

# Install go
wget https://dl.google.com/go/go1.17.6.linux-amd64.tar.gz (or latest version at https://golang.org/dl/)
sudo tar -xvf go1.17.6.linux-amd64.tar.gz
sudo mv go /usr/local

# Updates environmental variables to include go
cat <<EOF>> ~/.profile
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export GO111MODULE=on
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
EOF
source ~/.profile
```
To verify that Go is installed:
```bash
go version
# Should return go version go1.17.6 linux/amd64
```

## Install Kava
Install Kava using `git clone`. Note that version 0.16.1 is the correct version for mainnet.

```bash
git clone https://github.com/kava-labs/kava
cd kava
git checkout v0.16.1
make install
```
To verify that kava is installed:
```bash
kava version --long
# name: kava
# server_name: kava
# version: 0.16.1
# commit: c2257c409659e2a6c1c8907fe5ae596c5e46f19d

# build_tags: netgo,ledger
# go: go version go1.17.6 linux/amd64
```

## Configuring Your Node
Next, download the correct genesis file and sync your node with the Kava mainnet. To download the genesis file:
```bash
# First, initialize kava. Replace <name> with the public name of your node
kava init --chain-id kava-9 <name>
# Download the genesis file
wget https://kava-genesis-files.s3.amazonaws.com/kava-9/genesis.json -O ~/.kava/config/genesis.json
# Verify genesis hash
jq -S -c -M '' $HOME/.kava/config/genesis.json | shasum -a 256
# 5c688df5ae6cba9c9e5a9bab045eb367dd54ce9b7f5fab78cf3e636cf2e2b793
```
Next,  adjust some configurations. To open the config file:
```bash
vim $HOME/.kava/config/config.toml
```
At line 212, add [seeds](https://docs.google.com/spreadsheets/d/1j-e1HTkskNRS6mlhxLLQa0loB-ShcVUjz8WVVZxvPaY). These are used to connect to the peer-to-peer network:

At line 215, add some [persistent peers](https://docs.google.com/spreadsheets/d/1j-e1HTkskNRS6mlhxLLQa0loB-ShcVUjz8WVVZxvPaY), which help maintain a connection to the peer-to-peer network


Next, chose how much historical state you want to store. To open the application config file:
```bash
vim $HOME/.kava/config/app.toml
```
In this file, choose between `default`, `nothing`, and `everything`. To reduce hard drive storage, choose `everything` or `default`. To run an archival node, chose `nothing`.
```bash
pruning = "default"
```
In the same file, you will want to set minimum gas prices — setting a minimum prevents spam transactions:
```bash
minimum-gas-prices = "0.001ukava"
```
### Syncing Your Node
To sync your node, you will use systemd, which manages the Kava daemon and automatically restarts it in case of failure. To use systemd, you will create a service file. Be sure to replace `<your_user>` with the user on your server:
```bash
sudo tee /etc/systemd/system/kavad.service > /dev/null <<'EOF'
[Unit]
Description=Kava daemon
After=network-online.target

[Service]
User=<your_user>
ExecStart=/home/<your_user>/go/bin/kava start
Restart=on-failure
RestartSec=3
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
EOF
```
To start syncing:
```bash
# Start the node
sudo systemctl enable kavad
sudo systemctl start kavad
```
To check on the status of syncing:
```bash
kava status --log_format json | jq '.sync_info'
```
This will give output like:
```bash
{
  "latest_block_hash": "03BC172D991C3D66BAAA12A41CAA1A4718C63605B71FD9E0EF09EC38C2DA9F85",
  "latest_app_hash": "FBC020AC8258C4854D2D3384E78929384B0BFBA462F948D493E7883AD315B38C",
  "latest_block_height": "36",
  "latest_block_time": "2022-01-19T16:22:18.54790946Z",
  "earliest_block_hash": "C124499F70881EC71F41231DAB111F448558C5A7794D044F0613E1509FA4C8DF",
  "earliest_app_hash": "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
  "earliest_block_height": "1",
  "earliest_block_time": "2022-01-19T16:00:00Z",
  "catching_up": true
}
```
The main thing to watch is that the block height is increasing. Once you are caught up with the chain, `catching_up` will become false. At that point, you can start using your node to create a validator. If you need to sync using a snapshot, please use https://kava.quicksync.io/

To check the logs of the node:
```bash
sudo journalctl -u kavad -f
```

## Creating a Validator
First, create a wallet, which will give you a private key / public key pair for your node.
```bash
# Replace <your-key-name> with a name for your key that you will remember
kava keys add <your-key-name>
# To see a list of wallets on your node
kava keys list
```
**Be sure to write down the mnemonic for your wallet and store it securely. Losing your mnemonic could result in the irrecoverable loss of KAVA tokens.**

To see the options when creating a validator:
```bash
kava tx staking create-validator -h
```
An example of creating a validator with 50KAVA self-delegation and 10% commission:
```bash
# Replace <key_name> with the key you created previously
kava tx staking create-validator \
--amount=50000000ukava \
--pubkey=$(kava tendermint show-validator) \
--moniker="choose moniker" \
--website="optional website for your validator" \
--details="optional details for your validator" \
--commission-rate="0.10" \
--commission-max-rate="0.20" \
--commission-max-change-rate="0.01" \
--min-self-delegation="1" \
--from=<your-key-name> \
--chain-id=kava-9 \
--gas=auto \
--gas-adjustment=1.4
```
To check on the status of your validator:
```bash
kava status --log_format json | jq '.ValidatorInfo'
```
After you have completed this guide, your validator should be up and ready to receive delegations. Note that only the top 100 validators by weighted stake (self-delegations + other delegations) are eligible for block rewards. To view the current validator list, checkout one of the Kava block explorers:
- https://www.mintscan.io/kava
- https://kava.bigdipper.live/
- https://kavascan.com/

If you have questions, please join the active conversation in the #validators thread of the [__Kava Discord Channel__](https://discord.com/invite/kQzh3Uv).
