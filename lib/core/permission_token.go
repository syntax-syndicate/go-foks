// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func NewPermissionToken() (proto.PermissionToken, error) {
	ret, err := proto.RandomID16er[proto.PermissionToken]()
	if err != nil {
		var zed proto.PermissionToken
		return zed, err
	}
	return *ret, nil
}
