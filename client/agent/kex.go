// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import (
	"context"
	"errors"
	"sync"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/lib/merkle"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type kexProvisioneeEngine struct {
	kexCommonEngine
	fqr    proto.FQUserAndRole
	newKey *core.PrivateSuite25519
}

type kexCommonEngine struct {
	c             *AgentConn
	sess          *SessionBase
	displayCh     chan proto.KexSecret
	inputCh       chan proto.KexSecret
	inputCheckCh  chan<- error
	cancelInputCh chan error
	runCh         <-chan error
}

func newKexCommonEngine(c *AgentConn, s *SessionBase) *kexCommonEngine {
	return &kexCommonEngine{
		c:             c,
		sess:          s,
		displayCh:     make(chan proto.KexSecret),
		inputCh:       make(chan proto.KexSecret),
		cancelInputCh: make(chan error, 1),
	}
}

func newKexProvisioneeEngine(c *AgentConn, s *SignupSession, key *core.PrivateSuite25519) *kexProvisioneeEngine {
	return &kexProvisioneeEngine{
		kexCommonEngine: *newKexCommonEngine(c, &s.SessionBase),
		newKey:          key,
	}
}

func (e *kexCommonEngine) Router() libclient.KexRouter { return e }

func (e *kexCommonEngine) kexCli(m libclient.MetaContext) (rem.KexClient, error) {
	var ret rem.KexClient
	gcli, err := e.sess.RegGCli(m)
	if err != nil {
		return ret, err
	}
	return rem.KexClient{Cli: gcli, ErrorUnwrapper: core.StatusToError}, nil
}

func (e *kexCommonEngine) Send(ctx context.Context, arg rem.SendArg) error {
	m := e.c.MetaContext(ctx)
	cli, err := e.kexCli(m)
	if err != nil {
		m.Warnw("kexProvisionEngine.Send", "err", err, "stage", "kexCli")
		return err
	}

	select {
	case <-m.Ctx().Done():
		m.Warnw("kexProvisionEngine.Send", "ctx", "done even before send")
		return core.InternalError("ctx done before send")
	default:
	}

	err = cli.Send(m.Ctx(), arg)
	if err != nil {
		m.Warnw("kexProvisionEngine.Send", "err", err, "stage", "Send")
		return err
	}
	return nil
}

func (e *kexCommonEngine) Receive(ctx context.Context, arg rem.ReceiveArg) (rem.KexWrapperMsg, error) {
	m := e.c.MetaContext(ctx)
	cli, err := e.kexCli(m)
	var ret rem.KexWrapperMsg
	if err != nil {
		m.Warnw("kexProvisionEngine.Receive", "err", err, "stage", "kexCli")
		return ret, err
	}
	ret, err = cli.Receive(m.Ctx(), arg)
	if err != nil {
		m.Warnw("kexProvisionEngine.Receive", "err", err, "stage", "Receive")
		return ret, err
	}
	return ret, nil
}

func (e *kexCommonEngine) waitForMySecret(ctx context.Context) (proto.KexSecret, error) {
	select {
	case <-ctx.Done():
		return proto.KexSecret{}, ctx.Err()
	case secret := <-e.displayCh:
		return secret, nil
	}
}

func (e *kexProvisioneeEngine) UI() libclient.KexUIer {
	return e
}

func (e *kexCommonEngine) GetSessionFromUI(
	ctx context.Context,
) (
	proto.KexSecret,
	func(context.Context, error),
	error,
) {
	ret := <-e.inputCh
	return ret,
		func(ctx context.Context, err error) {
			e.inputCheckCh <- err
		},
		nil
}

func (e *kexCommonEngine) SendSessionToUI(ctx context.Context, secret proto.KexSecret) error {
	e.displayCh <- secret
	return nil
}

func (e *kexCommonEngine) pumpHESPThrough(hesp proto.KexHESP) error {
	var secret proto.KexSecret
	err := core.HESPToKexSecret(hesp, &secret)
	if err != nil {
		return err
	}
	err = e.pumpSecretThough(secret)
	if err != nil {
		return err
	}
	return nil
}

func (e *kexCommonEngine) pumpSecretThough(secret proto.KexSecret) error {
	ch := make(chan error)
	e.inputCheckCh = ch
	e.inputCh <- secret
	err := <-ch
	return err
}

func (e *kexProvisioneeEngine) EndSessionExchange(ctx context.Context, err error) error {
	e.kexCommonEngine.cancelInputCh <- err
	return nil
}

