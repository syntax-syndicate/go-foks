// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

import (
	"strings"
	"sync"
	"time"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/infra"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/invoice"
	"github.com/stripe/stripe-go/v81/price"
	"github.com/stripe/stripe-go/v81/product"
	"github.com/stripe/stripe-go/v81/subscription"
)

type RealStripe struct {
	sync.Mutex
	didInit bool
	initErr error
}

func NewRealStripe() *RealStripe {
	return &RealStripe{}
}

var _ Striper = (*RealStripe)(nil)

func (r *RealStripe) init(m MetaContext) error {
	r.Lock()
	defer r.Unlock()

	if r.didInit {
		return r.initErr
	}
	doInit := func() error {
		cfg, err := m.G().Config().StripeConfig(m.Ctx())
		if err != nil {
			return err
		}
		sk := cfg.SecretKey()
		if sk.IsZero() {
			return core.ConfigError("no secret stripe key")
		}
		stripe.Key = string(sk)
		return nil
	}
	r.initErr = doInit()
	r.didInit = true
	return r.initErr
}

func (r *RealStripe) CreateCustomer(
	m MetaContext,
	uid proto.UID,
	email proto.Email,
) (
	infra.StripeCustomerID,
	error,
) {
	var zed infra.StripeCustomerID
	err := r.init(m)
	if err != nil {
		return zed, err
	}
	params := stripe.CustomerParams{
		Email: stripe.String(email.String()),
	}
	result, err := customer.New(&params)
	if err != nil {
		return "", err
	}
	cid := infra.StripeCustomerID(result.ID)
	return cid, nil
}

func (r *RealStripe) LoadPaymentSuccess(
	m MetaContext,
	sessionId infra.StripeSessionID,
) (
	*PaymentSuccess,
	error,
) {
	err := r.init(m)
	if err != nil {
		return nil, err
	}
	params := stripe.CheckoutSessionParams{}
	params.AddExpand("subscription")
	params.AddExpand("subscription.latest_invoice")
	sess, err := session.Get(string(sessionId), &params)
	if err != nil {
		return nil, err
	}
	x := sess.Subscription
	if x == nil {
		return nil, stripeErr(core.NotFoundError("stripe subscription"))
	}
	subId := infra.StripeSubscriptionID(x.ID)
	if x.Items == nil || len(x.Items.Data) != 1 {
		return nil, stripeErr(core.NotFoundError("stripe subscription item"))
	}
	item := x.Items.Data[0]
	if item == nil || item.Price == nil {
		return nil, stripeErr(core.NotFoundError("stripe subscription item price"))
	}
	price := item.Price.ID
	if item.Plan.Product == nil {
		return nil, stripeErr(core.NotFoundError("stripe subscription item price product"))
	}
	prod := item.Plan.Product.ID

	invId := x.LatestInvoice.ID
	inv := x.LatestInvoice

	if inv.Charge == nil {
		return nil, stripeErr(core.NotFoundError("stripe invoice charge"))
	}

	return &PaymentSuccess{
		SubID:              subId,
		PriceID:            infra.StripePriceID(price),
		ProdID:             infra.StripeProdID(prod),
		SessionID:          sessionId,
		InvID:              infra.StripeInvoiceID(invId),
		ChargeID:           infra.StripeChargeID(inv.Charge.ID),
		CurrentPeriodStart: time.Unix(int64(x.CurrentPeriodStart), 0),
		CurrentPeriodEnd:   time.Unix(int64(x.CurrentPeriodEnd), 0),
		Amount:             infra.Cents(item.Price.UnitAmount),
	}, nil
}

func (r *RealStripe) ExpireSession(
	m MetaContext,
	sessionId infra.StripeSessionID,
) error {
	err := r.init(m)
	if err != nil {
		return err
	}
	params := stripe.CheckoutSessionExpireParams{}
	_, err = session.Expire(string(sessionId), &params)
	if err != nil {
		return err
	}
	return nil
}

func ImportInvoice(inv *stripe.Invoice) (*infra.StripeInvoice, error) {
	var desc string
	if len(inv.Lines.Data) > 0 {
		desc = inv.Lines.Data[0].Description
	}
	ret := infra.StripeInvoice{
		Id:   infra.StripeInvoiceID(inv.ID),
		Url:  proto.URLString(inv.HostedInvoiceURL),
		Time: proto.NewTimeFromSecs(inv.Created),
		Amt:  infra.Cents(inv.AmountPaid),
		Desc: desc,
	}
	return &ret, nil
}

