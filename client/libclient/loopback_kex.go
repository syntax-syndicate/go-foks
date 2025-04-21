// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"context"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

// loopback kex is used to provision a new yubikey device with either an existing
// yubikey or a local device key.
type LoobpackKexInterfacer interface {
	KexLocalKeyer
	Server(ctx context.Context) (KexServer, error)
}

type LoopbackKexRunner struct {
	g *GlobalContext

	// input
	existingKey core.PrivateSuiter
	newKey      core.PrivateSuiter
	subkey      core.EntityPrivate
	subkeyBox   *proto.Box
	fqu         proto.FQUser
	role        proto.Role
	dln         proto.DeviceLabelAndName
	lki         LoobpackKexInterfacer
	selfTok     proto.PermissionToken
	hepks       *core.HEPKSet
	pqhint      *proto.YubiSlotAndPQKeyID

	// state
	kglr *kexGenerateLinkRes
}

func NewLoopbackKexRunner(
	g *GlobalContext,
	existingKey core.PrivateSuiter,
	newKey core.PrivateSuiter,
	fqu proto.FQUser,
	role proto.Role,
	dln proto.DeviceLabelAndName,
	lki LoobpackKexInterfacer,
	tok proto.PermissionToken,
) *LoopbackKexRunner {
	return &LoopbackKexRunner{
		g:           g,
		existingKey: existingKey,
		newKey:      newKey,
		fqu:         fqu,
		role:        role,
		dln:         dln,
		lki:         lki,
		selfTok:     tok,
		hepks:       core.NewHEPKSet(),
	}
}

func (k *LoopbackKexRunner) WithSubkey(
	subkey core.EntityPrivate,
	subkeyBox *proto.Box,
) *LoopbackKexRunner {
	k.subkey = subkey
	k.subkeyBox = subkeyBox
	return k
}

func (k *LoopbackKexRunner) WithPQHint(
	pqhint *proto.YubiSlotAndPQKeyID,
) *LoopbackKexRunner {
	k.pqhint = pqhint
	return k
}

func (k *LoopbackKexRunner) checkDeviceName(ctx context.Context) error {
	err := k.lki.CheckDeviceName(ctx, &k.dln)
	if err != nil {
		return err
	}
	return nil
}

func (k *LoopbackKexRunner) generateLink(ctx context.Context) error {
	pubNew, err := k.newKey.Publicize(&k.fqu.HostID)
	if err != nil {
		return err
	}
	hepk, err := k.newKey.ExportHEPK()
	if err != nil {
		return err
	}
	err = k.hepks.Add(*hepk)
	if err != nil {
		return err
	}
	if k.newKey.HasSubkey() && k.subkey == nil {
		return core.KeyNotFoundError{Which: "subkey"}
	}

	kg := kexLinkGenerator{
		klk:          k.lki,
		existingPriv: k.existingKey,
		newPub:       pubNew,
		role:         k.role,
		uid:          k.fqu.Uid,
		host:         k.fqu.HostID,
		devLabel:     k.dln.Label,
		subkey:       k.subkey,
	}
	res, err := kg.gen(ctx)
	if err != nil {
		return err
	}
	k.kglr = res
	return nil
}

func (k *LoopbackKexRunner) sign(ctx context.Context) error {

	// Some tests intentionally break the loopback provision process to
	// simulate something like a YubiKey hardware error.
	if err := k.g.Testing.LoopbackSignError(); err != nil {
		return err
	}

	lo, err := core.CountersignProvisionLink(k.kglr.mlr.Link, k.newKey)
	if err != nil {
		return err
	}
	lo, err = core.CountersignProvisionLink(lo, k.existingKey)
	if err != nil {
		return err
	}
	k.kglr.mlr.Link = lo
	return nil
}

func (k *LoopbackKexRunner) Run(ctx context.Context) error {

	err := k.checkDeviceName(ctx)
	if err != nil {
		return err
	}

	err = k.generateLink(ctx)
	if err != nil {
		return err
	}

	err = k.sign(ctx)
	if err != nil {
		return err
	}

	err = k.post(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (k *LoopbackKexRunner) post(ctx context.Context) error {
	srv, err := k.lki.Server(ctx)
	if err != nil {
		return err
	}
	return srv.ProvisionDevice(
		ctx,
		rem.ProvisionDeviceArg{
			Link:     *k.kglr.mlr.Link,
			PukBoxes: *k.kglr.pukBoxes,
			Dlnc: rem.DeviceLabelNameAndCommitmentKey{
				Dln:           k.dln,
				CommitmentKey: *k.kglr.mlr.DevNameCommitmentKey,
			},
			NextTreeLocation: *k.kglr.mlr.NextTreeLocation,
			SubkeyBox:        k.subkeyBox,
			SelfToken:        k.selfTok,
			Hepks:            k.hepks.Export(),
			YubiPQhint:       k.pqhint,
		},
	)
}
