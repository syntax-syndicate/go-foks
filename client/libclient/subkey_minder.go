// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"errors"

	"github.com/foks-proj/go-foks/lib/chains"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type subkeyMinder struct {
	fqu       proto.FQUser
	parent    core.PrivateSuiter // a yubikey
	parentPub core.PublicSuiter
	parentEid proto.EntityID
	box       *proto.Box
	seed      proto.SecretSeed32
	subkey    core.EntityPrivate
	probe     *chains.Probe
	needSave  bool
}

func LoadSubkey(m MetaContext, u proto.UID, p *chains.Probe, key core.PrivateSuiter) (core.EntityPrivate, error) {
	minder := newSubkeyMinder(u, p, key)
	err := minder.run(m)
	if err != nil {
		return nil, err
	}
	return minder.subkey, nil
}

func newSubkeyMinder(uid proto.UID, p *chains.Probe, key core.PrivateSuiter) *subkeyMinder {
	return &subkeyMinder{
		fqu: proto.FQUser{
			Uid:    uid,
			HostID: p.Chain().HostID(),
		},
		parent: key,
		probe:  p,
	}
}

func (s *subkeyMinder) loadBox(m MetaContext) error {
	if s.box != nil {
		return nil
	}
	err := s.loadBoxFromDB(m)
	if err == nil {
		return nil
	}
	if !errors.Is(err, core.RowNotFoundError{}) {
		return err
	}
	err = s.loadBoxFromServer(m)
	if err != nil {
		return err
	}
	return nil
}

func (s *subkeyMinder) saveBoxToDB(m MetaContext) error {
	err := m.DbPut(DbTypeSoft, PutArg{
		Scope: &s.fqu.Uid,
		Typ:   lcl.DataType_SubkeyBox,
		Key:   s.parentEid,
		Val:   s.box,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *subkeyMinder) loadBoxFromServer(m MetaContext) error {
	cli, err := s.probe.RegCli(m)
	if err != nil {
		return err
	}
	chal, err := cli.GetSubkeyBoxChallenge(m.Ctx(), s.parentEid)
	if err != nil {
		return err
	}
	if !s.fqu.HostID.Eq(chal.Payload.HostID) ||
		!s.parentEid.Eq(chal.Payload.EntityID) ||
		!chal.Payload.Time.IsNowish() {
		return core.BadServerDataError("bad challenge")
	}
	sig, err := s.parent.Sign(&chal.Payload)
	if err != nil {
		return err
	}
	box, err := cli.LoadSubkeyBox(m.Ctx(), rem.LoadSubkeyBoxArg{
		Challenge: chal,
		Parent:    s.parentEid,
		Signature: *sig,
	})
	if err != nil {
		return err
	}
	s.box = &box
	s.needSave = true
	return nil
}

func (s *subkeyMinder) loadBoxFromDB(m MetaContext) error {

	var box proto.Box
	_, err := m.DbGet(
		&box,
		DbTypeSoft,
		&s.fqu.Uid,
		lcl.DataType_SubkeyBox,
		&s.parentEid,
	)
	if err != nil {
		return err
	}
	s.box = &box
	return nil
}

func (s *subkeyMinder) init(m MetaContext) error {
	var err error
	s.parentPub, err = s.parent.Publicize(&s.fqu.HostID)
	if err != nil {
		return err
	}
	s.parentEid = s.parentPub.GetEntityID()
	return nil
}

func (s *subkeyMinder) unbox(m MetaContext) error {
	var out proto.SubkeySeed
	err := core.SelfUnbox(s.parent, &out, *s.box)
	if err != nil {
		return err
	}
	if !s.parentPub.GetEntityID().Eq(out.Parent) {
		return core.KeyImportError("wrong parent key found")
	}
	s.seed = out.Seed
	subkeySeed, err := core.DeviceSigningSecretKey(s.seed)
	if err != nil {
		return err
	}
	s.subkey = core.NewEntityPrivateEd25519WithSeed(
		proto.EntityType_Subkey,
		*subkeySeed,
	)
	return nil
}

func (s *subkeyMinder) run(m MetaContext) error {
	err := s.init(m)
	if err != nil {
		return err
	}
	err = s.loadBox(m)
	if err != nil {
		return err
	}
	err = s.unbox(m)
	if err != nil {
		return err
	}
	if s.needSave {
		err = s.saveBoxToDB(m)
		if err != nil {
			return err
		}
	}
	return nil
}
