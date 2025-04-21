// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lib"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

// YMKMgr is a manager for YubiKey Management Keys (YMKs). It's used to store
// encryptions of the Yubi Management keys to the server, encrypted with the user's PUK.
// This allows the user to recover their management key if they ever botch their PIN
// enough times to lock. BTW, "PUK" here refers to the FOKS notion of PUK and not
// YubiKey's PUK. We largely ignore the yubi PUK since it too has the constraint that it
// can be locked out after enough failure attempts.

type ymkMgrKey struct {
	raw rem.YubiEncryptedManagementKey

	ymk  *proto.YubiManagementKey
	card proto.YubiCardID
	slot proto.YubiSlot
}

type YMKPull struct {
	uc *UserContext // one per user-context

	// usually a 1:1 map, but the same user might have multiple yubikeys.
	// some of them might not be currently available
	ymks []*ymkMgrKey
}

type YMKPushOutcome int

const (
	YMKPushOutcomeNone YMKPushOutcome = iota
	YMKPushOutcomeFresh
	YMKPushOutcomeNew
	YMKPushOutcomeNotAdmin
	YMKPushOutcomeNoManagementKey
	YMKPushOutcomeRefreshed
	YMKPushOutcomeErr
)

type YMKPush struct {
	uc *UserContext

	noPUKRefresh bool
	latestPUKgen proto.Generation
	pushYmk      *ymkMgrKey
	role         proto.Role
	stale        bool
	puk          core.SharedPrivateSuiter
	yi           *proto.YubiKeyInfoHybrid
	outcome      YMKPushOutcome
}

func NewYMKPush(uc *UserContext) *YMKPush {
	return &YMKPush{
		uc: uc,
	}
}

func (y *YMKPush) WithoutPUKRefresh() *YMKPush {
	y.noPUKRefresh = true
	return y
}

func (y *YMKPush) WithLatestPUKGen(g proto.Generation) *YMKPush {
	y.latestPUKgen = g
	return y
}

func NewYMKPull(uc *UserContext) *YMKPull {
	return &YMKPull{
		uc: uc,
	}
}

func (y *YMKPull) loadFromServer(m MetaContext) error {
	cli, err := y.uc.UserClient(m)
	if err != nil {
		return err
	}
	res, err := cli.GetAllYubiManagementKeys(m.Ctx())
	if err != nil {
		return err
	}
	y.ymks = make([]*ymkMgrKey, len(res))
	for i, ymk := range res {
		y.ymks[i] = &ymkMgrKey{
			raw: ymk,
		}
	}
	return nil
}

func (y *ymkMgrKey) unbox(m MetaContext, uc *UserContext) error {
	gt, err := y.raw.Role.GreaterThan(uc.Role())
	if err != nil {
		return err
	}
	if gt {
		return core.PermissionError("cannot unbox ymk with lower role")
	}
	pm := NewPUKMinder(uc)
	puk, err := pm.GetPUKAtRoleAndGeneration(m, y.raw.Role, y.raw.Gen)
	if err != nil {
		return err
	}

	key := puk.SecretBoxKey()

	var payload lib.YubiManagementKeyBoxPayload
	err = core.OpenSecretBoxInto(&payload, y.raw.Box, &key)
	if err != nil {
		return err
	}
	if !y.raw.Yk.Eq(payload.Yk) {
		return core.KeyMismatchError{}
	}
	y.ymk = &payload.Mk
	y.card = payload.Card
	y.slot = payload.Slot

	return nil
}

