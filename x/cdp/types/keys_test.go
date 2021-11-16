package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/tendermint/crypto/ed25519"
)

var addr = sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

func TestCdpKey(t *testing.T) {
	key := CdpKey("kava-a", 2)
	collateralType, id := SplitCdpKey(key)
	require.Equal(t, int(id), 2)
	require.Equal(t, "kava-a", collateralType)
}

func TestDenomIterKey(t *testing.T) {
	denomKey := DenomIterKey("kava-a")
	collateralType := SplitDenomIterKey(denomKey)
	require.Equal(t, "kava-a", collateralType)
}

func TestDepositKey(t *testing.T) {
	depositKey := DepositKey(2, addr)
	id, a := SplitDepositKey(depositKey)
	require.Equal(t, 2, int(id))
	require.Equal(t, a, addr)
}

func TestDepositIterKey(t *testing.T) {
	depositIterKey := DepositIterKey(2)
	id := SplitDepositIterKey(depositIterKey)
	require.Equal(t, 2, int(id))
}

func TestDepositIterKey_Invalid(t *testing.T) {
	require.Panics(t, func() { SplitDepositIterKey([]byte{0x03}) })
}

func TestCollateralRatioKey(t *testing.T) {
	collateralKey := CollateralRatioKey("kava-a", 2, sdk.MustNewDecFromStr("1.50"))
	collateralType, id, ratio := SplitCollateralRatioKey(collateralKey)
	require.Equal(t, "kava-a", collateralType)
	require.Equal(t, 2, int(id))
	require.Equal(t, ratio, sdk.MustNewDecFromStr("1.50"))
}

func TestCollateralRatioKey_BigRatio(t *testing.T) {
	bigRatio := sdk.OneDec().Quo(sdk.SmallestDec()).Mul(sdk.OneDec().Add(sdk.OneDec()))
	collateralKey := CollateralRatioKey("kava-a", 2, bigRatio)
	collateralType, id, ratio := SplitCollateralRatioKey(collateralKey)
	require.Equal(t, "kava-a", collateralType)
	require.Equal(t, 2, int(id))
	require.Equal(t, ratio, MaxSortableDec)
}

func TestCollateralRatioKey_Invalid(t *testing.T) {
	require.Panics(t, func() { SplitCollateralRatioKey(badRatioKey()) })
}

func TestCollateralRatioIterKey(t *testing.T) {
	collateralIterKey := CollateralRatioIterKey("kava-a", sdk.MustNewDecFromStr("1.50"))
	collateralType, ratio := SplitCollateralRatioIterKey(collateralIterKey)
	require.Equal(t, "kava-a", collateralType)
	require.Equal(t, ratio, sdk.MustNewDecFromStr("1.50"))
}

func TestCollateralRatioIterKey_Invalid(t *testing.T) {
	require.Panics(t, func() { SplitCollateralRatioIterKey(badRatioIterKey()) })
}

func badRatioKey() []byte {
	r := append(append(append(append([]byte{0x01}, sep...), []byte("nonsense")...), sep...), []byte{0xff}...)
	return r
}

func badRatioIterKey() []byte {
	r := append(append([]byte{0x01}, sep...), []byte("nonsense")...)
	return r
}