func (c *AgentConn) provisionGetSession(id proto.UISessionID) (*SignupSession, error) {
	sess, err := c.agent.sessions.Signup(id)
	if err != nil {
		return nil, err
	}
	if sess.kexEng == nil {
		return nil, core.InternalError("kex engine not found")
	}
	return sess, nil
}

func (c *AgentConn) GotKexInput(ctx context.Context, arg lcl.KexSessionAndHESP) error {
	id := arg.SessionId
	input := arg.Hesp
	sess, err := c.provisionGetSession(id)
	if err != nil {
		return err
	}
	return sess.kexEng.pumpHESPThrough(input)
}

func (c *AgentConn) StartKex(ctx context.Context, id proto.UISessionID) (proto.KexHESP, error) {
	m := c.MetaContext(ctx)
	sess, err := c.agent.sessions.Signup(id)
	if err != nil {
		return nil, err
	}

	host := sess.homeServer.Chain().HostID()
	ss := core.RandomSecretSeed32()
	role := proto.NewRoleDefault(proto.RoleType_OWNER)

	// Can't save it yet, since we don't know the UID!!! Need to get that via Kex.
	newkey, err := core.NewPrivateSuite25519(proto.EntityType_Device, role, ss, host)

	if err != nil {
		return nil, err
	}

	dnn, err := core.NormalizeDeviceName(sess.deviceName)
	if err != nil {
		return nil, err
	}

	dln := proto.DeviceLabelAndName{
		Label: proto.DeviceLabel{
			DeviceType: proto.DeviceType_Computer,
			Name:       dnn,
			Serial:     proto.FirstDeviceSerial,
		},
		Name: sess.deviceName,
	}

	sess.privKey = newkey
	eng := newKexProvisioneeEngine(c, sess, newkey)
	sess.kexEng = eng
	kx := libclient.NewKexProvisionee(eng, eng, dln, newkey)
	sess.kex = kx

	return eng.kexCommonEngine.runKex(m, kx)
}

func (e *kexCommonEngine) runKex(m libclient.MetaContext, kx libclient.KexActor) (proto.KexHESP, error) {

	timeout := m.G().Cfg().KexTimeout()
	m.Infow("StartKex", "timeout", timeout.String())

	completeCh := make(chan error)
	e.runCh = completeCh

	go func() {
		err := libclient.RunKex(context.Background(), kx, timeout)
		if err != nil {
			m.Errorw("kex failed", "err", err)
		}
		completeCh <- err
	}()

	// Almost immediately, the kex engine will return to us a kex secret to show into the
	// client UI.
	secret, err := e.waitForMySecret(m.Ctx())
	if err != nil {
		return nil, err
	}
	ret, err := core.KexSecretToHESP(secret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (e *kexCommonEngine) wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-e.runCh:
		return err
	}
}

type kexProvisionerEngine struct {
	kexCommonEngine
	uc *libclient.UserContext

	userMu  sync.Mutex
	user    *libclient.UserWrapper
	userErr error
}

func (e *kexProvisionerEngine) loadUser(ctx context.Context) error {
	e.userMu.Lock()
	defer e.userMu.Unlock()
	if e.user != nil {
		return nil
	}
	if e.userErr != nil {
		return e.userErr
	}
	m := e.c.MetaContext(ctx)
	u, err := libclient.LoadMe(m, e.uc)
	if err != nil {
		e.userErr = err
		return err
	}
	e.user = u
	return nil

}

func (e *kexProvisionerEngine) Router() libclient.KexRouter    { return e }
func (e *kexProvisionerEngine) UI() libclient.KexUIer          { return e }
func (e *kexProvisionerEngine) Keyer() libclient.KexLocalKeyer { return e }

func (e *kexProvisionerEngine) Server(ctx context.Context) (libclient.KexServer, error) {
	m := e.c.MetaContext(ctx)
	ret, err := e.uc.UserClient(m)
	return ret, err
}

func (e *kexProvisionerEngine) GetKexPPE(ctx context.Context) (*lcl.KexPPE, error) {
	m := e.c.MetaContext(ctx)
	return e.uc.GetKexPPE(m)
}

func (e *kexProvisionerEngine) FillChainer(ctx context.Context, res *proto.BaseChainer) error {
	err := e.loadUser(ctx)
	if err != nil {
		return err
	}
	m := e.c.MetaContext(ctx)
	ma, err := e.uc.MerkleAgent(m)
	if err != nil {
		return err
	}
	return kexFillChainer(m, e.user, ma, res)
}

