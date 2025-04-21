// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import (
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-snowpack-rpc/rpc"
)

func startLoopbackServer(m libclient.MetaContext) error {
	lb := rpc.NewLoopbackListener(m.Debugf)
	a := NewAgent(m)
	m.G().SetLoopback(a, lb)
	return a.ServeWithListener(m, lb)
}

func startStandalone(m libclient.MetaContext, opts StartupOpts) error {
	err := startLoopbackServer(m)
	if err != nil {
		return err
	}

	// On signup and simiilar commands, no need to load active user.
	if opts.NeedUser || opts.NeedUnlockedUser {
		lauOpts := libclient.LoadActiveUserOpts{NeedUnlocked: opts.NeedUnlockedUser}
		if opts.ForceYubiUnlock || (opts.NeedUnlockedUser && !m.G().Cfg().YubiNoForceUnlock()) {
			lauOpts.ForceYubiUnlock = true
		}
		err = m.LoadActiveUser(lauOpts)
		if err != nil {
			return err
		}
	}
	return nil
}

func connectToAgent(m libclient.MetaContext) error {
	file, err := m.G().Cfg().SocketFile()
	if err != nil {
		return err
	}
	i := libclient.NewIPCAgent(file)
	m.G().SetService(i)
	return nil
}

type StartupOpts struct {
	NeedUser         bool
	NeedUnlockedUser bool
	ForceYubiUnlock  bool
	GitRemoteHelper  bool
	NoStandalone     bool
}

func Startup(m libclient.MetaContext, opts StartupOpts) error {

	if opts.GitRemoteHelper {
		m.G().SetMode(libclient.GlobalContextModeGitRemhoteHelper)
	}

	err := m.Configure()
	if err != nil {
		return err
	}

	if m.G().Cfg().Standalone() && !opts.NoStandalone {
		err = startStandalone(m, opts)
	} else {
		err = connectToAgent(m)
	}

	if err != nil {
		return err
	}

	return nil
}
