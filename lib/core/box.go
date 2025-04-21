// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha512"

	proto "github.com/foks-proj/go-foks/proto/lib"
	"golang.org/x/crypto/nacl/box"
)

type BoxOpts struct {
	IncludePublicKey bool
}

type Boxer interface {
	BoxFor(CryptoPayloader, PublicBoxer, BoxOpts) (*proto.Box, error)
}

type DHPublicKey interface {
	ECDSA() *ecdsa.PublicKey
	Curve25519() *proto.Curve25519PublicKey
	Export() proto.DHPublicKey
}

type DHPublicKey25519 struct {
	*proto.Curve25519PublicKey
}

type DHPublicKeyYubi struct {
	*ecdsa.PublicKey
}

func (d DHPublicKey25519) ECDSA() *ecdsa.PublicKey                { return nil }
func (d DHPublicKey25519) Curve25519() *proto.Curve25519PublicKey { return d.Curve25519PublicKey }
func (d DHPublicKey25519) Export() proto.DHPublicKey {
	return proto.NewDHPublicKeyWithCurve25519(*d.Curve25519PublicKey)
}

func (d DHPublicKeyYubi) ECDSA() *ecdsa.PublicKey                { return d.PublicKey }
func (d DHPublicKeyYubi) Curve25519() *proto.Curve25519PublicKey { return nil }
func (d DHPublicKeyYubi) Export() proto.DHPublicKey {
	return proto.NewDHPublicKeyWithP256(proto.ExportECDSAPublic(d.PublicKey))
}

type PrivateBoxer interface {
	EntityPrivate
	DHType() proto.DHType
	BoxFor(CryptoPayloader, PublicBoxer, BoxOpts) (*proto.Box, error)
	UnboxFor(CryptoPayloader, proto.Box, PublicBoxer) (DHPublicKey, error)
	UnboxForEphemeral(CryptoPayloader, proto.Box, proto.DHPublicKey) error
	UnboxForIncludedEphemeral(CryptoPayloader, proto.Box) error
	EntityID() (proto.EntityID, error)
	ExportDHPublicKey(inContextOfSigKey bool) proto.DHPublicKey
	ExportHEPK() (*proto.HEPK, error)
	PublicizeToBoxer() (PublicBoxer, error)
}

type PublicBoxer interface {
	EntityPublic
	DHType() proto.DHType
	ECDSA() (*ecdsa.PublicKey, error)
	DHPublicKey() (*proto.DHPublicKey, error)
	Curve25519() (*proto.Curve25519PublicKey, error)
	Ephemeral() (EphemeralSender, error)
	GetHostID() *proto.HostID
	ExportToMember(h proto.HostID) (*proto.Member, *proto.HEPK, error)
	ExportToTarget(h proto.HostID) (*proto.SharedKeyBoxTarget, error)
	ExportHEPK() (*proto.HEPK, error)
	KemEncapKey() (*proto.KemEncapKey, error)
}

func checkDHTypeMatch(s PrivateBoxer, r PublicBoxer) bool {
	return s.DHType() == r.DHType()
}

func AssertDHTypeMatch(s PrivateBoxer, r PublicBoxer) error {
	if !checkDHTypeMatch(s, r) {
		return BoxError("DH type mismatch")
	}
	return nil
}

var _ PrivateBoxer = (*PrivateSuite25519)(nil)
var _ PublicBoxer = (*PublicSuite25519)(nil)
var _ PublicBoxer = (*PublicSuiteECDSA)(nil)

type EphemeralSender interface {
	BoxFor(CryptoPayloader, PublicBoxer, BoxOpts) (*proto.Box, error)
	Export() *proto.DHPublicKey
}

type EphemeralSender25519 struct {
	enc *proto.Curve25519PublicKey
	dec *proto.Curve25519SecretKey
}

var _ EphemeralSender = (*EphemeralSender25519)(nil)
var _ EphemeralSender = (*EphemeralSenderECDSA)(nil)

func NewEphemeralSender25519() (*EphemeralSender25519, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &EphemeralSender25519{
		enc: (*proto.Curve25519PublicKey)(pub),
		dec: (*proto.Curve25519SecretKey)(priv),
	}, nil
}

func (e *EphemeralSender25519) PublicKey() *proto.Curve25519PublicKey {
	return e.enc
}

func (e *EphemeralSender25519) SecretKey() *proto.Curve25519SecretKey {
	return e.dec
}

