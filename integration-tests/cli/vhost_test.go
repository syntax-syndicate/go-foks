// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cli

import (
	"testing"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/integration-tests/common"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/stretchr/testify/require"
)

func TestVhostSimpleSignup(t *testing.T) {
	defer common.DebugEntryAndExit()()

	env := globalTestEnv
	merklePoke(t)
	tvh := env.VHostInit(t, "bozo")

	agentOpts := agentOpts{dnsAliases: []proto.Hostname{tvh.Hostname}}

	x := newTestAgentWithOpts(t, agentOpts)
	x.runAgent(t)
	defer x.stop(t)

	// It's OK to reuse a yubi since we don't have any other
	// signups on this virtual host.
	uis := libclient.UIs{
		Signup: newMockSignupUI().
			withServer(tvh.ProbeAddr).
			withForceYubiReuse(),
	}
	x.runCmdWithUIs(t, uis, "--simple-ui", "signup")

	var st lcl.AgentStatus
	x.runCmdToJSON(t, &st, "status")
	require.Equal(t, len(st.Users), 1)
	require.Equal(t, st.Users[0].Info.Fqu.HostID, tvh.HostID.Id)

	// now try a provision up onto Y.
	y := newTestAgentWithOpts(t, agentOpts)
	y.runAgent(t)
	defer y.stop(t)

	runProvisionOnAgents(
		t,
		provOpts{
			enterHespOnX: true,
			x:            x,
			y:            y,
			noSignup:     true,
			probeAddr:    &tvh.ProbeAddr,
		},
	)

	y.runCmdToJSON(t, &st, "status")
	require.Equal(t, len(st.Users), 1)
	require.Equal(t, st.Users[0].Info.Fqu.HostID, tvh.HostID.Id)
}
