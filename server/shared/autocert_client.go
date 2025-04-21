// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

import (
	"time"

	"github.com/foks-proj/go-foks/proto/infra"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func DoAutocertViaClient(
	m MetaContext,
	waitFor time.Duration,
	pkg AutocertPackage,
) error {
	cli, closer, err := m.G().AutocertCli(m.Ctx())
	if err != nil {
		return err
	}
	defer closer()
	return cli.DoAutocert(m.Ctx(),
		infra.DoAutocertArg{
			WaitFor: proto.ExportDurationMilli(waitFor),
			Pkg: infra.AutocertPackage{
				Hostname: pkg.Hostname,
				Hostid:   pkg.HostID,
				Styp:     pkg.ServerType,
				IsVanity: pkg.IsVanity,
			},
		},
	)

}