func kexFillChainer(
	m libclient.MetaContext,
	u *libclient.UserWrapper,
	ma *merkle.Agent,
	res *proto.BaseChainer,
) error {
	nxt := u.Prot().Tail.Base.Seqno + 1
	prev := u.Prot().LastHash

	tr, err := ma.GetLatestTreeRootFromServer(m.Ctx())
	if err != nil {
		return err
	}
	*res = proto.BaseChainer{
		Prev:  &prev,
		Seqno: nxt,
		Root:  tr,
		Time:  proto.Now(),
	}
	return nil
}

func (e *kexProvisioneeEngine) StoreNewDeviceKey(
	ctx context.Context,
	fqr proto.FQUserAndRole,
	ppe *lcl.KexPPE,
	selfTok proto.PermissionToken,
) error {
	m := e.c.MetaContext(ctx)

	// Will error out with KeyExistsError if it's a repeat signup for this user.
	did, err := e.newKey.DeviceID()
	if err != nil {
		return err
	}
	row := lcl.LabeledSecretKeyBundle{
		Fqur:        fqr,
		KeyID:       did,
		SelfTok:     selfTok,
		Provisional: true,
	}
	skm, err := libclient.StoreSecretWithDefaults(m, row,
		e.newKey.Seed(), nil, ppe)

	if errors.Is(err, core.SecretKeyExistsError{}) {
		return core.DeviceAlreadyProvisionedError{}
	}

	if err != nil {
		return err
	}

	e.fqr = fqr
	e.sess.skm = skm
	e.sess.selfTok = selfTok
	return nil
}

func (e *kexProvisionerEngine) GetOrGeneratePUK(ctx context.Context, k core.RoleKey) (core.SharedPrivateSuiter, bool, error) {
	err := e.loadUser(ctx)
	if err != nil {
		return nil, false, err
	}
	m := e.c.MetaContext(ctx)
	return kexGetOrGeneratePUK(m, e.uc, e.user, k)
}

func kexGetOrGeneratePUK(
	m libclient.MetaContext,
	uc *libclient.UserContext,
	user *libclient.UserWrapper,
	k core.RoleKey,
) (core.SharedPrivateSuiter, bool, error) {

	pm := libclient.NewPUKMinder(uc).SetUser(user)
	r := k.Export()
	ps, err := pm.GetPUKSetForRole(m, r)
	if err != nil {
		return nil, false, err
	}
	if ps != nil {
		if ret := ps.Current(); ret != nil {
			return ret, false, nil
		}
	}
	pukSs := core.RandomSecretSeed32()
	newPuk, err := core.NewSharedPrivateSuite25519(
		proto.EntityType_User,
		r,
		pukSs,
		proto.Generation(0),
		uc.Info.Fqu.HostID,
	)
	if err != nil {
		return nil, false, err
	}
	return newPuk, true, nil
}

func (e *kexProvisionerEngine) GetAllDevicesForRole(ctx context.Context, rk core.RoleKey) ([]core.PublicBoxer, error) {
	err := e.loadUser(ctx)
	if err != nil {
		return nil, err
	}
	m := e.c.MetaContext(ctx)
	return kexGetAllDevicesForRole(m, e.uc, e.user, rk)
}

func kexGetAllDevicesForRole(
	m libclient.MetaContext,
	uc *libclient.UserContext,
	user *libclient.UserWrapper,
	rk core.RoleKey,
) (
	[]core.PublicBoxer,
	error,
) {
	r := rk.Export()
	var ret []core.PublicBoxer
	for _, dev := range user.Prot().Devices {
		lt, err := dev.Key.DstRole.LessThan(r)
		if err != nil {
			return nil, err
		}
		if !lt {
			pe, err := core.ImportPublicSuite(&dev.Key, user.Hepks, uc.Info.Fqu.HostID)
			if err != nil {
				return nil, err
			}
			ret = append(ret, pe)
		}
	}
	return ret, nil
}

func (e *kexProvisionerEngine) CheckDeviceName(ctx context.Context, dln *proto.DeviceLabelAndName) error {
	err := e.loadUser(ctx)
	if err != nil {
		return err
	}
	m := e.c.MetaContext(ctx)
	return kexCheckDeviceName(m, e.uc, e.user, dln)
}

