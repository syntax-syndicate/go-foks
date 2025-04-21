// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"context"
	"testing"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/client/libyubi"
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/stretchr/testify/require"
)

var _ libclient.LoobpackKexInterfacer = (*testKex)(nil)

func TestYubiAddYubi(t *testing.T) {
	tew := testEnvBeta(t)
	u := tew.NewTestUserFakeRoot(t)

	ctx := context.Background()
	ucli := tew.userCli(t, u)
	kexEng := newTestKex(u, ucli, nil)
	role := proto.OwnerRole
	host := u.host
	yubiDisp, err := libyubi.AllocDispatchTest()
	require.NoError(t, err)
	newKey := yubiDisp.NextTestKey(ctx, t, role, host)
	yinf, err := newKey.ExportToYubiKeyInfo(ctx)
	require.NoError(t, err)

	subkey, subkeyBox, err := core.MakeSubkey(newKey, u.host)
	require.NoError(t, err)

	dn := proto.DeviceName("yubi-3")
	dnn, err := core.NormalizeDeviceName(dn)
	require.NoError(t, err)
	dln := proto.DeviceLabelAndName{
		Name: dn,
		Label: proto.DeviceLabel{
			DeviceType: proto.DeviceType_YubiKey,
			Name:       dnn,
			Serial:     proto.FirstDeviceSerial,
		},
	}
	tok, err := core.NewPermissionToken()
	require.NoError(t, err)

	mu := tew.NewClientMetaContext(t, u)
	loopbackKex := libclient.NewLoopbackKexRunner(
		mu.G(),
		u.devices[0],
		newKey,
		u.FQUser(),
		role,
		dln,
		kexEng,
		tok,
	).WithSubkey(
		subkey,
		subkeyBox,
	).WithPQHint(
		&yinf.PqKey,
	)

	err = loopbackKex.Run(ctx)
	require.NoError(t, err)

	tew.DirectMerklePoke(t)

	lures1, err := libclient.LoadUser(mu,
		libclient.LoadUserArg{
			Uid:      u.uid,
			LoadMode: libclient.LoadModeSelf,
		},
	)
	require.NoError(t, err)
	require.Equal(t, 2, len(lures1.Prot().Devices))
}
