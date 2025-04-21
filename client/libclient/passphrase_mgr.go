// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"context"
	"fmt"
	"sync"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type PassphraseManager struct {
	sync.RWMutex
	user      proto.FQUser
	skmwkList []lcl.SKMWK
	sp        *StretchedPassphrase
	pi        *proto.PassphraseInfo
}

type RegServerInterface interface {
	GetLoginChallenge(context.Context, proto.UID) (rem.Challenge, error)
	Login(context.Context, rem.LoginArg) (rem.LoginRes, error)
}

type UserServerInterface interface {
	SetPassphrase(context.Context, rem.SetPassphraseArg) error
	ChangePassphrase(context.Context, rem.ChangePassphraseArg) error
	GetSalt(context.Context) (proto.PassphraseSalt, error)
	NextPassphraseGeneration(context.Context) (proto.PassphraseGeneration, error)
	StretchVersion(context.Context) (proto.StretchVersion, error)
	GetPpeParcel(context.Context) (proto.PpeParcel, error)
}

// When unlocking a passphrase, call into these external services. It will be a mix of calls
// to reg (for logged out calls) and user (for logged in calls).
type PassphraseManagerEngine interface {
	RegServer() RegServerInterface
	UserServer(uid proto.UID) UserServerInterface
	StretchOpts() StretchOpts
	MakeUserSettingsLink(ctx context.Context, info proto.PassphraseInfo) (*rem.PostGenericLinkArg, error)
	GetUserSettings(ctx context.Context) (*proto.PassphraseInfo, error)
}

type PassphraseManagerInterface interface {
	PME() PassphraseManagerEngine
	RawPassphrase() proto.Passphrase
	PUK() core.SharedPrivateSuiter
	Arg() *rem.SetPassphraseArg
}

func NewPassphraseManager(fqu proto.FQUser) *PassphraseManager {
	return &PassphraseManager{
		user: fqu,
	}
}

func (p *PassphraseManager) Logout() {
	p.Lock()
	defer p.Unlock()
	p.skmwkList = nil
	p.sp = nil
}

func (p *PassphraseManager) CheckFresh(ctx context.Context, pme PassphraseManagerEngine) (bool, error) {
	p.Lock()
	defer p.Unlock()

	if len(p.skmwkList) == 0 || p.sp == nil {
		return false, nil
	}
	usrv := pme.UserServer(p.user.Uid)
	if usrv == nil {
		return false, core.InternalError("no user server")
	}
	gen, err := usrv.NextPassphraseGeneration(ctx)
	if err != nil {
		return false, err
	}

	if len(p.skmwkList) < int(gen) {
		p.skmwkList = nil
		p.sp = nil
		return false, nil
	}

	return true, nil
}

func (p *PassphraseManager) GetSKMWK(g proto.PassphraseGeneration) (*lcl.SKMWK, error) {
	p.RLock()
	defer p.RUnlock()

	if len(p.skmwkList) == 0 {
		return nil, core.NeedLoginError{}
	}
	if !g.IsValid() {
		return nil, core.PassphraseError("invalid passphrase generation passed to GetSKMWK")
	}
	idx := g.ToIndex()
	if idx >= len(p.skmwkList) {
		return nil, core.InternalError("passphrase generation too far ahead")
	}
	if idx < 0 {
		return nil, core.InternalError("passphrase generation underflow")
	}
	ret := p.skmwkList[idx]
	return &ret, nil
}

func (p *PassphraseManager) IsLatest(ppg proto.PassphraseGeneration) bool {
	p.RLock()
	defer p.RUnlock()
	return int(ppg) == len(p.skmwkList)
}

func (p *PassphraseManager) GetLatest() (*lcl.SKMWK, proto.PassphraseGeneration, *proto.PassphraseSalt, proto.StretchVersion) {
	p.RLock()
	defer p.RUnlock()
	var sv proto.StretchVersion
	if len(p.skmwkList) == 0 || (p.sp == nil && p.pi == nil) {
		return nil, proto.PassphraseGeneration(0), nil, sv
	}
	ret := core.Last(p.skmwkList)
	g := proto.PassphraseGenerationFromIndex(len(p.skmwkList) - 1)

	var salt *proto.PassphraseSalt
	if p.sp != nil {
		salt = &p.sp.salt
	} else if p.pi != nil {
		salt = p.pi.Salt
	}
	if p.sp != nil {
		sv = p.sp.svers
	} else if p.pi != nil {
		sv = p.pi.Sv
	}

	return &ret, g, salt, sv
}

