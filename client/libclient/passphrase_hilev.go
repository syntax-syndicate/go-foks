// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"context"
	"errors"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type PMELoggedOut struct {
	r    rem.RegClient
	opts StretchOpts
}

func (p *PMELoggedOut) UserServer(uid proto.UID) UserServerInterface { return nil }
func (p *PMELoggedOut) RegServer() RegServerInterface                { return p.r }
func (p *PMELoggedOut) StretchOpts() StretchOpts                     { return p.opts }

func (s *PMELoggedOut) MakeUserSettingsLink(
	ctx context.Context,
	info proto.PassphraseInfo,
) (
	*rem.PostGenericLinkArg,
	error,
) {
	return nil, nil
}

func (s *PMELoggedOut) GetUserSettings(
	ctx context.Context,
) (
	*proto.PassphraseInfo,
	error,
) {
	return nil, nil
}

func NewPMELoggedOut(r rem.RegClient, opts StretchOpts) *PMELoggedOut {
	return &PMELoggedOut{
		r:    r,
		opts: opts,
	}
}

type PMELoggedIn struct {
	g    *GlobalContext
	uc   *UserContext
	reg  rem.RegClient
	user rem.UserClient
	opts StretchOpts
	uw   *UserWrapper
}

func NewPMELoggedIn(m MetaContext, au *UserContext) (*PMELoggedIn, error) {
	ucli, err := au.UserClient(m)
	if err != nil {
		return nil, err
	}
	rcli, err := au.RegClient(m)
	if err != nil {
		return nil, err
	}
	opts := StretchOpts{
		IsTest: m.G().Cfg().TestingMode(),
	}
	return &PMELoggedIn{
		g:    m.G(),
		uc:   au,
		reg:  *rcli,
		user: *ucli,
		opts: opts,
	}, nil
}

func (p *PMELoggedIn) SetUserWrapper(uw *UserWrapper) *PMELoggedIn {
	p.uw = uw
	return p
}

func (p *PMELoggedIn) RegServer() RegServerInterface            { return p.reg }
func (p *PMELoggedIn) UserServer(proto.UID) UserServerInterface { return p.user }
func (p *PMELoggedIn) StretchOpts() StretchOpts                 { return p.opts }

func (p *PMELoggedIn) MakeUserSettingsLink(
	ctx context.Context,
	info proto.PassphraseInfo,
) (
	*rem.PostGenericLinkArg,
	error,
) {
	m := NewMetaContext(ctx, p.g)
	payload := proto.NewGenericLinkPayloadWithUsersettings(
		proto.NewUserSettingsLinkWithPassphrase(
			info,
		),
	)
	res, err := MakeGenericLink(m, p.uc, payload)
	if err != nil {
		return nil, err
	}
	return &rem.PostGenericLinkArg{
		Link:             *res.Link,
		NextTreeLocation: *res.NextTreeLocation,
	}, nil
}

func (p *PMELoggedIn) GetUserSettings(
	ctx context.Context,
) (
	*proto.PassphraseInfo,
	error,
) {
	m := NewMetaContext(ctx, p.g)
	usl := NewUserSettingsLoader(p.uc)
	if p.uw != nil {
		usl.SetUserWrapper(p.uw)
	}
	chain, err := usl.Run(m)
	if err != nil {
		return nil, err
	}
	if chain == nil {
		return nil, nil
	}
	typ, err := chain.Payload.GetT()
	if err != nil {
		return nil, err
	}
	if typ != proto.ChainType_UserSettings {
		return nil, core.InternalError("unexpected payload type")
	}
	us := chain.Payload.Usersettings()
	return us.Passphrase, nil
}

func (s *SecretKeyMaterialManager) PassphraseGen() (proto.PassphraseGeneration, error) {
	var zed proto.PassphraseGeneration
	s.Lock()
	defer s.Unlock()
	if s.bun == nil {
		return zed, core.NotFoundError("no loaded bundle")
	}
	typ, err := s.bun.GetT()
	if err != nil {
		return zed, err
	}
	if typ != proto.SecretKeyStorageType_ENC_PASSPHRASE {
		return zed, core.SecretKeyStorageTypeError{Actual: typ}
	}
	bun := s.bun.EncPassphrase()
	return bun.Ppgen, nil
}

