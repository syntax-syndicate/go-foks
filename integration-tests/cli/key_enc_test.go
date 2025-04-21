// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cli

import (
	"testing"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/stretchr/testify/require"
)

func TestNoiseFileKeyEnc(t *testing.T) {
	testFileKeyEnc(t, "noise")
}

func TestMacOSFileKeyEnc(t *testing.T) {
	if !libclient.HasMacOSKeychain {
		t.Skip()
		return
	}
	testFileKeyEnc(t, "macos")
}

func testFileKeyEnc(t *testing.T, mode string) {
	agent := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: mode,
	})
	agent.runAgent(t)
	defer agent.stop(t)

	var signupUI mockSignupUI
	signupUI.useDeviceKey = true
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI,
	}
	hook := func(m libclient.MetaContext) error {
		m.G().SetUIs(uis)
		return nil
	}
	agent.runCmd(t, hook, "--simple-ui", "signup")
	tmp := getActiveUser(t, agent)
	userA := tmp
	require.True(t, userIsUnlocked(*tmp))

	merklePoke(t)

	agent.stop(t)
	agent.runAgent(t)
	tmp = getActiveUser(t, agent)
	require.True(t, tmp.Info.Fqu.Eq(userA.Info.Fqu))
	require.True(t, userIsUnlocked(*tmp))

	if mode == "macos" {
		agent.runCmd(t, hook, "test", "delete-macos-keychain-item")
	}
}

func randomPassphrase() proto.Passphrase {
	return proto.Passphrase(core.RandomBase62String(8))
}

type mockPassphraseUI struct {
	passphrase proto.Passphrase
	fail       bool
}

func (p *mockPassphraseUI) GetPassphrase(
	m libclient.MetaContext,
	uc proto.UserInfo,
	flags libclient.GetPassphraseFlags,
) (
	*proto.Passphrase,
	error,
) {
	ret := p.passphrase
	if p.fail {
		ret += "X"
	}
	return &ret, nil
}

func TestSignupPassphrase(t *testing.T) {
	agent := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	agent.runAgent(t)
	defer agent.stop(t)

	pp := randomPassphrase()
	var signupUI mockSignupUI
	signupUI.useDeviceKey = true
	signupUI.passphrase = pp
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI,
	}
	agent.runCmdWithUIs(t, uis, "--simple-ui", "signup")

	tmp := getActiveUser(t, agent)
	userA := tmp
	require.True(t, userIsUnlocked(*tmp))
	merklePoke(t)

	agent.stop(t)
	agent.runAgent(t)
	tmp = getActiveUser(t, agent)
	require.True(t, tmp.Info.Fqu.Eq(userA.Info.Fqu))
	require.False(t, userIsUnlocked(*tmp))
	require.IsType(t, core.PassphraseLockedError{}, core.StatusToError(tmp.LockStatus))

	mpui := &mockPassphraseUI{
		passphrase: pp,
		fail:       true,
	}
	uis = libclient.UIs{
		Passphrase: mpui,
		Terminal:   &terminalUI,
	}
	err := agent.runCmdErrWithUIs(uis, "passphrase", "unlock")
	require.Error(t, err)
	require.IsType(t, core.BadPassphraseError{}, err)

	mpui.fail = false
	err = agent.runCmdErrWithUIs(uis, "passphrase", "unlock")
	require.NoError(t, err)

	tmp = getActiveUser(t, agent)
	require.True(t, tmp.Info.Fqu.Eq(userA.Info.Fqu))
	require.True(t, userIsUnlocked(*tmp))
	require.Nil(t, core.StatusToError(tmp.LockStatus))
}

