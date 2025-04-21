// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"context"
	"net"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-snowpack-rpc/rpc"
)

type Dialer interface {
	Dial(context.Context) (net.Conn, error)
}

func NewLoopbackListener(m MetaContext) *rpc.LoopbackListener {
	return rpc.NewLoopbackListener(m.Debugf)
}

type IPCAgent struct {
	f core.Path
}

func (s IPCAgent) Dial(context.Context) (net.Conn, error) {
	return s.f.Dial()
}

func NewIPCAgent(f core.Path) IPCAgent {
	return IPCAgent{f: f}
}

var _ Dialer = (*rpc.LoopbackListener)(nil)
var _ Dialer = IPCAgent{}

func SocketDance(m MetaContext, f core.Path, fn func(MetaContext, core.Path) error) error {
	m.G().Lock()
	defer m.G().Unlock()

	m = m.WithLogTag("sockdance")
	m.Infow("SocketDance", "socket", f)

	// at ~103 bytes, the socket path is too long for unix sockets, so we have to
	// hack in a chdir and try that
	if len(f) >= 103 {
		m.Infow("SocketDance", "long_socket_len", len(f))
		cwd, err := core.Getwd()
		if err != nil {
			return err
		}
		dir := f.Dir()
		m.Infow("SocketDance", "chdir", dir, "cwd", cwd)
		err = dir.Chdir()
		if err != nil {
			return err
		}
		defer func() {
			err = cwd.Chdir()
			m.Infow("SocketDance", "restore", cwd)
			if err != nil {
				m.Errorw("SocketDance", "restore", cwd, "err", err)
			}
		}()

		f = f.Base()
	}

	err := fn(m, f)
	if err != nil {
		m.Errorw("SocketDance", "err", err)
		return err
	}
	return nil
}
