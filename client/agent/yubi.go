// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/client/libyubi"
	"github.com/foks-proj/go-foks/lib/chains"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	"github.com/foks-proj/go-foks/proto/lib"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

// YSPP = Yubi Set PIN and PUK
type YSPPSession struct {
	SessionBase
	id      proto.UISessionID
	currPUK *proto.YubiPUK
	mgmtKey *proto.YubiManagementKey
}

func (y *YSPPSession) Init(id proto.UISessionID) {
	y.SessionBase.Init()
	y.id = id
}

func LookupUsersForYubikey(m libclient.MetaContext) error {
	return nil
}

func (a *AgentConn) YubiUnlock(ctx context.Context) error {
	m := a.MetaContext(ctx)
	return a.yubiUnlock(m)
}

func (a *AgentConn) yubiUnlock(m libclient.MetaContext) error {
	u, err := m.ActiveConnectedUser(nil)
	if err != nil {
		return err
	}
	err = u.YubiUnlock(m)
	if err != nil {
		return err
	}
	err = u.PopulateWithDevkey(m)
	if err != nil {
		return err
	}
	err = u.SyncYubiManagementKey(m)
	if err != nil {
		return err
	}

	return nil
}

func (a *AgentConn) YubiListAllCards(ctx context.Context) ([]proto.YubiCardID, error) {
	m := a.MetaContext(ctx)
	v, err := m.G().YubiDispatch().ListCards(ctx)
	return v, err
}

func (a *AgentConn) YubiListAllSlots(ctx context.Context, serial proto.YubiSerial) (lcl.ListYubiSlotsRes, error) {
	var zed lcl.ListYubiSlotsRes
	m := a.MetaContext(ctx)
	card, err := m.G().YubiDispatch().FindCardBySerial(ctx, serial)
	if err != nil {
		return zed, err
	}
	ret := lcl.ListYubiSlotsRes{Device: card}
	return ret, nil
}

func (a *AgentConn) YubiMapSlotToUser(ctx context.Context, arg proto.YubiSerialSlotHost) (proto.LookupUserRes, error) {
	res, err := a.yubiMapSlotToUser(ctx, arg)
	if err != nil {
		return proto.LookupUserRes{}, err
	}
	return res.lur, nil
}

type YubiMapSlotToUserRes struct {
	lur proto.LookupUserRes
	ks  *libyubi.KeySuiteHybrid
	pr  *chains.Probe
}

func (a *AgentConn) yubiMapSlotToUser(
	ctx context.Context,
	arg proto.YubiSerialSlotHost,
) (
	*YubiMapSlotToUserRes,
	error,
) {

	var res YubiMapSlotToUserRes

	if arg.Host == "" {
		arg.Host = a.g.Cfg().HostsProbe()
	}
	if arg.Host == "" {
		return nil, core.NoDefaultHostError{}
	}

	m := a.MetaContext(ctx)
	pr, err := m.G().ProbeByAddr(m.Ctx(), arg.Host, 0)
	if err != nil {
		return nil, err
	}

	card, err := m.G().YubiDispatch().FindCardBySerial(ctx, arg.Serial)
	if err != nil {
		return nil, err
	}
	key, err := card.KeyAtSlot(arg.Slot)
	if err != nil {
		return nil, err
	}

	// Role doesn't matter here, so populate with owner role
	hostid := pr.Chain().HostID()
	ykClassical, err := m.G().YubiDispatch().Load(ctx, *key, proto.OwnerRole, hostid)
	if err != nil {
		return nil, err
	}

	err = pr.WithRegCli(m, func(regcli rem.RegClient) error {
		tmp, err := lookupDeviceOnServer(m, ykClassical, regcli, hostid)
		if err != nil {
			return err
		}
		res.lur = *tmp
		return nil
	})

	if err != nil {
		return nil, err
	}

	info, err := ykClassical.ExportToYubiKeyInfo(ctx)
	if err != nil {
		return nil, err
	}

	if res.lur.YubiPQHint == nil {
		return nil, core.YubiError("pq key hints not found on server")
	}

	info.PqKey = *res.lur.YubiPQHint

	kspq, err := m.G().YubiDispatch().LoadPQ(ctx, *info)
	if err != nil {
		return nil, err
	}
	hybrid := ykClassical.Fuse(kspq)
	res.ks = hybrid
	res.pr = pr
	return &res, nil
}

