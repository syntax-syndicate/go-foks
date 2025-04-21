// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

//go:build !darwin

package libclient

import (
	"context"

	"github.com/foks-proj/go-foks/lib/core"
	lcl "github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

var HasMacOSKeychain = false

func (s *SecretKeyMaterialManager) unlockMacOS(
	ctx context.Context,
	mac lcl.MacOSKeychainEncryptedSecretBundle,
) error {
	return core.PlatformError{}
}

func (s *SecretKeyMaterialManager) EncryptForMacOS(
	ctx context.Context,
	ss *SecretStore,
) error {
	return core.PlatformError{}
}

func (s *SecretKeyMaterialManager) ClearMacOSKeychain(
	mac lcl.MacOSKeychainEncryptedSecretBundle,
) error {
	return core.PlatformError{}
}

func EncryptSeedWithMacOSKeychain(
	ctx context.Context,
	seed proto.SecretSeed32,
	iid proto.LocalInstanceID,
	fqu proto.FQUser,
	role proto.Role,
	did proto.DeviceID,
	opts EncryptSeedMacOSOpts,
) (*lcl.StoredSecretKeyBundle, error) {
	return nil, core.PlatformError{}
}

func InitMacOSKeychainTest()    {}
func CleanupMacOSKeychainTest() {}