func (p *PassphraseManager) SetPassphrase(
	ctx context.Context,
	psi PassphraseManagerEngine,
	raw proto.Passphrase,
	puk core.SharedPrivateSuiter,
) error {

	var salt proto.PassphraseSalt
	ppgen := proto.FirstPassphraseGeneration

	p.Lock()
	defer p.Unlock()

	if p.sp != nil || len(p.skmwkList) > 0 {
		return core.PassphraseError("cannot call SetPassphrase if one is already set")
	}

	usrv := psi.UserServer(p.user.Uid)
	if usrv == nil {
		return core.InternalError("no user server")
	}

	sv, err := usrv.StretchVersion(ctx)
	if err != nil {
		return err
	}

	sopts := psi.StretchOpts()

	err = core.RandomFill(salt[:])
	if err != nil {
		return err
	}

	newSp, err := NewStretchedPassphrase(sopts, raw, salt, ppgen, sv)
	if err != nil {
		return err
	}

	eres, err := p.encrypt([]lcl.SKMWK{}, newSp, puk)
	if err != nil {
		return err
	}
	pgla, err := psi.MakeUserSettingsLink(ctx,
		proto.PassphraseInfo{
			Salt: &salt,
			Sv:   sv,
			Gen:  ppgen,
		},
	)

	if err != nil {
		return err
	}

	ppKey, err := newSp.PublicKeySuite()
	if err != nil {
		return err
	}
	eid := ppKey.GetEntityID()
	err = usrv.SetPassphrase(ctx, rem.SetPassphraseArg{
		Key:              eid,
		Salt:             salt,
		SkwkBox:          eres.skmwkListBox,
		PassphraseBox:    eres.passphraseBox,
		PukBox:           eres.pukBox,
		StretchVersion:   sv,
		UserSettingsLink: pgla,
	})
	if err != nil {
		return err
	}

	p.sp = newSp
	p.skmwkList = eres.lst

	return nil
}

func (p *PassphraseManager) RefreshWithUserSettings(
	ctx context.Context,
	pme PassphraseManagerEngine,
	puks []core.SharedPrivateSuiter,
) error {
	p.Lock()
	defer p.Unlock()
	_, err := p.refreshWithUserSettingsLocked(ctx, pme, puks, false)
	return err
}

func (p *PassphraseManager) refreshWithUserSettingsLocked(
	ctx context.Context,
	pme PassphraseManagerEngine,
	puks []core.SharedPrivateSuiter,
	needRes bool,
) (
	*lcl.UnlockedSKMWK,
	error,
) {
	pi, err := pme.GetUserSettings(ctx)
	if err != nil {
		return nil, err
	}
	n := len(p.skmwkList)
	if !needRes && n > 0 && pi != nil && int(pi.Gen) <= n {
		// Already up-to-date
		return nil, nil
	}
	pres, err := p.refreshSKWKList(ctx, pme, puks)
	if err != nil {
		return nil, err
	}
	n = len(p.skmwkList)
	if pi != nil && int(pi.Gen) > n {
		return nil, core.BadServerDataError("passphrase refresh was stale relative to user settings")
	}
	return pres, nil
}

func (p *PassphraseManager) RotateWithPUK(
	ctx context.Context,
	pme PassphraseManagerEngine,
	puks []core.SharedPrivateSuiter,
) (*rem.SetPassphraseAnnex, error) {
	p.Lock()
	defer p.Unlock()
	pres, err := p.refreshWithUserSettingsLocked(ctx, pme, puks, true)
	if err != nil {
		return nil, err
	}
	ppGen := pres.ExpectedGen + 1

	pub, err := core.ImportPublicSuiterFromHEPK(pres.VerifyKey, &pres.Ppk)
	if err != nil {
		return nil, err
	}
	eres, err := p.encryptWithPassphrasePublicKey(pres.Lst, pub, ppGen, core.Last(puks))
	if err != nil {
		return nil, err
	}
	pgla, err := pme.MakeUserSettingsLink(ctx,
		proto.PassphraseInfo{
			Salt: &pres.Salt,
			Sv:   pres.Sv,
			Gen:  ppGen,
		},
	)
	if err != nil {
		return nil, err
	}
	cparg := rem.ChangePassphraseArg{
		Key:              pres.VerifyKey,
		PpGen:            ppGen,
		SkwkBox:          eres.skmwkListBox,
		PassphraseBox:    eres.passphraseBox,
		PukBox:           eres.pukBox,
		StretchVersion:   pres.Sv,
		UserSettingsLink: pgla,
	}
	return &rem.SetPassphraseAnnex{
		Arg:  cparg,
		Link: *pgla,
	}, nil
}