func verifyPassphraseLockedThenUnlock(
	t *testing.T,
	agent *testAgent,
	user *proto.UserContext,
	pp proto.Passphrase,
	expPpGen proto.PassphraseGeneration,
) {
	// verify that the user account is locked with a passphrase
	tmp := getActiveUser(t, agent)
	require.True(t, tmp.Info.Fqu.Eq(user.Info.Fqu))
	require.False(t, userIsUnlocked(*tmp))
	require.IsType(t, core.PassphraseLockedError{}, core.StatusToError(tmp.LockStatus))

	var terminalUI terminalUI
	mpui := mockPassphraseUI{
		passphrase: pp,
	}
	uis := libclient.UIs{
		Passphrase: &mpui,
		Terminal:   &terminalUI,
	}

	// verify that bad passowrd fails as expected
	mpui.fail = true
	err := agent.runCmdErrWithUIs(uis, "passphrase", "unlock")
	require.Error(t, err)
	require.IsType(t, core.BadPassphraseError{}, err)

	// verify that unlock works
	mpui.fail = false
	agent.runCmdWithUIs(t, uis, "passphrase", "unlock")
	tmp = getActiveUser(t, agent)
	require.True(t, tmp.Info.Fqu.Eq(user.Info.Fqu))
	require.True(t, userIsUnlocked(*tmp))
	require.Nil(t, core.StatusToError(tmp.LockStatus))

	var info lcl.StoredSecretKeyBundle
	agent.runCmdToJSON(t, &info, "skm", "info")
	typ, err := info.GetT()
	require.NoError(t, err)
	require.Equal(t, proto.SecretKeyStorageType_ENC_PASSPHRASE, typ)
	require.Equal(t, proto.StretchVersion_TEST, info.EncPassphrase().StretchVersion)
	require.Equal(t, expPpGen, info.EncPassphrase().Ppgen)
}

func TestSignupPassphraseModeNoPassphrase(t *testing.T) {
	agent := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	agent.runAgent(t)
	defer agent.stop(t)

	// Even though we are in passphrase mode, we don't set a passphrase.
	// Cancel out of there via "empty" input
	pp := proto.Passphrase("")
	var signupUI mockSignupUI
	signupUI.useDeviceKey = true
	signupUI.passphrase = pp
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI,
	}
	agent.runCmdWithUIs(t, uis, "--simple-ui", "signup")

	tmp := getActiveUser(t, agent)
	userA := tmp
	require.True(t, userIsUnlocked(*tmp))
	require.Nil(t, core.StatusToError(tmp.LockStatus))

	stopper := runMerkleActivePoker(t)
	defer stopper()

	// Now we go ahead and set a passphrase.
	pp = randomPassphrase()
	ppui := &mockPassphraseUI{
		passphrase: pp,
	}
	uis.Passphrase = ppui
	agent.runCmdWithUIs(t, uis, "passphrase", "set")
	merklePoke(t)

	// reset the agent
	agent.stop(t)
	agent.runAgent(t)

	verifyPassphraseLockedThenUnlock(t, agent, userA, pp, 1)

}

func TestSignupPassphraseNoTestingFailStretch(t *testing.T) {
	agent := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
		noTestingFlag:            true,
	})
	agent.runAgent(t)
	defer agent.stop(t)

	pp := randomPassphrase()
	var signupUI mockSignupUI
	signupUI.useDeviceKey = true
	signupUI.passphrase = pp
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI,
	}
	err := agent.runCmdErrWithUIs(uis, "--simple-ui", "signup")
	require.Error(t, err)
	require.Equal(t, core.TestingOnlyError{}, err)
}

func TestPassphraseSetViaKex(t *testing.T) {

	x := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	x.runAgent(t)
	defer x.stop(t)

	y := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	y.runAgent(t)
	defer y.stop(t)

	pp := core.RandomPassphrase()

	runProvisionOnAgents(
		t,
		provOpts{
			enterHespOnX: true,
			x:            x,
			y:            y,
			signupHook: func(x *mockSignupUI) *mockSignupUI {
				x.passphrase = pp
				return x
			},
		},
	)

	var info lcl.StoredSecretKeyBundle
	y.runCmdToJSON(t, &info, "skm", "info")
	typ, err := info.GetT()
	require.NoError(t, err)
	require.Equal(t, proto.SecretKeyStorageType_ENC_PASSPHRASE, typ)
	require.Equal(t, proto.StretchVersion_TEST, info.EncPassphrase().StretchVersion)
	require.Equal(t, proto.PassphraseGeneration(1), info.EncPassphrase().Ppgen)
	userY := getActiveUser(t, y)

	y.stop(t)
	y.runAgent(t)

	verifyPassphraseLockedThenUnlock(t, y, userY, pp, 1)
}

