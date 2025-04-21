// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import (
	"context"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/proto/lcl"
)

func (c *AgentConn) TriggerBgUserRefresh(ctx context.Context) error {
	m := c.MetaContext(ctx)
	mgr := c.agent.bg
	return mgr.Bump(m, libclient.BgJobTypeUserRefresh)
}

func (c *AgentConn) TriggerBgClkr(ctx context.Context) error {
	m := c.MetaContext(ctx)
	mgr := c.agent.bg
	return mgr.Bump(m, libclient.BgJobTypeCLKR)
}

var _ lcl.UtilInterface = (*AgentConn)(nil)
