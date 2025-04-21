// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"errors"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/infra"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/server/shared"
)

type UsageRow struct {
	id            proto.PartyID
	IsTeam        bool
	Id            proto.EntityIDString
	Name          proto.NameUtf8
	Usage         proto.Size
	SrcRole       proto.Role
	DstRole       proto.Role
	IsClaimed     bool
	IsClaimedByMe bool
}

type UsageSummary struct {
	Total     proto.Size
	NumTeams  uint64
	OverQuota bool
}

type UsageData struct {
	Mine     []UsageRow
	Others   []UsageRow
	UserPlan *infra.UserPlan
	QCfg     infra.QuotaConfig
	Summary  UsageSummary
}

type FindRowRes struct {
	Row  UsageRow
	Prev proto.EntityIDString
	Mine bool
}

func (u UsageData) MaxTeams() int {
	if u.UserPlan != nil && u.UserPlan.IsLive() {
		return int(u.UserPlan.Plan.MaxSeats)
	}
	return int(u.QCfg.NoPlanMaxTeams)
}

func (u *UsageData) CanAddMoreTeams() bool {
	return int(u.Summary.NumTeams) < u.MaxTeams()
}

func (u *UsageData) HasTheirs() bool { return len(u.Others) > 0 }

func (u *UsageData) FindRow(pid proto.PartyID) *FindRowRes {

	find := func(vec []UsageRow, Mine bool) *FindRowRes {
		for i, r := range vec {
			if !r.id.Eq(pid) {
				continue
			}
			ret := FindRowRes{
				Row:  r,
				Mine: Mine,
			}
			if i > 0 {
				ret.Prev = vec[i-1].Id
			}
			return &ret
		}
		return nil
	}
	ret := find(u.Mine, true)
	if ret != nil {
		return ret
	}
	return find(u.Others, false)
}

func LoadUsageData(
	m shared.MetaContext,
	username proto.NameUtf8,
) (
	*UsageData,
	error,
) {
	claimedByMe := make(map[proto.FixedPartyID]bool)
	qu, err := shared.LoadQuotaUserByKVParty(m, m.ShortHostID(), m.UID().ToPartyID())
	if err != nil {
		return nil, err
	}

	add := func(p proto.PartyID, m map[proto.FixedPartyID]bool) error {
		feid, err := p.Fixed()
		if err != nil {
			return err
		}
		m[feid] = true
		return nil
	}

	err = add(m.UID().ToPartyID(), claimedByMe)
	if err != nil {
		return nil, err
	}

	for _, t := range qu.Teams {
		err = add(t.ToPartyID(), claimedByMe)
		if err != nil {
			return nil, err
		}
	}

	allTeams, err := shared.GetNamedTeamListForUser(m)
	if err != nil {
		return nil, err
	}

	allParties := make([]proto.PartyID, len(allTeams)+1)
	allParties[0] = m.UID().ToPartyID()
	for i, t := range allTeams {
		allParties[i+1] = t.Te.Id.ToPartyID()
	}

	usageMap, err := shared.LoadUsageForParties(m, allParties)
	if err != nil {
		return nil, err
	}
	meFixed, err := m.UID().ToPartyID().Fixed()
	if err != nil {
		return nil, err
	}
	meS, err := m.UID().EntityID().ToEntityIDString()
	if err != nil {
		return nil, err
	}

	myRow := UsageRow{
		Id:            meS,
		IsTeam:        false,
		Name:          username,
		Usage:         usageMap[meFixed],
		IsClaimed:     true,
		IsClaimedByMe: true,
		SrcRole:       proto.OwnerRole,
		DstRole:       proto.OwnerRole,
	}

	var mine []UsageRow
	var others []UsageRow
	tot := myRow.Usage
	var numTeams uint64 // start at 0 for personal

	for _, t := range allTeams {

		idf, err := t.Te.Id.ToPartyID().Fixed()
		if err != nil {
			return nil, err
		}
		ids, err := t.Te.Id.EntityID().ToEntityIDString()
		if err != nil {
			return nil, err
		}

		qr := UsageRow{
			id:        t.Te.Id.ToPartyID(),
			Id:        ids,
			IsTeam:    true,
			Name:      t.Name,
			Usage:     usageMap[idf],
			IsClaimed: (t.QuotaMaster != nil),
			SrcRole:   t.Te.SrcRole,
			DstRole:   t.Te.DstRole,
		}
		if claimedByMe[idf] {
			qr.IsClaimedByMe = true
			mine = append(mine, qr)
			tot += qr.Usage
			numTeams++

		} else {
			others = append(others, qr)
		}
	}

	err = shared.LoadOverQuota(m)
	var over bool
	switch {
	case errors.Is(err, core.UserNotFoundError{}):
		over = false
	case errors.Is(err, core.OverQuotaError{}):
		over = true
	case err != nil:
		return nil, err
	}

	return &UsageData{
		Mine:     append([]UsageRow{myRow}, mine...),
		Others:   others,
		UserPlan: qu.UserPlan,
		QCfg:     qu.QCfg,
		Summary: UsageSummary{
			Total:     tot,
			NumTeams:  numTeams,
			OverQuota: over,
		},
	}, nil
}
