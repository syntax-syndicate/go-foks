// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

import (
	"encoding/json"
	"slices"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/infra"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/jackc/pgx/v5"
)

type PlanMaker struct {
	p    *infra.Plan
	opts infra.MakePlanOpts
}

type PlanMakerOpts struct {
	UseStripe bool
}

func NewPlanMaker(p *infra.Plan, opts infra.MakePlanOpts) *PlanMaker {
	return &PlanMaker{p: p, opts: opts}
}

func (p *PlanMaker) makeIDs() error {
	if p.p.Id.IsZero() {
		err := p.p.RandomID()
		if err != nil {
			return err
		}
	}
	prices := make([]infra.PlanPrice, 0, len(p.p.Prices))
	for _, pr := range p.p.Prices {
		pr := pr
		if pr.Id.IsZero() {
			err := pr.RandomID()
			if err != nil {
				return err
			}
		}
		prices = append(prices, pr)
	}

	p.p.Prices = prices
	return nil

}

func (p *PlanMaker) checkArgs(m MetaContext) error {
	plan := p.p
	switch plan.Scope {
	case infra.QuotaScope_None:
		return core.BadArgsError("no scope in plan")
	case infra.QuotaScope_VHost:
		if plan.MaxVhosts == 0 {
			return core.BadArgsError("no max vhosts in plan")
		}
	case infra.QuotaScope_Teams:
		// noop
	}

	if plan.MaxSeats == 0 {
		return core.BadArgsError("no max seats in plan")
	}

	if plan.MonthlyPrice() == nil {
		return core.BadArgsError("no monthly price in plan; it's required")
	}

	return nil
}

func (p *PlanMaker) Run(m MetaContext) error {

	err := p.checkArgs(m)
	if err != nil {
		return err
	}

	err = p.makeIDs()
	if err != nil {
		return err
	}

	if len(p.p.Prices) == 0 {
		return core.BadServerDataError("no prices for plan")
	}

	err = p.doStripe(m)
	if err != nil {
		return err
	}

	err = p.dbStore(m)
	if err != nil {
		return err
	}
	return nil
}

func (p *PlanMaker) Obj() *infra.Plan {
	return p.p
}

func (p *PlanMaker) createPrices(m MetaContext) error {
	prices := make([]infra.PlanPrice, 0, len(p.p.Prices))
	for _, pr := range p.p.Prices {
		price, err := m.Stripe().CreatePrice(m, p.p.StripeProdId, pr.Cents, pr.Pi)
		if err != nil {
			return err
		}
		pr.StripePriceId = price
		prices = append(prices, pr)
	}
	p.p.Prices = prices
	return nil
}

func (p *PlanMaker) createProduct(m MetaContext) error {

	prod, err := m.Stripe().CreatePlan(m, p.p.DisplayName, p.p.Points)
	if err != nil {
		return err
	}
	p.p.StripeProdId = prod
	return nil
}

func (p *PlanMaker) doStripe(m MetaContext) error {

	err := p.createProduct(m)
	if err != nil {
		return err
	}
	err = p.createPrices(m)
	if err != nil {
		return err
	}
	return nil
}

