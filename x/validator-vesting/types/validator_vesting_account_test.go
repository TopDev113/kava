package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto/multisig"
	tmtime "github.com/tendermint/tendermint/types/time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
)

var (
	stakeDenom = "stake"
	feeDenom   = "fee"
)

func TestNewAccount(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour).Unix()
	periods := vestingtypes.Periods{
		vestingtypes.Period{Length: int64(12 * 60 * 60), Amount: cs(c(feeDenom, 500), c(stakeDenom, 50))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
	}

	testAddr := CreateTestAddrs(1)[0]
	testPk := CreateTestPubKeys(1)[0]
	testConsAddr := sdk.ConsAddress(testPk.Address())
	origCoins := cs(c(feeDenom, 1000), c(stakeDenom, 100))
	bacc := auth.NewBaseAccountWithAddress(testAddr)
	bacc.SetCoins(origCoins)
	bva, _ := vesting.NewBaseVestingAccount(&bacc, origCoins, endTime)
	require.NotPanics(t, func() { NewValidatorVestingAccountRaw(bva, now.Unix(), periods, testConsAddr, nil, 90) })
	vva := NewValidatorVestingAccountRaw(bva, now.Unix(), periods, testConsAddr, nil, 90)
	vva.PubKey = testPk
	_, err := vva.MarshalYAML()
	require.NoError(t, err)
}

func TestGetVestedCoinsValidatorVestingAcc(t *testing.T) {
	now := tmtime.Now()
	periods := vestingtypes.Periods{
		vestingtypes.Period{Length: int64(12 * 60 * 60), Amount: cs(c(feeDenom, 500), c(stakeDenom, 50))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
	}

	testAddr := CreateTestAddrs(1)[0]
	testPk := CreateTestPubKeys(1)[0]
	testConsAddr := sdk.ConsAddress(testPk.Address())
	origCoins := cs(c(feeDenom, 1000), c(stakeDenom, 100))
	bacc := auth.NewBaseAccountWithAddress(testAddr)
	bacc.SetCoins(origCoins)
	vva := NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)

	// require no coins vested at the beginning of the vesting schedule
	vestedCoins := vva.GetVestedCoins(now)
	require.Nil(t, vestedCoins)

	// require no coins vested during first vesting period
	vestedCoins = vva.GetVestedCoins(now.Add(6 * time.Hour))
	require.Nil(t, vestedCoins)

	// require 50% of coins vested after successful period 1 vesting
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vestedCoins = vva.GetVestedCoins(now.Add(12 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 500), c(stakeDenom, 50)), vestedCoins)

	// require 50% of coins vested after unsuccessful period 1 vesting
	// NOTE: There is a fairly important semantic distinction here. It seems tempting to say that a failed vesting period should mean that 'GetVestedCoins' should not return those coins. While the point of a validator vesting account is to 'seize' or 'burn' unsuccessfully vested coins, they do in fact vest and become spendable. The intuition is that they have to be spendable in order for the bank keeper to allow us to send/burn them. If they were not vested, then a validator vesting account that failed all of it's vesting periods would never return/burn the coins because it would never have a spendable balance by which to do so. They way we prevent them from being spent in a way other than return/burn is by sending them in the BeginBlock and thus beating any other transfers that would otherwise occur.
	vva.VestingPeriodProgress[0] = VestingProgress{true, false}
	vestedCoins = vva.GetVestedCoins(now.Add(12 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 500), c(stakeDenom, 50)), vestedCoins)

	// require period 2 coins don't vest until period is over
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	// even if the vesting period was somehow successful, should still only return 50% of coins as vested, since the second vesting period hasn't completed.
	vva.VestingPeriodProgress[1] = VestingProgress{true, true}
	vestedCoins = vva.GetVestedCoins(now.Add(15 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 500), c(stakeDenom, 50)), vestedCoins)

	// require 75% of coins vested after successful period 2
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.VestingPeriodProgress[1] = VestingProgress{true, true}
	vestedCoins = vva.GetVestedCoins(now.Add(18 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 750), c(stakeDenom, 75)), vestedCoins)

	// require 75% of coins vested after successful period 1 and unsuccessful period 2.
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.VestingPeriodProgress[1] = VestingProgress{true, false}
	vestedCoins = vva.GetVestedCoins(now.Add(18 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 750), c(stakeDenom, 75)), vestedCoins)

	// require 100% of coins vested after all periods complete successfully
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.VestingPeriodProgress[1] = VestingProgress{true, true}
	vva.VestingPeriodProgress[2] = VestingProgress{true, true}

	vestedCoins = vva.GetVestedCoins(now.Add(48 * time.Hour))
	require.Equal(t, origCoins, vestedCoins)

	// require 100% of coins vested after all periods complete unsuccessfully
	vva.VestingPeriodProgress[0] = VestingProgress{true, false}
	vva.VestingPeriodProgress[1] = VestingProgress{true, false}
	vva.VestingPeriodProgress[2] = VestingProgress{true, false}

	vestedCoins = vva.GetVestedCoins(now.Add(48 * time.Hour))
	require.Equal(t, origCoins, vestedCoins)
}