func populateUserContext(
	lur *proto.LookupUserRes,
	pr *chains.Probe,
	dev core.PrivateSuiter,
	kg proto.KeyGenus,
	yinf *proto.YubiKeyInfoHybrid,
) (*libclient.UserContext, error) {
	keyID, err := dev.EntityID()
	if err != nil {
		return nil, err
	}
	uctx := &libclient.UserContext{
		Info: proto.UserInfo{
			Fqu: lur.Fqu,
			Username: proto.NameBundle{
				Name:     lur.Username,
				NameUtf8: lur.UsernameUtf8,
			},
			HostAddr: pr.CanonicalAddr(),
			Role:     lur.Role,
			KeyGenus: kg,
			Key:      keyID,
		},
		PrivKeys: libclient.UserPrivateKeys{
			Devkey: dev,
		},
	}
	if yinf != nil {
		uctx.Info.YubiInfo = yinf
		uctx.Yubi = dev
	}
	uctx.SetHomeServer(pr)
	return uctx, nil
}

func (a *AgentConn) YubiProvision(ctx context.Context, arg proto.YubiSerialSlotHost) error {
	res, err := a.yubiMapSlotToUser(ctx, arg)
	if err != nil {
		return err
	}

	yinf, err := res.ks.ExportToYubiKeyInfo(ctx)
	if err != nil {
		return err
	}

	uctx, err := populateUserContext(&res.lur, res.pr, res.ks, proto.KeyGenus_Yubi, yinf)
	if err != nil {
		return err
	}

	m := a.MetaContext(ctx)
	err = libclient.SetActiveUser(m, uctx)
	if err != nil {
		return err
	}
	err = uctx.PopulateWithDevkey(m)
	if err != nil {
		return err
	}

	return nil
}

func prepareNewYubi(
	m libclient.MetaContext,
	au *libclient.UserContext,
	arg lcl.YubiNewArg,
	hostid proto.HostID,
) (
	*libyubi.KeySuiteHybrid,
	error,
) {
	ret, err := (&libyubi.Prepper{
		Disp:        m.G().YubiDispatch(),
		Host:        hostid,
		Role:        arg.Role,
		Serial:      arg.Ss.Serial,
		Slot:        arg.Ss.Slot,
		PQSlot:      arg.PqSlot,
		Pin:         arg.Pin,
		LockWithPIN: arg.LockWithPin,
	}).Run(m.Ctx())
	if err != nil {
		return nil, err
	}

	rcli, err := au.RegClient(m)
	if err != nil {
		return nil, err
	}
	_, err = lookupDeviceOnServer(m, &ret.KeySuite, *rcli, hostid)
	if err == nil {
		return nil, core.KeyInUseError{}
	}
	if !errors.Is(err, core.KeyNotFoundError{}) {
		return nil, err
	}
	return ret, nil
}

type prepareNewDeviceRes struct {
	au     *libclient.UserContext
	devKey core.PrivateSuiter
	tok    proto.PermissionToken
	lke    *loopbackKexEngine
	hostID proto.HostID
}

func (a *AgentConn) prepareNewDevice(
	m libclient.MetaContext,
	role proto.Role,
) (
	*prepareNewDeviceRes,
	error,
) {
	au, err := m.ActiveConnectedUser(nil)
	if err != nil {
		return nil, err
	}

	gt, err := role.GreaterThan(au.Role())
	if err != nil {
		return nil, err
	}
	if gt {
		return nil, core.RoleError("cannot provision a key with a role greater than current active user")
	}
	hostid := au.HostID()

	lke := newLoopbackKexEngine(m.G(), au)
	err = lke.init(m)
	if err != nil {
		return nil, err
	}

	devkey, err := au.Devkey(m.Ctx())
	if err != nil {
		return nil, err
	}

	tok, err := core.NewPermissionToken()
	if err != nil {
		return nil, err
	}
	return &prepareNewDeviceRes{
		au:     au,
		devKey: devkey,
		tok:    tok,
		lke:    lke,
		hostID: hostid,
	}, nil

}

func (a *AgentConn) YubiNew(ctx context.Context, arg lcl.YubiNewArg) error {

	m := a.MetaContext(ctx)

	return a.yubiNew(m,
		arg.Role,
		arg.Dln,
		func(r *prepareNewDeviceRes) (*libyubi.KeySuiteHybrid, error) {
			return prepareNewYubi(m, r.au, arg, r.hostID)
		},
	)
}

