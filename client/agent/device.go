// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import (
	"context"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func (c *AgentConn) SelfProvision(ctx context.Context, arg lcl.SelfProvisionArg) error {

	m := c.MetaContext(ctx)
	au, err := m.ActiveConnectedUser(&libclient.ACUOpts{AssertUnlocked: true})
	if err != nil {
		return err
	}

	err = libclient.AssertNotAlreadyProvisioned(m)
	if err != nil {
		return err
	}

	gt, err := arg.Role.GreaterThan(au.Role())
	if err != nil {
		return err
	}
	if gt {
		return core.RoleError("cannot provision a key with a role greater than current active user")
	}

	ss := core.RandomSecretSeed32()
	newKey, err := core.NewPrivateSuite25519(
		proto.EntityType_Device,
		arg.Role,
		ss,
		au.HostID(),
	)
	if err != nil {
		return err
	}

	newKeyEID, err := newKey.EntityID()
	if err != nil {
		return err
	}
	newKeyID, err := newKeyEID.ToDeviceID()
	if err != nil {
		return err
	}

	// Might fail if we alredy have a key provisioned for this device for this
	// user X role combo.
	ppe, err := au.GetKexPPE(m)
	if err != nil {
		return err
	}
	tok, err := core.NewPermissionToken()
	if err != nil {
		return err
	}
	row := lcl.LabeledSecretKeyBundle{
		Fqur: proto.FQUserAndRole{
			Fqu:  au.FQU(),
			Role: arg.Role,
		},
		KeyID:       newKeyID,
		SelfTok:     tok,
		Provisional: true,
	}
	skm, err := libclient.StoreSecretWithDefaults(m, row, ss, nil, ppe)
	if err != nil {
		return err
	}

	lke := newLoopbackKexEngine(m.G(), au)
	err = lke.init(m)
	if err != nil {
		return err
	}

	existingKey, err := au.Devkey(ctx)
	if err != nil {
		return err
	}

	lkr := libclient.NewLoopbackKexRunner(
		m.G(),
		existingKey,
		newKey,
		au.FQU(),
		arg.Role,
		arg.Dln,
		lke,
		tok,
	)

	err = lkr.Run(ctx)
	if err != nil {
		return err
	}

	info := au.Info // copy the current userInfo, it's mainly correct
	info.YubiInfo = nil
	info.KeyGenus = proto.KeyGenus_Device
	info.Role = arg.Role
	info.Key = newKeyEID

	uc := libclient.UserContext{
		Info:    info,
		Devname: arg.Dln.Name,
	}
	uc.PrivKeys.SetDevkey(newKey)
	uc.SetHomeServer(au.HomeServer())
	uc.SetSkmm(skm)

	pm := libclient.NewPUKMinder(&uc).SetUser(lke.user)
	pukset, err := pm.GetPUKSetForRole(m, arg.Role)
	if err != nil {
		return err
	}
	uc.PrivKeys.SetPUKs(pukset)

	var ltx libclient.LocalDbTx
	err = ltx.PutUser(info, false)
	if err != nil {
		return err
	}

	err = libclient.SetActiveUser(m, &uc)
	if err != nil {
		return err
	}

	// Confirm that provisioning worked via user load
	user, err := libclient.LoadMe(m, &uc)
	if err != nil {
		return err
	}

	if !user.HasDeviceID(newKeyID) {
		return core.InternalError("provisioned key not found in reloaded user; something went wrong")
	}

	if err := m.G().Testing.SelfProvisionCrash(); err != nil {
		return err
	}

	// Clear provisional bits in the secret store; we checked right above that the server agrees
	// the key is provisioned for us.
	err = skm.ClearProvisionalBit(m.Ctx(), m.G().SecretStore())
	if err != nil {
		return err
	}
	return nil
}

var _ lcl.DeviceInterface = (*AgentConn)(nil)
