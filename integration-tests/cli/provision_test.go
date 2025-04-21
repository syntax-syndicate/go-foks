// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cli

import (
	"errors"
	"testing"
	"time"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/integration-tests/common"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/stretchr/testify/require"
)

type hespAndErr struct {
	hesp *proto.KexHESP
	err  error
}

type mockDeviceAssistUI struct {
	user        proto.UserInfo
	hespInputCh chan hespAndErr
	hespOutpuCh chan proto.KexHESP
}

func newMockDeviceAssistUI() *mockDeviceAssistUI {
	return &mockDeviceAssistUI{
		hespInputCh: make(chan hespAndErr, 1),
		hespOutpuCh: make(chan proto.KexHESP, 1),
	}
}

func (m *mockDeviceAssistUI) ConfirmActiveUser(
	_ libclient.MetaContext,
	u proto.UserInfo,
) error {
	m.user = u
	return nil
}

func (m *mockDeviceAssistUI) GetKexHESP(
	_ libclient.MetaContext,
	ourHesp proto.KexHESP,
	lastErr error,
) (
	*proto.KexHESP,
	error,
) {
	if m.hespInputCh == nil || m.hespOutpuCh == nil {
		return nil, core.NotImplementedError{}
	}
	m.hespOutpuCh <- ourHesp
	he := <-m.hespInputCh
	return he.hesp, he.err
}

var _ libclient.DeviceAssistUIer = (*mockDeviceAssistUI)(nil)

func runMerkleActivePoker(t *testing.T) func() {
	activePoker := common.NewLoopUntilStop(func() {
		time.Sleep(10 * time.Millisecond)
		merklePoke(t)
	})
	go activePoker.Run()
	return func() {
		activePoker.Stop()
	}
}

func TestProvisionEnterHespOnX(t *testing.T) { testProvision(t, true) }
func TestProvisionEnterHespOnY(t *testing.T) { testProvision(t, false) }

func testProvision(t *testing.T, enterHespOnX bool) {

	x := newTestAgent(t)
	y := newTestAgent(t)
	x.runAgent(t)
	y.runAgent(t)
	runProvisionOnAgents(t, provOpts{enterHespOnX: enterHespOnX, x: x, y: y})
	x.stop(t)
	y.stop(t)
}

func TestProvisionMacOSTwoDevicesOneKeychain(t *testing.T) {
	if !libclient.HasMacOSKeychain {
		t.Skip("macOS keychain not available")
	}
	opts := agentOpts{
		defaultKeyEncryptionMode: "macos",
	}
	x := newTestAgentWithOpts(t, opts)
	y := newTestAgentWithOpts(t, opts)
	x.runAgent(t)
	y.runAgent(t)
	runProvisionOnAgents(t, provOpts{x: x, y: y})
	x.stop(t)
	y.stop(t)
}

type provOpts struct {
	x            *testAgent
	y            *testAgent
	nameY        proto.DeviceName
	enterHespOnX bool
	signupHook   func(*mockSignupUI) *mockSignupUI
	noSignup     bool
	probeAddr    *proto.TCPAddr
}

func runProvisionOnAgents(t *testing.T, opts provOpts) {
	err := runProvisionOnAgentsWithErr(t, opts)
	require.NoError(t, err)
}