func TestPassphraseSetViaYubiProvision(t *testing.T) {
	agent := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	agent.runAgent(t)
	defer agent.stop(t)
	var signupUI mockSignupUI
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI,
	}
	agent.runCmdWithUIs(t, uis, "--simple-ui", "signup")
	tmp := getActiveUser(t, agent)
	userA := tmp
	require.True(t, userIsUnlocked(*tmp))
	merklePoke(t)

	stopper := runMerkleActivePoker(t)
	defer stopper()

	pp := core.RandomPassphrase()
	mpui := &mockPassphraseUI{
		passphrase: pp,
	}
	uis = libclient.UIs{
		Passphrase: mpui,
		Terminal:   &terminalUI,
	}

	agent.runCmdWithUIs(t, uis, "passphrase", "set")

	// ensure that we are on a Yubi device
	var klres lcl.KeyListRes
	agent.runCmdToJSON(t, &klres, "key", "ls", "--json")
	require.Equal(t, 1, len(klres.CurrUserAllKeys))
	require.Equal(t, proto.DeviceType_YubiKey, klres.CurrUserAllKeys[0].Di.Dn.Label.DeviceType)

	uis.Passphrase = nil

	// now make the agent a bonafide FOKS device
	agent.runCmdWithUIs(t, uis, "key", "dev", "perm", "--name", "zodobomb 3.14+")
	merklePoke(t)

	// Assert that we switch over the device in a self-provision to the computer device
	agent.runCmdToJSON(t, &klres, "key", "ls", "--json")
	require.Equal(t, 2, len(klres.CurrUserAllKeys))
	for _, x := range klres.CurrUserAllKeys {
		if x.Active {
			require.Equal(t, proto.DeviceType_Computer, x.Di.Dn.Label.DeviceType)
		}
	}

	agent.stop(t)
	agent.runAgent(t)

	// After stopping and starting the agent, we should be able to unlock our dev
	// key with whatever we got sent over.
	verifyPassphraseLockedThenUnlock(t, agent, userA, pp, 1)
}

func TestPasssphraseLockAndUnlock(t *testing.T) {
	x := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	x.runAgent(t)
	defer x.stop(t)
	pp := randomPassphrase()
	var signupUI mockSignupUI
	signupUI.useDeviceKey = true
	signupUI.passphrase = pp
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI,
	}
	x.runCmdWithUIs(t, uis, "--simple-ui", "signup")
	stopper := runMerkleActivePoker(t)
	defer stopper()

	tmp := getActiveUser(t, x)
	x.runCmd(t, nil, "key", "lock")
	verifyPassphraseLockedThenUnlock(t, x, tmp, pp, 1)
}
func TestPassphraseLockAndUnlockOnProvisionedDevice(t *testing.T) {

	x := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	x.runAgent(t)
	defer x.stop(t)

	y := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	y.runAgent(t)
	defer y.stop(t)

	stopper := runMerkleActivePoker(t)
	defer stopper()

	pp := core.RandomPassphrase()

	runProvisionOnAgents(
		t,
		provOpts{
			enterHespOnX: true,
			x:            x,
			y:            y,
			signupHook: func(x *mockSignupUI) *mockSignupUI {
				x.passphrase = pp
				return x
			},
		},
	)
	userY := getActiveUser(t, y)
	x.runCmd(t, nil, "key", "lock")
	verifyPassphraseLockedThenUnlock(t, x, userY, pp, 1)
}

