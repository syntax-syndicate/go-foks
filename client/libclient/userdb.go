// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"errors"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type LocalDbTx struct {
	arg []PutArg
}

func (t *LocalDbTx) Arg() []PutArg { return t.arg }

func (t *LocalDbTx) PutUserInfo(u proto.UserInfo) error {
	key := u.ToLocalUserIndexAtHost()
	t.arg = append(t.arg, PutArg{
		Scope: &u.Fqu.HostID,
		Typ:   lcl.DataType_User,
		Key:   &key,
		Val:   &u,
	})
	return nil
}

func (t *LocalDbTx) PutUsername(u proto.UserInfo) error {
	tmp := string(u.Username.Name)
	uid := u.Fqu.Uid
	t.arg = append(t.arg, PutArg{
		Scope: &u.Fqu.HostID,
		Key:   tmp,
		Typ:   lcl.DataType_UsernameLookup,
		Val:   &uid,
	})
	return nil
}

func (t *LocalDbTx) PutAllUsers(u proto.UserInfo) error {
	val := u.ToLocalUserIndex()
	t.arg = append(t.arg, PutArg{
		Key: core.KVKeyAllUsers,
		Val: &val,
		Set: true,
	})
	return nil
}

func (t *LocalDbTx) PutActiveUser(u proto.UserInfo) error {
	val := u.ToLocalUserIndex()
	t.arg = append(t.arg, PutArg{
		Key: core.KVKeyCurrentUser,
		Val: &val,
	})
	return nil
}

func (t *LocalDbTx) PutUser(u proto.UserInfo, active bool) error {
	err := t.PutUserInfo(u)
	if err != nil {
		return err
	}
	err = t.PutUsername(u)
	if err != nil {
		return err
	}
	err = t.PutAllUsers(u)
	if err != nil {
		return err
	}
	if active {
		err = t.PutActiveUser(u)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *LocalDbTx) PutPukParcel(u proto.FQUser, pp proto.SharedKeyParcel) error {
	t.arg = append(t.arg, PutArg{
		Scope: &u,
		Typ:   lcl.DataType_SharedKeyCacheEntry,
		Key:   core.RoleDBKey(pp.Box.Role),
		Val:   &pp,
	})
	return nil
}

func (t *LocalDbTx) Exec(m MetaContext) error {
	return m.DbPutTx(DbTypeHard, t.arg)
}

func LoadUserFromDB(m MetaContext, u proto.LocalUserIndex) (*proto.UserInfo, error) {
	var uinfo proto.UserInfo
	_, err := m.DbGet(&uinfo, DbTypeHard, &u.Host, lcl.DataType_User, &u.Rest)
	if err != nil {
		return nil, err
	}
	return &uinfo, nil
}

func LoadAllUsersFromDB(m MetaContext) ([]proto.UserInfo, error) {

	v, _, err := DbGetGlobalSet[proto.LocalUserIndex](m, DbTypeHard, core.KVKeyAllUsers)
	if err != nil {
		return nil, err
	}

	var ret []proto.UserInfo
	for _, idx := range v {
		tmp, err := LoadUserFromDB(m, idx)
		if errors.Is(err, core.RowNotFoundError{}) {
			continue
		}
		if err != nil {
			return nil, err
		}
		ret = append(ret, *tmp)
	}
	return ret, nil
}

func LoadCurrentUserFromDB(m MetaContext) (*proto.UserInfo, error) {
	var idx proto.LocalUserIndex
	_, err := m.DbGetGlobalKV(&idx, DbTypeHard, core.KVKeyCurrentUser)
	if err != nil {
		return nil, err
	}
	return LoadUserFromDB(m, idx)
}

func LookupUIDFromDB(m MetaContext, hostid proto.HostID, username proto.Name) (*proto.UID, error) {
	var ret proto.UID
	_, err := m.DbGet(&ret, DbTypeHard, &hostid, lcl.DataType_UsernameLookup, string(username))
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func StoreCurrentUserToDB(m MetaContext, u *proto.UserInfo) error {
	var ltx LocalDbTx

	// Don't store backup key usages to the local DB. They should be ephemeral.
	if u.KeyGenus == proto.KeyGenus_Backup {
		return nil
	}
	err := ltx.PutUser(*u, true)
	if err != nil {
		return err
	}
	return ltx.Exec(m)
}

func StoreUserToDB(m MetaContext, u *proto.UserInfo) error {
	var ltx LocalDbTx

	// Don't store backup key usages to the local DB. They should be ephemeral.
	if u.KeyGenus == proto.KeyGenus_Backup {
		return nil
	}

	err := ltx.PutUser(*u, false)
	if err != nil {
		return err
	}
	return ltx.Exec(m)
}

func DeleteUserFromDB(m MetaContext, idx proto.LocalUserIndex, keyID proto.EntityID) error {
	uinf, err := LoadUserFromDB(m, idx)
	if err != nil {
		return err
	}
	if !uinf.Key.Eq(keyID) {
		return core.KeyMismatchError{}
	}
	hsh, err := core.PrefixedHash(&idx)
	if err != nil {
		return err
	}
	return m.DbDeleteFromGlobalSet(DbTypeHard, core.KVKeyAllUsers, *hsh)
}
