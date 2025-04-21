// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libyubi

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/go-piv/piv-go/v2/piv"
)

type KeySuiteCore struct {
	ch   *Handle
	slot piv.Slot
	id   proto.YubiID
	pk   crypto.PublicKey
}

type KeySuite struct {
	KeySuiteCore
	role proto.Role
	hid  proto.HostID
}

type KeySuitePQ struct {
	KeySuiteCore
	dec proto.KemDecapKey
	enc *proto.KemEncapKey
}

type KeySuiteHybrid struct {
	KeySuite
	Pq KeySuitePQ
}

type keySuiter interface {
	ExportHEPK() (*proto.HEPK, error)
	EntityID() (proto.EntityID, error)
	ID() proto.YubiID
	HostID() proto.HostID
	Role() proto.Role
}

func (k *KeySuite) HostID() proto.HostID { return k.hid }
func (k *KeySuite) ID() proto.YubiID     { return k.id }
func (k *KeySuite) Role() proto.Role     { return k.role }

var _ keySuiter = (*KeySuite)(nil)

func (k *KeySuite) Fuse(kks *KeySuitePQ) *KeySuiteHybrid {
	return &KeySuiteHybrid{
		KeySuite: *k,
		Pq:       *kks,
	}
}

func NewKeySuite(ksc *KeySuiteCore, h proto.HostID, r proto.Role) *KeySuite {
	return &KeySuite{
		KeySuiteCore: *ksc,
		role:         r,
		hid:          h,
	}
}

func NewKeySuitePQ(ksc *KeySuiteCore, ss *proto.SecretSeed32) (*KeySuitePQ, error) {
	enc, dec, err := core.GenPQKemKeysFromSeed(*ss)
	if err != nil {
		return nil, err
	}
	return &KeySuitePQ{
		KeySuiteCore: *ksc,
		enc:          enc,
		dec:          dec,
	}, nil
}

func (w *KeySuiteCore) auth() piv.KeyAuth {
	var ret piv.KeyAuth
	if w.ch != nil && w.ch.pin != nil {
		ret.PIN = w.ch.pin.String()
	}
	return ret
}

