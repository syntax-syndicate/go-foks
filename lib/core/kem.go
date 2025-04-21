// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"bytes"

	"filippo.io/mlkem768"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type KEM interface {
	Decap(key proto.KemDecapKey, ctext proto.KemCiphertext) (proto.KemSharedKey, error)
	Encap(key proto.KemEncapKey) (proto.KemCiphertext, proto.KemSharedKey, error)
	GenKey() (*proto.KemEncapKey, proto.KemDecapKey, error)
	KeyFromSeed(proto.KemSeed) (*proto.KemEncapKey, proto.KemDecapKey, error)
	Type() proto.KEMType
	SeedSize() int
	DeriveSeed(s proto.SecretSeed32) (proto.KemSeed, error)
}

type Mlkem768 struct{}

var _ KEM = (*Mlkem768)(nil)

var PQKemAlgo = &Mlkem768{}

func (k *Mlkem768) DeriveSeed(s proto.SecretSeed32) (proto.KemSeed, error) {
	mlkem, err := DeriveMlkemSeed(s)
	if err != nil {
		return nil, err
	}
	return proto.KemSeed(mlkem[:]), nil
}

func (k *Mlkem768) Decap(seed proto.KemDecapKey, ctext proto.KemCiphertext) (proto.KemSharedKey, error) {
	key, err := mlkem768.NewKeyFromSeed(seed[:])
	if err != nil {
		return nil, err
	}
	ret, err := mlkem768.Decapsulate(key, ctext[:])
	if err != nil {
		return nil, err
	}
	return proto.KemSharedKey(ret), nil
}

func (k *Mlkem768) Encap(key proto.KemEncapKey) (proto.KemCiphertext, proto.KemSharedKey, error) {
	typ, err := key.GetT()
	if err != nil {
		return nil, nil, err
	}
	if typ != proto.KEMType_Mlkem768 {
		return nil, nil, KeyMismatchError{}
	}
	raw := key.Mlkem768()
	ctext, shared, err := mlkem768.Encapsulate(raw)
	if err != nil {
		return nil, nil, err
	}
	return proto.KemCiphertext(ctext), proto.KemSharedKey(shared), nil
}

func (k *Mlkem768) GenKey() (*proto.KemEncapKey, proto.KemDecapKey, error) {
	decap, err := mlkem768.GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	encap := decap.EncapsulationKey()
	eRet := proto.NewKemEncapKeyWithMlkem768(encap)
	return &eRet, proto.KemDecapKey(decap.Bytes()), nil
}

func (k *Mlkem768) KeyFromSeed(seed proto.KemSeed) (*proto.KemEncapKey, proto.KemDecapKey, error) {
	if len(seed) != k.SeedSize() {
		return nil, nil, InternalError("bad seed size for MLKEM-768 (should be 64 bytes)")
	}
	decap, err := mlkem768.NewKeyFromSeed(seed[:])
	if err != nil {
		return nil, nil, err
	}

	// it turns out the exported key is exactly the same as the seed,
	// so just assert that this is the case.
	decapExport := decap.Bytes()
	if !bytes.Equal(seed, decapExport) {
		return nil, nil, InternalError("decap key seed mismatch")
	}

	encap := decap.EncapsulationKey()
	eRet := proto.NewKemEncapKeyWithMlkem768(encap)
	return &eRet, proto.KemDecapKey(decapExport), nil
}

func (k *Mlkem768) Type() proto.KEMType {
	return proto.KEMType_Mlkem768
}

func (k *Mlkem768) SeedSize() int {
	return mlkem768.SeedSize
}

func kemGenKeysFromSeed(algo KEM, s proto.SecretSeed32) (*proto.KemEncapKey, proto.KemDecapKey, error) {
	seed, err := algo.DeriveSeed(s)
	if err != nil {
		return nil, nil, err
	}
	return algo.KeyFromSeed(seed)
}

func GenPQKemKeysFromSeed(s proto.SecretSeed32) (*proto.KemEncapKey, proto.KemDecapKey, error) {
	return kemGenKeysFromSeed(PQKemAlgo, s)
}