var _ PassphraseManagerEngine = (*PMELoggedOut)(nil)
var _ PassphraseManagerEngine = (*PMELoggedIn)(nil)

func PassphraseUnlockCurrentUser(
	m MetaContext,
	pp proto.Passphrase,
) error {
	uc := m.G().ActiveUser()
	if uc == nil {
		return core.NoActiveUserError{}
	}

	store := m.G().SecretStore()
	err := store.Load(m.Ctx())
	if err != nil {
		return err
	}

	skmm := uc.Skmm()
	if skmm == nil {
		skmm = NewSecretKeyMaterialManager(uc.FQU(), uc.Role())
		uc.SetSkmm(skmm)
	}
	err = skmm.Load(m.Ctx(), store, SecretStoreGetOpts{NoProvisional: true})
	if err != nil {
		return err
	}

	err = skmm.ReadySeed(m.Ctx())
	if err == nil {
		return nil
	}
	if !errors.Is(err, core.PassphraseLockedError{}) {
		return err
	}

	rcli, err := uc.RegClient(m)
	if err != nil {
		return err
	}
	psi := NewPMELoggedOut(*rcli, StretchOpts{IsTest: m.G().Cfg().TestingMode()})
	pm := NewPassphraseManager(uc.FQU())
	err = pm.Login(m.Ctx(), psi, pp, skmm.Salt(), skmm.StretchVersion())
	if err != nil {
		return err
	}
	err = skmm.DecryptWithPassphrase(m.Ctx(), pm, store)
	if err != nil {
		return err
	}
	err = uc.PopulateWithDevkey(m)
	if err != nil {
		return err
	}

	pmli, err := NewPMELoggedIn(m, uc)
	if err != nil {
		return err
	}

	ppgen, err := skmm.PassphraseGen()
	if err != nil {
		return err
	}

	err = skmm.RefreshUserSettingsMaybeUpdatePassphraseEncryption(
		m.Ctx(),
		pm,
		pmli,
		ppgen,
		uc.PUKs(),
		store,
	)
	if err != nil {
		return err
	}

	return nil
}

func ReencryptIfPassphraseLocked(
	m MetaContext,
	uc *UserContext,
	pm *PassphraseManager,
) error {
	ss := m.G().SecretStore()
	err := ss.Load(m.Ctx())
	if err != nil {
		return err
	}
	skmm := uc.SkmmGetOrMake()
	return skmm.ReencryptIfPassphraseLocked(m.Ctx(), pm, ss)
}

func BgRefreshPassphraseEncryption(
	m MetaContext,
	uc *UserContext,
	uw *UserWrapper,
) error {

	m.Infow("BgRefreshPassphraseEncryption", "stage", "enter")
	skmm, err := uc.LoadSkkm(m)
	if err != nil {
		if _, ok := err.(core.NotFoundError); ok {
			m.Infow("BgRefreshPassphraseEncryption", "stage", "no_skmm")
			return nil
		}
		return err
	}

	pukSet, err := uc.RefreshPUKs(m)
	if err != nil {
		return err
	}
	puks := pukSet.All()
	if len(puks) == 0 {
		return core.KeyNotFoundError{Which: "puks"}
	}

	ss := m.G().SecretStore()

	pm := NewPassphraseManager(uc.FQU())
	pme, err := NewPMELoggedIn(m, uc)
	if err != nil {
		return err
	}
	pme.SetUserWrapper(uw)
	typ, err := skmm.UnlockSeed(m.Ctx(), pme, puks)
	if err != nil {
		return err
	}
	if typ != proto.SecretKeyStorageType_ENC_PASSPHRASE {
		return nil
	}

	ppgen, err := skmm.PassphraseGen()
	if err != nil {
		return err
	}

	return skmm.RefreshUserSettingsMaybeUpdatePassphraseEncryption(
		m.Ctx(),
		pm,
		pme,
		ppgen,
		puks,
		ss,
	)
}
