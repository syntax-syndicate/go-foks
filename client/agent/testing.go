// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import (
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
)

func InitTesting(m libclient.MetaContext) error {
	g := m.G()

	if g.Cfg().GetTestKillNetwork() {
		g.SetNetworkConditioner(
			core.CatastrophicNetworkConditions{
				On: true,
			},
		)
	}

	return nil
}