func TestGetVestingCoinsValidatorVestingAcc(t *testing.T) {
	now := tmtime.Now()
	periods := vestingtypes.Periods{
		vestingtypes.Period{Length: int64(12 * 60 * 60), Amount: cs(c(feeDenom, 500), c(stakeDenom, 50))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
	}

	testAddr := CreateTestAddrs(1)[0]
	testPk := CreateTestPubKeys(1)[0]
	testConsAddr := sdk.ConsAddress(testPk.Address())
	origCoins := cs(c(feeDenom, 1000), c(stakeDenom, 100))
	bacc := auth.NewBaseAccountWithAddress(testAddr)
	bacc.SetCoins(origCoins)
	vva := NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)

	// require all coins vesting at the beginning of the vesting schedule
	vestingCoins := vva.GetVestingCoins(now)
	require.Equal(t, origCoins, vestingCoins)

	// require all coins vesting during first vesting period
	vestingCoins = vva.GetVestingCoins(now.Add(6 * time.Hour))
	require.Equal(t, origCoins, vestingCoins)

	// require 50% of coins vesting after successful period 1 vesting
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vestingCoins = vva.GetVestingCoins(now.Add(12 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 500), c(stakeDenom, 50)), vestingCoins)

	// require 50% of coins vesting after unsuccessful period 1 vesting
	vva.VestingPeriodProgress[0] = VestingProgress{true, false}
	vestingCoins = vva.GetVestingCoins(now.Add(12 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 500), c(stakeDenom, 50)), vestingCoins)

	// require period 2 coins still vesting until period is over
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	// should never happen, but still won't affect vesting balance
	vva.VestingPeriodProgress[1] = VestingProgress{true, true}
	vestingCoins = vva.GetVestingCoins(now.Add(15 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 500), c(stakeDenom, 50)), vestingCoins)

	// require 25% of coins vesting after successful period 2
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.VestingPeriodProgress[1] = VestingProgress{true, true}
	vestingCoins = vva.GetVestingCoins(now.Add(18 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 250), c(stakeDenom, 25)), vestingCoins)

	// require 25% of coins vesting after successful period 1 and unsuccessful period 2
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.VestingPeriodProgress[1] = VestingProgress{true, false}
	vestingCoins = vva.GetVestingCoins(now.Add(18 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 250), c(stakeDenom, 25)), vestingCoins)

	// require no coins vesting after all periods complete successfully
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.VestingPeriodProgress[1] = VestingProgress{true, true}
	vva.VestingPeriodProgress[2] = VestingProgress{true, true}

	vestingCoins = vva.GetVestingCoins(now.Add(48 * time.Hour))
	require.Nil(t, vestingCoins)

	// require no coins vesting after all periods complete unsuccessfully
	vva.VestingPeriodProgress[0] = VestingProgress{true, false}
	vva.VestingPeriodProgress[1] = VestingProgress{true, false}
	vva.VestingPeriodProgress[2] = VestingProgress{true, false}

	vestingCoins = vva.GetVestingCoins(now.Add(48 * time.Hour))
	require.Nil(t, vestingCoins)
}

