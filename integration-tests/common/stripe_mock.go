// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package common

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/infra"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/server/shared"
	"github.com/stretchr/testify/require"
)

type FakeStripeSession struct {
	Id   infra.StripeSessionID
	Arg  shared.CheckoutArg
	Sub  infra.StripeSubscriptionID
	Plan infra.StripeProdID
	Ps   *shared.PaymentSuccess
}

type FakeInvoice struct {
	Inv      infra.StripeInvoice
	ChargeId infra.StripeChargeID
}

type FakeStripeSubscription struct {
	Id            infra.StripeSubscriptionID
	Cus           infra.StripeCustomerID
	Plan          infra.StripeProdID
	Price         infra.StripePriceID
	Start         time.Time
	PaidThrough   time.Time
	Invoice       *FakeInvoice
	CancelPending bool
	Canceled      bool
}

type FakeStripeCustomer struct {
	Id        infra.StripeCustomerID
	Email     proto.Email
	ActiveSub *FakeStripeSubscription
	Invoices  []*FakeInvoice
}

type FakePlan struct {
	Id      infra.StripeProdID
	Name    string
	Details []string
	Prices  []infra.StripePriceID
}

type FakeStripe struct {
	sync.Mutex
	customers   map[infra.StripeCustomerID]*FakeStripeCustomer
	emails      map[proto.Email]infra.StripeCustomerID
	uids        map[proto.UID]infra.StripeCustomerID
	sessions    map[infra.StripeSessionID]*FakeStripeSession
	subs        map[infra.StripeSubscriptionID]*FakeStripeSubscription
	priceToPlan map[infra.StripePriceID]infra.StripeProdID
	prices      map[infra.StripePriceID]*infra.PlanPrice
	plans       map[infra.StripeProdID]*FakePlan
}

func NewFakeStripe() *FakeStripe {
	ret := &FakeStripe{
		customers:   make(map[infra.StripeCustomerID]*FakeStripeCustomer),
		emails:      make(map[proto.Email]infra.StripeCustomerID),
		uids:        make(map[proto.UID]infra.StripeCustomerID),
		sessions:    make(map[infra.StripeSessionID]*FakeStripeSession),
		priceToPlan: make(map[infra.StripePriceID]infra.StripeProdID),
		subs:        make(map[infra.StripeSubscriptionID]*FakeStripeSubscription),
		prices:      make(map[infra.StripePriceID]*infra.PlanPrice),
		plans:       make(map[infra.StripeProdID]*FakePlan),
	}
	return ret
}

func MakeFakeStripeID(t *testing.T, prfx string) string {
	ret, err := fakeID(prfx)
	require.NoError(t, err)
	return ret
}

func fakeID(prfx string) (string, error) {
	var sffx [10]byte
	err := core.RandomFill(sffx[:])
	if err != nil {
		return "", err
	}
	s := core.B62Encode(sffx[:])
	return prfx + "_FAKE" + s, nil
}

func makeFakeInvoice(amt infra.Cents) (*FakeInvoice, error) {
	tmp, err := fakeID("inv")
	if err != nil {
		return nil, err
	}
	charge, err := fakeID("ch")
	if err != nil {
		return nil, err
	}
	url := proto.URLString("https://example.com/invoice/" + tmp)
	return &FakeInvoice{
		ChargeId: infra.StripeChargeID(charge),
		Inv: infra.StripeInvoice{
			Id:   infra.StripeInvoiceID(tmp),
			Url:  url,
			Amt:  amt,
			Time: proto.ExportTime(time.Now()),
			Desc: "Fake Invoice",
		},
	}, nil
}

func (f *FakeStripe) CreateCustomer(
	m shared.MetaContext,
	uid proto.UID,
	em proto.Email,
) (
	infra.StripeCustomerID,
	error,
) {
	f.Lock()
	defer f.Unlock()
	if _, found := f.emails[em]; found {
		return "", errors.New("email already exists")
	}
	id, err := fakeID("cus")
	if err != nil {
		return "", err
	}
	ret := infra.StripeCustomerID(id)
	f.customers[ret] = &FakeStripeCustomer{
		Email: em,
	}
	f.emails[em] = ret
	f.uids[uid] = ret
	return ret, nil
}