func (k *KeySuiteCore) privateKey(ctx context.Context) (crypto.PrivateKey, Card, func(), error) {
	card, close, err := k.ch.Card(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	priv, err := card.PrivateKey(k.slot, k.pk, k.auth())
	if err != nil {
		close()
		return nil, nil, nil, err
	}
	return priv, card, close, nil
}

// Outputs g^x^2, which is roughly as secret as x if g^x isn't used anywhere else.
// We can in turn use this 32-byte secret as a seed for a PQ key. I wish there were
// a better way to put/get a secret to a yubikey, but so far, this is the best bet.
// The issue here is that this key might be used down the line somewhere else, opening existing
// FOKS data to quantum attacks. But the hope here is that all necessary FOKS information
// is contained on the yubikey, and never written down to storage locally, so this is
// a reasonable compromise, for now. Once yubikeys support PQ algorithms, we can do way
// better.
func (k *KeySuiteCore) GenerateSelfSecret(ctx context.Context) (*proto.SecretSeed32, error) {
	privGen, card, close, err := k.privateKey(ctx)
	if err != nil {
		return nil, err
	}
	defer close()
	rcvr, ok := k.pk.(*ecdsa.PublicKey)
	if !ok {
		return nil, core.YubiError("failed to cast key to ecdsa.PublicKey")
	}
	sk, err := card.SharedKey(privGen, rcvr)
	if err != nil {
		return nil, err
	}
	var ret proto.SecretSeed32
	if len(sk) != len(ret) {
		return nil, core.YubiError("bad shared key length")
	}
	copy(ret[:], sk)

	return &ret, nil
}

func (k *KeySuite) privateKeyNoCtx() (crypto.PrivateKey, Card, func(), error) {
	return k.privateKey(context.Background())
}

func (k *KeySuite) Sign(obj core.Verifiable) (*proto.Signature, error) {
	priv, _, close, err := k.privateKeyNoCtx()
	if err != nil {
		return nil, err
	}
	defer close()
	s, ok := priv.(crypto.Signer)
	if !ok {
		return nil, core.YubiError("failed to cast key to Signer")
	}
	data, err := core.ECDSASigPayload(obj)
	if err != nil {
		return nil, err
	}
	out, err := s.Sign(rand.Reader, data, crypto.SHA256)
	if err != nil {
		return nil, err
	}
	ret := proto.NewSignatureWithEcdsa(proto.ECDSASignature(out))
	return &ret, nil
}

func (k *KeySuite) boxForCommon(
	r core.PublicBoxer,
) (
	proto.DHSharedKey,
	*ecdsa.PublicKey,
	error,
) {
	err := core.AssertDHTypeMatch(k, r)
	if err != nil {
		return nil, nil, err
	}
	receiver, err := r.ECDSA()
	if err != nil {
		return nil, nil, err
	}
	xPriv, card, close, err := k.privateKeyNoCtx()
	if err != nil {
		return nil, nil, err
	}
	defer close()
	shared, err := card.SharedKey(xPriv, receiver)
	if err != nil {
		return nil, nil, err
	}
	return proto.DHSharedKey(shared), receiver, nil

}

func (k *KeySuite) BoxFor(o core.CryptoPayloader, r core.PublicBoxer, opts core.BoxOpts) (*proto.Box, error) {
	shared, receiver, err := k.boxForCommon(r)
	if err != nil {
		return nil, err
	}
	return core.SealIntoECDSABox(o, shared, receiver, opts)
}

func (k *KeySuiteHybrid) BoxFor(o core.CryptoPayloader, rec core.PublicBoxer, opts core.BoxOpts) (*proto.Box, error) {
	shared, _, err := k.boxForCommon(rec)
	if err != nil {
		return nil, err
	}
	senderPk := k.ExportDHPublicKey(false)
	return core.HybridEncryptCommon(
		o,
		shared,
		proto.DHType_P256,
		senderPk,
		rec,
		opts,
	)
}

func (k *KeySuite) DHType() proto.DHType                     { return proto.DHType_P256 }
func (k *KeySuite) EntityID() (proto.EntityID, error)        { return proto.EntityID(k.id), nil }
func (k *KeySuite) GetRole() proto.Role                      { return k.role }
func (k *KeySuite) RollingEntityID() (proto.EntityID, error) { return k.EntityID() }

func (k *KeySuite) EntityPublic() (core.EntityPublic, error) {
	eid, err := k.EntityID()
	if err != nil {
		return nil, err
	}
	return core.EntityPublicECDSA{EntityID: eid}, nil
}

func (k *KeySuite) ExportDHPublicKey(inContextOfSigKey bool) proto.DHPublicKey {
	return proto.NewDHPublicKeyWithP256(k.id.CompressedPublicKey())
}

func (k *KeySuite) ExportHEPK() (*proto.HEPK, error) {
	ret := proto.NewHEPKWithV1(
		proto.HEPKv1{
			Classical: proto.NewDHPublicKeyWithP256(
				k.id.CompressedPublicKey(),
			),
		},
	)
	return &ret, nil
}

func (k *KeySuiteHybrid) ExportHEPK() (*proto.HEPK, error) {
	dh := k.KeySuite.ExportDHPublicKey(false)
	ret := proto.NewHEPKWithV1(
		proto.HEPKv1{
			Classical: dh,
			Pqkem:     *k.Pq.enc,
		},
	)
	return &ret, nil
}

func exportKeySuite(k keySuiter) (*proto.KeySuite, error) {
	eid, err := k.EntityID()
	if err != nil {
		return nil, err
	}
	hepk, err := k.ExportHEPK()
	if err != nil {
		return nil, err
	}
	return &proto.KeySuite{
		Entity: eid,
		Hepk:   *hepk,
	}, nil
}

func (k *KeySuite) ExportKeySuite() (*proto.KeySuite, error)             { return exportKeySuite(k) }
func (k *KeySuiteHybrid) ExportKeySuite() (*proto.KeySuite, error)       { return exportKeySuite(k) }
func (k *KeySuite) ExportToMember(h proto.HostID) (*proto.Member, error) { return exportToMember(k, h) }
func (k *KeySuiteHybrid) ExportToMember(h proto.HostID) (*proto.Member, error) {
	return exportToMember(k, h)
}

func exportToMember(k keySuiter, h proto.HostID) (*proto.Member, error) {
	hepk, err := k.ExportHEPK()
	if err != nil {
		return nil, err
	}
	fp, err := core.HEPK(hepk).Fingerprint()
	if err != nil {
		return nil, err
	}

	id := k.ID()
	keyHost := k.HostID()

	return &proto.Member{
		Id: proto.FQEntityInHostScope{
			Entity: proto.EntityID(id),
			Host:   core.PickHostIDInScope(&keyHost, h),
		},
		Keys: proto.NewMemberKeysWithUser(
			proto.UserMemberKeys{
				HepkFp: *fp,
			},
		),
	}, nil
}

func (k *KeySuite) PrivateKeyForCert() (crypto.PrivateKey, error) {
	return nil, core.YubiError("cannot use a yubikey for mTLS; use a delegated subkey instead")
}

func (k *KeySuite) PublicizeToBoxer() (core.PublicBoxer, error) {
	return k.Publicize(nil)
}
func (k *KeySuiteHybrid) PublicizeToBoxer() (core.PublicBoxer, error) {
	return k.Publicize(nil)
}

func (k *KeySuite) Publicize(hostID *proto.HostID) (core.PublicSuiter, error) {
	return publicize(k, hostID)
}
func (k *KeySuiteHybrid) Publicize(hostID *proto.HostID) (core.PublicSuiter, error) {
	return publicize(k, hostID)
}

func publicize(k keySuiter, hostID *proto.HostID) (core.PublicSuiter, error) {
	ent, err := k.EntityID()
	if err != nil {
		return nil, err
	}
	hepk, err := k.ExportHEPK()
	if err != nil {
		return nil, err
	}
	role := k.Role()
	return &core.PublicSuiteECDSA{
		EntityPublicECDSA: core.EntityPublicECDSA{EntityID: ent},
		HostID:            hostID,
		Role:              role,
		Hepk:              hepk,
	}, nil
}

func (k *KeySuiteHybrid) KemDecap() proto.KemDecapKey { return k.Pq.dec }

func (k *KeySuiteHybrid) UnboxFor(
	o core.CryptoPayloader,
	box proto.Box,
	sender core.PublicBoxer,
) (
	core.DHPublicKey,
	error,
) {
	v1, sDh, err := core.OpenBoxAndSenderHybrid(box, sender)
	if err != nil {
		return nil, err
	}
	return k.unboxWithSenderDH(o, v1, sDh)
}

func (k *KeySuiteHybrid) UnboxForEphemeral(
	o core.CryptoPayloader,
	box proto.Box,
	sender proto.DHPublicKey,
) error {
	v1, err := core.OpenBoxHybrid(box)
	if err != nil {
		return err
	}
	_, err = k.unboxWithSenderDH(o, v1, &sender)
	return err
}

func (k *KeySuiteHybrid) unboxWithSenderDH(
	o core.CryptoPayloader,
	v1 *proto.BoxHybridV1,
	sDh *proto.DHPublicKey,
) (
	core.DHPublicKey,
	error,
) {
	sDhTyp, err := sDh.GetT()
	if err != nil {
		return nil, err
	}
	if sDhTyp != proto.DHType_P256 {
		return nil, core.BoxError("got non-P256 sender key for yubikey")
	}
	sDhECSA, err := sDh.P256().ImportToECDSAPublic()
	if err != nil {
		return nil, err
	}
	if v1.DhType != proto.DHType_P256 {
		return nil, core.BoxError("got non-P256 box for yubikey")
	}
	xPriv, card, close, err := k.privateKeyNoCtx()
	if err != nil {
		return nil, err
	}
	defer close()

	dhSharedKey, err := card.SharedKey(xPriv, sDhECSA)
	if err != nil {
		return nil, err
	}
	rHepk, err := k.ExportHEPK()
	if err != nil {
		return nil, err
	}

	err = core.HybridUnboxCommon(
		o,
		v1,
		dhSharedKey,
		sDh,
		&k.Pq.dec,
		*rHepk,
	)
	if err != nil {
		return nil, err
	}
	return core.DHPublicKeyYubi{PublicKey: sDhECSA}, nil
}

func (k *KeySuite) UnboxFor(
	o core.CryptoPayloader,
	box proto.Box,
	sender core.PublicBoxer,
) (
	core.DHPublicKey,
	error,
) {
	var sndrEcdsa *ecdsa.PublicKey
	xPriv, card, close, err := k.privateKeyNoCtx()
	if err != nil {
		return nil, err
	}
	defer close()
	if sender != nil {
		err := core.AssertDHTypeMatch(k, sender)
		if err != nil {
			return nil, err
		}
		sndrEcdsa, err = sender.ECDSA()
		if err != nil {
			return nil, err
		}
	}
	ret, err := core.OpenECDHBox(o, box, xPriv, card, sndrEcdsa)
	if err != nil {
		return nil, err
	}
	return core.DHPublicKeyYubi{PublicKey: ret}, nil
}

func (k *KeySuite) UnboxForIncludedEphemeral(
	o core.CryptoPayloader,
	box proto.Box,
) error {
	xPriv, card, close, err := k.privateKeyNoCtx()
	if err != nil {
		return err
	}
	defer close()
	_, err = core.OpenECDHBox(o, box, xPriv, card, nil)
	if err != nil {
		return err
	}
	return nil
}

func (k *KeySuite) UnboxForEphemeral(
	o core.CryptoPayloader,
	box proto.Box,
	sender proto.DHPublicKey,
) error {
	typ, err := sender.GetT()
	if err != nil {
		return err
	}
	if typ != proto.DHType_P256 {
		return core.BoxError("got non-P256 sender key")
	}
	xPriv, card, close, err := k.privateKeyNoCtx()
	if err != nil {
		return err
	}
	defer close()
	sndrEcdsa, err := sender.P256().ImportToECDSAPublic()
	if err != nil {
		return err
	}
	_, err = core.OpenECDHBox(o, box, xPriv, card, sndrEcdsa)
	if err != nil {
		return err
	}
	return nil
}

func (k *KeySuite) HasSubkey() bool { return true }
func (k *KeySuite) CertSigner() (core.EntityPrivate, error) {
	return nil, core.YubiError("cannot use yubikey as a cert signer; need subkey")
}

func (k *KeySuite) Verify(s proto.Signature, obj core.Verifiable) error {
	pub, ok := k.pk.(*ecdsa.PublicKey)
	if !ok {
		return core.YubiError("type assertion failed")
	}
	return core.VerifyWithECDSAPublic(pub, s, obj)
}

func (k *KeySuiteCore) ExportToYubiKeyInfo(ctx context.Context) (*proto.YubiKeyInfoHybrid, error) {
	card, close, err := k.ch.Card(ctx)
	if err != nil {
		return nil, err
	}
	defer close()
	serial, err := card.Serial()
	if err != nil {
		return nil, err
	}

	return &proto.YubiKeyInfoHybrid{
		Card: proto.YubiCardID{
			Name:   k.ch.nm,
			Serial: serial,
		},
		Key: proto.YubiSlotAndKeyID{
			Id:   k.id,
			Slot: proto.YubiSlot(k.slot.Key),
		},
	}, nil
}

func (k *KeySuitePQ) PQKeyID() (*proto.YubiPQKeyID, error) {
	epk, ok := k.KeySuiteCore.pk.(*ecdsa.PublicKey)
	if !ok {
		return nil, core.InternalError("failed to cast key to ecdsa.PublicKey")
	}
	return core.ComputeYubiPQKeyID(epk)
}

func (k *KeySuitePQ) ExportToYubiSlotAndPQKey() (*proto.YubiSlotAndPQKeyID, error) {
	id, err := k.PQKeyID()
	if err != nil {
		return nil, err
	}
	ret := proto.YubiSlotAndPQKeyID{
		Slot: proto.YubiSlot(k.slot.Key),
		Id:   *id,
	}
	return &ret, nil
}

func (k *KeySuiteHybrid) ExportToYubiKeyInfo(ctx context.Context) (*proto.YubiKeyInfoHybrid, error) {
	base, err := k.KeySuiteCore.ExportToYubiKeyInfo(ctx)
	if err != nil {
		return nil, err
	}
	ret := proto.YubiKeyInfoHybrid{
		Card: base.Card,
		Key:  base.Key,
	}
	pq, err := k.Pq.ExportToYubiSlotAndPQKey()
	if err != nil {
		return nil, err
	}
	ret.PqKey = *pq
	return &ret, nil
}

var _ core.PrivateSuiter = (*KeySuite)(nil)
var _ core.PrivateSuiter = (*KeySuiteHybrid)(nil)
