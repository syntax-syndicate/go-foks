// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import (
	"context"
	"os"

	"github.com/foks-proj/go-foks/proto/lcl"
)

func (a *AgentConn) Shutdown(ctx context.Context) (uint64, error) {
	a.agent.TriggerStop()
	ret := uint64(os.Getpid())
	return ret, nil
}

func (a *AgentConn) PingAgent(ctx context.Context) (uint64, error) {
	return uint64(os.Getpid()), nil
}

var _ lcl.CtlInterface = (*AgentConn)(nil)