func (f *FakeStripe) CheckoutSession(
	m shared.MetaContext,
	arg shared.CheckoutArg,
) (
	infra.StripeSessionID,
	proto.URLString,
	error,
) {
	f.Lock()
	defer f.Unlock()
	id, err := fakeID("sess")
	var url proto.URLString
	if err != nil {
		return "", url, err
	}
	ret := infra.StripeSessionID(id)
	plan, found := f.priceToPlan[arg.PriceID]
	if !found {
		return "", url, core.NotFoundError("price not found")
	}
	tmp, err := fakeID("sub")
	if err != nil {
		return "", url, err
	}
	subId := infra.StripeSubscriptionID(tmp)
	sess := &FakeStripeSession{
		Id:   ret,
		Arg:  arg,
		Sub:  subId,
		Plan: plan,
	}
	f.sessions[ret] = sess
	sub := &FakeStripeSubscription{
		Id:    subId,
		Cus:   arg.CustomerID,
		Plan:  plan,
		Price: arg.PriceID,
	}
	f.subs[subId] = sub
	url = proto.URLString("https://fake.stripe.com/checkout/v1")
	return ret, url, nil
}

func (f *FakeStripe) ExpireSession(m shared.MetaContext, sid infra.StripeSessionID) error {
	f.Lock()
	defer f.Unlock()
	_, ok := f.sessions[sid]
	if !ok {
		return core.NotFoundError("session not found")
	}
	delete(f.sessions, sid)
	return nil
}

func (f *FakeStripe) LoadPaymentSuccess(
	m shared.MetaContext,
	sessId infra.StripeSessionID,
) (*shared.PaymentSuccess, error) {
	f.Lock()
	defer f.Unlock()
	sess := f.sessions[sessId]
	if sess == nil {
		return nil, core.NotFoundError("session not found")
	}
	if sess.Ps != nil {
		return sess.Ps, nil
	}
	sub := f.subs[sess.Sub]
	if sub == nil {
		return nil, core.NotFoundError("subscription not found")
	}
	price := f.prices[sub.Price]
	if price == nil {
		return nil, core.NotFoundError("price not found")
	}
	dur := price.Pi.Interval.Duration()
	start := m.Now()
	expire := start.Add(dur)
	sub.Start = start
	sub.PaidThrough = expire
	inv, err := makeFakeInvoice(price.Cents)
	if err != nil {
		return nil, err
	}
	sub.Invoice = inv
	cust := f.customers[sub.Cus]
	if cust == nil {
		return nil, core.NotFoundError("customer not found")
	}
	cust.Invoices = append(cust.Invoices, inv)
	cust.ActiveSub = sub

	ret := &shared.PaymentSuccess{
		SessionID:          sessId,
		ProdID:             sub.Plan,
		PriceID:            sub.Price,
		SubID:              sub.Id,
		CurrentPeriodEnd:   expire,
		CurrentPeriodStart: start,
		InvID:              inv.Inv.Id,
		ChargeID:           inv.ChargeId,
		Amount:             price.Cents,
		CustomerID:         sub.Cus,
	}
	sess.Ps = ret
	return ret, nil
}

func (f *FakeStripe) LoadInvoices(
	m shared.MetaContext,
	custId infra.StripeCustomerID,
	pag shared.StripePaginate,
) ([]infra.StripeInvoice, error) {
	f.Lock()
	defer f.Unlock()
	cust := f.customers[custId]
	if cust == nil {
		return nil, core.NotFoundError("customer not found")
	}
	ret := core.Map(cust.Invoices, func(i *FakeInvoice) infra.StripeInvoice {
		return i.Inv
	})
	return ret, nil
}
func (f *FakeStripe) UpdateCancelAtPeriodEnd(
	m shared.MetaContext,
	subId infra.StripeSubscriptionID,
	val bool,
) error {
	f.Lock()
	defer f.Unlock()
	sub := f.subs[subId]
	if sub == nil {
		return core.NotFoundError("subscription not found")
	}
	if sub.Canceled {
		return errors.New("subscription already canceled")
	}
	sub.CancelPending = val
	return nil
}

func (f *FakeStripe) CancelSubscription(
	m shared.MetaContext,
	subId infra.StripeSubscriptionID,
) error {
	f.Lock()
	defer f.Unlock()
	sub := f.subs[subId]
	if sub == nil {
		return core.NotFoundError("subscription not found")
	}
	if sub.Canceled {
		return errors.New("subscription already canceled")
	}
	sub.Canceled = true
	return nil
}

func (f *FakeStripe) CreatePrice(
	m shared.MetaContext,
	prodId infra.StripeProdID,
	price infra.Cents,
	pi infra.PaymentInterval,
) (infra.StripePriceID, error) {
	f.Lock()
	defer f.Unlock()
	id, err := fakeID("price")
	if err != nil {
		return "", err
	}
	prod := f.plans[prodId]
	if prod == nil {
		return "", core.NotFoundError("product not found")
	}
	ret := infra.StripePriceID(id)
	f.priceToPlan[ret] = prodId
	f.prices[ret] = &infra.PlanPrice{
		Cents:         price,
		Pi:            pi,
		StripePriceId: ret,
	}
	prod.Prices = append(prod.Prices, ret)
	return ret, nil
}

