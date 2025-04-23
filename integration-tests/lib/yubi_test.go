// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"context"
	"testing"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/client/libyubi"
	"github.com/foks-proj/go-foks/integration-tests/common"
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
	"github.com/foks-proj/go-foks/server/shared"
	"github.com/stretchr/testify/require"
)

func (u *TestUser) SignupYubi(ctx context.Context, t *testing.T, cli *rem.RegClient, d *libyubi.Dispatch) {
	m := shared.NewMetaContext(ctx, G)
	opts := &TestUserOpts{
		KeyConstructor: func(role proto.Role, host proto.HostID) (core.PrivateSuiter, error) {
			return d.NextTestKey(ctx, t, role, host), nil
		},
	}
	u.SignupWithOpts(t, m, cli, opts)
}

func GenerateNewTestUserYubi(t *testing.T, d *libyubi.Dispatch) *TestUser {
	ctx := context.Background()
	cli, closeFn, err := newRegClient(ctx)
	require.NoError(t, err)
	defer closeFn()
	testUser := NewTestUser(t)
	testUser.SignupYubi(ctx, t, cli, d)
	testUser.GetCert(ctx, t, cli)
	return testUser
}

func GenerateNewTestUserYubiWithRegCli(t *testing.T, te *common.TestEnv, cli *rem.RegClient) *TestUser {
	ctx := context.Background()
	testUser := NewTestUser(t)
	testUser.SignupYubiWithRegCli(t, ctx, te, cli)
	return testUser
}

func allocYubiDispatch(t *testing.T) *libyubi.Dispatch {
	d, err := libyubi.AllocDispatchTest()
	require.NoError(t, err)
	return d
}

func TestSignupYubi(t *testing.T) {
	u := GenerateNewTestUserYubi(t, allocYubiDispatch(t))
	defer u.Cleanup()
	ctx := context.Background()
	crt := u.ClientCertRobust(ctx, t)
	ucli, userCloseFn, err := newUserClient(ctx, crt)
	defer userCloseFn()
	require.NoError(t, err)
	_, err = ucli.Ping(ctx)
	require.NoError(t, err)
	subkey := u.GetSubkey(t, ctx)
	require.NotNil(t, subkey)
}

func TestSignupYubiPingWithSubkeyAuth(t *testing.T) {
	u := GenerateNewTestUserYubi(t, allocYubiDispatch(t))
	defer u.Cleanup()
	ctx := context.Background()
	crt := u.ClientSubkeyCertRobust(ctx, t)
	ucli, userCloseFn, err := newUserClient(ctx, crt)
	defer userCloseFn()
	require.NoError(t, err)
	_, err = ucli.Ping(ctx)
	require.NoError(t, err)
}

func TestSignupYubiPingWithSubkeyAuthBadUser(t *testing.T) {
	disp := allocYubiDispatch(t)
	u := GenerateNewTestUserYubi(t, disp)
	v := GenerateNewTestUserYubi(t, disp)
	defer u.Cleanup()
	defer v.Cleanup()
	ctx := context.Background()
	subkey := u.GetSubkey(t, ctx)
	cli, closeFn, err := newRegClient(ctx)
	require.NoError(t, err)
	defer closeFn()
	pubkey, err := subkey.EntityPublic()
	require.NoError(t, err)
	eid := pubkey.GetEntityID()
	// U tries to get a cert as v, should fail.
	_, err = cli.GetClientCertChain(ctx, rem.GetClientCertChainArg{
		Uid: v.uid,
		Key: eid,
	})
	require.Error(t, err)
	require.IsType(t, core.AuthError{}, err)
}

func setupKexTestYubi(t *testing.T, ctx context.Context, xIsYubi bool, yIsYubi bool, disp *libyubi.Dispatch) (
	*TestUser, func(), *testKex, *testKex,
	*libclient.KexProvisioner, *libclient.KexProvisionee,
) {
	var u *TestUser
	if xIsYubi {
		u = GenerateNewTestUserYubi(t, disp)
	} else {
		u = GenerateNewTestUser(t)
	}
	require.NotNil(t, u)
	crt := u.ClientCertRobust(ctx, t)
	ucli, userCloseFn, err := newUserClient(ctx, crt)
	require.NoError(t, err)
	kexcli, kexCloseFn, err := newKexClient(ctx)
	require.NoError(t, err)

	kiX := newTestKex(u, ucli, kexcli)
	kiY := newTestKex(u, nil, kexcli)

	X := libclient.NewKexProvisioner(kiX, u.uid, u.host,
		proto.NewRoleDefault(proto.RoleType_OWNER),
		u.eldest)

	ss := core.RandomSecretSeed32()
	role := proto.NewRoleDefault(proto.RoleType_OWNER)
	var newkey core.PrivateSuiter
	if yIsYubi {
		ks := disp.NextTestKey(ctx, t, role, u.host)
		newkey = ks
	} else {
		newkey, err = core.NewPrivateSuite25519(proto.EntityType_Device, role, ss, u.host)
		require.NoError(t, err)
	}

	dln := makeDeviceLabelAndName(t, "galaxy", proto.DeviceType_Mobile)

	Y := libclient.NewKexProvisionee(kiY, kiY, dln, newkey)
	cleanup := core.Compose(userCloseFn, kexCloseFn)
	cleanup = core.Compose(cleanup, u.Cleanup)

	return u, cleanup, kiX, kiY, X, Y
}

