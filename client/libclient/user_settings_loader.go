// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/lib/merkle"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type UserSettingsLoaderTesting struct {
	Base                  ChainLoaderTesting
	FetchHook             func(m MetaContext, arg rem.LoadGenericChainArg) (rem.GenericChain, error)
	MutateUserWrapperHook func(uw *UserWrapper)
}

type UserSettingsLoader struct {
	gcl      *GenericChainLoader
	au       *UserContext
	uw       *UserWrapper
	settings *lcl.UserSettingsChainPayload
	testing  *UserSettingsLoaderTesting
}

var _ ChainLoaderSubclass = (*UserSettingsLoader)(nil)

func NewUserSettingsLoader(au *UserContext) *UserSettingsLoader {
	ret := &UserSettingsLoader{au: au}
	ret.gcl = NewGenericChainLoader(au.UID().EntityID(), ret)
	return ret
}

func (u *UserSettingsLoader) SetTesting(t *UserSettingsLoaderTesting) {
	if t == nil {
		u.gcl.SetTesting(nil)
	} else {
		u.gcl.SetTesting(&t.Base)
		u.testing = t
	}
}

func (u *UserSettingsLoader) GenericChainLoader() *GenericChainLoader {
	return u.gcl
}

func (u *UserSettingsLoader) SetUserWrapper(uw *UserWrapper) {
	u.uw = uw
}

func (u *UserSettingsLoader) init(m MetaContext) error {
	if u.uw != nil {
		return nil
	}
	uw, err := LoadMe(m, u.au)
	if err != nil {
		return err
	}
	if u.testing != nil && u.testing.MutateUserWrapperHook != nil {
		u.testing.MutateUserWrapperHook(uw)
	}
	u.uw = uw
	return nil
}

func (u *UserSettingsLoader) Run(m MetaContext) (*lcl.GenericChainState, error) {
	err := u.init(m)
	if err != nil {
		return nil, err
	}
	err = u.gcl.Run(m)
	if err != nil {
		return nil, err
	}
	return u.gcl.res, nil
}

func (u *UserSettingsLoader) LatestTreeRoot(m MetaContext) (*proto.TreeRoot, error) {
	ma := u.gcl.ma
	if ma != nil {
		var err error
		ma, err = u.MerkleAgent(m)
		if err != nil {
			return nil, err
		}
		defer ma.Shutdown()
	}
	mr, err := ma.GetLatestRootFromCache(m.Ctx())
	if err != nil {
		return nil, err
	}
	ret, err := merkle.ToTreeRoot(mr)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (u *UserSettingsLoader) Fetch(m MetaContext, arg rem.LoadGenericChainArg) (rem.GenericChain, error) {
	if u.testing != nil && u.testing.FetchHook != nil {
		return u.testing.FetchHook(m, arg)
	}
	var ret rem.GenericChain
	ucli, err := u.au.UserClient(m)
	if err != nil {
		return ret, err
	}
	return ucli.LoadGenericChain(m.Ctx(), arg)
}

func (u *UserSettingsLoader) SeedCommitment() *proto.TreeLocationCommitment {
	return &u.uw.prot.Sctlsc
}
func (u *UserSettingsLoader) Type() proto.ChainType {
	return proto.ChainType_UserSettings
}
func (u *UserSettingsLoader) Scoper() Scoper {
	fqu := u.au.FQU()
	return &fqu
}

func (u *UserSettingsLoader) MerkleAgent(m MetaContext) (*merkle.Agent, error) {
	ma, err := u.au.MerkleAgent(m)
	if err != nil {
		return nil, err
	}
	return ma, nil
}

func (u *UserSettingsLoader) LoadState(p lcl.GenericChainStatePayload) error {
	typ, err := p.GetT()
	if err != nil {
		return err
	}
	if typ != proto.ChainType_UserSettings {
		return core.ChainLoaderError{
			Err: core.UserSettingsError("wrong chain type stored"),
		}
	}
	tmp := p.Usersettings()
	u.settings = &tmp
	return nil
}

func (u *UserSettingsLoader) PlayLink(m MetaContext, l proto.LinkOuter, g proto.GenericLinkPayload) error {
	typ, err := g.GetT()
	if err != nil {
		return err
	}
	if typ != proto.ChainType_UserSettings {
		return core.ChainLoaderError{
			Err: core.UserSettingsError("wrong chain type in playback"),
		}
	}
	usl := g.Usersettings()
	styp, err := usl.GetT()
	if err != nil {
		return err
	}
	switch styp {
	case proto.UserSettingsType_Passphrase:
		ppi := usl.Passphrase()
		if u.settings == nil {
			u.settings = &lcl.UserSettingsChainPayload{}
		}
		if u.settings.Passphrase == nil {
			u.settings.Passphrase = &proto.PassphraseInfo{}
		}
		if u.settings.Passphrase.Gen > ppi.Gen {
			return core.ChainLoaderError{
				Err: core.UserSettingsError("passphrase gen went backwards"),
			}
		}
		u.settings.Passphrase.Gen = ppi.Gen
		if ppi.Salt != nil {
			u.settings.Passphrase.Salt = ppi.Salt
		}
		return nil
	default:
		return core.ChainLoaderError{
			Err: core.NotImplementedError{},
		}
	}
}
func (u *UserSettingsLoader) SaveState() (lcl.GenericChainStatePayload, error) {
	return lcl.NewGenericChainStatePayloadWithUsersettings(*u.settings), nil
}

func (u *UserSettingsLoader) BookendSigningKey(
	m MetaContext,
	owner proto.FQEntity,
	key proto.EntityID,
	epno proto.MerkleEpno,
) (
	*KeyBookends,
	error,
) {
	fqe := u.au.FQU().ToFQEntity()
	if !fqe.Eq(owner) {
		return nil, core.PermissionError("wrong owner")
	}
	return u.uw.BookendSigningKey(proto.FQEntity{Entity: key, Host: owner.Host}, epno)
}

func PassphrseInfoFromUserSettings(g *lcl.GenericChainState) (*proto.PassphraseInfo, error) {
	if g == nil {
		return nil, nil
	}
	typ, err := g.Payload.GetT()
	if err != nil {
		return nil, err
	}
	if typ != proto.ChainType_UserSettings {
		return nil, nil
	}
	us := g.Payload.Usersettings()
	return us.Passphrase, nil
}