type EphemeralSenderECDSA struct {
	priv *ecdsa.PrivateKey
	pub  *ecdsa.PublicKey
}

func NewEphemeralSenderECDSA() (*EphemeralSenderECDSA, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	pub := priv.PublicKey
	return &EphemeralSenderECDSA{
		pub:  &pub,
		priv: priv,
	}, nil
}

func (e *EphemeralSenderECDSA) Export() *proto.DHPublicKey {
	key := proto.ExportECDSAPublic(e.pub)
	ret := proto.NewDHPublicKeyWithP256(key)
	return &ret
}

func (e *EphemeralSender25519) Export() *proto.DHPublicKey {
	ret := proto.NewDHPublicKeyWithCurve25519(*e.enc)
	return &ret
}

func (e *EphemeralSender25519) BoxFor(o CryptoPayloader, r PublicBoxer, opts BoxOpts) (*proto.Box, error) {
	if r.DHType() != proto.DHType_Curve25519 {
		return nil, BoxError("DH type mismatch")
	}
	return hybridEncrypt25519(o, e.dec, e.enc, r, opts)
}

type ECDHSharedKeyer interface {
	SharedKey(crypto.PrivateKey, *ecdsa.PublicKey) ([]byte, error)
}

func OpenECDHBox(
	o CryptoPayloader,
	box proto.Box,
	privKey crypto.PrivateKey,
	skyer ECDHSharedKeyer,
	sndr *ecdsa.PublicKey,
) (
	*ecdsa.PublicKey,
	error,
) {
	t, err := box.GetT()
	if err != nil {
		return nil, err
	}
	if t != proto.BoxType_YUBI {
		return nil, BoxError("unsupported box type")
	}
	ybox := box.Yubi()
	if sndr == nil && ybox.Pk == nil {
		return nil, BoxError("no sender key")
	}
	if sndr == nil && ybox.Pk != nil {
		sndr, err = ybox.Pk.ImportToECDSAPublic()
		if err != nil {
			return nil, err
		}
	}
	if sndr == nil {
		return nil, InternalError("no sender key")
	}
	shared, err := skyer.SharedKey(privKey, sndr)
	if err != nil {
		return nil, err
	}
	sum := sha512.Sum512_256(shared)
	sboxKey := proto.SecretBoxKey(sum)
	err = OpenSecretBoxInto(o, ybox.SecretBox, &sboxKey)
	if err != nil {
		return nil, err
	}
	return sndr, nil
}

func SealIntoECDSABox(o CryptoPayloader, sharedKey []byte, receiver *ecdsa.PublicKey, opts BoxOpts) (*proto.Box, error) {
	sum := sha512.Sum512_256(sharedKey)
	sboxKey := proto.SecretBoxKey(sum)
	sbox, err := SealIntoSecretBox(o, &sboxKey)
	if err != nil {
		return nil, err
	}
	ybox := proto.YubiBox{SecretBox: *sbox}
	if opts.IncludePublicKey {
		compressed := proto.ExportECDSAPublic(receiver)
		ybox.Pk = &compressed
	}
	ret := proto.NewBoxWithYubi(ybox)
	return &ret, nil
}

func (e *EphemeralSenderECDSA) BoxFor(o CryptoPayloader, receiver PublicBoxer, opts BoxOpts) (*proto.Box, error) {
	if receiver.DHType() != proto.DHType_P256 {
		return nil, BoxError("DH type mismatch")
	}
	return hybridEncryptECDSA(o, e.priv, e.pub, receiver, opts)
}

var _ EphemeralSender = (*EphemeralSender25519)(nil)
var _ EphemeralSender = (*EphemeralSenderECDSA)(nil)

type SharedKeyBoxer struct {
	id     proto.BoxSetID
	host   proto.HostID
	sender PrivateBoxer
	eph    EphemeralSender
	boxes  []proto.SharedKeyBox
}

func BoxOne(h proto.HostID, puk SharedPrivateSuiter, s PrivateBoxer, r PublicBoxer) (*proto.SharedKeyBoxSet, error) {
	skb, err := NewSharedKeyBoxer(h, s)
	if err != nil {
		return nil, err
	}
	err = skb.Box(puk, r)
	if err != nil {
		return nil, err
	}
	return skb.Finish()
}