func runProvisionOnAgentsWithErr(t *testing.T, opts provOpts) error {

	devX := opts.x
	devY := opts.y

	nameX := proto.DeviceName("YodoDyne A 4.1")
	nameY := opts.nameY
	if nameY == "" {
		nameY = proto.DeviceName("Pickle Violet B 0.1.3")
	}

	signupUiX := newMockSignupUI().withDeviceKey().withDeviceName(nameX)
	if opts.signupHook != nil {
		signupUiX = opts.signupHook(signupUiX)
	}
	var terminalUiX terminalUI

	makeHook := func(s *mockSignupUI) func(libclient.MetaContext) error {
		return func(m libclient.MetaContext) error {
			m.G().SetUIs(libclient.UIs{
				Signup:   s,
				Terminal: &terminalUiX,
			})
			return nil
		}
	}

	if !opts.noSignup {
		devX.runCmd(t, makeHook(signupUiX), "--simple-ui", "signup")
	}

	provUi := newMockSignupUI().
		withDeviceKey().
		withDeviceName(nameY).
		withUsername(signupUiX.username)

	if opts.probeAddr != nil {
		provUi.withServer(*opts.probeAddr)
	}

	assistUi := newMockDeviceAssistUI()

	assistHook := func(m libclient.MetaContext) error {
		m.G().SetUIs(libclient.UIs{Assist: assistUi})
		return nil
	}

	merklePoke(t)

	stopper := runMerkleActivePoker(t)

	yDone := make(chan error)
	xDone := make(chan error)

	go func() {
		err := devY.runCmdErr(makeHook(provUi), "--simple-ui", "key", "dev", "provision")
		yDone <- err
	}()
	go func() {
		err := devX.runCmdErr(assistHook, "--simple-ui", "key", "assist")
		xDone <- err
	}()

	if opts.enterHespOnX {
		hesp := <-assistUi.hespOutpuCh
		provUi.hespInputCh <- hespAndErr{hesp: &hesp}
		err := <-yDone
		if err != nil {
			return err
		}
		assistUi.hespInputCh <- hespAndErr{err: core.CanceledInputError{}}
		err = <-xDone
		if err != nil {
			return err
		}
	} else {
		hesp := <-provUi.hespOutpuCh
		assistUi.hespInputCh <- hespAndErr{hesp: &hesp}
		err := <-xDone
		if err != nil {
			return err
		}
		provUi.hespInputCh <- hespAndErr{err: core.CanceledInputError{}}
		err = <-yDone
		if err != nil {
			return err
		}
	}

	stopper()

	var st lcl.AgentStatus
	devY.runCmdToJSON(t, &st, "status")
	require.Equal(t, 1, len(st.Users))
	require.True(t, st.Users[0].Info.Active)
	require.Equal(t, 1, len(st.Users[0].Puks))
	if !opts.noSignup {
		require.Equal(t, signupUiX.username, st.Users[0].Info.Username.NameUtf8)
	}
	require.Equal(t, nameY, st.Users[0].Devname)

	return nil
}

func TestIssue5(t *testing.T) {
	if !libclient.HasMacOSKeychain {
		t.Skip("macOS keychain not available")
	}
	opts := agentOpts{
		defaultKeyEncryptionMode: "macos",
	}
	x := newTestAgentWithOpts(t, opts)
	y := newTestAgentWithOpts(t, opts)
	x.runAgent(t)
	y.runAgent(t)

	defer func() {
		x.stop(t)
		y.stop(t)
	}()

	runProvisionOnAgents(t, provOpts{x: x, y: y})
	yInfo := getActiveUser(t, y)
	yKey, err := yInfo.Key.StringErr()
	require.NoError(t, err)

	stopper := runMerkleActivePoker(t)
	defer stopper()

	// ensure we find the key in the macos keychain

	// Revoke y's key.
	x.runCmd(t, nil, "key", "revoke", yKey)

	// Trigger clean-up job
	y.runCmd(t, nil, "test", "trigger-bg-user-job")

	runProvisionOnAgents(t, provOpts{x: x, y: y})
}
func secretStoreDump(t *testing.T, x *testAgent) []lcl.LabeledSecretKeyBundle {
	var ss lcl.SecretStore
	x.runCmdToJSON(t, &ss, "test", "dump-secret-store")
	v, err := ss.GetV()
	require.NoError(t, err)
	require.Equal(t, lcl.SecretStoreVersion_V2, v)
	rows := ss.V2().Keys
	return rows
}