func (p *PlanMaker) dbStore(m MetaContext) error {

	insPlan := func(tx pgx.Tx) error {
		byt, err := json.Marshal(p.p.Points)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(
			m.Ctx(),
			`INSERT INTO quota_plans
			  (plan_id, name, display_name, quota_scope, max_seats, max_vhosts, 
			   quota, details, stripe_prod_id, promoted, sso_support, ctime)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())`,
			p.p.Id.ExportToDB(),
			p.p.Name,
			p.p.DisplayName,
			p.p.Scope.String(),
			p.p.MaxSeats,
			p.p.MaxVhosts,
			p.p.Quota,
			string(byt),
			p.p.StripeProdId.String(),
			p.p.Promoted,
			p.p.Sso,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return core.InsertError("makePlan")
		}
		return nil
	}

	insPrice := func(tx pgx.Tx, pr infra.PlanPrice) error {
		tag, err := tx.Exec(
			m.Ctx(),
			`INSERT INTO quota_plan_prices 
			  (plan_id, price_id, stripe_price_id, interval, interval_count, price_cents, promoted, pri)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8)`,
			p.p.Id.ExportToDB(),
			pr.Id.ExportToDB(),
			pr.StripePriceId.String(),
			pr.Pi.Interval.String(),
			pr.Pi.Count,
			pr.Cents,
			pr.Promoted,
			pr.Pri,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return core.InsertError("price")
		}
		return nil
	}

	return RetryTxUserDB(m, "PlanMaker.dbStore", func(m MetaContext, tx pgx.Tx) error {
		err := insPlan(tx)
		if err != nil {
			return err
		}
		for _, pr := range p.p.Prices {
			err = insPrice(tx, pr)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func SetPlanViaCli(
	m MetaContext,
	fqu proto.FQUser,
	planID proto.PlanID,
	repl bool,
) (*proto.CancelID, error) {
	connector := NewBackendClient(m.G(), proto.ServerType_Quota, proto.ServerType_Tools, nil)
	defer connector.Close()
	gcli, err := connector.Cli(m.Ctx())
	if err != nil {
		return nil, err
	}
	cli := infra.QuotaClient{Cli: gcli, ErrorUnwrapper: core.StatusToError}

	pcid, err := cli.SetPlan(m.Ctx(), infra.SetPlanArg{
		Fqu:     fqu,
		Plan:    planID,
		Replace: repl,
	})
	if err != nil {
		return nil, err
	}
	var ret *proto.CancelID
	if !pcid.IsZero() {
		ret = &pcid
	}
	return ret, nil
}

func readQuotaPlanFromDB(
	planId []byte,
	name string,
	displayName string,
	quotaScopeRaw string,
	maxTeams uint64,
	maxVhosts uint64,
	quota int64,
	details []byte,
	stripeProdId string,
	promoted bool,
	ssoEnabled bool,
) (
	*infra.Plan,
	error,
) {
	var points []string
	err := json.Unmarshal(details, &points)
	if err != nil {
		return nil, err
	}

	plan := infra.Plan{
		Name:         name,
		DisplayName:  displayName,
		MaxSeats:     maxTeams,
		Quota:        proto.Size(quota),
		StripeProdId: infra.StripeProdID(stripeProdId),
		Points:       points,
		Promoted:     promoted,
		MaxVhosts:    maxVhosts,
		Sso:          ssoEnabled,
	}

	err = plan.Id.ImportFromDB(planId)
	if err != nil {
		return nil, err
	}
	err = plan.Scope.ImportFromDB(quotaScopeRaw)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func loadPricesIntoPlans(
	m MetaContext,
	qry Querier,
	plans map[proto.PlanID]*infra.Plan,
) error {
	ids := make([][]byte, 0, len(plans))
	for id := range plans {
		ids = append(ids, id.ExportToDB())
	}

	rows, err := qry.Query(
		m.Ctx(),
		`SELECT plan_id, price_id, stripe_price_id, interval, interval_count, price_cents, promoted, pri
		 FROM quota_plan_prices WHERE plan_id = ANY($1) ORDER by pri`,
		ids,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var planIdRaw, priceId []byte
		var stripePriceId, stripeInterval string
		var intervalCount, priceCents int64
		var promoted bool
		var pri int
		err := rows.Scan(&planIdRaw, &priceId, &stripePriceId, &stripeInterval, &intervalCount, &priceCents, &promoted, &pri)
		if err != nil {
			return err
		}
		var planID proto.PlanID
		err = planID.ImportFromDB(planIdRaw)
		if err != nil {
			return err
		}
		plan, ok := plans[planID]
		if !ok {
			return core.BadServerDataError("price without plan")
		}

		price := infra.PlanPrice{
			StripePriceId: infra.StripePriceID(stripePriceId),
			Pi: infra.PaymentInterval{
				Count: uint64(intervalCount),
			},
			Cents: infra.Cents(priceCents),
			Pri:   int64(pri),
		}

		err = price.Id.ImportFromDB(priceId)
		if err != nil {
			return err
		}
		err = price.Pi.Interval.ImportFromDB(stripeInterval)
		if err != nil {
			return err
		}
		plan.Prices = append(plan.Prices, price)
	}

	for _, plan := range plans {
		if len(plan.Prices) == 0 {
			return core.BadServerDataError("plan without prices")
		}
	}

	return nil
}

func LoadPromotedPlans(
	m MetaContext,
) (
	[]infra.Plan,
	error,
) {
	db, err := m.Db(DbTypeUsers)
	if err != nil {
		return nil, err
	}
	defer db.Release()

	hcfg, err := m.HostConfig()
	if err != nil {
		return nil, err
	}

	scope := infra.QuotaScopeFromHostType(hcfg.Typ)
	if scope == infra.QuotaScope_None {
		return nil, core.InternalError("current host type does not have associated plans")
	}

	rows, err := db.Query(
		m.Ctx(),
		`SELECT plan_id, name, display_name, quota_scope, max_seats, max_vhosts,
		    quota, details, stripe_prod_id, promoted, sso_support
		 FROM quota_plans
		 WHERE promoted=true AND quota_scope=$1`,
		scope.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	retMap := make(map[proto.PlanID]*infra.Plan)

	for rows.Next() {
		var quotaScopeRaw string
		var planId []byte
		var name, displayName string
		var maxTeams, maxVhosts uint64
		var quota int64
		var details []byte
		var stripeProdId string
		var promoted, ssoSupport bool
		err := rows.Scan(&planId, &name, &displayName, &quotaScopeRaw, &maxTeams,
			&maxVhosts, &quota, &details, &stripeProdId, &promoted, &ssoSupport)
		if err != nil {
			return nil, err
		}
		plan, err := readQuotaPlanFromDB(planId, name, displayName, quotaScopeRaw, maxTeams, maxVhosts,
			quota, details, stripeProdId, promoted, ssoSupport)
		if err != nil {
			return nil, err
		}
		retMap[plan.Id] = plan
	}

	err = loadPricesIntoPlans(m, db, retMap)
	if err != nil {
		return nil, err
	}

	ret := make([]infra.Plan, 0, len(retMap))
	for _, plan := range retMap {
		ret = append(ret, *plan)
	}

	// Sort from lowest to highest price
	slices.SortFunc(ret, func(a, b infra.Plan) int {
		if len(a.Prices) == 0 {
			return -1
		}
		if len(b.Prices) == 0 {
			return 1
		}
		return int(a.Prices[0].Cents - b.Prices[0].Cents)
	})

	return ret, nil
}
