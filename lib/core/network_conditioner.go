// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type NetworkConditioner interface {
	// FailConnect returns non-nil if we're going to fail
	// a connection to the given address as it's being made
	// The failure can be immediate so as not to slow
	// down tests.
	FailConnect(a proto.TCPAddr) error
}

type ClearNetworkConditions struct {
}

func (c ClearNetworkConditions) FailConnect(a proto.TCPAddr) error {
	return nil
}

var _ NetworkConditioner = ClearNetworkConditions{}

type CatastrophicNetworkConditions struct {
	On bool
}

func (c CatastrophicNetworkConditions) FailConnect(a proto.TCPAddr) error {
	if c.On {
		return NewConnectError(
			"catastrophic network conditions",
			NetworkConditionerError{},
		)
	}
	return nil
}
