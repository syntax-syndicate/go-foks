// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package ui

import (
	"time"

	"github.com/foks-proj/go-foks/client/libclient"
)

func debugSpinners(m libclient.MetaContext) {
	if m.G().Cfg().DebugSpinners() {
		<-time.After(800 * time.Millisecond)
	}
}
