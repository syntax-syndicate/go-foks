// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"crypto/ecdsa"

	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type SharedKeyMetadata struct {
	Gen proto.Generation
}

type SharedPublicSuite struct {
	proto.SharedKey
	HEPK proto.HEPK
}

func ImportSharedPublicSuite(o *proto.SharedKey, k *proto.HEPK) (*SharedPublicSuite, error) {
	return &SharedPublicSuite{SharedKey: *o, HEPK: *k}, nil
}

func (s *SharedPublicSuite) Verify(sig proto.Signature, obj Verifiable) error {
	return VerifyWithEd25519Public(s.VerifyKey.PublicKeyEd25519(), sig, obj)
}

type SharedPrivateSuite25519 struct {
	PrivateSuite25519
	Md     SharedKeyMetadata
	sbox   proto.SecretBoxKey
	appKey proto.SecretSeed32
}

type SharedPrivateSuiter interface {
	PrivateSuiter
	ExportToSharedKey() (*proto.SharedKey, *proto.HEPK, error)
	ExportToBoxCleartext(e proto.FQEntity) proto.SharedKeySeed
	Metadata() SharedKeyMetadata
	SecretBoxKey() proto.SecretBoxKey
	AppKey() proto.SecretSeed32
}

func (s *SharedPrivateSuite25519) Metadata() SharedKeyMetadata {
	return s.Md
}

func (s *SharedPrivateSuite25519) ExportToBoxCleartext(fqe proto.FQEntity) proto.SharedKeySeed {
	return proto.SharedKeySeed{
		Fqe:  fqe,
		Gen:  s.Md.Gen,
		Role: s.Role,
		Seed: s.seed,
	}
}

func ImportSharedPrivateSuite25519(typ proto.EntityType, sks proto.SharedKeySeed) (*SharedPrivateSuite25519, error) {
	return NewSharedPrivateSuite25519(typ, sks.Role, sks.Seed, sks.Gen, sks.Fqe.Host)
}

func NewSharedPrivateSuite25519(typ proto.EntityType, role proto.Role, s proto.SecretSeed32, g proto.Generation, h proto.HostID) (*SharedPrivateSuite25519, error) {
	ps, err := NewPrivateSuite25519(typ, role, s, h)
	if err != nil {
		return nil, err
	}
	sbox, err := DeriveSecretBoxKey(s)
	if err != nil {
		return nil, err
	}
	app, err := DeriveAppKey(s)
	if err != nil {
		return nil, err
	}
	return &SharedPrivateSuite25519{
		PrivateSuite25519: *ps,
		Md:                SharedKeyMetadata{Gen: g},
		sbox:              *sbox,
		appKey:            *app,
	}, nil
}

func (s *SharedPrivateSuite25519) SecretBoxKey() proto.SecretBoxKey {
	return s.sbox
}

func (s *SharedPrivateSuite25519) AppKey() proto.SecretSeed32 {
	return s.appKey
}

func (s *SharedPrivateSuite25519) HEPK() (*proto.HEPK, error) {
	return s.ExportHEPK()
}

func (s *SharedPrivateSuite25519) HEPKFingerprint() (*proto.HEPKFingerprint, error) {
	hepk, err := s.ExportHEPK()
	if err != nil {
		return nil, err
	}
	return HEPK(hepk).Fingerprint()
}

func (s *SharedPrivateSuite25519) ExportToSharedKey() (*proto.SharedKey, *proto.HEPK, error) {

	vk, err := s.RollingEntityID()
	if err != nil {
		return nil, nil, err
	}
	hepk, err := s.ExportHEPK()
	if err != nil {
		return nil, nil, err
	}
	fp, err := HEPK(hepk).Fingerprint()
	if err != nil {
		return nil, nil, err
	}
	ret := proto.SharedKey{
		Gen:       s.Md.Gen,
		Role:      s.Role,
		VerifyKey: vk,
		HepkFp:    *fp,
	}
	return &ret, hepk, nil
}

var _ SharedPrivateSuiter = (*SharedPrivateSuite25519)(nil)

func FindLatestSharedKeyForRole(keys []proto.SharedKey, role proto.Role) (*proto.SharedKey, error) {
	var ret *proto.SharedKey
	for _, k := range keys {
		eq, err := k.Role.Eq(role)
		if err != nil {
			return nil, err
		}
		if eq && (ret == nil || k.Gen > ret.Gen) {
			ret = &k
		}
	}
	return ret, nil
}

func ImportSharedPublicSuiteFromDB(
	vkRaw []byte,
	hpekFpRaw []byte,
	hpekRaw []byte,
	role proto.Role,
	gen int,
) (
	*SharedPublicSuite,
	error,
) {
	vk, err := proto.ImportEntityIDFromBytes(vkRaw)
	if err != nil {
		return nil, err
	}
	var hpek proto.HEPK
	err = DecodeFromBytes(&hpek, hpekRaw)
	if err != nil {
		return nil, err
	}
	hpek1, err := HEPK(&hpek).Open()
	if err != nil {
		return nil, err
	}
	typ, err := hpek1.Classical.GetT()
	if err != nil {
		return nil, err
	}
	if typ != proto.DHType_Curve25519 {
		return nil, KeyImportError("bad dh type")
	}

	var fp proto.HEPKFingerprint
	err = fp.ImportFromDB(hpekFpRaw)
	if err != nil {
		return nil, err
	}

	fpExp, err := HEPK(&hpek).Fingerprint()
	if err != nil {
		return nil, err
	}

	if !fpExp.Eq(&fp) {
		return nil, HEPKFingerprintError{}
	}

	ret := SharedPublicSuite{
		SharedKey: proto.SharedKey{
			VerifyKey: vk,
			HepkFp:    fp,
			Role:      role,
			Gen:       proto.Generation(gen),
		},
		HEPK: hpek,
	}
	return &ret, nil
}

