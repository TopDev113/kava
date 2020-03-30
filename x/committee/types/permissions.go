package types

import (
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/params"
	sdtypes "github.com/kava-labs/kava/x/shutdown/types"
)

func init() {
	// Gov proposals need to be registered on gov's ModuleCdc.
	// But since proposals contain Permissions, those types also need registering.
	gov.ModuleCdc.RegisterInterface((*Permission)(nil), nil)
	gov.RegisterProposalTypeCodec(GodPermission{}, "kava/GodPermission")
	gov.RegisterProposalTypeCodec(ParamChangePermission{}, "kava/ParamChangePermission")
	gov.RegisterProposalTypeCodec(ShutdownPermission{}, "kava/ShutdownPermission")
}

// GodPermission allows any governance proposal. It is used mainly for testing.
type GodPermission struct{}

var _ Permission = GodPermission{}

func (GodPermission) Allows(gov.Content) bool { return true }

func (GodPermission) MarshalYAML() (interface{}, error) {
	valueToMarshal := struct {
		Type string `yaml:"type"`
	}{
		Type: "god_permission",
	}
	return valueToMarshal, nil
}

// ParamChangeProposal only allows changes to certain params
type ParamChangePermission struct {
	AllowedParams AllowedParams `json:"allowed_params" yaml:"allowed_params"`
}

var _ Permission = ParamChangePermission{}

func (perm ParamChangePermission) Allows(p gov.Content) bool {
	proposal, ok := p.(params.ParameterChangeProposal)
	if !ok {
		return false
	}
	for _, change := range proposal.Changes {
		if !perm.AllowedParams.Contains(change) {
			return false
		}
	}
	return true
}

func (perm ParamChangePermission) MarshalYAML() (interface{}, error) {
	valueToMarshal := struct {
		Type          string        `yaml:"type"`
		AllowedParams AllowedParams `yaml:"allowed_params`
	}{
		Type:          "param_change_permission",
		AllowedParams: perm.AllowedParams,
	}
	return valueToMarshal, nil
}

type AllowedParam struct {
	Subspace string `json:"subspace" yaml:"subspace"`
	Key      string `json:"key" yaml:"key"`
	Subkey   string `json:"subkey,omitempty" yaml:"subkey,omitempty"`
}
type AllowedParams []AllowedParam

func (allowed AllowedParams) Contains(paramChange params.ParamChange) bool {
	for _, p := range allowed {
		if paramChange.Subspace == p.Subspace && paramChange.Key == p.Key && paramChange.Subkey == p.Subkey {
			return true
		}
	}
	return false
}

// ShutdownPermission allows certain message types to be disabled
type ShutdownPermission struct {
	MsgRoute sdtypes.MsgRoute `json:"msg_route" yaml:"msg_route"`
}

var _ Permission = ShutdownPermission{}

func (perm ShutdownPermission) Allows(p gov.Content) bool {
	proposal, ok := p.(sdtypes.ShutdownProposal)
	if !ok {
		return false
	}
	for _, r := range proposal.MsgRoutes {
		if r == perm.MsgRoute {
			return true
		}
	}
	return false
}

func (perm ShutdownPermission) MarshalYAML() (interface{}, error) {
	valueToMarshal := struct {
		Type     string           `yaml:"type"`
		MsgRoute sdtypes.MsgRoute `yaml:"msg_route"`
	}{
		Type:     "shutdown_permission",
		MsgRoute: perm.MsgRoute,
	}
	return valueToMarshal, nil
}

// TODO add more permissions?
// - limit parameter changes to be within small ranges
// - allow community spend proposals
// - allow committee change proposals
