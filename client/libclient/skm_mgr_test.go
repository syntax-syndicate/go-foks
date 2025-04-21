// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	skmm *SecretKeyMaterialManager
	ss   *SecretStore
	pm   *PassphraseManager
}

func (tc *testClient) reboot() {
	tc.skmm.Clear()
	tc.pm.Logout()
}

func TestLockSecretSimpleHappy(t *testing.T) {

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

	ctx := context.Background()
	err := tc.ss.LoadOrCreate(ctx)
	require.NoError(t, err)
	tok, err := core.NewPermissionToken()
	require.NoError(t, err)

	row := lcl.LabeledSecretKeyBundle{
		Fqur: proto.FQUserAndRole{
			Fqu:  u,
			Role: role,
		},
		KeyID:   did,
		SelfTok: tok,
		Bundle: lcl.NewStoredSecretKeyBundleWithPlaintext(
			lcl.NewSecretKeyBundleWithV1(
				seed,
			),
		),
	}

	err = tc.ss.Put(row)
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

	pp := core.RandomPassphrase()

	err = tc.pm.SetPassphrase(ctx, srv, pp, nil)
	require.NoError(t, err)

	err = tc.skmm.EncryptWithPassphrase(ctx, tc.pm, tc.ss)
	require.NoError(t, err)

	tc.reboot()

	loginSequence := func(passphrase proto.Passphrase) {

		err = tc.skmm.Load(ctx, tc.ss, SecretStoreGetOpts{NoProvisional: true})
		require.NoError(t, err)
		err = tc.skmm.ReadySeed(ctx)
		require.Error(t, err)
		require.IsType(t, core.PassphraseLockedError{}, err)

		err = tc.pm.Login(ctx, srv, passphrase, tc.skmm.Salt(), nil)
		require.NoError(t, err)

		err = tc.skmm.DecryptWithPassphrase(ctx, tc.pm, tc.ss)
		require.NoError(t, err)
		err = tc.skmm.ReadySeed(ctx)
		require.NoError(t, err)
		err = tc.skmm.MaybeUpdatePassphraseEncryption(ctx, tc.pm, tc.ss, *tc.skmm.ppgen)
		require.NoError(t, err)
	}

	loginSequence(pp)

	require.Equal(t, proto.PassphraseGeneration(1), tc.skmm.bun.EncPassphrase().Ppgen)

	tc.reboot()

	// Simulate a pssphrase change on a separate computer
	pm2 := &PassphraseManager{user: u}
	err = pm2.Login(ctx, srv, pp, nil, nil)
	require.NoError(t, err)
	pp2 := core.RandomPassphrase()
	err = pm2.ChangePassphrase(ctx, srv, pp2, nil)
	require.NoError(t, err)

	loginSequence(pp2)

	tc.reboot()

	loginSequence(pp2)

	require.Equal(t, proto.PassphraseGeneration(2), tc.skmm.bun.EncPassphrase().Ppgen)
}

func flipFileAtBit(fn string, indx int) error {

	fh, err := os.OpenFile(fn, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if fh != nil {
			fh.Close()
		}
	}()

	bytePos := indx >> 3
	bitPos := indx & 7

	buf := make([]byte, 1)
	_, err = fh.ReadAt(buf, int64(bytePos))
	if err != nil {
		return err
	}
	buf[0] ^= 1 << uint(bitPos)
	_, err = fh.WriteAt(buf, int64(bytePos))
	if err != nil {
		return err
	}
	err = fh.Close()
	fh = nil
	if err != nil {
		return err
	}
	return nil
}

func TestNoiseFile(t *testing.T) {

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

	sskb, err := EncryptSeedWithNoiseFile(ctx, seed, tc.ss.Dir())
	require.NoError(t, err)
	tok, err := core.NewPermissionToken()
	require.NoError(t, err)

	row := lcl.LabeledSecretKeyBundle{
		Fqur: proto.FQUserAndRole{
			Fqu:  u,
			Role: role,
		},
		KeyID:   did,
		SelfTok: tok,
		Bundle:  *sskb,
	}

	err = tc.ss.Put(row)
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

	tc.ss.clearForTest(t)
	tc.skmm.Clear()
	err = tc.ss.Load(ctx)
	require.NoError(t, err)

	bun := tc.ss.data.Keys[0].Bundle
	typ, err := bun.GetT()
	require.NoError(t, err)
	require.Equal(t, proto.SecretKeyStorageType_ENC_NOISE_FILE, typ)
	stem := bun.EncNoiseFile().Filename
	fn := filepath.Join(tc.ss.Dir(), stem)
	err = flipFileAtBit(fn, 10043)
	require.NoError(t, err)

	err = tc.skmm.Load(ctx, tc.ss, SecretStoreGetOpts{NoProvisional: true})
	require.NoError(t, err)
	err = tc.skmm.ReadySeed(ctx)
	require.Error(t, err)
	require.IsType(t, core.DecryptionError{}, err)

	err = tc.skmm.Delete(ctx, tc.ss, did)
	require.NoError(t, err)

	_, err = os.Stat(fn)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

}