func TestPassphraseChangeOnDeviceAUnlockOnDeviceB(t *testing.T) {

	x := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	x.runAgent(t)
	defer x.stop(t)

	y := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	y.runAgent(t)
	defer y.stop(t)

	stopper := runMerkleActivePoker(t)
	defer stopper()

	pp := core.RandomPassphrase()

	runProvisionOnAgents(
		t,
		provOpts{
			enterHespOnX: true,
			x:            x,
			y:            y,
			signupHook: func(x *mockSignupUI) *mockSignupUI {
				x.passphrase = pp
				return x
			},
		},
	)
	userX := getActiveUser(t, x)

	pp1 := core.RandomPassphrase()
	var terminalUI terminalUI
	mpui := &mockPassphraseUI{
		passphrase: pp1,
	}
	uis := libclient.UIs{
		Passphrase: mpui,
		Terminal:   &terminalUI,
	}
	x.runCmdWithUIs(t, uis, "passphrase", "change")
	x.stop(t)
	x.runAgent(t)
	verifyPassphraseLockedThenUnlock(t, x, userX, pp1, 2)

	y.stop(t)
	y.runAgent(t)
	verifyPassphraseLockedThenUnlock(t, y, userX, pp1, 2)

}

func TestDisblePassphraseUseNoise(t *testing.T) {
	x := newTestAgentWithOpts(t, agentOpts{
		defaultKeyEncryptionMode: "passphrase",
	})
	x.runAgent(t)
	defer x.stop(t)
	pp := core.RandomPassphrase()
	var signupUI mockSignupUI
	signupUI.useDeviceKey = true
	signupUI.passphrase = pp
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI,
	}
	x.runCmdWithUIs(t, uis, "--simple-ui", "signup")
	stopper := runMerkleActivePoker(t)
	defer stopper()

	userX := getActiveUser(t, x)

	x.stop(t)
	x.runAgent(t)
	verifyPassphraseLockedThenUnlock(t, x, userX, pp, 1)

	x.runCmd(t, nil, "skm", "set-mode", "noise")

	var info lcl.StoredSecretKeyBundle
	x.runCmdToJSON(t, &info, "skm", "info")
	typ, err := info.GetT()
	require.NoError(t, err)
	require.Equal(t, proto.SecretKeyStorageType_ENC_NOISE_FILE, typ)

	x.stop(t)
	x.runAgent(t)

	tmp := getActiveUser(t, x)
	require.True(t, tmp.Info.Fqu.Eq(tmp.Info.Fqu))
	require.True(t, userIsUnlocked(*tmp))

	x.runCmd(t, nil, "skm", "set-mode", "passphrase")
	x.stop(t)
	x.runAgent(t)
	verifyPassphraseLockedThenUnlock(t, x, userX, pp, 1)

	x.runCmd(t, nil, "skm", "set-mode", "none")
	x.stop(t)
	x.runAgent(t)

	tmp = getActiveUser(t, x)
	require.True(t, tmp.Info.Fqu.Eq(tmp.Info.Fqu))
	require.True(t, userIsUnlocked(*tmp))

	x.runCmdToJSON(t, &info, "skm", "info")
	typ, err = info.GetT()
	require.NoError(t, err)
	require.Equal(t, proto.SecretKeyStorageType_PLAINTEXT, typ)
}

func TestProvisionAfterPassphraseRemoval(t *testing.T) {
	x := newTestAgentWithOpts(t, agentOpts{})
	x.runAgent(t)
	defer x.stop(t)

	var signupUI mockSignupUI
	signupUI.useDeviceKey = true
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI,
	}
	x.runCmdWithUIs(t, uis, "--simple-ui", "signup")
	stopper := runMerkleActivePoker(t)
	defer stopper()

	pp := core.RandomPassphrase()
	mpui := &mockPassphraseUI{
		passphrase: pp,
	}
	uis.Passphrase = mpui
	x.runCmdWithUIs(t, uis, "passphrase", "set")

	x.runCmd(t, nil, "skm", "set-mode", "none")

	y := newTestAgentWithOpts(t, agentOpts{})
	y.runAgent(t)
	defer y.stop(t)

	runProvisionOnAgents(t, provOpts{x: x, y: y})
}