func (p *PassphraseManager) UnlockWithPUKs(ctx context.Context, psi PassphraseManagerEngine, puks []core.SharedPrivateSuiter) error {
	p.Lock()
	defer p.Unlock()
	_, err := p.refreshSKWKList(ctx, psi, puks)
	if err != nil {
		return err
	}
	return err
}

func (p *PassphraseManager) refreshSKWKList(
	ctx context.Context,
	psi PassphraseManagerEngine,
	puks []core.SharedPrivateSuiter,
) (
	*lcl.UnlockedSKMWK,
	error,
) {
	usrv := psi.UserServer(p.user.Uid)
	if usrv == nil {
		return nil, core.InternalError("no user server")
	}

	res, err := GetUnlockedSKMWK(
		ctx,
		usrv,
		p.user,
		puks,
	)
	if err != nil {
		return nil, err
	}
	p.skmwkList = res.Lst
	p.pi = &proto.PassphraseInfo{
		Salt: &res.Salt,
		Sv:   res.Sv,
		Gen:  res.ExpectedGen,
	}
	return res, nil
}

func GetUnlockedSKMWK(
	ctx context.Context,
	usr UserServerInterface,
	fqu proto.FQUser,
	puks []core.SharedPrivateSuiter,
) (
	*lcl.UnlockedSKMWK,
	error,
) {

	parcel, err := usr.GetPpeParcel(ctx)
	if err != nil {
		return nil, err
	}

	if parcel.PukBox == nil {
		return nil, core.PassphraseError("didn't get PUK-backup for passphrase")
	}

	gen := parcel.PukBox.PukGen
	idx := gen.ToIndex()
	if idx >= len(puks) {
		return nil, core.PassphraseError("went off end of PUK vector")
	}
	if idx < 0 {
		return nil, core.PassphraseError("underflowed the PUK vector")
	}
	puk := puks[idx]
	if puk.Metadata().Gen != gen {
		return nil, core.PassphraseError("PUK generation mismatch")
	}

	sboxkey := puk.SecretBoxKey()
	expectedGen := parcel.PpGen

	var gPayload lcl.PpePUKBoxPayload
	err = core.OpenSecretBoxInto(&gPayload, parcel.PukBox.Box, &sboxkey)
	if err != nil {
		return nil, err
	}
	if !gPayload.Gen.IsValid() {
		return nil, core.PassphraseError("invalid passphrase generation number boxed into gPayload")
	}
	if gPayload.Gen != expectedGen {
		return nil, core.PassphraseError("bad passphrase generation number")
	}

	var lst lcl.SKMWKList
	err = core.OpenSecretBoxInto(&lst, parcel.SkwkBox, (*proto.SecretBoxKey)(&gPayload.Sesskey))
	if err != nil {
		return nil, err
	}
	if !lst.Fqu.Eq(fqu) {
		return nil, core.PassphraseError("SKWK list is for wrong user")
	}
	if len(lst.Keys) != int(parcel.PpGen) {
		return nil, core.PassphraseError(
			fmt.Sprintf("SKWK list has wrong length (%d, expected %d)",
				len(lst.Keys),
				parcel.PpGen,
			),
		)
	}

	return &lcl.UnlockedSKMWK{
		Lst:         lst.Keys,
		Salt:        parcel.Salt,
		ExpectedGen: expectedGen,
		Ppk:         gPayload.Passphrase,
		Sv:          parcel.Sv,
		VerifyKey:   parcel.VerifyKey,
	}, nil

}

func (p *PassphraseManager) ChangePassphraseWithPUK(
	ctx context.Context,
	psi PassphraseManagerEngine,
	raw proto.Passphrase,
	puks []core.SharedPrivateSuiter,
) error {
	p.Lock()
	defer p.Unlock()

	pres, err := p.refreshSKWKList(ctx, psi, puks)
	if err != nil {
		return err
	}

	lastPuk := puks[len(puks)-1]

	err = p.changePassphraseWithLock(
		ctx,
		psi,
		raw,
		lastPuk,
		pres.Salt,
		pres.ExpectedGen,
	)

	if err != nil {
		return err
	}

	return nil
}

