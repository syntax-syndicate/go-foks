// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cli

import (
	"testing"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/proto/lcl"
)

func TestSelfProvision(t *testing.T) {
	alice := makeAliceAndHerAgent(t)
	a := alice.agent
	stopper := runMerkleActivePoker(t)
	a.runCmd(t, nil, "key", "dev", "perm", "--name", "bozo-phone 4")
	stopper()
}

func TestLoadMe(t *testing.T) {
	agent := newTestAgentWithOpts(t, agentOpts{})
	agent.runAgent(t)
	defer agent.stop(t)
	var signupUI mockSignupUI
	signupUI.useDeviceKey = true
	var terminalUI terminalUI
	uis := libclient.UIs{
		Signup:   &signupUI,
		Terminal: &terminalUI,
	}
	agent.runCmdWithUIs(t, uis, "--simple-ui", "signup")
	stopper := runMerkleActivePoker(t)
	defer stopper()

	var res lcl.UserMetadataAndSigchainState
	agent.runCmdToJSON(t, &res, "user", "load-me")
}
