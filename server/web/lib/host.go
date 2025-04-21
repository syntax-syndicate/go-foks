// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/server/shared"
)

type Host struct {
	core.HostID
	Name proto.Hostname
}

func LoadHostByShortID(
	m shared.MetaContext,
	shid core.ShortHostID,
) (
	*Host,
	error,
) {
	himap := m.G().HostIDMap()
	hid, err := himap.LookupByShortID(m, shid)
	if err != nil {
		return nil, err
	}
	hn, err := himap.Hostname(m, hid.Short)
	if err != nil {
		return nil, err
	}
	ret := Host{
		HostID: *hid,
		Name:   hn,
	}
	return &ret, nil
}
