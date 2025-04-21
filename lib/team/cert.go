// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package team

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

func MakeTeamCert(
	team proto.FQTeam,
	ptk1 core.SharedPrivateSuiter,
	ptkCurr core.SharedPrivateSuiter,
	name proto.NameUtf8,
) (
	*rem.TeamCert,
	error,
) {
	ptkPub, hepk, err := ptkCurr.ExportToSharedKey()
	if err != nil {
		return nil, err
	}

	c := rem.TeamCertV1Payload{
		Team: team,
		Ptk:  *ptkPub,
		Tm:   proto.Now(),
		Hepk: *hepk,
		Name: name,
	}
	signers := []core.Signer{ptkCurr}
	if ptk1 == nil {
		return nil, core.InternalError("nil 1-gen PTK passed to MakeTeamCert")
	}

	if !ptkCurr.Metadata().Gen.IsFirst() {
		signers = append(signers, ptk1)
	}
	bl, err := c.EncodeTyped(core.EncoderFactory{})
	if err != nil {
		return nil, err
	}

	v1 := rem.TeamCertV1Signed{
		Payload: *bl,
	}
	err = core.SignStacked(&v1, signers)
	if err != nil {
		return nil, err
	}
	ret := rem.NewTeamCertWithV1(v1)
	return &ret, nil
}

func OpenTeamCert(
	c rem.TeamCert,
) (
	*rem.TeamCertV1Payload,
	error,
) {
	v, err := c.GetV()
	if err != nil {
		return nil, err
	}
	if v != rem.TeamCertVersion_V1 {
		return nil, core.VersionNotSupportedError("team cert != v1")
	}
	v1 := c.V1()

	var verifiers []core.Verifier
	cert, err := v1.Payload.AllocAndDecode(core.DecoderFactory{})
	if err != nil {
		return nil, err
	}
	if !cert.Ptk.Gen.IsFirst() {
		ep, err := core.ImportEntityPublic(cert.Ptk.VerifyKey)
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, ep)
	}
	ep, err := core.ImportEntityPublicSubclass(cert.Team.Team)
	if err != nil {
		return nil, err
	}
	verifiers = append(verifiers, ep)
	err = core.VerifyStackedSignature(&v1, verifiers)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

// Example: YcarI5JTCwt7ZSjjMWYAx5S9tUvx8bQNiDSu6Ta2M2xavlj8lp8a1vRfsu5uj54p7kpi1n93Ld3jlR8YNFv7sv3Rktl8U08OKBw8CDLOoVis
// (101 chars)
func ExportTeamCert(c rem.TeamCert) (string, error) {
	b, err := core.EncodeToBytes(&c)
	if err != nil {
		return "", err
	}
	return core.B62Encode(b), nil
}

func ExportTeamInvite(i proto.TeamInvite) (string, error) {
	b, err := core.EncodeToBytes(&i)
	if err != nil {
		return "", err
	}
	return core.B62Encode(b), nil
}

func ImportTeamCert(s string) (rem.TeamCert, error) {
	b, err := core.B62Decode(s)
	if err != nil {
		return rem.TeamCert{}, err
	}
	var ret rem.TeamCert
	err = core.DecodeFromBytes(&ret, b)
	if err != nil {
		return rem.TeamCert{}, err
	}
	return ret, nil
}

func ImportTeamInvite(s string) (*proto.TeamInvite, error) {
	b, err := core.B62Decode(s)
	if err != nil {
		return nil, err
	}
	var ret proto.TeamInvite
	err = core.DecodeFromBytes(&ret, b)
	if err != nil {
		return nil, err
	}
	return &ret, nil

}
