// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func ExportTmpDHKeySignedToDB(t *proto.TempDHKeySigned) ([]byte, error) {
	if t == nil {
		return nil, nil
	}
	return core.EncodeToBytes(t)
}