func kexCheckDeviceName(
	m libclient.MetaContext,
	uc *libclient.UserContext,
	user *libclient.UserWrapper,
	dln *proto.DeviceLabelAndName,
) error {

	if !core.IsDeviceNameFixed(dln.Name) {
		return core.NameError("device name isn't 'fixed'")
	}
	ndn, err := core.NormalizeDeviceName(dln.Name)
	if err != nil {
		return err
	}
	if ndn != dln.Label.Name {
		return proto.NormalizationError("device name isn't normalized")
	}

	for _, dev := range user.Prot().Devices {
		if dev.Dn == nil {
			continue
		}
		eq, err := dev.Dn.NormEq(*dln)
		if err != nil {
			return err
		}
		if eq {
			return core.NameError("device name already exists")
		}
	}
	return nil

}

func newKexProvisionerEngine(
	c *AgentConn,
	sess *AssistSession,
	au *libclient.UserContext,
) *kexProvisionerEngine {
	ret := &kexProvisionerEngine{
		kexCommonEngine: *newKexCommonEngine(c, &sess.SessionBase),
		uc:              au,
	}
	return ret
}

var _ libclient.KexFullInterfacer = (*kexProvisionerEngine)(nil)

type AssistSession struct {
	SessionBase
	id     proto.UISessionID
	user   *libclient.UserContext
	kexEng *kexProvisionerEngine
}

func (a *AssistSession) Init(id proto.UISessionID) {
	a.SessionBase.Init()
	a.id = id
}

func (a *AgentConn) AssistInit(ctx context.Context, id proto.UISessionID) (proto.UserInfo, error) {
	var ret proto.UserInfo
	m := a.MetaContext(ctx)
	au, err := m.ActiveConnectedUser(&libclient.ACUOpts{AssertUnlocked: true})
	if err != nil {
		return ret, err
	}
	sess, err := a.assistGetSession(id, false)

	eng := newKexProvisionerEngine(a, sess, au)
	sess.kexEng = eng
	if err != nil {
		return ret, err
	}
	if au.Info.Fqu.Uid.IsZero() {
		return ret, core.UserNotFoundError{}
	}
	sess.user = au
	sess.homeServer = au.HomeServer()
	ret = au.UserInfo()

	return ret, nil
}

func (a *AgentConn) assistGetSession(id proto.UISessionID, needKexEngine bool) (*AssistSession, error) {
	sess, err := a.agent.sessions.Assist(id)
	if err != nil {
		return nil, err
	}
	if sess.kexEng == nil && needKexEngine {
		return nil, core.InternalError("no kex engine")
	}
	return sess, nil
}

func (a *AgentConn) AssistStartKex(ctx context.Context, id proto.UISessionID) (proto.KexHESP, error) {
	var ret proto.KexHESP
	m := a.MetaContext(ctx)

	sess, err := a.assistGetSession(id, true)
	if err != nil {
		return ret, err
	}
	cu := sess.user
	role := proto.NewRoleDefault(proto.RoleType_OWNER)
	kex := libclient.NewKexProvisioner(
		sess.kexEng,
		cu.Info.Fqu.Uid,
		cu.Info.Fqu.HostID,
		role,
		cu.PrivKeys.GetDevkey(),
	)
	hesp, err := sess.kexEng.kexCommonEngine.runKex(m, kex)
	if err != nil {
		return ret, err
	}
	return hesp, nil
}

func (e *kexProvisionerEngine) EndSessionExchange(ctx context.Context, err error) error {
	e.kexCommonEngine.cancelInputCh <- err
	return nil
}

func (a *AgentConn) AssistGotKexInput(ctx context.Context, arg lcl.KexSessionAndHESP) error {
	sess, err := a.assistGetSession(arg.SessionId, true)
	if err != nil {
		return err
	}
	return sess.kexEng.pumpHESPThrough(arg.Hesp)
}

