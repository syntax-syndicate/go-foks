// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cli

import (
	"fmt"
	"testing"
	"time"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/integration-tests/common"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/stretchr/testify/require"
)

type backupUI struct {
	hesp lcl.BackupHESP
}

func (b *backupUI) CheckedServer(m libclient.MetaContext, addr proto.TCPAddr, e error) error {
	return nil
}

func (b *backupUI) GetBackupKeyHESP(m libclient.MetaContext) (lcl.BackupHESP, error) {
	return b.hesp, nil
}

func (b *backupUI) PickServer(
	m libclient.MetaContext,
	def proto.TCPAddr,
	timeout time.Duration,
) (
	*proto.TCPAddr,
	error,
) {
	return nil, nil
}

func TestBackupNew(t *testing.T) {

	bob := makeBobAndHisAgent(t)
	b := bob.agent
	defer b.stop(t)
	merklePoke(t)

	var res lcl.BackupHESP
	b.runCmdToJSON(t, &res, "backup", "new")
	fmt.Printf("%+v\n", res)

	merklePoke(t)

	c := newTestAgent(t)
	c.runAgent(t)
	defer c.stop(t)

	backupUI := backupUI{hesp: res}

	hook := func(m libclient.MetaContext) error {
		m.G().SetUIs(libclient.UIs{Backup: &backupUI})
		return nil
	}
	c.runCmd(t, hook, "backup", "load")

	var bk core.BackupKey
	err := bk.Import(res)
	require.NoError(t, err)
	ks, err := bk.KeySuite(proto.OwnerRole, proto.HostID{})
	require.NoError(t, err)
	bkid, err := ks.EntityID()
	require.NoError(t, err)

	var klres lcl.KeyListRes
	c.runCmdToJSON(t, &klres, "key", "list")
	require.Equal(t, 2, len(klres.CurrUserAllKeys))
	require.Equal(t, 1, len(klres.AllUsers))
	nm := bk.Name()

	active := func(lst []lcl.ActiveDeviceInfo) lcl.ActiveDeviceInfo {
		for _, d := range lst {
			if d.Active {
				return d
			}
		}
		t.Fatal("no active device")
		return lcl.ActiveDeviceInfo{}
	}

	ad := active(klres.CurrUserAllKeys)
	require.Equal(t, nm, ad.Di.Dn.Name)
	require.Equal(t, bkid, ad.Di.Key.Member.Id.Entity)
	require.Equal(t, proto.DeviceType_Backup, ad.Di.Dn.Label.DeviceType)

	newDevName := proto.DeviceName("dodo0")
	stopper := runMerkleActivePoker(t)
	c.runCmd(t, nil, "key", "dev", "perm", "--name", string(newDevName), "--role", "o")
	stopper()

	c.runCmdToJSON(t, &klres, "key", "list")
	require.Equal(t, 3, len(klres.CurrUserAllKeys))
	ad = active(klres.CurrUserAllKeys)
	require.Equal(t, newDevName, ad.Di.Dn.Name)
	require.Equal(t, proto.DeviceType_Computer, ad.Di.Dn.Label.DeviceType)
}

func TestNewBackupWithDeviceKey(t *testing.T) {
	defer common.DebugEntryAndExit()()
	merklePoke(t)
	env := globalTestEnv
	tvh := env.VHostInit(t, "zed-40")
	agentOpts := agentOpts{dnsAliases: []proto.Hostname{tvh.Hostname}}

	x := newTestAgentWithOpts(t, agentOpts)
	x.runAgent(t)
	defer x.stop(t)
	merklePoke(t)

	// first sign up for a user on the base host
	signupUI := &mockSignupUI{}
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   signupUI,
		Terminal: &terminalUI,
	}
	hook := func(m libclient.MetaContext) error {
		m.G().SetUIs(uis)
		return nil
	}
	x.runCmd(t, hook, "--simple-ui", "signup")

	signupUI = signupUI.withDeviceKey().withServer(tvh.ProbeAddr)
	uis.Signup = signupUI

	x.runCmd(t, hook, "--simple-ui", "signup")
	merklePoke(t)

	var res lcl.BackupHESP
	x.runCmdToJSON(t, &res, "backup", "new")
	fmt.Printf("%+v\n", res)
	merklePoke(t)

	c := newTestAgentWithOpts(t, agentOpts)
	c.runAgent(t)
	defer c.stop(t)

	backupUI := backupUI{hesp: res}

	hook = func(m libclient.MetaContext) error {
		m.G().SetUIs(libclient.UIs{Backup: &backupUI})
		return nil
	}
	c.runCmd(t, hook, "backup", "load", "--host", tvh.ProbeAddr.String())
}