func TestSpendableCoinsValidatorVestingAccount(t *testing.T) {
	now := tmtime.Now()
	periods := vestingtypes.Periods{
		vestingtypes.Period{Length: int64(12 * 60 * 60), Amount: cs(c(feeDenom, 500), c(stakeDenom, 50))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
	}

	testAddr := CreateTestAddrs(1)[0]
	testPk := CreateTestPubKeys(1)[0]
	testConsAddr := sdk.ConsAddress(testPk.Address())
	origCoins := cs(c(feeDenom, 1000), c(stakeDenom, 100))
	bacc := auth.NewBaseAccountWithAddress(testAddr)
	bacc.SetCoins(origCoins)
	vva := NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)

	// require that there exist no spendable coins at the beginning of the vesting schedule
	spendableCoins := vva.SpendableCoins(now)
	require.Nil(t, spendableCoins)

	// require that all vested coins (50%) are spendable when period 1 completes successfully
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	spendableCoins = vva.SpendableCoins(now.Add(12 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 500), c(stakeDenom, 50)), spendableCoins)

	// require that 50% of coins are spendable after period 1 completes unsuccessfully. See note above. The reason the coins are still 'spendable' is that we need to be able to transfer the coins to the return address/burn them. Making them not spendable means that it would be impossible to recover the debt for a validator vesting account for which all periods failed.
	vva.VestingPeriodProgress[0] = VestingProgress{true, false}
	spendableCoins = vva.SpendableCoins(now.Add(12 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 500), c(stakeDenom, 50)), spendableCoins)

	// receive some coins
	recvAmt := cs(c(stakeDenom, 50))
	vva.SetCoins(vva.GetCoins().Add(recvAmt...))

	// require that all vested coins (50%) are spendable plus any received after period 1 completes successfully
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	spendableCoins = vva.SpendableCoins(now.Add(12 * time.Hour))
	require.Equal(t, cs(c(feeDenom, 500), c(stakeDenom, 100)), spendableCoins)

	// spend all spendable coins
	vva.SetCoins(vva.GetCoins().Sub(spendableCoins))

	// require that no more coins are spendable
	spendableCoins = vva.SpendableCoins(now.Add(12 * time.Hour))
	require.Nil(t, spendableCoins)
}

func TestGetFailedVestedCoins(t *testing.T) {
	now := tmtime.Now()
	periods := vestingtypes.Periods{
		vestingtypes.Period{Length: int64(12 * 60 * 60), Amount: cs(c(feeDenom, 500), c(stakeDenom, 50))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
	}

	testAddr := CreateTestAddrs(1)[0]
	testPk := CreateTestPubKeys(1)[0]
	testConsAddr := sdk.ConsAddress(testPk.Address())
	origCoins := cs(c(feeDenom, 1000), c(stakeDenom, 100))
	bacc := auth.NewBaseAccountWithAddress(testAddr)
	bacc.SetCoins(origCoins)
	vva := NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)

	vva.VestingPeriodProgress[0] = VestingProgress{true, false}
	// require that period 1 coins are failed if the period completed unsuccessfully.
	require.Equal(t,
		cs(c(feeDenom, 500), c(stakeDenom, 50)),
		vva.GetFailedVestedCoins(),
	)

	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	require.Equal(t,
		sdk.Coins(nil),
		vva.GetFailedVestedCoins(),
	)

}
func TestTrackDelegationValidatorVestingAcc(t *testing.T) {
	now := tmtime.Now()
	periods := vestingtypes.Periods{
		vestingtypes.Period{Length: int64(12 * 60 * 60), Amount: cs(c(feeDenom, 500), c(stakeDenom, 50))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
	}

	testAddr := CreateTestAddrs(1)[0]
	testPk := CreateTestPubKeys(1)[0]
	testConsAddr := sdk.ConsAddress(testPk.Address())
	origCoins := cs(c(feeDenom, 1000), c(stakeDenom, 100))
	bacc := auth.NewBaseAccountWithAddress(testAddr)
	bacc.SetCoins(origCoins)
	vva := NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)

	vva.TrackDelegation(now, origCoins)
	require.Equal(t, origCoins, vva.DelegatedVesting)
	require.Nil(t, vva.DelegatedFree)

	// all periods pass successfully
	bacc.SetCoins(origCoins)
	vva = NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.VestingPeriodProgress[1] = VestingProgress{true, true}
	vva.VestingPeriodProgress[2] = VestingProgress{true, true}
	vva.TrackDelegation(now.Add(48*time.Hour), origCoins)
	// require all delegated coins are free
	require.Equal(t, origCoins, vva.DelegatedFree)
	require.Nil(t, vva.DelegatedVesting)

	// require the ability to delegate all vesting coins (50%) and all vested coins (50%)
	bacc.SetCoins(origCoins)
	vva = NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)
	vva.TrackDelegation(now.Add(12*time.Hour), cs(c(stakeDenom, 50)))
	require.Equal(t, cs(c(stakeDenom, 50)), vva.DelegatedVesting)
	require.Nil(t, vva.DelegatedFree)

	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.TrackDelegation(now.Add(12*time.Hour), cs(c(stakeDenom, 50)))
	require.Equal(t, cs(c(stakeDenom, 50)), vva.DelegatedVesting)
	require.Equal(t, cs(c(stakeDenom, 50)), vva.DelegatedFree)

	// require no modifications when delegation amount is zero or not enough funds
	bacc.SetCoins(origCoins)
	vva = NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)
	require.Panics(t, func() {
		vva.TrackDelegation(now.Add(24*time.Hour), cs(c(stakeDenom, 1000000)))
	})
	require.Nil(t, vva.DelegatedVesting)
	require.Nil(t, vva.DelegatedFree)
}