func NewSharedKeyBoxer(h proto.HostID, s PrivateBoxer) (*SharedKeyBoxer, error) {
	ret := SharedKeyBoxer{
		host:   h,
		sender: s,
	}
	err := RandomFill(ret.id[:])
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (s *SharedKeyBoxer) Box(puk SharedPrivateSuiter, rcvr PublicBoxer) error {

	var boxer Boxer
	if checkDHTypeMatch(s.sender, rcvr) {
		boxer = s.sender
	} else {
		if s.eph == nil {
			var err error
			s.eph, err = rcvr.Ephemeral()
			if err != nil {
				return err
			}
		}
		boxer = s.eph
	}
	rhid := rcvr.GetHostID()
	if rhid == nil {
		return BoxError("cannot box without a receiver host")
	}

	fqe := proto.FQEntity{
		Entity: rcvr.GetEntityID(),
		Host:   *rhid,
	}

	seed := puk.ExportToBoxCleartext(fqe)

	box, err := boxer.BoxFor(&seed, rcvr, BoxOpts{IncludePublicKey: false})
	if err != nil {
		return err
	}
	skb := proto.SharedKeyBox{
		Gen:  seed.Gen,
		Role: seed.Role,
		Box:  *box,
	}
	targ, err := rcvr.ExportToTarget(s.host)
	if err != nil {
		return err
	}
	skb.Targ = *targ

	s.boxes = append(s.boxes, skb)
	return nil
}

func (s *SharedKeyBoxer) Finish() (*proto.SharedKeyBoxSet, error) {
	ret := proto.SharedKeyBoxSet{
		Id:    s.id,
		Boxes: s.boxes,
	}
	if s.eph == nil {
		return &ret, nil
	}
	ekey := s.eph.Export()
	if ekey == nil {
		return nil, BoxError("failed to export ephemeral key")
	}
	tdhk := proto.TempDHKey{
		Key:  *ekey,
		Time: proto.Now(),
	}
	signer, err := s.sender.EntityID()
	if err != nil {
		return nil, err
	}
	template := proto.TempDHKeySigTemplate{
		TempDHKey: tdhk,
		Signer: proto.FQEntity{
			Host:   s.host,
			Entity: signer,
		},
		BoxSetId: s.id,
	}
	sig, err := s.sender.Sign(&template)
	if err != nil {
		return nil, err
	}
	ret.TempDHKeySigned = &proto.TempDHKeySigned{
		TempDHKey: tdhk,
		Sig:       *sig,
	}
	return &ret, nil
}

func OpenBoxInSet(
	obj CryptoPayloader,
	box proto.Box,
	tdh *proto.TempDHKeySigned,
	bsid *proto.BoxSetID,
	sndr PublicBoxer,
	rcvr PrivateSuiter,
) error {

	if sndr == nil {
		return InternalError("OpenBoxWithSender expects a sender")
	}

	if sndr.DHType() == rcvr.DHType() {
		_, err := rcvr.UnboxFor(obj, box, sndr)
		if err != nil {
			return err
		}

		return nil
	}

	if tdh == nil || bsid == nil {
		return BoxError("must use a TempDHKey with mismatched DH types")
	}
	host := sndr.GetHostID()
	if host == nil {
		return BoxError("cannot open box without a sender host")
	}

	template := proto.TempDHKeySigTemplate{
		TempDHKey: tdh.TempDHKey,
		Signer: proto.FQEntity{
			Host:   *host,
			Entity: sndr.GetEntityID(),
		},
		BoxSetId: *bsid,
	}
	err := sndr.Verify(tdh.Sig, &template)
	if err != nil {
		return err
	}
	err = rcvr.UnboxForEphemeral(obj, box, tdh.TempDHKey.Key)
	if err != nil {
		return err
	}
	return nil
}

func SelfBox(b PrivateSuiter, obj CryptoPayloader, hostID proto.HostID) (*proto.Box, error) {
	pub, err := b.Publicize(&hostID)
	if err != nil {
		return nil, err
	}
	return b.BoxFor(obj, pub, BoxOpts{})
}

func SelfUnbox(b PrivateSuiter, obj CryptoPayloader, box proto.Box) error {
	pub, err := b.Publicize(nil)
	if err != nil {
		return err
	}
	_, err = b.UnboxFor(obj, box, pub)
	return err
}

func BoxForEmphemeral(
	obj CryptoPayloader,
	rcvr PublicBoxer,
	opts BoxOpts,
) (
	*proto.Box,
	error,
) {
	eph, err := rcvr.Ephemeral()
	if err != nil {
		return nil, err
	}
	ret, err := eph.BoxFor(obj, rcvr, opts)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