func (f *FakeStripe) CreatePlan(
	m shared.MetaContext,
	name string,
	details []string,
) (infra.StripeProdID, error) {
	f.Lock()
	defer f.Unlock()
	id, err := fakeID("prod")
	if err != nil {
		return "", err
	}
	ret := infra.StripeProdID(id)
	f.plans[ret] = &FakePlan{
		Id:      ret,
		Name:    name,
		Details: details,
	}
	return ret, nil
}

func (f *FakeStripe) SessionSuccess(id infra.StripeSessionID) (proto.URLString, error) {
	f.Lock()
	defer f.Unlock()
	sess := f.sessions[id]
	if sess == nil {
		return "", core.NotFoundError("session not found")
	}
	url := sess.Arg.SuccessURL
	ret := strings.Replace(url.String(), "{CHECKOUT_SESSION_ID}", string(id), 1)
	return proto.URLString(ret), nil
}

func (f *FakeStripe) SessionCancel(id infra.StripeSessionID) (proto.URLString, error) {
	f.Lock()
	defer f.Unlock()
	sess := f.sessions[id]
	if sess == nil {
		return "", core.NotFoundError("session not found")
	}
	url := sess.Arg.CancelURL
	ret := strings.Replace(url.String(), "{CHECKOUT_SESSION_ID}", string(id), 1)
	return proto.URLString(ret), nil
}

func (f *FakeStripe) LoadSubscription(
	m shared.MetaContext,
	subId infra.StripeSubscriptionID,
) (*shared.Subscription, error) {
	f.Lock()
	defer f.Unlock()
	sub := f.subs[subId]
	if sub == nil {
		return nil, core.NotFoundError("subscription not found")
	}
	return &shared.Subscription{
		CurrentPeriodEnd: sub.PaidThrough,
		ProdID:           sub.Plan,
		PriceID:          sub.Price,
	}, nil
}

func (f *FakeStripe) Renew(
	m shared.MetaContext,
	uid proto.UID,
) error {
	f.Lock()
	defer f.Unlock()

	customerID, ok := f.uids[uid]
	if !ok {
		return core.NotFoundError("customer ID")
	}
	cust, ok := f.customers[customerID]
	if !ok {
		return core.NotFoundError("customer")
	}
	if cust.ActiveSub == nil {
		return core.NotFoundError("active subscription")
	}
	priceID := cust.ActiveSub.Price
	price, ok := f.prices[priceID]
	if !ok {
		return core.NotFoundError("plan")
	}
	term := price.Pi.Duration()
	now := m.Now()
	expire := now.Add(term)
	cust.ActiveSub.PaidThrough = expire
	return nil
}

func (f *FakeStripe) PreviewProration(
	m shared.MetaContext,
	arg shared.PreviewProrationArg,
) (*shared.ProrationData, error) {
	f.Lock()
	defer f.Unlock()

	cust, ok := f.customers[arg.CustomerID]
	if !ok {
		return nil, core.NotFoundError("customer")
	}

	if cust.ActiveSub == nil {
		return nil, core.NotFoundError("active subscription")
	}

	if cust.ActiveSub.Id != arg.SubID {
		return nil, core.NotFoundError("subscription mismatch")
	}

	oldPrice := cust.ActiveSub.Price
	oldPriceData := f.prices[oldPrice]
	newPriceData := f.prices[arg.NewPlan.PriceID]

	if oldPriceData == nil || newPriceData == nil {
		return nil, core.NotFoundError("price data")
	}

	now := m.Now()
	nextBillTime := cust.ActiveSub.PaidThrough

	// Calculate prorated adjustment
	oldAmount := oldPriceData.Cents
	newAmount := newPriceData.Cents
	remainingDays := nextBillTime.Sub(now).Hours() / 24
	totalDays := oldPriceData.Pi.Duration().Hours() / 24
	prorationAmount := infra.SignedCents((float64(oldAmount-newAmount) * remainingDays) / totalDays)

	return &shared.ProrationData{
		Time: now,
		Adj: []shared.ProrationAdjustment{
			{
				Amount: prorationAmount,
				Desc:   "Prorated adjustment",
			},
		},
		NextBill: shared.Billing{
			Time:     proto.ExportTime(nextBillTime),
			Subtotal: infra.SignedCents(newAmount),
			Total:    infra.SignedCents(newAmount),
		},
	}, nil
}

func (f *FakeStripe) ApplyProration(
	m shared.MetaContext,
	arg shared.PreviewProrationArg,
) error {
	cust, ok := f.customers[arg.CustomerID]
	if !ok {
		return core.NotFoundError("customer")
	}
	cust.ActiveSub.Price = arg.NewPlan.PriceID
	cust.ActiveSub.Plan = arg.NewPlan.ProdID
	return nil
}

var _ shared.Striper = (*FakeStripe)(nil)
