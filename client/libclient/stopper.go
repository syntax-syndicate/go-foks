// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"strconv"
	"time"

	"github.com/foks-proj/go-foks/lib/core"
)

type StopperFile struct {
	core.Path
}

func (f StopperFile) Touch() error {
	now := time.Now().UTC().UnixNano()
	return f.WriteFile([]byte(strconv.FormatInt(now, 10)), 0644)
}
