// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

// Hostname might be non-nil in the case of a vhost registration
func BeaconRegisterCli(m MetaContext, vhn proto.Hostname, gs *GlobalService) error {
	var err error

	if gs == nil {
		gs, err = m.G().Config().BeaconGlobalService(m.Ctx())
		if err != nil {
			return err
		}
	}

	rm := RpcClientMetaContext{MetaContext: m}
	gcli := core.NewRpcClient(
		rm,
		gs.Addr,
		gs.CAs,
		nil,
		nil,
	)

	_, ext, _, err := m.G().ListenParams(m.Ctx(), proto.ServerType_Probe)
	if err != nil {
		return err
	}

	hn, port, err := ext.Split()
	if err != nil {
		return err
	}
	if !vhn.IsZero() {
		hn = vhn
	}
	if port == nil {
		tmp := proto.Port(443)
		port = &tmp
	}

	cli := core.NewBeaconClient(gcli, m)
	err = cli.BeaconRegister(m.Ctx(), rem.BeaconRegisterArg{
		Host:   hn,
		Port:   *port,
		HostID: m.HostID().Id,
	})

	if err != nil {
		return err
	}

	return nil
}
