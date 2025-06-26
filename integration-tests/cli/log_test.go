package cli

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/integration-tests/common"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	"github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/server/shared"
	"github.com/keybase/clockwork"
	"github.com/stretchr/testify/require"
)

func getLogRotate(m libclient.MetaContext) *libclient.LogRotate {
	for range 10 {
		lr := m.G().LogRotate()
		if lr != nil {
			return lr
		}
		time.Sleep(10 * time.Millisecond)
	}
	panic("log rotate not initialized; waited 100msec")
}

func advancePastNextRotation(
	t *testing.T,
	m libclient.MetaContext,
	lr *libclient.LogRotate,
	st *core.ClockWrapState,
) {
	cw := m.G().ClockWrap()
	nxt := lr.NextRotation()
	err := cw.PushTo(nxt, st)
	require.NoError(t, err)
}

func TestLogRotations(t *testing.T) {
	defer common.DebugEntryAndExit()()
	agent := newTestAgent(t)
	agent.runAgent(t)
	defer agent.stop(t)

	m := libclient.NewMetaContextBackground(agent.g)

	line := 1
	spew := func() {
		m.Infow("test log", "cmd", line)
		line++
	}
	cl := clockwork.NewFakeClock()
	orig := m.G().Clock()
	m.G().SetClock(cl)
	defer m.G().SetClock(orig)

	lr := getLogRotate(m)

	// After setting the new clock, we need to poke the log rotate loop
	// to pick up the new clock.
	lr.Poke(m)

	for range 10 {
		spew()
	}

	countLines := func(b []byte) int {
		lines := 0
		for _, c := range b {
			if c == '\n' {
				lines++
			}
		}
		return lines
	}

	gunzip := func(p core.Path) []byte {
		file, err := p.ReadFile()
		require.NoError(t, err)
		var b bytes.Buffer
		_, err = b.Write(file)
		require.NoError(t, err)
		r, err := gzip.NewReader(&b)
		require.NoError(t, err)
		defer r.Close()
		data, err := io.ReadAll(r)
		require.NoError(t, err)
		return data
	}

	countRotatedLogs := func(expected int) {
		file, err := m.G().OutLogPath()
		require.NoError(t, err)
		logs, err := libclient.FindRotatedLogs(m, file)
		require.NoError(t, err)
		require.Equal(t, expected, len(logs))
		for _, log := range logs {
			data := gunzip(log.Path)
			lines := countLines(data)
			require.Greater(t, lines, 1) // need at least two lines in each rotated log
		}
	}

	readLog := func(low int, high int) {
		_ = m.G().LogSync()
		file, err := m.G().OutLogPath()
		require.NoError(t, err)
		data, err := file.ReadFile()
		require.NoError(t, err)
		lines := countLines(data)
		require.GreaterOrEqual(t, lines, low)
		require.LessOrEqual(t, lines, high)
	}

	readLog(10, 20)
	countRotatedLogs(0)

	ch := make(chan struct{})
	lr.WaitForNextRotate(ch)
	var clst core.ClockWrapState
	advancePastNextRotation(t, m, lr, &clst)

	<-ch // Wait for the log rotation to complete

	for range 3 {
		spew()
	}

	// log rotation should add one line to the new log
	readLog(4, 6)
	countRotatedLogs(1)

}

type mockLogSendUI struct {
	logs lcl.LogSendSet
	res  *lcl.LogSendRes
}

func (u *mockLogSendUI) ApproveLogs(
	m libclient.MetaContext,
	logs lcl.LogSendSet,
) error {
	u.logs = logs
	return nil
}

func (u *mockLogSendUI) ShowLogSendRes(
	m libclient.MetaContext,
	res lcl.LogSendRes,
) error {
	u.res = &res
	return nil
}

func (u *mockLogSendUI) ShowStartSend(m libclient.MetaContext) error {
	return nil
}

func (u *mockLogSendUI) ShowCompleteSend(m libclient.MetaContext, err error) error {
	return nil
}

var _ libclient.LogSendUIer = (*mockLogSendUI)(nil)

func TestLogSend(t *testing.T) {
	defer common.DebugEntryAndExit()()
	bob := makeBobAndHisAgent(t)
	agent := bob.agent
	status := agent.status(t)
	require.Equal(t, 1, len(status.Users))
	require.True(t, status.Users[0].Info.Active)
	fqu := status.Users[0].Info.Fqu
	defer agent.stop(t)

	m := libclient.NewMetaContextBackground(agent.g)
	cl := clockwork.NewFakeClock()
	orig := m.G().Clock()
	m.G().SetClock(cl)
	defer m.G().SetClock(orig)

	lr := getLogRotate(m)

	// After setting the new clock, we need to poke the log rotate loop
	// to pick up the new clock.

	lr.Poke(m)
	m.Infow("test log", "stage", "post poke")

	// force one log rotation
	ch := make(chan struct{})
	lr.WaitForNextRotate(ch)

	var clst core.ClockWrapState
	advancePastNextRotation(t, m, lr, &clst)

	<-ch // Wait for the log rotation to complete

	m.Infow("test log", "stage", "post rotation")

	uis := m.G().UIs()
	mlsui := &mockLogSendUI{}
	uis.LogSend = mlsui
	agent.runCmdWithUIs(t, uis, "logsend")

	require.Equal(t, 2, len(mlsui.logs.Files), "should have two log files")
	require.Equal(t, core.Path("agent.log"), core.ImportPath(mlsui.logs.Files[0]).Base())

	sm := shared.NewMetaContextBackground(agent.testEnv().G)
	set, err := shared.LogSendReassemble(sm, mlsui.res.Id)
	require.NoError(t, err)
	require.Equal(t, 2, len(set.Files), "should have two log files in the set")
	require.Equal(t, lib.LocalFSPath("agent.log.gz"), set.Files[0].Name, "first file name should match")
	for _, f := range set.Files {
		require.NotEmpty(t, f.Name, "file name should not be empty")
		require.NotEmpty(t, f.RawData, "raw data should not be empty")
		require.NotEmpty(t, f.ExpandedData, "expanded data should not be empty")
	}

	// because we called logsend after login, the logsend should include
	// our UID.
	require.NotNil(t, set.UID)
	require.Equal(t, fqu.Uid, *set.UID)
	require.Equal(t, fqu.HostID, set.HostID)
}

func TestLogSendAfterFailedSignup(t *testing.T) {

	a := newTestAgent(t)
	a.runAgent(t)
	defer a.stop(t)

	myErr := errors.New("bailing out for test")

	signupUI := mockSignupUI{
		deviceErr: myErr,
	}
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI{},
	}
	err := a.runCmdErrWithUIs(uis, "signup", "--simple-ui")
	require.Error(t, err)
	require.Equal(t, myErr, err)

	mlsui := &mockLogSendUI{}
	uis.LogSend = mlsui
	a.runCmdWithUIs(t, uis, "logsend")

	require.Equal(t, 1, len(mlsui.logs.Files), "should have two log files")
	require.Equal(t, core.Path("agent.log"), core.ImportPath(mlsui.logs.Files[0]).Base())

	sm := shared.NewMetaContextBackground(a.testEnv().G)
	set, err := shared.LogSendReassemble(sm, mlsui.res.Id)
	require.NoError(t, err)
	require.Equal(t, 1, len(set.Files), "should have one log file")
	require.Nil(t, set.UID, "UID should be nil after failed signup")
}
