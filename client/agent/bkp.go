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

type LoadBackupSession struct {
	SessionBase
}

func (l *LoadBackupSession) Init(id proto.UISessionID) {
	l.SessionBase.Init()
}

var _ Sessioner = (*LoadBackupSession)(nil)

func (a *AgentConn) BackupNew(
	ctx context.Context,
	role proto.Role,
) (
	lcl.BackupHESP,
	error,
) {
	var zed lcl.BackupHESP
	m := a.MetaContext(ctx)
	r, err := a.prepareNewDevice(m, role)
	if err != nil {
		return zed, err
	}

	bkp, err := core.NewBackupKey()
	if err != nil {
		return zed, err
	}
	ks, err := bkp.KeySuite(role, r.hostID)
	if err != nil {
		return zed, err
	}
	dln, err := bkp.DeviceLabelAndName()
	if err != nil {
		return zed, err
	}

	lkr := libclient.NewLoopbackKexRunner(
		m.G(),
		r.devKey,
		ks,
		r.au.FQU(),
		role,
		*dln,
		r.lke,
		r.tok,
	)
	err = lkr.Run(ctx)
	if err != nil {
		return zed, err
	}
	ret, err := bkp.Export()
	if err != nil {
		return zed, err
	}
	return ret, nil
}

func (a *AgentConn) BackupLoadPutHESP(
	ctx context.Context,
	arg lcl.BackupLoadPutHESPArg,
) error {
	m := a.MetaContext(ctx)
	sess, err := a.agent.sessions.LoadBackup(arg.SessionId)
	if err != nil {
		return err
	}
	var bk core.BackupKey
	err = bk.Import(arg.Hesp)
	if err != nil {
		return err
	}
	ks, err := bk.KeySuite(proto.OwnerRole, sess.homeServer.Chain().HostID())
	if err != nil {
		return err
	}
	lres, err := a.lookupDeviceOnServer(m, &sess.SessionBase, ks)
	if err != nil {
		return err
	}

	uctx, err := populateUserContext(lres, sess.homeServer, ks, proto.KeyGenus_Backup, nil)
	if err != nil {
		return err
	}

	// Fill in device name based on the HESP (the first two tokens in it)
	uctx.Devname = bk.Name()

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
