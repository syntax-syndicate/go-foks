// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"encoding/binary"
	"fmt"

	proto "github.com/foks-proj/go-foks/proto/lib"
)

type DbKeyer interface {
	DbKey() (proto.DbKey, error)
}

func NewDbKey(a any) (proto.DbKey, error) {

	if dbk, ok := a.(DbKeyer); ok {
		return dbk.DbKey()
	}

	if ta, ok := a.(interface{ Bytes() []byte }); ok {
		if raw := ta.Bytes(); raw != nil {
			return proto.DbKey(raw), nil
		}
	}

	if ta, ok := a.(CryptoPayloader); ok {
		ret := make([]byte, 32)
		err := PrefixedHashInto(ta, ret)
		if err != nil {
			return nil, err
		}
		return proto.DbKey(ret), nil
	}

	if s, ok := a.(string); ok {
		return proto.DbKey([]byte(s)), nil
	}

	if b, ok := a.([]byte); ok {
		return proto.DbKey(b), nil
	}

	if s, ok := a.(interface{ String() string }); ok {
		return proto.DbKey([]byte(s.String())), nil
	}

	if i, ok := a.(interface{ Uint16() uint16 }); ok {
		var tmp [2]byte
		binary.BigEndian.PutUint16(tmp[:], i.Uint16())
		return proto.DbKey(tmp[:]), nil
	}

	return nil, DbError(fmt.Sprintf("cannot coerce type (%T) into a DbKey", a))
}

type EmptyKey struct{}

func (e EmptyKey) DbKey() (proto.DbKey, error) {
	tmp := []byte{0}
	return proto.DbKey(tmp), nil
}

var _ DbKeyer = EmptyKey{}

type KVKey string

const (
	KVKeyCurrentUser KVKey = "current_user_v2"
	KVKeyAllUsers    KVKey = "all_users_v2"
)
