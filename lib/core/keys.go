// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/x509"

	proto "github.com/foks-proj/go-foks/proto/lib"
)

func DeriveKey32(s proto.SecretSeed32, obj proto.KeyDerivation) (*proto.HMAC, error) {
	key := proto.HMACKey(s)
	tmp, err := Hmac(&obj, &key)
	if err != nil {
		return nil, err
	}
	return tmp, nil
}

func GenericDeriveKey32(s proto.SecretSeed32, obj CryptoPayloader) (*proto.SecretSeed32, error) {
	key := proto.HMACKey(s)
	tmp, err := Hmac(obj, &key)
	if err != nil {
		return nil, err
	}
	ret := (*proto.SecretSeed32)(tmp)
	return ret, nil
}

func DeriveSecretBoxKey(s proto.SecretSeed32) (*proto.SecretBoxKey, error) {
	tmp, err := DeriveKey32(s, proto.NewKeyDerivationDefault(proto.KeyDerivationType_SecretBoxKey))
	if err != nil {
		return nil, err
	}
	ret := proto.SecretBoxKey(*tmp)
	return &ret, nil
}

func DeriveMlkemSeed(s proto.SecretSeed32) (*proto.MlkemSeed, error) {
	var ret proto.MlkemSeed
	var offset int
	for i := 0; i < 2; i++ {
		key, err := DeriveKey32(s, proto.NewKeyDerivationWithMlkem(uint64(i)))
		if err != nil {
			return nil, err
		}
		copy(ret[offset:], key[:])
		offset += len(key)
	}
	return &ret, nil
}

func DeriveAppKey(s proto.SecretSeed32) (*proto.SecretSeed32, error) {
	tmp, err := DeriveKey32(s, proto.NewKeyDerivationDefault(proto.KeyDerivationType_AppKey))
	if err != nil {
		return nil, err
	}
	ret := (*proto.SecretSeed32)(tmp)
	return ret, nil
}

func DeviceSigningSecretKey(s proto.SecretSeed32) (*proto.Ed25519SecretKey, error) {
	tmp, err := DeriveKey32(s, proto.NewKeyDerivationDefault(proto.KeyDerivationType_Signing))
	if err != nil {
		return nil, err
	}
	ret := proto.Ed25519SecretKey(*tmp)
	return &ret, nil
}

func DeviceDHSecretKey(s proto.SecretSeed32) (*proto.Curve25519SecretKey, error) {
	tmp, err := DeriveKey32(s, proto.NewKeyDerivationDefault(proto.KeyDerivationType_DH))
	if err != nil {
		return nil, err
	}
	ret := proto.Curve25519SecretKey(*tmp)
	return &ret, nil
}

func TreeLocationRFKey(s proto.SecretSeed32) (*proto.HMACKey, error) {
	k, err := DeriveKey32(s, proto.NewKeyDerivationDefault(proto.KeyDerivationType_TreeLocationRF))
	if err != nil {
		return nil, err
	}
	tmp := proto.HMACKey(*k)
	return &tmp, nil
}

type DevicePublicKeyer interface {
	ExportToDB() []byte
	ExportToPublicKey() (crypto.PublicKey, error)
	Type() proto.EntityType
}

var _ DevicePublicKeyer = proto.DeviceID{}
var _ DevicePublicKeyer = proto.EntityID{}

func DPKEq(k1, k2 DevicePublicKeyer) bool {
	return k1.Type() == k2.Type() && hmac.Equal(k1.ExportToDB(), k2.ExportToDB())
}

func ImportDeviceIDFromCryptoPublicKey(k crypto.PublicKey) (proto.EntityID, error) {
	ret, err := proto.EntityType_Device.ImportFromPublicKey(k)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func SubchainTreeLocation(s proto.TreeLocation, typ proto.ChainType) (*proto.TreeLocation, error) {
	key := proto.HMACKey(s)
	obj := proto.NewChainLocationDerivationDefault(typ)
	tmp, err := Hmac(&obj, &key)
	if err != nil {
		return nil, err
	}
	ret := proto.TreeLocation(*tmp)
	return &ret, nil
}

func YubiIDtoYubiPQKeyID(
	i proto.YubiID,
) (
	*proto.YubiPQKeyID,
	error,
) {
	var ret proto.YubiPQKeyID
	ckey := i.CompressedPublicKey()
	err := PrefixedHashInto(&ckey, ret[:])
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func ComputeYubiPQKeyID(
	pk *ecdsa.PublicKey,
) (
	*proto.YubiPQKeyID,
	error,
) {
	var ret proto.YubiPQKeyID
	pkc := proto.ExportECDSAPublic(pk)
	err := PrefixedHashInto(&pkc, ret[:])
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

// ComputePKIKeyID is used for keys that were generated somewhere else, like in the autocert/ACME library.
// We still want to hash to get a key ID (as we do X509CertIDs), but do it via the Goland Marshal library,
// so we don't have to open up the key specifics.
func ComputePKIXCertID(
	pub crypto.PublicKey,
) (proto.PKIXCertID, error) {
	bytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	raw := proto.PKIXKeyBytes(bytes)
	ret := make([]byte, 33)
	ret[0] = byte(proto.EntityType_PKIXCert)
	err = PrefixedHashInto(&raw, ret[1:])
	if err != nil {
		return nil, err
	}
	return proto.PKIXCertID(ret), nil
}
