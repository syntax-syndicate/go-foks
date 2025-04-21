// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	proto "github.com/foks-proj/go-foks/proto/lib"
)

type PrivateSuiteHybridECDSA struct {
	seed     proto.SecretSeed32
	pub      *ecdsa.PublicKey
	priv     *ecdsa.PrivateKey
	kemEncap proto.KemEncapKey
	kemDecap proto.KemDecapKey
}

var _ PrivateBoxer = (*PrivateSuiteHybridECDSA)(nil)

func NewPrivateSuiteECDSA(s proto.SecretSeed32) (*PrivateSuiteHybridECDSA, error) {
	kemEnc, kemDec, err := GenPQKemKeysFromSeed(s)
	if err != nil {
		return nil, err
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	pub := priv.PublicKey
	return &PrivateSuiteHybridECDSA{
		seed:     s,
		pub:      &pub,
		priv:     priv,
		kemEncap: *kemEnc,
		kemDecap: kemDec,
	}, nil
}

func (p *PrivateSuiteHybridECDSA) DHType() proto.DHType { return proto.DHType_P256 }
func (p *PrivateSuiteHybridECDSA) EntityID() (proto.EntityID, error) {
	cpkid := proto.ExportECDSAPublic(p.pub)
	return proto.EntityType_Yubi.MakeEntityID(cpkid)
}
func (p *PrivateSuiteHybridECDSA) EntityPublic() (EntityPublic, error) {
	eid, err := p.EntityID()
	if err != nil {
		return nil, err
	}
	return EntityPublicECDSA{EntityID: eid}, nil
}
func (p *PrivateSuiteHybridECDSA) ExportDHPublicKey(inContextOfSigKey bool) proto.DHPublicKey {
	return proto.NewDHPublicKeyWithP256(proto.ExportECDSAPublic(p.pub))
}
func (p *PrivateSuiteHybridECDSA) ExportHEPK() (*proto.HEPK, error) {
	dh := p.ExportDHPublicKey(false)
	ret := proto.NewHEPKWithV1(
		proto.HEPKv1{
			Classical: dh,
			Pqkem:     p.kemEncap,
		},
	)
	return &ret, nil
}

func (p *PrivateSuiteHybridECDSA) PrivateKeyForCert() (crypto.PrivateKey, error) {
	return p.priv, nil
}
func (p *PrivateSuiteHybridECDSA) Sign(obj Verifiable) (*proto.Signature, error) {
	return nil, NotImplementedError{}
}

func (p *PrivateSuiteHybridECDSA) PublicizeToBoxer() (PublicBoxer, error) {
	eid, err := p.EntityID()
	if err != nil {
		return nil, err
	}
	hepk, err := p.ExportHEPK()
	if err != nil {
		return nil, err
	}
	return &PublicSuiteECDSA{
		EntityPublicECDSA: EntityPublicECDSA{EntityID: eid},
		Role:              proto.OwnerRole,
		Hepk:              hepk,
	}, nil
}

func hybridKeySwizzle(p *proto.HybridSecretKeySHA3Payload) (*proto.SecretBoxKey, error) {
	var ret proto.SecretBoxKey
	err := PrefixedSHA3HashInto(p, ret[:])
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (p *PrivateSuiteHybridECDSA) BoxFor(c CryptoPayloader, rec PublicBoxer, boxOpts BoxOpts) (*proto.Box, error) {
	err := AssertDHTypeMatch(p, rec)
	if err != nil {
		return nil, err
	}
	return hybridEncryptECDSA(c, p.priv, p.pub, rec, boxOpts)
}

func hybridEncryptEDCSAInnerDH(
	dec *ecdsa.PrivateKey, rec *ecdsa.PublicKey,
) (
	proto.DHSharedKey, error,
) {
	y, err := rec.ECDH()
	if err != nil {
		return nil, err
	}
	x, err := dec.ECDH()
	if err != nil {
		return nil, err
	}
	dhKey, err := x.ECDH(y)
	if err != nil {
		return nil, err
	}
	return proto.DHSharedKey(dhKey), nil
}

func hybridEncryptECDSA(
	c CryptoPayloader, dec *ecdsa.PrivateKey, enc *ecdsa.PublicKey,
	rec PublicBoxer, boxOpts BoxOpts,
) (*proto.Box, error) {
	if rec.DHType() != proto.DHType_P256 {
		return nil, BoxError("cannot box for non-P256 recipient")
	}
	recECDSA, err := rec.ECDSA()
	if err != nil {
		return nil, err
	}
	dhKey, err := hybridEncryptEDCSAInnerDH(dec, recECDSA)
	if err != nil {
		return nil, err
	}
	sPub := proto.NewDHPublicKeyWithP256(proto.ExportECDSAPublic(enc))
	return HybridEncryptCommon(c, dhKey, proto.DHType_P256, sPub, rec, boxOpts)
}

func hybridEncrypt25519(
	c CryptoPayloader, dec *proto.Curve25519SecretKey, enc *proto.Curve25519PublicKey,
	rec PublicBoxer, boxOpts BoxOpts,
) (*proto.Box, error) {

	rDh, err := rec.Curve25519()
	if err != nil {
		return nil, err
	}
	dhKey, err := Curve25519DHExchange(dec, rDh)
	if err != nil {
		return nil, err
	}
	return HybridEncryptCommon(c, dhKey.ToDHSharedKey(), proto.DHType_Curve25519,
		proto.NewDHPublicKeyWithCurve25519(*enc),
		rec, boxOpts,
	)
}

func HybridEncryptCommon(
	c CryptoPayloader, dhKey proto.DHSharedKey, dhType proto.DHType,
	sPub proto.DHPublicKey, rec PublicBoxer,
	boxOpts BoxOpts,
) (*proto.Box, error) {

	rKem, err := rec.KemEncapKey()
	if err != nil {
		return nil, err
	}

	kemCtext, kemKey, err := PQKemAlgo.Encap(*rKem)
	if err != nil {
		return nil, err
	}

	hepk, err := rec.ExportHEPK()
	if err != nil {
		return nil, err
	}
	payload := proto.HybridSecretKeySHA3Payload{
		PqKemKey:    kemKey,
		DhSharedKey: dhKey,
		Version:     proto.BoxHybridVersion_V1,
		Rcvr:        *hepk,
		Sndr:        sPub,
	}
	finalKey, err := hybridKeySwizzle(&payload)
	if err != nil {
		return nil, err
	}

	sbox, err := SealIntoSecretBox(c, finalKey)
	if err != nil {
		return nil, err
	}

	v1 := proto.BoxHybridV1{
		KemCtext: kemCtext,
		DhType:   dhType,
		Sbox:     *sbox,
	}
	if boxOpts.IncludePublicKey {
		v1.Sender = &sPub
	}

	ret := proto.NewBoxWithHybrid(
		proto.NewBoxHybridWithV1(v1),
	)
	return &ret, nil
}

func OpenBoxHybrid(b proto.Box) (*proto.BoxHybridV1, error) {
	typ, err := b.GetT()
	if err != nil {
		return nil, err
	}
	if typ != proto.BoxType_HYBRID {
		return nil, BoxError("got non-hybrid box")
	}
	bh := b.Hybrid()
	v, err := bh.GetV()
	if err != nil {
		return nil, err
	}
	if v != proto.BoxHybridVersion_V1 {
		return nil, VersionNotSupportedError("hybrid box from future")
	}
	ret := bh.V1()
	return &ret, nil
}

func OpenDHSender(v1 *proto.BoxHybridV1, sender PublicBoxer) (*proto.DHPublicKey, error) {
	var sDh *proto.DHPublicKey
	var err error
	if sender != nil {
		sDh, err = sender.DHPublicKey()
		if err != nil {
			return nil, err
		}
	} else {
		sDh = v1.Sender
	}
	if sDh == nil {
		return nil, BoxError("missing sender public key")
	}
	return sDh, nil
}

func OpenBoxAndSenderHybrid(b proto.Box, sender PublicBoxer) (*proto.BoxHybridV1, *proto.DHPublicKey, error) {
	ret, err := OpenBoxHybrid(b)
	if err != nil {
		return nil, nil, err
	}
	sDh, err := OpenDHSender(ret, sender)
	if err != nil {
		return nil, nil, err
	}
	return ret, sDh, nil
}

func HybridUnboxCommon(
	c CryptoPayloader,
	v1 *proto.BoxHybridV1,
	dhKey proto.DHSharedKey,
	senderDh *proto.DHPublicKey,
	dec *proto.KemDecapKey,
	rcvrPk proto.HEPK,
) error {

	kemSk, err := PQKemAlgo.Decap(*dec, v1.KemCtext)
	if err != nil {
		return err
	}

	payload := proto.HybridSecretKeySHA3Payload{
		PqKemKey:    kemSk,
		DhSharedKey: dhKey,
		Version:     proto.BoxHybridVersion_V1,
		Rcvr:        rcvrPk,
		Sndr:        *senderDh,
	}
	finalKey, err := hybridKeySwizzle(&payload)
	if err != nil {
		return err
	}

	err = OpenSecretBoxInto(c, v1.Sbox, finalKey)
	if err != nil {
		return err
	}
	return nil
}

func (p *PrivateSuiteHybridECDSA) UnboxFor(
	c CryptoPayloader, b proto.Box, sender PublicBoxer,
) (DHPublicKey, error) {

	v1, sDh, err := OpenBoxAndSenderHybrid(b, sender)
	if err != nil {
		return nil, err
	}

	if v1.DhType != proto.DHType_P256 {
		return nil, BoxError("got non-ECDSA box")
	}
	sDhTyp, err := sDh.GetT()
	if err != nil {
		return nil, err
	}
	if sDhTyp != proto.DHType_P256 {
		return nil, BoxError("got non-ECDSA sender key")
	}

	sCompressed := sDh.P256()
	sECDSA, err := sCompressed.ImportToECDSAPublic()
	if err != nil {
		return nil, err
	}

	skDH, err := hybridEncryptEDCSAInnerDH(p.priv, sECDSA)
	if err != nil {
		return nil, err
	}
	rHepk, err := p.ExportHEPK()
	if err != nil {
		return nil, err
	}

	err = HybridUnboxCommon(c, v1, skDH, sDh, &p.kemDecap, *rHepk)
	if err != nil {
		return nil, err
	}
	return DHPublicKeyYubi{PublicKey: sECDSA}, nil
}

func (p *PrivateSuiteHybridECDSA) UnboxForEphemeral(
	o CryptoPayloader,
	box proto.Box,
	sender proto.DHPublicKey,
) error {
	return NotImplementedError{}
}

func (p *PrivateSuiteHybridECDSA) UnboxForIncludedEphemeral(
	o CryptoPayloader,
	box proto.Box,
) error {
	return NotImplementedError{}
}

func (e PrivateSuiteHybridECDSA) Verify(s proto.Signature, obj Verifiable) error {
	return NotImplementedError{}
}