func TestTrackUndelegationPeriodicVestingAcc(t *testing.T) {
	now := tmtime.Now()
	periods := vestingtypes.Periods{
		vestingtypes.Period{Length: int64(12 * 60 * 60), Amount: cs(c(feeDenom, 500), c(stakeDenom, 50))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
	}

	testAddr := CreateTestAddrs(1)[0]
	testPk := CreateTestPubKeys(1)[0]
	testConsAddr := sdk.ConsAddress(testPk.Address())
	origCoins := cs(c(feeDenom, 1000), c(stakeDenom, 100))
	bacc := auth.NewBaseAccountWithAddress(testAddr)
	bacc.SetCoins(origCoins)
	vva := NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)

	// require ability to delegate then undelegate all coins.
	vva.TrackDelegation(now, origCoins)
	vva.TrackUndelegation(origCoins)
	require.Nil(t, vva.DelegatedFree)
	require.Nil(t, vva.DelegatedVesting)

	// require the ability to delegate all coins after they have successfully vested
	bacc.SetCoins(origCoins)
	vva = NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.VestingPeriodProgress[1] = VestingProgress{true, true}
	vva.VestingPeriodProgress[2] = VestingProgress{true, true}
	vva.TrackDelegation(now.Add(24*time.Hour), origCoins)
	vva.TrackUndelegation(origCoins)
	require.Nil(t, vva.DelegatedFree)
	require.Nil(t, vva.DelegatedVesting)

	// require panic and no modifications when attempting to undelegate zero coins
	bacc.SetCoins(origCoins)
	vva = NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)
	require.Panics(t, func() {
		vva.TrackUndelegation(sdk.Coins{c(stakeDenom, 0)})
	})
	require.Nil(t, vva.DelegatedFree)
	require.Nil(t, vva.DelegatedVesting)

	// successfully vest period 1 and delegate to two validators
	vva = NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 90)
	vva.VestingPeriodProgress[0] = VestingProgress{true, true}
	vva.TrackDelegation(now.Add(12*time.Hour), cs(c(stakeDenom, 50)))
	vva.TrackDelegation(now.Add(12*time.Hour), cs(c(stakeDenom, 50)))

	// undelegate from one validator that got slashed 50%
	vva.TrackUndelegation(cs(c(stakeDenom, 25)))
	require.Equal(t, cs(c(stakeDenom, 25)), vva.DelegatedFree)
	require.Equal(t, cs(c(stakeDenom, 50)), vva.DelegatedVesting)

	// undelegate from the other validator that did not get slashed
	vva.TrackUndelegation(cs(c(stakeDenom, 50)))
	require.Nil(t, vva.DelegatedFree)
	require.Equal(t, cs(c(stakeDenom, 25)), vva.DelegatedVesting)
}

