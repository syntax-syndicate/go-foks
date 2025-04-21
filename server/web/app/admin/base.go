// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package admin

import (
	"net/http"
	"time"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/server/shared"
	"github.com/foks-proj/go-foks/server/web/lib"
)

type BaseHandler struct {
	*lib.BaseHandler
}

func (b *BaseHandler) LoggedInUser(
	m shared.MetaContext,
	r *http.Request,
) (
	*lib.User,
	error,
) {
	return lib.LoadUserFromCookie(m, r)
}

type ServeOpts struct {
	CSRFSend        bool
	CSRFCheck       bool
	CheckVHostAdmin bool
}

func (b *BaseHandler) insertCSRFToken(
	m shared.MetaContext, w http.ResponseWriter, user *lib.User,
) error {
	tok, err := lib.CreateCSRFToken(m, user.Uid)
	if err != nil {
		return err
	}
	user.CSRFToken = tok
	return nil
}

func (b *BaseHandler) checkCSRFHeader(
	m shared.MetaContext,
	r *http.Request,
	user *lib.User,
) error {
	tok := r.Header.Get("X-CSRF-Token")
	if tok == "" {
		return core.NewHttp401Error(core.NotFoundError("param"), "no CSRF token found")
	}
	err := lib.CheckCSRFToken(m, user.Uid, lib.CSRFToken(tok))
	if err != nil {
		return core.NewHttp401Error(err, "CSRF token validation failed")
	}
	return nil
}

func (b *BaseHandler) checkVHostAdmin(
	m shared.MetaContext,
	r *http.Request,
	user *lib.User,
) error {
	hid, err := lib.NewArgs(r).StdTargetHostID()
	if err != nil {
		return err
	}
	chid, err := user.CheckIsAdminOf(m, *hid)
	if err != nil {
		return err
	}
	user.AdminOfHost = chid
	return nil
}

func (b *BaseHandler) ServeLoggedIn(
	w http.ResponseWriter,
	r *http.Request,
	opts ServeOpts,
	h func(m shared.MetaContext, user *lib.User) error,
) {
	b.Serve(w, r, func(m shared.MetaContext) error {
		user, err := b.LoggedInUser(m, r)
		if err != nil {
			return IndexHandler{}.RedirectTo()
		}
		m = m.WithUserHost(user.ToUHC())

		if opts.CSRFSend {
			err = b.insertCSRFToken(m, w, user)
			if err != nil {
				return err
			}
		}
		if opts.CSRFCheck {
			err = b.checkCSRFHeader(m, r, user)
			if err != nil {
				return err
			}
		}
		if opts.CheckVHostAdmin {
			err = b.checkVHostAdmin(m, r, user)
			if err != nil {
				return err
			}
		}

		return h(m, user)
	})
}

func (b *BaseHandler) ServeWithDataLoad(
	w http.ResponseWriter,
	r *http.Request,
	so ServeOpts,
	lo lib.DataLoadOpts,
	h func(m shared.MetaContext, pd *lib.AdminPageData) error,
) {
	b.ServeLoggedIn(w, r, so, func(m shared.MetaContext, user *lib.User) error {
		pd, err := lib.LoadAdminPageData(m, user, lo)
		if err != nil {
			return err
		}
		return h(m, pd)
	})
}

func (b *BaseHandler) ServeWithVHostDetails(
	w http.ResponseWriter,
	r *http.Request,
	h func(m shared.MetaContext, pd *lib.AdminPageData, vhr *lib.VHostRow) error,
) {
	b.ServeWithDataLoad(w, r,
		ServeOpts{CSRFCheck: true, CheckVHostAdmin: true},
		lib.DataLoadOpts{
			Usage:    true,
			Headers:  true,
			VHosts:   true,
			UserPlan: true,
		},
		func(m shared.MetaContext, pd *lib.AdminPageData) error {
			doDebugDelay(m)
			m = m.WithHostID(pd.User.AdminOfHost)
			vhr := pd.VHosts.FindVHostRow(pd.User.AdminOfHost.VId)
			if vhr == nil {
				return core.NewHttp422Error(core.NotFoundError("vhost"), "no such vhost")
			}
			err := vhr.LoadDetails(m)
			if err != nil {
				return err
			}
			return h(m, pd, vhr)
		})
}

func newBaseHandler(g *shared.GlobalContext) *BaseHandler {
	return &BaseHandler{
		BaseHandler: lib.NewBaseHandlerWithErr(g, HandleErr),
	}
}

func doDebugDelay(m shared.MetaContext) {
	wcfg, err := m.G().Config().WebConfig(m.Ctx())
	if err != nil {
		m.Errorw("doDebugDelay", "err", err)
	}
	dd := wcfg.DebugDelay()
	if dd > 0 {
		time.Sleep(dd)
	}
}
