// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package admin

import (
	"github.com/foks-proj/go-foks/server/shared"
	"github.com/go-chi/chi/v5"
)

func InitRouter(g *shared.GlobalContext, prefix string, scfg shared.StripeConfigger, mux chi.Router) {
	stripeCallbacks := scfg.Callbacks()

	p := func(s string) string {
		return prefix + s
	}
	mux.Get(p(""), TopHandler(g))
	mux.Get(p("/plans"), PlansTopHandler(g))
	mux.Get(p("/main"), MainPanelHandler(g))
	mux.Get(p("/plans/main"), PlansMainPanelHandler(g))
	mux.Get(p("/plans/invoices"), InvoicesHandler(g))
	mux.Get(p("/plans/active"), PlanManageHandler(g))
	mux.Put(p("/plans/active/cancel"), PlanEditHandler(g, shared.PlanEditOpCancel))
	mux.Put(p("/plans/active/resume"), PlanEditHandler(g, shared.PlanEditOpResume))
	mux.Patch(p("/plans/active"), PlanChangeHandler(g))
	mux.Delete(p("/plans/active"), PlanEditHandler(g, shared.PlanEditOpRageQuit))
	mux.Post(p("/plans/manage/{planId}"), PlanManageHandler(g))
	mux.Post(p("/plans/subscribe/{planId}"), PlanSubscribeHandler(g))
	mux.Post(p("/plans/prorate/{planId}"), PlanPreviewProrateHandler(g))
	mux.Put(p("/claim/{tid}"), ClaimHandler(g, true))
	mux.Delete(p("/claim/{tid}"), ClaimHandler(g, false))
	mux.Get(stripeCallbacks.Success().String(), StripeSuccessHandler(g))
	mux.Get(stripeCallbacks.Cancel().String(), StripeCancelHandler(g))
	mux.Post(stripeCallbacks.Webhook().String(), StripeWebhookHandler(g))
	mux.Delete(p("/stripe/session/{sessionId}"), StripeDeleteSessionHandler(g))
	mux.Post(p("/vhost"), NewVHostPostHandler(g).ServeHTTP)
	mux.Get(p("/vhost/{vhostId}"), VHostDetailsHandler(g))
	mux.Get(p("/vhost/{vhostId}/check"), VHostCheckHandler(g))
	mux.Delete(p("/vhost/{vhostId}"), VHostDeleteHandler(g))
	mux.Delete(p("/vhost/{hostId}/invite/{inviteCode}"), VHostDeleteInviteHandler(g))
	mux.Post(p("/vhost/{hostId}/invite"), VHostNewInviteHandler(g))
	mux.Patch(p("/vhost/{hostId}/user/viewership"), VHostSetUserViewershipHandler(g))
	mux.Put(p("/vhost/{hostId}/sso"), VHostSetSSOHandler(g))

}
