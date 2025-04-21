// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"errors"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type SeqnoDbKey proto.Seqno

func (s SeqnoDbKey) DbKey() (proto.DbKey, error) {
	tmp := proto.Seqno(s)
	b, err := core.EncodeToBytes(&tmp)
	if err != nil {
		return nil, err
	}
	return proto.DbKey(b), nil
}

func (u *UserContext) storeTreeLocMem(s proto.Seqno, loc proto.TreeLocation) {
	if u.treeLocs == nil {
		u.treeLocs = make(map[proto.Seqno]proto.TreeLocation)
	}
	u.treeLocs[s] = loc
}

func (u *UserContext) StoreTreeLocation(m MetaContext, loc rem.TreeLocationPair) error {
	u.treeLocMu.Lock()
	defer u.treeLocMu.Unlock()
	u.storeTreeLocMem(loc.Seqno, loc.Loc)
	return u.storeTreeLocDB(m, loc.Seqno, loc.Loc)
}

func (u *UserContext) storeTreeLocDB(m MetaContext, s proto.Seqno, loc proto.TreeLocation) error {
	uid := u.Info.Fqu.Uid
	return m.DbPut(DbTypeSoft, PutArg{
		Scope: &uid,
		Typ:   lcl.DataType_TreeLocation,
		Key:   SeqnoDbKey(s),
		Val:   &loc,
	})
}

func (u *UserContext) getTreeLocDB(m MetaContext, s proto.Seqno) (*proto.TreeLocation, error) {
	uid := u.Info.Fqu.Uid
	var loc proto.TreeLocation
	_, err := m.DbGet(
		&loc,
		DbTypeSoft,
		&uid,
		lcl.DataType_TreeLocation,
		SeqnoDbKey(s),
	)
	if err == nil {
		return &loc, nil
	}
	if errors.Is(err, core.RowNotFoundError{}) {
		return nil, nil
	}
	return nil, err
}

func (u *UserContext) getTreeLocMem(s proto.Seqno) *proto.TreeLocation {
	if u.treeLocs == nil {
		return nil
	}
	if loc, ok := u.treeLocs[s]; ok {
		return &loc
	}
	return nil
}

func (u *UserContext) getTreeLocServer(m MetaContext, s proto.Seqno) (*proto.TreeLocation, error) {
	cli, err := u.UserClient(m)
	if err != nil {
		return nil, err
	}
	tmp, err := cli.GetTreeLocation(m.Ctx(), s)
	if err != nil {
		return nil, err
	}
	return &tmp, nil
}

func (u *UserContext) GetTreeLocation(m MetaContext, s proto.Seqno) (*proto.TreeLocation, error) {
	u.treeLocMu.Lock()
	defer u.treeLocMu.Unlock()

	loc := u.getTreeLocMem(s)
	if loc != nil {
		return loc, nil
	}
	loc, err := u.getTreeLocDB(m, s)
	if err != nil {
		return nil, err
	}
	if loc != nil {
		u.storeTreeLocMem(s, *loc)
		return loc, nil
	}
	loc, err = u.getTreeLocServer(m, s)
	if err != nil {
		return nil, err
	}
	u.storeTreeLocMem(s, *loc)
	err = u.storeTreeLocDB(m, s, *loc)
	if err != nil {
		return nil, err
	}

	return loc, nil
}