func PublicizeToSPSBoxer(s SharedPrivateSuiter, owner proto.FQParty) (*SPSBoxer, error) {
	sps, hepk, err := s.ExportToSharedKey()
	if err != nil {
		return nil, err
	}
	ret := SPSBoxer{
		SharedPublicSuite: SharedPublicSuite{SharedKey: *sps, HEPK: *hepk},
		Parent:            owner,
	}
	return &ret, nil
}

// SPSBoxer = SharedPublicSuite + Boxer
type SPSBoxer struct {
	SharedPublicSuite
	Parent proto.FQParty
}

func (s *SPSBoxer) KemEncapKey() (*proto.KemEncapKey, error) {
	return HEPK(&s.HEPK).KemEncapKey()
}

func (s *SPSBoxer) DHType() proto.DHType { return proto.DHType_Curve25519 }

func (s *SPSBoxer) ECDSA() (*ecdsa.PublicKey, error) {
	return nil, KeyNotFoundError{Which: "ECDSA DH key"}
}

func (s *SPSBoxer) Curve25519() (*proto.Curve25519PublicKey, error) {
	return HEPK(&s.HEPK).ExtractCurve25519()
}

func (s *SPSBoxer) ExportHEPK() (*proto.HEPK, error) {
	return &s.HEPK, nil
}

func (s *SPSBoxer) Ephemeral() (EphemeralSender, error) {
	return NewEphemeralSender25519()
}

func (s *SPSBoxer) DHPublicKey() (*proto.DHPublicKey, error) {
	return HEPK(&s.HEPK).DHPublicKey()
}

func (s *SPSBoxer) ExportToMember(h proto.HostID) (*proto.Member, *proto.HEPK, error) {
	ret := proto.Member{
		Id: proto.FQEntityInHostScope{
			Entity: s.Parent.Party.EntityID(),
			Host:   PickHostIDInScope(&s.Parent.Host, h),
		},
		Keys: proto.NewMemberKeysWithTeam(
			proto.TeamMemberKeys{
				VerifyKey: s.VerifyKey,
				HepkFp:    s.HepkFp,
				Gen:       s.Gen,
			},
		),
	}
	return &ret, &s.HEPK, nil
}

func (s *SPSBoxer) ExportToTarget(h proto.HostID) (*proto.SharedKeyBoxTarget, error) {
	return &proto.SharedKeyBoxTarget{
		Role: s.Role,
		Gen:  s.Gen,
		Eid:  s.Parent.Party.EntityID(),
		Host: PickHostIDInScope(&s.Parent.Host, h),
	}, nil
}

func (s *SPSBoxer) GetEntityID() proto.EntityID {
	return s.Parent.Party.EntityID()
}
func (s *SPSBoxer) GetHostID() *proto.HostID {
	return &s.Parent.Host
}

func ImportSPSBoxerFromTeamCert(
	cert *rem.TeamCertV1Payload,
) (
	*SPSBoxer,
	error,
) {
	sps, err := ImportSharedPublicSuite(&cert.Ptk, &cert.Hepk)
	if err != nil {
		return nil, err
	}
	ret := SPSBoxer{
		SharedPublicSuite: *sps,
		Parent:            cert.Team.FQParty(),
	}
	return &ret, nil
}

func ImportSPSBoxer(
	eid proto.FQEntity,
	hepks *HEPKSet,
	tmk proto.TeamMemberKeys,
	role proto.Role,
) (
	*SPSBoxer,
	error,
) {
	hepk, ok := hepks.Lookup(&tmk.HepkFp)
	if !ok {
		return nil, KeyNotFoundError{Which: "hepk"}
	}
	sk := proto.SharedKey{
		Gen:       tmk.Gen,
		Role:      role,
		HepkFp:    tmk.HepkFp,
		VerifyKey: tmk.VerifyKey,
	}
	pid, err := eid.Entity.ToPartyID()
	if err != nil {
		return nil, err
	}
	return &SPSBoxer{
		Parent: proto.FQParty{
			Party: pid,
			Host:  eid.Host,
		},
		SharedPublicSuite: SharedPublicSuite{
			SharedKey: sk,
			HEPK:      *hepk.Obj(),
		},
	}, nil
}

var _ PublicBoxer = &SPSBoxer{}

func RandomPUKVerifyKey() (EntityPublic, error) {
	var pk proto.Ed25519PublicKey
	err := RandomFill(pk[:])
	if err != nil {
		return nil, err
	}
	eid, err := proto.EntityType_PUKVerify.MakeEntityIDFromKey(pk)
	if err != nil {
		return nil, err
	}
	ret, err := ImportEntityPublic(eid)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *SharedPrivateSuite25519) AssertEqPub(pub proto.SharedKey, sendErr error) error {
	sk, _, err := s.ExportToSharedKey()
	if err != nil {
		return err
	}
	err = sk.Role.AssertEq(pub.Role, sendErr)
	if err != nil {
		return err
	}
	if sk.Gen != pub.Gen {
		return sendErr
	}
	if !sk.VerifyKey.Eq(pub.VerifyKey) {
		return sendErr
	}
	myFp, err := s.HEPKFingerprint()
	if err != nil {
		return err
	}
	if !myFp.Eq(&pub.HepkFp) {
		return sendErr
	}
	return nil
}
