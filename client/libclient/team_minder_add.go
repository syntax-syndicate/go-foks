// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type teamAdder struct {
	tm      *TeamMinder
	tr      *TeamRecord
	tok     *rem.TeamBearerToken
	mrs     []proto.MemberRole
	arg     lcl.TeamAddArg
	hepks   *core.HEPKSet
	dstRole proto.Role
}

func (t *teamAdder) hostID() proto.HostID {
	return t.tr.FQT().Host
}

func newTeamAdder(tm *TeamMinder, tr *TeamRecord, tok *rem.TeamBearerToken, arg lcl.TeamAddArg) *teamAdder {
	return &teamAdder{
		tm:    tm,
		tr:    tr,
		tok:   tok,
		arg:   arg,
		hepks: core.NewHEPKSet(),
	}
}

func (t *TeamMinder) requireOpenViewership(m MetaContext) error {
	ucli, err := t.au.UserClient(m)
	if err != nil {
		return err
	}
	cfg, err := ucli.GetHostConfig(m.Ctx())
	if err != nil {
		return err
	}
	if cfg.Viewership.User != proto.ViewershipMode_OpenToAll {
		return core.PermissionError("host does not allow open viewership; must use 3-way invitation flow")
	}
	return nil
}

func (t *teamAdder) loadUser(m MetaContext, u lcl.FQPartyParsedAndRole) error {
	uw, err := LoadUserByFQPartyParsed(m, u.Fqp)
	if err != nil {
		return err
	}
	if !uw.fqu.HostID.Eq(t.hostID()) {
		return core.HostMismatchError{}
	}
	srcRole := proto.OwnerRole
	if u.Role != nil {
		srcRole = *u.Role
	}
	id := uw.fqu.ToFQEntity().AtHost(t.hostID())
	rk, err := core.ImportRole(srcRole)
	if err != nil {
		return err
	}
	tmk, hepk, err := uw.TeamMemberKeys(*rk)
	if err != nil {
		return err
	}
	err = t.hepks.Add(*hepk)
	if err != nil {
		return err
	}
	mr := proto.MemberRole{
		DstRole: t.dstRole,
		Member: proto.Member{
			Id:      id,
			SrcRole: srcRole,
			Keys:    proto.NewMemberKeysWithTeam(*tmk),
		},
	}
	t.mrs = append(t.mrs, mr)
	return nil
}

func (t *teamAdder) loadUsers(m MetaContext) error {
	for _, u := range t.arg.Members {
		err := t.loadUser(m, u)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *teamAdder) post(m MetaContext) error {
	tr := t.tr
	tr.Lock()
	defer tr.Unlock()

	ed := teamEditorFromTeamRecord(tr)
	ed.tok = t.tok
	ed.changes = t.mrs
	ed.hepks = t.hepks

	for _, mr := range t.mrs {
		pid, err := mr.Member.Id.Entity.ToPartyID()
		if err != nil {
			return err
		}
		ed.lvpf = append(ed.lvpf, pid)
	}
	return ed.Run(m)
}

func (t *teamAdder) loadDstRole(m MetaContext) error {
	t.dstRole = proto.DefaultRole
	if t.arg.DstRole != nil {
		t.dstRole = *t.arg.DstRole
	}
	return nil
}

func (t *teamAdder) run(m MetaContext) error {
	err := t.loadDstRole(m)
	if err != nil {
		return err
	}
	err = t.loadUsers(m)
	if err != nil {
		return err
	}
	err = t.post(m)
	if err != nil {
		return err
	}
	return nil
}

func (t *TeamMinder) Add(m MetaContext, arg lcl.TeamAddArg) error {

	err := t.requireOpenViewership(m)
	if err != nil {
		return err
	}

	return t.withLoadedTeamAndAdminToken(
		m,
		arg.Team,
		LoadTeamOpts{LoadMembers: false, Refresh: true},
		func(m MetaContext, tr *TeamRecord, tok *rem.TeamBearerToken) error {
			adder := newTeamAdder(t, tr, tok, arg)
			return adder.run(m)
		},
	)
}