func (a *AgentConn) AssistWaitForKexComplete(ctx context.Context, id proto.UISessionID) error {
	sess, err := a.assistGetSession(id, true)
	if err != nil {
		return err
	}
	err = sess.kexEng.wait(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (a *AgentConn) finishKexProvision(ctx context.Context, sess *SignupSession) error {
	m := a.MetaContext(ctx)

	uc, err := a.commonFinish(m, sess)
	if err != nil {
		return err
	}

	// now should do a user load to get the user's device name and utf8-username
	// from the source of truth (sigchain/server)
	user, err := libclient.LoadMe(m, uc)
	if err != nil {
		return err
	}

	uc.Info.Username = user.Prot().Username.B

	pe, err := sess.kex.MyKey().EntityPublic()
	if err != nil {
		return err
	}
	eid := pe.GetEntityID()
	dev := user.Prot().LookupDevice(eid)

	if dev == nil {
		return core.KeyNotFoundError{}
	}

	uc.Devname = dev.Dn.Name

	uinf := uc.Info

	err = m.DbPutUserInfo(&uinf, true)
	if err != nil {
		return err
	}

	pm := libclient.NewPUKMinder(uc).SetUser(user)
	pukset, err := pm.GetPUKSetForRole(m, sess.role)
	if err != nil {
		return err
	}
	uc.PrivKeys.SetPUKs(pukset)

	if sess.skm == nil {
		return core.InternalError("no secret key manager in finishKexProvision")
	}
	devid := sess.skm.DeviceID()

	if !user.HasDeviceID(devid) {
		return core.InternalError("provisioned key not found in reloaded user; something went wrong")
	}

	ss := m.G().SecretStore()

	err = sess.skm.ClearProvisionalBit(m.Ctx(), ss)
	if err != nil {
		return err
	}
	return nil
}

func (a *AgentConn) WaitForKexComplete(ctx context.Context, id proto.UISessionID) error {
	sess, err := a.provisionGetSession(id)
	if err != nil {
		return err
	}
	err = sess.kexEng.wait(ctx)
	if err != nil {
		return err
	}

	// We got this earlier when storing the secret key. Now plumb it through
	// for when we set the active user session, etc.
	sess.fqu = sess.kexEng.fqr.Fqu
	sess.role = sess.kexEng.fqr.Role

	err = a.finishKexProvision(ctx, sess)
	if err != nil {
		return err
	}
	return nil
}

func (a *AgentConn) KexCancelInput(ctx context.Context, id proto.UISessionID) error {
	sess, err := a.provisionGetSession(id)
	if err != nil {
		return err
	}
	err = <-sess.kexEng.cancelInputCh
	return err
}

func (a *AgentConn) AssistKexCancelInput(ctx context.Context, id proto.UISessionID) error {
	sess, err := a.assistGetSession(id, true)
	if err != nil {
		return err
	}
	err = <-sess.kexEng.cancelInputCh
	return err
}

type loopbackKexEngine struct {
	uc   *libclient.UserContext
	ucli *rem.UserClient
	user *libclient.UserWrapper
	g    *libclient.GlobalContext
}

func newLoopbackKexEngine(
	g *libclient.GlobalContext,
	uc *libclient.UserContext,
) *loopbackKexEngine {
	return &loopbackKexEngine{g: g, uc: uc}
}

func (e *loopbackKexEngine) metaContext(ctx context.Context) libclient.MetaContext {
	return libclient.NewMetaContext(ctx, e.g)
}

func (e *loopbackKexEngine) init(m libclient.MetaContext) error {
	var err error
	e.ucli, err = e.uc.UserClient(m)
	if err != nil {
		return err
	}
	u, err := libclient.LoadMe(m, e.uc)
	if err != nil {
		return err
	}
	e.user = u
	return nil
}

func (e *loopbackKexEngine) CheckDeviceName(
	ctx context.Context,
	dn *proto.DeviceLabelAndName,
) error {
	m := e.metaContext(ctx)
	return kexCheckDeviceName(m, e.uc, e.user, dn)
}

func (e *loopbackKexEngine) GetKexPPE(ctx context.Context) (*lcl.KexPPE, error) {
	m := e.metaContext(ctx)
	return e.uc.GetKexPPE(m)
}

func (e *loopbackKexEngine) FillChainer(ctx context.Context, res *proto.BaseChainer) error {
	m := e.metaContext(ctx)
	ma, err := e.uc.MerkleAgent(m)
	if err != nil {
		return err
	}
	return kexFillChainer(m, e.user, ma, res)
}

func (e *loopbackKexEngine) GetAllDevicesForRole(
	ctx context.Context,
	rk core.RoleKey,
) (
	[]core.PublicBoxer,
	error,
) {
	m := e.metaContext(ctx)
	return kexGetAllDevicesForRole(m, e.uc, e.user, rk)
}

func (e *loopbackKexEngine) GetOrGeneratePUK(
	ctx context.Context,
	k core.RoleKey,
) (
	core.SharedPrivateSuiter,
	bool,
	error,
) {
	m := e.metaContext(ctx)
	return kexGetOrGeneratePUK(m, e.uc, e.user, k)
}

func (e *loopbackKexEngine) Server(ctx context.Context) (libclient.KexServer, error) {
	return *e.ucli, nil
}

var _ libclient.LoobpackKexInterfacer = (*loopbackKexEngine)(nil)
