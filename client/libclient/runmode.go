// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

type RunMode uint

const (
	RunModeProd  RunMode = 0
	RunModeDevel RunMode = 1
)

func (r RunMode) ToString() string {
	switch r {
	case RunModeDevel:
		return "devel"
	case RunModeProd:
		return "prod"
	default:
		return "none"
	}
}