func (r *RealStripe) LoadInvoices(
	m MetaContext,
	customer infra.StripeCustomerID,
	page StripePaginate,
) (
	[]infra.StripeInvoice,
	error,
) {
	err := r.init(m)
	if err != nil {
		return nil, err
	}
	lim := int64(page.Limit)
	params := &stripe.InvoiceListParams{
		ListParams: stripe.ListParams{
			Limit:         &lim,
			StartingAfter: page.StartingAfter,
			EndingBefore:  page.EndingBefore,
		},
		Customer: customer.StringP(),
	}
	iter := invoice.List(params)
	var ret []infra.StripeInvoice
	for iter.Next() {
		raw := iter.Invoice()

		// Don't include invoices that haven't been finalized.
		if raw.Status == stripe.InvoiceStatusOpen || raw.Status == stripe.InvoiceStatusDraft {
			continue
		}
		inv, err := ImportInvoice(iter.Invoice())
		if err != nil {
			return nil, err
		}
		ret = append(ret, *inv)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *RealStripe) UpdateCancelAtPeriodEnd(
	m MetaContext,
	subId infra.StripeSubscriptionID,
	cancel bool,
) error {
	err := r.init(m)
	if err != nil {
		return err
	}
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: &cancel,
	}
	_, err = subscription.Update(subId.String(), params)
	if err != nil {
		return err
	}
	return nil
}

func (r *RealStripe) CancelSubscription(
	m MetaContext,
	subId infra.StripeSubscriptionID,
) error {
	err := r.init(m)
	if err != nil {
		return err
	}
	params := &stripe.SubscriptionCancelParams{}
	_, err = subscription.Cancel(subId.String(), params)
	if err != nil {
		return err
	}
	return nil
}

func (r *RealStripe) PreviewProration(
	m MetaContext,
	arg PreviewProrationArg,
) (
	*ProrationData,
	error,
) {
	err := r.init(m)
	if err != nil {
		return nil, err
	}

	prorateNow := m.Now()

	subscription, err := subscription.Get(arg.SubID.String(), nil)
	if err != nil {
		return nil, err
	}

	// See what the next invoice would look like with a price switch
	// and proration set:
	items := []*stripe.InvoiceUpcomingSubscriptionDetailsItemParams{
		{
			ID:    stripe.String(subscription.Items.Data[0].ID),
			Price: stripe.String(arg.NewPlan.PriceID.String()),
		},
	}

	params := &stripe.InvoiceUpcomingParams{
		Customer:     arg.CustomerID.StringP(),
		Subscription: stripe.String(arg.SubID.String()),
		SubscriptionDetails: &stripe.InvoiceUpcomingSubscriptionDetailsParams{
			Items:             items,
			ProrationDate:     stripe.Int64(prorateNow.UTC().Unix()),
			ProrationBehavior: stripe.String(ProrationBehaviorCreateProrations),
		},
	}
	inv, err := invoice.Upcoming(params)
	if err != nil {
		return nil, err
	}

	ret := ProrationData{
		Time:  prorateNow.UTC(), // ignore arg.Time
		SubID: arg.SubID,
	}

	for _, line := range inv.Lines.Data {
		adj := ProrationAdjustment{
			Amount: infra.SignedCents(line.Amount),
			Desc:   line.Description,
		}
		ret.Adj = append(ret.Adj, adj)
	}

	ret.NextBill = Billing{
		Subtotal: infra.SignedCents(inv.Subtotal),
		Tax:      infra.SignedCents(inv.Tax),
		Total:    infra.SignedCents(inv.Total),
		Time:     proto.NewTimeFromSecs(inv.NextPaymentAttempt),
	}

	return &ret, nil
}

const (
	ProrationBehaviorCreateProrations = "create_prorations"
)

