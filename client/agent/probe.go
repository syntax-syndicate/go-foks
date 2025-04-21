// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/chains"
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func (a *AgentConn) Probe(ctx context.Context, addr proto.TCPAddr) (proto.PublicZone, error) {
	var zed proto.PublicZone
	m := a.MetaContext(ctx)
	prb, err := a.probe(m, addr, 30*time.Second)
	if err != nil {
		return zed, err
	}
	ret := prb.PublicZone()
	return *ret, nil
}

func (a *AgentConn) probe(
	m libclient.MetaContext,
	addr proto.TCPAddr,
	timeout time.Duration,
) (*chains.Probe, error) {
	if addr == "" {
		addr = m.G().Cfg().HostsProbe()
	}
	if addr == "" {
		return nil, core.NoDefaultHostError{}
	}
	addr, err := addr.Portify(proto.DefProbePort)
	if err != nil {
		return nil, err
	}
	cas, err := m.G().ProbeRootCAs(m.Ctx())
	if err != nil {
		return nil, err
	}
	return probe(m, timeout, addr, cas)
}

func probe(
	m libclient.MetaContext,
	timeout time.Duration,
	probeSrv proto.TCPAddr,
	cas *x509.CertPool,
) (
	*chains.Probe,
	error,
) {
	return m.G().ProbeByAddr(m.Ctx(), probeSrv, timeout)
}