func (a *AgentConn) yubiNew(
	m libclient.MetaContext,
	role proto.Role,
	dln proto.DeviceLabelAndName,
	prep func(r *prepareNewDeviceRes) (*libyubi.KeySuiteHybrid, error),
) error {

	r, err := a.prepareNewDevice(m, role)
	if err != nil {
		return err
	}

	ykw, err := prep(r)
	if err != nil {
		return err
	}
	subkey, subkeyBox, err := core.MakeSubkey(ykw, r.hostID)
	if err != nil {
		return err
	}
	yinf, err := ykw.ExportToYubiKeyInfo(m.Ctx())
	if err != nil {
		return err
	}

	lkr := libclient.NewLoopbackKexRunner(
		m.G(),
		r.devKey,
		ykw,
		r.au.FQU(),
		role,
		dln,
		r.lke,
		r.tok,
	).WithSubkey(
		subkey,
		subkeyBox,
	).WithPQHint(
		&yinf.PqKey,
	)
	err = lkr.Run(m.Ctx())
	if err != nil {
		return err
	}

	return nil
}

func (a *AgentConn) ValidateCurrentPIN(
	ctx context.Context,
	arg lcl.ValidateCurrentPINArg,
) error {
	m := a.MetaContext(ctx)
	return a.withBaseSession(arg.SessionId, func(sess *SessionBase) error {
		if sess.activeYubiDevice == nil {
			return core.InternalError("no card in session")
		}
		pin := arg.Pin
		err := m.G().YubiDispatch().ValidatePIN(
			ctx,
			sess.activeYubiDevice.Id,
			pin,
			arg.DoUnlock,
		)
		if err != nil {
			return err
		}
		sess.currPIN = &pin
		return nil
	})
}

func (a *AgentConn) ValidateCurrentPUK(
	ctx context.Context,
	arg lcl.ValidateCurrentPUKArg,
) error {
	m := a.MetaContext(ctx)
	return withGenericSession(a, arg.SessionId, func(sess *YSPPSession) error {
		if sess.activeYubiDevice == nil {
			return core.InternalError("no card in session")
		}
		puk := arg.Puk
		err := m.G().YubiDispatch().ValidatePUK(ctx, sess.activeYubiDevice.Id, puk)
		if err != nil {
			return err
		}
		sess.currPUK = &puk
		return nil
	})
}

func (a *AgentConn) SetPIN(
	ctx context.Context,
	arg lcl.SetPINArg,
) error {
	m := a.MetaContext(ctx)
	return withGenericSession(a, arg.SessionId, func(sess *YSPPSession) error {
		if sess.activeYubiDevice == nil {
			return core.InternalError("no card in session")
		}
		if sess.currPIN == nil {
			return core.InternalError("no current pin in session")
		}
		pin := arg.Pin
		err := m.G().YubiDispatch().SetPIN(ctx, sess.activeYubiDevice.Id, *sess.currPIN, pin)
		if err != nil {
			return err
		}
		sess.currPIN = &pin
		return nil
	})
}

func (a *AgentConn) SetPUK(
	ctx context.Context,
	arg lcl.SetPUKArg,
) error {
	m := a.MetaContext(ctx)
	return withGenericSession(a, arg.SessionId, func(sess *YSPPSession) error {
		if sess.activeYubiDevice == nil {
			return core.InternalError("no card in session")
		}
		if sess.currPUK == nil {
			return core.InternalError("no current PUK in session")
		}
		return m.G().YubiDispatch().SetPUK(ctx, sess.activeYubiDevice.Id, *sess.currPUK, arg.New)
	})
}

func (c *AgentConn) ListAllLocalYubiDevices(ctx context.Context, arg proto.UISessionID) ([]proto.YubiCardID, error) {
	m := c.MetaContext(ctx)
	sess := c.agent.sessions.Get(arg)
	if sess == nil {
		return nil, core.SessionNotFoundError(arg)
	}
	v, err := m.G().YubiDispatch().ListCards(ctx)

	// If we get a YubiBusError, that likely means on linux that pcscd is not running.
	// For now, just swallow the error so that signup can continue, though we can consider
	// better online documentation here.
	if ybe, ok := err.(core.YubiBusError); ok {
		m.Warnw("ListAllLocalYubiDevices", "err", ybe.Err, "type", "yubi bus error")
		err = nil
	}
	if err != nil {
		return nil, err
	}
	sess.Base().yubiDevices = v
	return v, nil
}

