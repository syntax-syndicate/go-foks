// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

//go:build darwin

package libclient

import (
	"context"
	"testing"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/stretchr/testify/require"
)

func TestMacKeychain(t *testing.T) {

	srv := newPassphraseServerMock()
	u, seed, _, did := srv.makeNewUser(t)
	role := proto.OwnerRole

	tc := &testClient{
		skmm: NewSecretKeyMaterialManager(u, role),
		ss:   newTestSecretStore(),
		pm: &PassphraseManager{
			user: u,
		},
	}
	tc.skmm.isTest = true

	ctx := context.Background()
	err := tc.ss.LoadOrCreate(ctx)
	require.NoError(t, err)
	tok, err := core.NewPermissionToken()
	require.NoError(t, err)

	err = tc.ss.Put(
		lcl.LabeledSecretKeyBundle{
			Fqur: proto.FQUserAndRole{
				Fqu:  u,
				Role: role,
			},
			KeyID: did,
			Bundle: lcl.NewStoredSecretKeyBundleWithPlaintext(
				lcl.NewSecretKeyBundleWithV1(
					seed,
				),
			),
			SelfTok: tok,
		},
	)
	require.NoError(t, err)

	err = tc.ss.Save(ctx)
	require.NoError(t, err)
	tc.ss.clearForTest(t)
	err = tc.ss.Load(ctx)
	require.NoError(t, err)

	err = tc.skmm.Load(ctx, tc.ss, SecretStoreGetOpts{NoProvisional: true})
	require.NoError(t, err)
	err = tc.skmm.ReadySeed(ctx)
	require.NoError(t, err)

	err = tc.skmm.EncryptForMacOS(ctx, tc.ss)
	require.NoError(t, err)
	tc.skmm.Clear()

	err = tc.skmm.Load(ctx, tc.ss, SecretStoreGetOpts{NoProvisional: true})
	require.NoError(t, err)
	err = tc.skmm.ReadySeed(ctx)
	require.NoError(t, err)
	require.Equal(t, seed, *tc.skmm.seed)

	err = tc.skmm.UnlockToPlaintext(ctx, tc.ss)
	require.NoError(t, err)

	tc.skmm.Clear()

	err = tc.skmm.Load(ctx, tc.ss, SecretStoreGetOpts{NoProvisional: true})
	require.NoError(t, err)
	err = tc.skmm.ReadySeed(ctx)
	require.NoError(t, err)
	require.Equal(t, seed, *tc.skmm.seed)

	err = tc.skmm.Delete(ctx, tc.ss, did)
	require.NoError(t, err)
}

func TestMacKeychainDelete(t *testing.T) {

	srv := newPassphraseServerMock()
	u, seed, _, did := srv.makeNewUser(t)
	role := proto.OwnerRole

	tc := &testClient{
		skmm: NewSecretKeyMaterialManager(u, role),
		ss:   newTestSecretStore(),
		pm: &PassphraseManager{
			user: u,
		},
	}
	tc.skmm.isTest = true

	ctx := context.Background()
	err := tc.ss.LoadOrCreate(ctx)
	require.NoError(t, err)
	liid, err := tc.ss.LocalInstanceID()
	require.NoError(t, err)

	bun, err := EncryptSeedWithMacOSKeychain(
		ctx,
		seed,
		*liid,
		u,
		role,
		did,
		EncryptSeedMacOSOpts{
			IsTest: true,
			Vers:   EncryptSeedMacOSVersion2,
		},
	)
	require.NoError(t, err)

	tok, err := core.NewPermissionToken()
	require.NoError(t, err)

	err = tc.ss.Put(
		lcl.LabeledSecretKeyBundle{
			Fqur: proto.FQUserAndRole{
				Fqu:  u,
				Role: role,
			},
			KeyID:   did,
			Bundle:  *bun,
			SelfTok: tok,
		},
	)
	require.NoError(t, err)
	err = tc.ss.Save(ctx)
	require.NoError(t, err)

	tc.ss.clearForTest(t)
	err = tc.ss.Load(ctx)
	require.NoError(t, err)

	err = tc.skmm.Load(ctx, tc.ss, SecretStoreGetOpts{NoProvisional: true})
	require.NoError(t, err)
	err = tc.skmm.ReadySeed(ctx)
	require.NoError(t, err)

	typ, err := tc.skmm.bun.GetT()
	require.NoError(t, err)
	require.Equal(t, proto.SecretKeyStorageType_ENC_MACOS_KEYCHAIN, typ)
	params := tc.skmm.bun.EncMacosKeychain()
	err = tc.skmm.unlockMacOS(ctx, params)
	require.NoError(t, err)

	err = tc.skmm.Delete(ctx, tc.ss, did)
	require.NoError(t, err)

	err = tc.skmm.unlockMacOS(ctx, params)
	require.Error(t, err)
	require.Equal(t, core.MacOSKeychainError("not found"), err)
}