func (r *RealStripe) CheckoutSession(
	m MetaContext,
	arg CheckoutArg,
) (
	infra.StripeSessionID,
	proto.URLString,
	error,
) {
	err := r.init(m)
	if err != nil {
		return "", "", err
	}
	var zed infra.StripeSessionID
	exp := arg.Expire.Unix()
	params := stripe.CheckoutSessionParams{
		SuccessURL: arg.SuccessURL.StringP(),
		CancelURL:  arg.CancelURL.StringP(),
		Customer:   arg.CustomerID.StringP(),
		ExpiresAt:  &exp,
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    arg.PriceID.StringP(),
				Quantity: stripe.Int64(1),
			},
		},
	}

	sess, err := session.New(&params)
	if err != nil {
		return zed, "", err
	}
	id := infra.StripeSessionID(sess.ID)
	url := proto.URLString(sess.URL)
	return id, url, nil
}

func (r *RealStripe) ApplyProration(
	m MetaContext,
	arg PreviewProrationArg,
) error {
	err := r.init(m)
	if err != nil {
		return err
	}

	sub, err := subscription.Get(arg.SubID.String(), nil)
	if err != nil {
		return err
	}

	var itemId *string
	for _, item := range sub.Items.Data {
		if item.Plan.Product.ID == arg.CurrPlan.ProdID.String() &&
			item.Price.ID == arg.CurrPlan.PriceID.String() {
			itemId = &item.ID
			break
		}
	}
	if itemId == nil {
		return core.NotFoundError("subscription item")
	}

	secs := arg.Time.UnixSeconds()

	params := &stripe.SubscriptionParams{
		ProrationBehavior: stripe.String(ProrationBehaviorCreateProrations),
		ProrationDate:     stripe.Int64(secs),
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    itemId,
				Price: arg.NewPlan.PriceID.StringP(),
			},
		},
	}

	_, err = subscription.Update(arg.SubID.String(), params)
	if err != nil {
		return err
	}
	return nil

}

func (r *RealStripe) CreatePrice(
	m MetaContext,
	prod infra.StripeProdID,
	cents infra.Cents,
	intvl infra.PaymentInterval,
) (
	infra.StripePriceID,
	error,
) {
	tmp := int64(intvl.Count)
	params := &stripe.PriceParams{
		Product:    prod.StringP(),
		UnitAmount: cents.Int64P(),
		Currency:   stripe.String("usd"),
		Recurring: &stripe.PriceRecurringParams{
			Interval:      intvl.Interval.StringP(),
			IntervalCount: &tmp,
		},
	}

	var zed infra.StripePriceID

	stripePr, err := price.New(params)
	if err != nil {
		return zed, err
	}
	ret := infra.StripePriceID(stripePr.ID)
	return ret, nil
}

func (r *RealStripe) CreatePlan(
	m MetaContext,
	name string,
	details []string,
) (
	infra.StripeProdID,
	error,
) {
	var zed infra.StripeProdID
	err := r.init(m)
	if err != nil {
		return zed, err
	}
	desc := strings.Join(
		core.Map(details, func(s string) string {
			return "- " + s
		}), "\n")
	params := &stripe.ProductParams{
		Name:        stripe.String(name),
		Description: stripe.String(desc),
	}

	prod, err := product.New(params)
	if err != nil {
		return zed, err
	}
	ret := infra.StripeProdID(prod.ID)
	return ret, nil
}

func (r *RealStripe) LoadSubscription(
	m MetaContext,
	subId infra.StripeSubscriptionID,
) (
	*Subscription,
	error,
) {
	err := r.init(m)
	if err != nil {
		return nil, err
	}
	params := stripe.SubscriptionParams{
		Expand: []*string{stripe.String("items.data.plan")},
	}
	sub, err := subscription.Get(subId.String(), &params)
	if err != nil {
		return nil, err
	}
	ret := Subscription{
		CurrentPeriodEnd: time.Unix(int64(sub.CurrentPeriodEnd), 0),
		ProdID:           infra.StripeProdID(sub.Items.Data[0].Plan.Product.ID),
		PriceID:          infra.StripePriceID(sub.Items.Data[0].Plan.ID),
	}
	return &ret, nil
}

var _ Striper = (*RealStripe)(nil)

func StripeHandleWebhookEvent(
	m MetaContext,
	event stripe.Event,
) error {
	switch event.Type {
	case "invoice.payment_succeeded":
		return handleInvoicePaymentSucceeded(m, event)
	default:
		m.Warnw("HandleWebhookEvent", "err", "unhandled event", "type", event.Type)
	}
	return nil
}