func (p *PassphraseManager) ChangePassphrase(
	ctx context.Context,
	psi PassphraseManagerEngine,
	raw proto.Passphrase,
	puk core.SharedPrivateSuiter,
) error {
	p.Lock()
	defer p.Unlock()
	if p.sp == nil {
		return core.PassphraseError("cannot call ChangePassphrase if no current passphrase")
	}
	return p.changePassphraseWithLock(ctx, psi, raw, puk, p.sp.salt, p.sp.ppgen)
}

func (p *PassphraseManager) changePassphraseWithLock(
	ctx context.Context,
	pme PassphraseManagerEngine,
	raw proto.Passphrase,
	puk core.SharedPrivateSuiter,
	salt proto.PassphraseSalt,
	ppgenLast proto.PassphraseGeneration,
) error {

	if len(p.skmwkList) == 0 {
		return core.PassphraseError("cannot call ChangePassphrase if no current passphrase")
	}

	usrv := pme.UserServer(p.user.Uid)
	if usrv == nil {
		return core.InternalError("no user server")
	}

	sv, err := usrv.StretchVersion(ctx)
	if err != nil {
		return err
	}
	newPpgen := ppgenLast + 1
	newSp, err := NewStretchedPassphrase(pme.StretchOpts(), raw, salt, newPpgen, sv)
	if err != nil {
		return err
	}

	lstCopy := append([]lcl.SKMWK{}, p.skmwkList...)

	eres, err := p.encrypt(lstCopy, newSp, puk)
	if err != nil {
		return err
	}

	pgla, err := pme.MakeUserSettingsLink(ctx,
		proto.PassphraseInfo{
			Salt: &salt,
			Sv:   sv,
			Gen:  newPpgen,
		},
	)

	if err != nil {
		return err
	}

	newPPKey, err := newSp.PublicKeySuite()
	if err != nil {
		return err
	}
	eid := newPPKey.GetEntityID()
	err = usrv.ChangePassphrase(ctx, rem.ChangePassphraseArg{
		Key:              eid,
		PpGen:            newSp.ppgen,
		SkwkBox:          eres.skmwkListBox,
		PassphraseBox:    eres.passphraseBox,
		PukBox:           eres.pukBox,
		StretchVersion:   sv,
		UserSettingsLink: pgla,
	})
	if err != nil {
		return err
	}

	p.sp = newSp
	p.skmwkList = eres.lst

	return nil
}

type encryptRes struct {
	lst           []lcl.SKMWK
	skmwkListBox  proto.SecretBox
	passphraseBox proto.PpePassphraseBox
	pukBox        *proto.PpePUKBox
}

func (p *PassphraseManager) encrypt(
	lst []lcl.SKMWK,
	newSp *StretchedPassphrase,
	puk core.SharedPrivateSuiter,
) (
	*encryptRes,
	error,
) {

	pk, err := newSp.PublicKeySuite()
	if err != nil {
		return nil, err
	}
	return p.encryptWithPassphrasePublicKey(
		lst,
		pk,
		newSp.ppgen,
		puk,
	)
}

