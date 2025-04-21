// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"github.com/foks-proj/go-foks/proto/infra"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/server/shared"
)

type Plan struct {
	infra.Plan
	Active bool
	Verb   string
	Url    string
	Ids    proto.ID16String
}

func DecoratePlans(
	m shared.MetaContext,
	userPlan *infra.UserPlan,
	plans []infra.Plan,
) (
	[]Plan,
	error,
) {
	ret := make([]Plan, 0, len(plans))

	for _, x := range plans {
		x := x
		s, err := x.Id.ToID16().ID16StringErr()
		if err != nil {
			return nil, err
		}

		tmp := Plan{
			Plan: x,
			Ids:  s,
			Url:  "subscribe",
		}
		if userPlan == nil || !userPlan.IsLive() {
			tmp.Verb = "subscribe"
		} else if userPlan != nil && userPlan.IsLive() && userPlan.Plan.Id.Eq(x.Id) {
			tmp.Active = true
			tmp.Verb = "manage"
			tmp.Url = "manage"
		} else if userPlan.Plan.MonthlyCents() < x.MonthlyCents() {
			tmp.Verb = "upgrade"
			tmp.Url = "prorate"
		} else {
			tmp.Verb = "downgrade"
			tmp.Url = "prorate"
		}
		ret = append(ret, tmp)
	}
	return ret, nil
}

type ProrationData struct {
}