func TestGenesisAccountValidate(t *testing.T) {
	now := tmtime.Now()
	periods := vestingtypes.Periods{
		vestingtypes.Period{Length: int64(12 * 60 * 60), Amount: cs(c(feeDenom, 500), c(stakeDenom, 50))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
	}

	testAddr := CreateTestAddrs(1)[0]
	testPk := CreateTestPubKeys(1)[0]
	testConsAddr := sdk.ConsAddress(testPk.Address())
	origCoins := cs(c(feeDenom, 1000), c(stakeDenom, 100))
	bacc := auth.NewBaseAccountWithAddress(testAddr)
	bacc.SetCoins(origCoins)

	invAcc := NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 100)
	invAcc.DebtAfterFailedVesting = sdk.Coins{sdk.Coin{Denom: "KAVA", Amount: sdk.OneInt()}}

	tests := []struct {
		name    string
		acc     authexported.GenesisAccount
		expPass bool
	}{
		{
			"valid validator vesting account",
			NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 100),
			true,
		},
		{
			"empty validator address",
			NewValidatorVestingAccount(&bacc, now.Unix(), periods, nil, nil, 100),
			false,
		},
		{
			"invalid signing threshold",
			NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, -1),
			false,
		},
		{
			"invalid signing threshold",
			NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, nil, 120),
			false,
		},
		{
			"invalid return address",
			NewValidatorVestingAccount(&bacc, now.Unix(), periods, testConsAddr, testAddr, 90),
			false,
		},
		{
			"invalid debt after failed vesting ",
			invAcc,
			false,
		},
	}
	for _, tt := range tests {
		err := tt.acc.Validate()
		if tt.expPass {
			require.NoError(t, err, tt.name)
		} else {
			require.Error(t, err, tt.name)
		}
	}
}

func TestMarshalJSON(t *testing.T) {
	testTime := tmtime.Now()
	periods := vestingtypes.Periods{
		vestingtypes.Period{Length: int64(12 * 60 * 60), Amount: cs(c(feeDenom, 500), c(stakeDenom, 50))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
		vestingtypes.Period{Length: int64(6 * 60 * 60), Amount: cs(c(feeDenom, 250), c(stakeDenom, 25))},
	}
	testAddrs := CreateTestAddrs(2)
	testPk := CreateTestPubKeys(1)[0]
	testConsAddr := sdk.ConsAddress(testPk.Address())
	testMultisigPk := multisig.NewPubKeyMultisigThreshold(6, CreateTestPubKeys(10))

	testCases := []struct {
		name    string
		account *ValidatorVestingAccount
	}{
		{
			name: "normal",
			account: NewValidatorVestingAccount(
				auth.NewBaseAccount(
					testAddrs[0],
					cs(c(stakeDenom, 100)),
					testPk,
					0,
					1,
				),
				testTime.Unix(),
				periods,
				testConsAddr,
				testAddrs[1],
				90,
			),
		},
		{
			name: "multisig",
			account: NewValidatorVestingAccount(
				auth.NewBaseAccount(
					testAddrs[0],
					cs(c(stakeDenom, 1)),
					testMultisigPk,
					412,
					67,
				),
				testTime.Unix(),
				periods,
				testConsAddr,
				testAddrs[1],
				67,
			),
		},
	}

	for _, tc := range testCases {
		bz, err := ModuleCdc.MarshalJSON(tc.account)
		require.NoError(t, err)

		var reconstructedAccount *ValidatorVestingAccount
		err = ModuleCdc.UnmarshalJSON(bz, &reconstructedAccount)
		require.NoError(t, err)

		// first check debt coins using coin equality function, then overwrite to avoid problems with testify's Equality check
		require.Truef(t,
			tc.account.DebtAfterFailedVesting.IsEqual(reconstructedAccount.DebtAfterFailedVesting),
			"DebtAfterFailedVesting not equal, expected: %+v, actual: %+v",
			tc.account.DebtAfterFailedVesting,
			reconstructedAccount.DebtAfterFailedVesting,
		)
		reconstructedAccount.DebtAfterFailedVesting = tc.account.DebtAfterFailedVesting

		require.Equal(t, tc.account, reconstructedAccount)
	}
}