func (p *PassphraseManager) encryptWithPassphrasePublicKey(
	lst []lcl.SKMWK,
	pk core.PublicSuiter,
	ppgen proto.PassphraseGeneration,
	puk core.SharedPrivateSuiter,
) (
	*encryptRes,
	error,
) {
	if !ppgen.IsValid() {
		return nil, core.PassphraseError("invalid passphrase generation in encryptWithPassphrasePublicKey")
	}

	var skmwk lcl.SKMWK
	var sessionKey lcl.PpeSessionKey

	t, err := pk.Ephemeral()
	if err != nil {
		return nil, err
	}

	err = core.RandomFill(skmwk[:])
	if err != nil {
		return nil, err
	}
	err = core.RandomFill(sessionKey[:])
	if err != nil {
		return nil, err
	}

	lst = append(lst, skmwk)

	payload := lcl.SKMWKList{
		Fqu:  p.user,
		Keys: lst,
	}

	// Encrypt e_i (see ppe.snowp)
	sbox, err := core.SealIntoSecretBox(&payload, (*proto.SecretBoxKey)(&sessionKey))
	if err != nil {
		return nil, err
	}
	ret := encryptRes{lst: lst, skmwkListBox: *sbox}

	// f_i payload
	fPayload := lcl.PpePassphraseBoxPayload{
		Gen:     ppgen,
		Sesskey: sessionKey,
	}

	fBox, err := t.BoxFor(
		&fPayload,
		pk,
		core.BoxOpts{IncludePublicKey: true},
	)
	if err != nil {
		return nil, err
	}

	ret.passphraseBox.Box = *fBox

	// If no PUK, we can't make a backup box, so we can leave right here
	if puk == nil {
		return &ret, nil
	}

	hepk, err := pk.ExportHEPK()
	if err != nil {
		return nil, err
	}

	// g_i payload, includes the passphrase (in public key form) so we can rotate
	// the session key later without prompting for the passphrase.
	gPayload := lcl.PpePUKBoxPayload{
		Gen:        ppgen,
		Sesskey:    sessionKey,
		Passphrase: *hepk,
	}

	pukKey := puk.SecretBoxKey()
	gBox, err := core.SealIntoSecretBox(&gPayload, &pukKey)
	if err != nil {
		return nil, err
	}

	ret.pukBox = &proto.PpePUKBox{
		Box:     *gBox,
		PukGen:  puk.Metadata().Gen,
		PukRole: puk.GetRole(),
	}

	return &ret, nil
}

func (p *PassphraseManager) Login(
	ctx context.Context,
	psi PassphraseManagerEngine,
	raw proto.Passphrase,
	salt *proto.PassphraseSalt,
	sv *proto.StretchVersion,
) error {

	var err error

	// If the user set a passphrase on machine A but then moved to machine B,
	// or if the user on machine A had a passphrase lock and then removed it,
	// the client might not know the salt. The server does. In either case,
	// the client has an unlocked deviceKey so can grab the salt from the
	// server. Note that in Keybase, salts were given out publicly, but here
	// we'd like to avoid that if possible.
	if salt == nil {
		usrv := psi.UserServer(p.user.Uid)
		if usrv == nil {
			return core.InternalError("no user server")
		}
		tmp, err := usrv.GetSalt(ctx)
		if err != nil {
			return err
		}
		salt = &tmp
	}

	// Same as above for stretch version
	if sv == nil {
		usrv := psi.UserServer(p.user.Uid)
		if usrv == nil {
			return core.InternalError("no user server")
		}
		tmp, err := usrv.StretchVersion(ctx)
		if err != nil {
			return err
		}
		sv = &tmp
	}

	chal, err := psi.RegServer().GetLoginChallenge(ctx, p.user.Uid)
	if err != nil {
		return err
	}

	p.Lock()
	defer p.Unlock()

	var ppgen proto.PassphraseGeneration
	sp, err := NewStretchedPassphrase(psi.StretchOpts(), raw, *salt, ppgen, *sv)
	if err != nil {
		return err
	}

	priv, err := sp.SecretKeySuite()
	if err != nil {
		return err
	}

	chal, err = psi.RegServer().GetLoginChallenge(ctx, p.user.Uid)
	if err != nil {
		return err
	}
	sig, err := priv.Sign(&chal.Payload)
	if err != nil {
		return err
	}

	res, err := psi.RegServer().Login(ctx, rem.LoginArg{
		Challenge: chal,
		Uid:       p.user.Uid,
		Signature: *sig,
	})

	if err != nil {
		return err
	}

	var fPayload lcl.PpePassphraseBoxPayload
	err = priv.UnboxForIncludedEphemeral(&fPayload, res.PassphraseBox.Box)
	if err != nil {
		return err
	}
	if fPayload.Gen != res.PpGen {
		return core.PassphraseError("bad passphrase generation")
	}

	var ePayload lcl.SKMWKList
	err = core.OpenSecretBoxInto(&ePayload, res.SkwkBox, (*proto.SecretBoxKey)(&fPayload.Sesskey))
	if err != nil {
		return err
	}
	eq, err := core.Eq(&ePayload.Fqu, &p.user)
	if err != nil {
		return err
	}
	if !eq {
		return core.WrongUserError{}

	}
	lst := ePayload.Keys

	if len(lst) == 0 {
		return core.BadServerDataError("did not expect 0-length SKMWK list")
	}

	p.skmwkList = lst
	sp.ppgen = res.PpGen
	p.sp = sp
	return nil
}
