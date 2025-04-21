// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

import (
	"bytes"

	proto "github.com/foks-proj/go-foks/proto/lib"
)

var LocalHost = proto.LocalHost

func ExportLocalHost() []byte {
	return LocalHost
}

func ExportHostP(hid *proto.HostID) []byte {
	if hid == nil {
		return LocalHost
	}
	return hid.ExportToDB()
}

func ExportHostInScope(m MetaContext, hid proto.HostID) []byte {
	local := m.HostID().Id
	if local.Eq(hid) {
		return LocalHost
	}
	return hid.ExportToDB()
}

func ImportHost(m MetaContext, raw []byte) (proto.HostID, error) {
	if len(raw) == len(LocalHost) && bytes.Equal(LocalHost, raw) {
		return m.HostID().Id, nil
	}
	var ret proto.HostID
	err := ret.ImportFromBytes(raw)
	return ret, err
}

func ImportHostInScope(raw []byte) (*proto.HostID, error) {
	if len(raw) == len(LocalHost) && bytes.Equal(LocalHost, raw) {
		return nil, nil
	}
	var ret proto.HostID
	err := ret.ImportFromBytes(raw)
	if err != nil {
		return nil, err
	}
	return &ret, nil

}