func (c *AgentConn) UseYubi(ctx context.Context, arg lcl.UseYubiArg) error {
	m := c.MetaContext(ctx)
	m.Infow("UseYubi", "arg", fmt.Sprintf("%+v", arg))

	return c.withBaseSession(arg.SessionId, func(sess *SessionBase) error {
		idx := int(arg.Idx)
		if idx > len(sess.yubiDevices) {
			return core.InternalError("yubi index out of range")
		}
		id := sess.yubiDevices[idx]
		ydisp := m.G().YubiDispatch()
		chosen, err := ydisp.Explore(ctx, id)
		if err != nil {
			return err
		}
		sess.activeYubiDevice = chosen
		m.Infow("picked yubi device", "yubi", chosen, "session", arg.SessionId)
		return nil
	})
}

func (c *AgentConn) SetOrGetManagementKey(
	ctx context.Context,
	id lib.UISessionID,
) (
	lcl.SetOrGetManagementKeyRes,
	error,
) {
	var ret lcl.SetOrGetManagementKeyRes
	m := c.MetaContext(ctx)
	err := withGenericSession(c, id, func(sess *YSPPSession) error {
		if sess.activeYubiDevice == nil {
			return core.InternalError("no card in session")
		}
		if sess.currPIN == nil {
			return core.InternalError("no current pin in session")
		}
		mk, wasMade, err := m.G().YubiDispatch().SetOrGetManagementKey(
			ctx,
			sess.activeYubiDevice.Id,
			*sess.currPIN,
		)
		if err != nil {
			return err
		}
		sess.mgmtKey = mk
		ret.Key = *mk
		ret.WasMade = wasMade
		return nil

	})
	return ret, err
}

func (c *AgentConn) InputPIN(
	ctx context.Context,
	arg lcl.InputPINArg,
) (
	proto.ManagementKeyState,
	error,
) {
	var mks proto.ManagementKeyState
	var ycid *proto.YubiCardID
	m := c.MetaContext(ctx)

	if arg.SessionId.IsZero() {
		au := m.G().ActiveUser()
		if au == nil {
			return mks, core.NoActiveUserError{}
		}
		yi := au.Info.YubiInfo
		if yi == nil {
			return mks, core.KeyNotFoundError{Which: "yubi"}
		}
		ycid = &yi.Card
	} else {
		err := c.withBaseSession(arg.SessionId, func(sess *SessionBase) error {
			if sess.activeYubiDevice == nil {
				return core.InternalError("no actie yubi device to set pin on")
			}
			ycid = &sess.activeYubiDevice.Id
			return nil
		})
		if err != nil {
			return mks, err
		}
	}

	if ycid == nil {
		return mks, core.InternalError("no yubi device in session")
	}

	mks, err := m.G().YubiDispatch().InputPIN(ctx, *ycid, arg.Pin)
	if err != nil {
		return mks, err
	}
	return mks, nil
}

func (c *AgentConn) ManagementKeyState(
	ctx context.Context,
	id lib.UISessionID,
) (
	proto.ManagementKeyState,
	error,
) {
	var ret proto.ManagementKeyState
	err := c.withBaseSession(id, func(sess *SessionBase) error {
		if sess.activeYubiDevice == nil {
			return core.InternalError("no active yubi device to check management key")
		}
		ret = sess.activeYubiDevice.Mks
		return nil
	})
	return ret, err
}

func (c *AgentConn) ProtectKeyWithPIN(
	ctx context.Context,
	id lib.UISessionID,
) error {
	return c.withBaseSession(id, func(sess *SessionBase) error {
		if sess.activeYubiDevice == nil {
			return core.InternalError("no active yubi device to set pin on")
		}
		sess.doPinProtect = true
		return nil
	})
}

func (c *AgentConn) RecoverManagementKey(
	ctx context.Context,
	arg lcl.RecoverManagementKeyArg,
) error {
	m := c.MetaContext(ctx)
	return libclient.RecoverYubiManagementKey(
		m,
		arg.Serial,
		arg.Pin,
		arg.Puk,
		arg.Mk,
	)
}

var _ lcl.YubiInterface = (*AgentConn)(nil)
