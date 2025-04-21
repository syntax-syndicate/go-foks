// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"strings"

	lcl "github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type BackupKey struct {
	name  []string
	seed  proto.BackupSeed
	words []string
}

// Secret seed:
// == 8*11+7*13 = 179 bits of entropy
// Note that it's 8*11 and not 9*11, since the first word and number are known to the
// server as a "device key name".
//
// Total seed with public name:
// = 9*11+8*13 = 203 bits of entropy
// = 25.375 bytes
var BackupSeedHESPConfig = NewHESPConfig(9, 13)

func (k *BackupKey) FromSeed(s proto.BackupSeed) error {
	copy(k.seed[:], s[:])
	return k.fillWords()
}

func GenerateBackupSeed() (*proto.BackupSeed, error) {
	var ret proto.BackupSeed
	err := BackupSeedHESPConfig.GenerateSecret(ret[:])
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func NewBackupKey() (*BackupKey, error) {
	seed, err := GenerateBackupSeed()
	if err != nil {
		return nil, err
	}
	ret := &BackupKey{seed: *seed}
	err = ret.fillWords()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (k *BackupKey) SeedBytes() int { return 23 }

func (k *BackupKey) Import(words lcl.BackupHESP) error {

	if len(words) < 1 {
		return HESPError("empty HESP")
	}

	hesp := NewHESP(BackupSeedHESPConfig)
	err := hesp.FromTokens(words[:])
	if err != nil {
		return err
	}
	if len(hesp.raw) != BackupSeedHESPConfig.TotalBytes() {
		return HESPError("wrong length for HESP backup seed (1)")
	}
	if len(hesp.raw) != len(k.seed) {
		return HESPError("wrong length for HESP backup seed (2)")
	}
	k.name = words[0:2]
	k.words = words
	copy(k.seed[:], hesp.raw)
	return nil
}

func (k *BackupKey) Name() proto.DeviceName {
	return proto.DeviceName(strings.Join(k.name, " "))
}

func (k *BackupKey) SecretSeed32(out *proto.SecretSeed32) error {
	return PrefixedHashInto(&k.seed, (*out)[:])
}

func (k *BackupKey) fillWords() error {
	hesp := NewHESP(BackupSeedHESPConfig)
	err := hesp.Import(k.seed[:])
	if err != nil {
		return err
	}
	k.words = hesp.Export()
	k.name = k.words[:2]
	return nil
}

func (k *BackupKey) Export() (lcl.BackupHESP, error) {
	err := k.fillWords()
	if err != nil {
		return nil, err
	}
	return k.words, nil
}

func (k *BackupKey) KeySuite(
	role proto.Role,
	hid proto.HostID,
) (
	*PrivateSuite25519,
	error,
) {
	var ss proto.SecretSeed32
	err := k.SecretSeed32(&ss)
	if err != nil {
		return nil, err
	}
	return NewPrivateSuite25519(proto.EntityType_BackupKey, role, ss, hid)
}

func (k *BackupKey) DeviceLabelAndName() (
	*proto.DeviceLabelAndName,
	error,
) {
	nm := k.Name()
	nnm, err := NormalizeDeviceName(nm)
	if err != nil {
		return nil, err
	}
	ret := proto.DeviceLabelAndName{
		Label: proto.DeviceLabel{
			DeviceType: proto.DeviceType_Backup,
			Name:       nnm,
			Serial:     proto.FirstDeviceSerial,
		},
		Nv:   proto.NormalizationVersion_V0,
		Name: nm,
	}
	return &ret, nil
}

func ValidateBackupHESP(s lcl.BackupHESPString) error {
	var tmp BackupKey
	return tmp.Import(s.Split())
}
