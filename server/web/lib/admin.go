// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/infra"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/server/shared"
)

type PageSelector struct {
	HostType proto.HostType
	PageType PageType
}

type PageType int

const (
	PageTypeUsage PageType = 1
	PageTypePlans PageType = 2
)

type TypedAdminPageMain struct {
	HostType proto.HostType
}

type PageDataer interface {
	GetHead() *HeaderData
	GetCSRFToken() *CSRFToken
}

type AdminPageData struct {
	User *User

	Which PageSelector

	// Can be nil or zero-valied dedpending on how the data is loaded
	HostConfig *proto.HostConfig
	Usage      *UsageData
	AllPlans   []Plan
	ActiveSess infra.StripeSessionID
	Head       *HeaderData
	Invoices   *shared.StripeInvoices
	VHosts     *VHostData
	UserPlan   *infra.UserPlan
}

var _ PageDataer = (*AdminPageData)(nil)

func (d *AdminPageData) GetHead() *HeaderData {
	if d == nil {
		return nil
	}
	return d.Head
}

func (d *AdminPageData) GetCSRFToken() *CSRFToken {
	if d == nil || d.User == nil {
		return nil
	}
	tok := d.User.CSRFToken
	if tok == "" {
		return nil
	}
	return &tok
}

type DataLoadOpts struct {
	Which       PageSelector
	Usage       bool
	UserPlan    bool
	Plans       bool
	Sess        bool
	Headers     bool
	VHosts      bool
	DefInvoices bool
	Invoices    *shared.StripePaginate
	PageTitle   string
}

func (a *AdminPageData) CanAddMoreVHosts() bool {
	if a.UserPlan == nil || a.VHosts == nil {
		return false
	}
	return a.VHosts.NumHosts() < a.UserPlan.MaxVHosts()
}

func (a *AdminPageData) CanDoSSO() bool {
	if a.UserPlan == nil {
		return false
	}
	return a.UserPlan.Plan.Sso
}

func LoadAdminPageData(
	m shared.MetaContext,
	u *User,
	opts DataLoadOpts,
) (
	*AdminPageData,
	error,
) {
	ret := AdminPageData{
		User:  u,
		Which: opts.Which,
	}

	err := ret.load(m, opts)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (d *AdminPageData) loadHeaders(
	m shared.MetaContext,
	title string,
) error {
	if title == "" {
		title = "FOKS Admin Control Panel"
	}
	d.Head = NewHeaderData(m.Ctx(), title)
	return nil

}

func (d *AdminPageData) LoadUsageData(m shared.MetaContext) error {
	urs, err := LoadUsageData(m, d.User.Name)
	if err != nil {
		return err
	}
	d.Usage = urs
	d.UserPlan = urs.UserPlan
	return nil
}

func (d *AdminPageData) loadPlans(m shared.MetaContext) error {
	plans, err := shared.LoadPromotedPlans(m)
	if err != nil {
		return err
	}
	dplans, err := DecoratePlans(m, d.UserPlan, plans)
	if err != nil {
		return err
	}
	d.AllPlans = dplans
	return nil
}

func (d *AdminPageData) loadSession(
	m shared.MetaContext,
) error {
	err := shared.CheckForOutstandingSessions(m, d.User.Uid)
	switch te := err.(type) {
	case core.StripeSessionExistsError:
		d.ActiveSess = te.Id
	case nil:
	default:
		return err
	}
	return nil
}

func (d *AdminPageData) loadInvoices(
	m shared.MetaContext,
	opts *shared.StripePaginate,
) error {
	if opts == nil {
		opts = &shared.StripePaginate{Limit: 20}
	}

	inv, err := shared.LoadStripeInvoices(m, d.User.Uid, *opts)
	if err != nil {
		return err
	}
	d.Invoices = inv
	return nil
}

func (d *AdminPageData) loadUserPlan(
	m shared.MetaContext,
) error {

	// Might have already been set as a result of loading usage data
	if d.UserPlan != nil {
		return nil
	}
	db, err := m.Db(shared.DbTypeUsers)
	if err != nil {
		return err
	}
	defer db.Release()

	plan, err := shared.LoadPlanForUser(m, db, m.ShortHostID(), d.User.Uid)

	switch err.(type) {
	case nil:
		d.UserPlan = plan
	case core.NoActivePlanError:
	default:
		return err
	}

	return nil
}

func (d *AdminPageData) LoadVHosts(
	m shared.MetaContext,
) error {
	vd, err := LoadVHostData(m)
	if err != nil {
		return err
	}
	d.VHosts = vd
	return nil
}

func (d *AdminPageData) load(
	m shared.MetaContext,
	opts DataLoadOpts,
) error {

	if opts.Headers {
		err := d.loadHeaders(m, opts.PageTitle)
		if err != nil {
			return err
		}
	}

	hc, err := m.HostConfig()
	if err != nil {
		return err
	}
	d.HostConfig = hc
	d.Which.HostType = hc.Typ

	if d.User == nil {
		return nil
	}

	if d.Which.HostType == proto.HostType_BigTop && opts.Usage {
		err := d.LoadUsageData(m)
		if err != nil {
			return err
		}
	}

	if opts.Sess {
		err := d.loadSession(m)
		if err != nil {
			return err
		}
	}
	if d.Which.HostType == proto.HostType_VHostManagement && opts.VHosts {
		err := d.LoadVHosts(m)
		if err != nil {
			return err
		}
	}

	if opts.DefInvoices || opts.Invoices != nil {
		err := d.loadInvoices(m, opts.Invoices)
		if err != nil {
			return err
		}
	}

	// If we're loading plans, we need to show which plan is active,
	// so we have to load the user's plan. But note we might have already
	// loaded that as part of UsageData.
	if opts.UserPlan || opts.Plans {
		err = d.loadUserPlan(m)
		if err != nil {
			return err
		}
	}

	if opts.Plans {
		err := d.loadPlans(m)
		if err != nil {
			return err
		}
	}

	return nil
}