func (y *YMKPull) unbox(m MetaContext) error {
	for _, ymk := range y.ymks {
		err := ymk.unbox(m, y.uc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (y *YMKPull) run(m MetaContext) error {

	err := y.loadFromServer(m)
	if err != nil {
		return err
	}

	err = y.unbox(m)
	if err != nil {
		return err
	}

	return nil
}

func (y *YMKPull) Recover(
	m MetaContext,
	id proto.YubiCardID,
) (
	*proto.YubiManagementKey,
	error,
) {
	err := y.run(m)
	if err != nil {
		return nil, err
	}
	for _, ymk := range y.ymks {
		if ymk.card.Eq(id) {
			return ymk.ymk, nil
		}
	}
	return nil, core.KeyNotFoundError{Which: "YubiManagementKey"}
}

func (y *YMKPush) syncOne(m MetaContext) error {
	cli, err := y.uc.UserClient(m)
	if err != nil {
		return err
	}
	id := y.yi.Key.Id
	res, err := cli.GetYubiManagementKey(m.Ctx(), id)

	if _, ok := err.(core.KeyNotFoundError); ok {
		m.Infow("YMKMgr.syncOne", "id", id, "status", "not_found")
		return nil
	}

	if err != nil {
		return err
	}
	y.pushYmk = &ymkMgrKey{
		raw: res,
	}
	err = y.pushYmk.unbox(m, y.uc)
	if err != nil {
		return err
	}
	return nil
}

func (y *YMKPush) getPUK(
	m MetaContext,
	role proto.Role,
) (
	core.SharedPrivateSuiter,
	error,
) {

	eq, err := role.Eq(y.role)
	if err != nil {
		return nil, err
	}
	latest := y.uc.PrivKeys.LatestPuk()
	var ret core.SharedPrivateSuiter

	if eq && y.noPUKRefresh && latest != nil {
		ret = latest
	} else if eq && y.latestPUKgen.IsValid() &&
		latest != nil && latest.Metadata().Gen == y.latestPUKgen {
		ret = latest
	} else {
		pm := NewPUKMinder(y.uc)
		pks, err := pm.GetPUKSetForRole(m, role)
		if err != nil {
			return nil, err
		}
		ret = pks.Current()
	}
	if ret == nil {
		return nil, core.KeyNotFoundError{Which: "latest PUK"}
	}
	return ret, nil
}

func (y *YMKPush) setupNew(m MetaContext) error {

	latest, err := y.getPUK(m, y.role)
	if err != nil {
		return err
	}
	y.pushYmk = &ymkMgrKey{}
	y.puk = latest
	return nil
}

func (y *YMKPush) checkEncFresh(m MetaContext, mk *proto.YubiManagementKey) error {
	if y.pushYmk == nil {
		return core.InternalError("no push ymk")
	}
	latest, err := y.getPUK(m, y.pushYmk.raw.Role)
	if err != nil {
		return err
	}
	gen := latest.Metadata().Gen

	// Reencrypt in 2 scenarios: (1) if the PUK has been rolled/upgraded;
	// or (2) if the YubiManagementKey has changed.
	if gen > y.pushYmk.raw.Gen || !mk.Eq(*y.pushYmk.ymk) {
		y.stale = true
		y.puk = latest
	}

	return nil
}

func (y *YMKPush) box(m MetaContext, mk *lib.YubiManagementKey) error {
	pylod := lib.YubiManagementKeyBoxPayload{
		Mk:   *mk,
		Card: y.yi.Card,
		Slot: y.yi.Key.Slot,
		Yk:   y.yi.Key.Id,
	}
	key := y.puk.SecretBoxKey()
	enc, err := core.SealIntoSecretBox(&pylod, &key)
	if err != nil {
		return err
	}
	y.pushYmk.raw = rem.YubiEncryptedManagementKey{
		Box:  *enc,
		Role: y.puk.GetRole(),
		Gen:  y.puk.Metadata().Gen,
		Yk:   y.yi.Key.Id,
	}

	// Set internal fields as if unbox has just happened
	y.pushYmk.card = y.yi.Card
	y.pushYmk.slot = y.yi.Key.Slot
	y.pushYmk.ymk = mk

	return nil
}

func (y *YMKPush) push(m MetaContext) error {
	cli, err := y.uc.UserClient(m)
	if err != nil {
		return err
	}
	err = cli.PutYubiManagementKey(m.Ctx(), y.pushYmk.raw)
	if err != nil {
		return err
	}
	return nil
}

func (y *YMKPush) Run(m MetaContext) (err error) {

	m = m.WithLogTag("ymkpush")
	defer func() {
		if err != nil {
			y.outcome = YMKPushOutcomeErr
			m.Warnw("YMKPush", "err", err)
		} else {
			m.Infow("YMKPush", "status", "success")
		}
	}()
	yi := y.uc.Info.YubiInfo
	y.yi = yi

	if yi == nil {
		m.Infow("YMKPush", "exit", "no_yubi_info")
		return nil
	}

	m.Infow("YMKPush", "ymk", yi.Key.Id, "slot", yi.Key.Slot)

	var admin bool
	role := y.uc.Role()
	admin, err = role.IsAdminOrAbove()
	if err != nil {
		return err
	}
	y.role = role

	if !admin {
		m.Infow("YMKPush", "exit", "no admin")
		y.outcome = YMKPushOutcomeNotAdmin
		return nil
	}

	mk, err := m.G().YubiDispatch().GetManagementKey(m.Ctx(), yi.Card)
	if err != nil {
		return err
	}

	if mk == nil {
		m.Infow("YMKPush", "exit", "no mk")
		y.outcome = YMKPushOutcomeNoManagementKey
		return nil
	}

	err = y.syncOne(m)
	if err != nil {
		return err
	}

	outcome := YMKPushOutcomeNew

	if y.pushYmk != nil {

		err = y.checkEncFresh(m, mk)
		if err != nil {
			return err
		}

		if !y.stale {
			m.Infow("YMKPush", "exit", "not stale")
			y.outcome = YMKPushOutcomeFresh
			return nil
		}
		outcome = YMKPushOutcomeRefreshed

	} else {
		err = y.setupNew(m)
		if err != nil {
			return err
		}
	}

	err = y.box(m, mk)
	if err != nil {
		return err
	}

	err = y.push(m)
	if err != nil {
		return err
	}
	y.outcome = outcome

	return nil
}

func (y *YMKPush) Outcome() YMKPushOutcome {
	return y.outcome
}

func (u *UserContext) SyncYubiManagementKey(m MetaContext) error {
	return NewYMKPush(u).WithoutPUKRefresh().Run(m)
}

func BgRotateYubiManagementKey(m MetaContext, uc *UserContext, uw *UserWrapper) error {
	g, err := uw.LatestPUKGenForRole(uc.Role())
	if err != nil {
		return err
	}
	return NewYMKPush(uc).WithLatestPUKGen(g).Run(m)
}

// RecoverYubiManagementKey resets the YubiKey PIN and PUK to the new values. It needs
// a management key to do this. If the user didn't specify a management key (passed in
// via mk), then we try to recover it via the server-sync method. If that fails, we return an
// error. Requires the YubiKey to be present, and the user to be logged in and active.
func RecoverYubiManagementKey(
	m MetaContext,
	serial proto.YubiSerial,
	newPin proto.YubiPIN,
	newPuk proto.YubiPUK,
	mk *proto.YubiManagementKey,
) error {
	disp := m.G().YubiDispatch()
	cardID, err := disp.FindCardIDBySerial(m.Ctx(), serial)
	if err != nil {
		return err
	}

	// If the user didn't specify a management key, then we have to
	// try to recover it via the server-sync method.
	if mk == nil {
		au := m.G().ActiveUser()
		if au == nil {
			return core.NoActiveUserError{}
		}
		tmp, err := NewYMKPull(au).Recover(m, *cardID)
		if err != nil {
			return err
		}
		mk = tmp
	}

	if mk == nil {
		return core.InternalError("no management key")
	}

	err = disp.ResetPINandPUK(m.Ctx(), *cardID, *mk, newPin, newPuk)
	if err != nil {
		return err
	}
	return nil
}