func TestRecoveryFromAgentCrashAfterProvision(t *testing.T) {
	defer common.DebugEntryAndExit()()

	x := newTestAgent(t)
	x.runAgent(t)
	defer x.stop(t)

	name := proto.DeviceName("device A.1")
	vh := vHost(t, 1)
	signupUi := newMockSignupUI().withDeviceName(name).withServer(vh.Addr)
	x.runCmdWithUIs(t, libclient.UIs{Signup: signupUi}, "--simple-ui", "signup")

	crashErr := errors.New("computer crahsed!")
	x.g.Testing.SetSelfProvisionCrash(crashErr)

	stopper := runMerkleActivePoker(t)
	defer stopper()

	err := x.runCmdErr(nil, "key", "dev", "perm", "--name", "device B.2")
	require.Error(t, err)
	require.Equal(t, crashErr, err)

	secretStoreDump := func() []lcl.LabeledSecretKeyBundle {
		return secretStoreDump(t, x)
	}

	ss := secretStoreDump()
	require.Equal(t, 1, len(ss))
	require.True(t, ss[0].Provisional)

	x.stop(t)

	x.runAgent(t)

	st := x.status(t)
	require.Equal(t, 1, len(st.Users))
	require.True(t, st.Users[0].Info.Active)
	require.Equal(t, 1, len(st.Users[0].Puks))

	ss = secretStoreDump()
	require.Equal(t, 1, len(ss))
	require.False(t, ss[0].Provisional)
}

func TestBrokenSelfProvisionThenRecover(t *testing.T) {
	defer common.DebugEntryAndExit()()

	x := newTestAgent(t)
	x.runAgent(t)
	defer x.stop(t)
	name := proto.DeviceName("device A.1")
	vh := vHost(t, 1)
	signupUi := newMockSignupUI().withDeviceName(name).withServer(vh.Addr)
	x.runCmdWithUIs(t, libclient.UIs{Signup: signupUi}, "--simple-ui", "signup")

	yerr := errors.New("weird yubi failure")
	x.g.Testing.SetLoopbackSignError(yerr)

	stopper := runMerkleActivePoker(t)
	defer stopper()

	err := x.runCmdErr(nil, "key", "dev", "perm", "--name", "device B.2")
	require.Error(t, err)
	require.Equal(t, yerr, err)

	// check that there is 1 key in the secret store and that it's marked
	// provisional, since provision never succeeded.

	secretStoreDump := func() []lcl.LabeledSecretKeyBundle {
		return secretStoreDump(t, x)
	}
	rows := secretStoreDump()
	require.Equal(t, 1, len(rows))
	require.True(t, rows[0].Provisional)

	x.g.Testing.SetLoopbackSignError(nil)

	x.runCmd(t, nil, "key", "dev", "perm", "--name", "device B.2")

	rows = secretStoreDump()
	require.Equal(t, 2, len(rows))
	require.True(t, rows[0].Provisional)
	require.False(t, rows[1].Provisional)

	// 2nd self-provision should fail
	err = x.runCmdErr(nil, "key", "dev", "perm", "--name", "device c.3")
	require.Error(t, err)
	require.Equal(t, core.DeviceAlreadyProvisionedError{}, err)
}

func TestProvisionFailOnSpam(t *testing.T) {

	x := newTestAgent(t)
	y := newTestAgent(t)
	x.runAgent(t)
	defer x.stop(t)
	y.runAgent(t)
	defer y.stop(t)
	runProvisionOnAgents(t, provOpts{enterHespOnX: true, x: x, y: y})

	st := x.status(t)
	require.Equal(t, 1, len(st.Users))
	name := st.Users[0].Info.Username.NameUtf8

	provUi := newMockSignupUI().withUsername(name).withDeviceKey().withDeviceName("spam crap")
	assistUi := newMockDeviceAssistUI()
	assistHook := func(m libclient.MetaContext) error {
		m.G().SetUIs(libclient.UIs{Assist: assistUi})
		return nil
	}
	provHook := func(m libclient.MetaContext) error {
		m.G().SetUIs(libclient.UIs{Signup: provUi})
		return nil
	}

	stopper := runMerkleActivePoker(t)
	defer stopper()

	yDone := make(chan error)
	xDone := make(chan error, 1)

	go func() {
		err := y.runCmdErr(provHook, "--simple-ui", "key", "dev", "provision")
		yDone <- err
	}()
	go func() {
		err := x.runCmdErr(assistHook, "--simple-ui", "key", "assist")
		xDone <- err
	}()

	hesp := <-assistUi.hespOutpuCh
	provUi.hespInputCh <- hespAndErr{hesp: &hesp}
	err := <-yDone
	require.Error(t, err)
	require.Equal(t, core.DeviceAlreadyProvisionedError{}, err)
}