func TestKexProvisionXYubiDriving(t *testing.T) {
	ctx := context.Background()
	_, cleanup, kiX, kiY, X, Y := setupKexTestYubi(t, ctx, true, false, allocYubiDispatch(t))
	defer cleanup()
	waitFn := runBoth(t, ctx, X, Y)

	secret := <-kiX.mySessionCh
	kiY.theirSessionCh <- secret

	waitFn()
}

// Note that the case when Y is a yubikey doesn't make sense, it's not a
// network kex provision. Rather, it's a Loopback Kex provision, which is
// tested elsewhere

func TestYubiReuse(t *testing.T) {

	disp := allocYubiDispatch(t)

	reg := func(
		kg func(role proto.Role, host proto.HostID) (core.PrivateSuiter, error),
	) error {
		ctx := context.Background()
		cli, closeFn, err := newRegClient(ctx)
		require.NoError(t, err)
		defer closeFn()
		testUser := NewTestUser(t)
		opts := &TestUserOpts{KeyConstructor: kg}
		m := shared.NewMetaContext(ctx, G)
		return testUser.SignupWithOptsAndError(t, m, cli, opts)
	}

	var k0 *libyubi.KeySuiteHybrid
	err := reg(
		func(role proto.Role, host proto.HostID) (core.PrivateSuiter, error) {
			ctx := context.Background()
			k0 = disp.NextTestKey(ctx, t, role, host)
			return k0, nil
		},
	)
	require.NoError(t, err)

	err = reg(
		func(role proto.Role, host proto.HostID) (core.PrivateSuiter, error) {
			return k0, nil
		},
	)
	require.Error(t, err)
	require.Equal(t, core.KeyInUseError{}, err)
}

func TestYubiManagementKeyPushPull(t *testing.T) {
	tew := testEnvBeta(t)
	a := tew.NewTestUserYubi(t)
	tew.DirectMerklePoke(t)
	mc := tew.NewClientMetaContextWithEracer(t, a)
	yd := mc.G().YubiDispatch()
	au := mc.G().ActiveUser()
	require.NotNil(t, au)
	yinfo := au.Info.YubiInfo
	require.NotNil(t, yinfo)
	pin := proto.YubiPIN("123412")

	var defpin proto.YubiPIN
	err := yd.SetPIN(mc.Ctx(), yinfo.Card, defpin, pin)
	require.NoError(t, err)

	mk, didSet, err := yd.SetOrGetManagementKey(mc.Ctx(), yinfo.Card, pin)
	require.NoError(t, err)
	require.True(t, didSet)
	require.NotNil(t, mk)

	err = yd.ValidatePIN(mc.Ctx(), yinfo.Card, pin, true)
	require.NoError(t, err)

	pusher := libclient.NewYMKPush(au)
	err = pusher.Run(mc)
	require.NoError(t, err)
	require.Equal(t, libclient.YMKPushOutcomeNew, pusher.Outcome())

	mk2, err := libclient.NewYMKPull(au).Recover(mc, yinfo.Card)
	require.NoError(t, err)
	require.NotNil(t, mk2)
	require.Equal(t, *mk, *mk2)

	pusher = libclient.NewYMKPush(au)
	err = pusher.Run(mc)
	require.NoError(t, err)
	require.Equal(t, libclient.YMKPushOutcomeFresh, pusher.Outcome())

	// spam the sigchain to force a PUK rotation
	rabbit := a.ProvisionNewDevice(t, a.eldest, "rabbit", proto.DeviceType_Computer, proto.OwnerRole)
	tew.DirectDoubleMerklePokeInTest(t)
	a.RevokeDevice(t, a.eldest, rabbit)
	tew.DirectMerklePoke(t)

	pusher = libclient.NewYMKPush(au)
	err = pusher.Run(mc)
	require.NoError(t, err)
	require.Equal(t, libclient.YMKPushOutcomeRefreshed, pusher.Outcome())

	var mk3 proto.YubiManagementKey
	err = core.RandomFill(mk3[:])
	require.NoError(t, err)

	// change management key to a new random management key; this clears
	// out all secrets we have stored in the card Handle, and we have to unlock
	// the card again.
	err = yd.SetManagementKey(mc.Ctx(), yinfo.Card, mk, mk3)
	require.NoError(t, err)

	// This unlocks the card and repopulates the management key into the Handle
	err = yd.ValidatePIN(mc.Ctx(), yinfo.Card, pin, true)
	require.NoError(t, err)

	pusher = libclient.NewYMKPush(au)
	err = pusher.Run(mc)
	require.NoError(t, err)
	require.Equal(t, libclient.YMKPushOutcomeRefreshed, pusher.Outcome())

	mk4, err := libclient.NewYMKPull(au).Recover(mc, yinfo.Card)
	require.NoError(t, err)
	require.NotNil(t, mk4)
	require.Equal(t, *mk4, mk3)

	var stopped bool
	for range 10 {
		err = yd.ValidatePIN(mc.Ctx(), yinfo.Card, "000000xx", true)
		require.Error(t, err)
		yerr, ok := err.(core.YubiAuthError)
		require.True(t, ok)
		if yerr.Retries == 0 {
			stopped = true
			break
		}
	}
	require.True(t, stopped)

	newPin := proto.YubiPIN("234567")
	err = libclient.RecoverYubiManagementKey(mc, yinfo.Card.Serial, newPin, "45678912", nil)
	require.NoError(t, err)

	err = yd.ValidatePIN(mc.Ctx(), yinfo.Card, newPin, true)
	require.NoError(t, err)
}
