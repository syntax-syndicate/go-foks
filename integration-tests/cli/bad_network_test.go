// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cli

import (
	"testing"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	"github.com/stretchr/testify/require"
)

func TestBadNetworkThenRecovery(t *testing.T) {
	bob := makeBobAndHisAgent(t)
	b := bob.agent
	defer b.stop(t)

	merklePoke(t)

	netStatus := func() error {
		var status lcl.AgentStatus
		b.runCmdToJSON(t, &status, "status")
		require.Equal(t, 1, len(status.Users))
		return core.StatusToError(status.Users[0].NetworkStatus)
	}

	require.Equal(t, nil, netStatus())

	b.stop(t)
	b.opts.killNetwork = true
	b.setFlags(t)
	b.runAgent(t)

	requireNetErr := func(err error) {
		require.Error(t, err)
		require.Equal(t,
			core.NewConnectError(
				"catastrophic network conditions",
				core.NetworkConditionerError{},
			),
			err,
		)

	}

	err := netStatus()
	requireNetErr(err)

	err = b.runCmdErr(nil, "key", "list")
	requireNetErr(err)

	b.runCmd(t, nil, "test", "set-network-conditions", "clear")
	b.runCmd(t, nil, "key", "list")

	err = netStatus()
	require.NoError(t, err)
}
