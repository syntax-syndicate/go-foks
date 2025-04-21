// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package admin

import (
	"net/http"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/server/shared"
	"github.com/foks-proj/go-foks/server/web/lib"
	"github.com/foks-proj/go-foks/server/web/templates"
)

type VHostPostHandler struct {
	*BaseHandler
}

func NewVHostPostHandler(g *shared.GlobalContext) *VHostPostHandler {
	return &VHostPostHandler{
		BaseHandler: newBaseHandler(g),
	}
}

func (v *vhostPostHandlerSession) postBYOD(
	m shared.MetaContext,
) error {
	hn, err := v.args.NonApexHostname(lib.ParamSourceForm, "byod")
	if err != nil {
		return err
	}
	vm := shared.VanityMinder{
		Vstem: hn,
	}
	err = vm.Stage1(m)
	if err != nil {
		return err
	}
	return nil
}

func (v *vhostPostHandlerSession) render(
	m shared.MetaContext,
) error {
	if v.pageArgs.Err != nil {
		v.w.WriteHeader(http.StatusUnprocessableEntity)
	}
	return templates.VHostsMain(v.pd, &v.pageArgs).Render(m.Ctx(), v.w)
}

type vhostPostHandlerSession struct {
	w        http.ResponseWriter
	r        *http.Request
	pd       *lib.AdminPageData
	args     *lib.Args
	pageArgs lib.VHostAddArgs
}

func (v *vhostPostHandlerSession) serveWithErr(m shared.MetaContext) error {
	byod, err := v.args.IsChecked(lib.ParamSourceForm, "toggle-byod")
	if err != nil {
		return err
	}
	v.pageArgs.IsBYOD = byod
	if byod {
		return v.postBYOD(m)
	}
	return v.postCanned(m)
}

func (v *vhostPostHandlerSession) postCanned(m shared.MetaContext) error {
	domain, err := v.args.Hostname(lib.ParamSourceForm, "canned-domain")
	if err != nil {
		return err
	}
	host, err := v.args.HostnamePart(lib.ParamSourceForm, "canned-name")
	if err != nil {
		return err
	}
	err = lib.MakeCannedVHost(m, host, domain)
	if err != nil {
		return err
	}
	return nil
}

func (v *vhostPostHandlerSession) handleErr(m shared.MetaContext, err error) error {
	if err == nil {
		return nil
	}
	v.pageArgs.Err = &lib.VHostSetupError{Err: err}
	return nil
}

func (v *vhostPostHandlerSession) serve(m shared.MetaContext) error {

	err := v.serveWithErr(m)

	err = v.handleErr(m, err)
	if err != nil {
		return err
	}
	// Reload data since we're updating the page as a result of any
	// vhost additions we made.
	err = v.pd.LoadVHosts(m)
	if err != nil {
		return err
	}
	return v.render(m)
}

func (v *VHostPostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.ServeWithDataLoad(w, r,
		ServeOpts{CSRFSend: true, CSRFCheck: true},
		lib.DataLoadOpts{
			Usage:    true,
			Headers:  true,
			VHosts:   true,
			UserPlan: true,
		},
		func(m shared.MetaContext, pd *lib.AdminPageData) error {
			doDebugDelay(m)
			sess := &vhostPostHandlerSession{
				w:    w,
				r:    r,
				pd:   pd,
				args: lib.NewArgs(r),
			}
			return sess.serve(m)
		},
	)
}

func vhostDetails(
	g *shared.GlobalContext,
	isEdit bool,
	h func(shared.MetaContext, http.ResponseWriter, *lib.AdminPageData, *lib.VHostRow) error,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		newBaseHandler(g).ServeWithDataLoad(w, r,
			ServeOpts{CSRFSend: true, CSRFCheck: isEdit},
			lib.DataLoadOpts{
				Usage:    true,
				Headers:  true,
				VHosts:   true,
				UserPlan: true,
			},
			func(m shared.MetaContext, pd *lib.AdminPageData) error {
				doDebugDelay(m)
				args := lib.NewArgs(r)
				vhostID, err := args.VHostID(lib.ParamSourceURL, "vhostId")
				if err != nil {
					return err
				}
				vhr := pd.VHosts.FindVHostRow(*vhostID)
				if vhr == nil {
					return core.NewHttp422Error(core.NotFoundError("vhost"), "no such vhost")
				}
				return h(m, w, pd, vhr)
			},
		)
	}
}

func VHostDetailsHandler(g *shared.GlobalContext) func(w http.ResponseWriter, r *http.Request) {
	return vhostDetails(
		g,
		false,
		func(m shared.MetaContext, w http.ResponseWriter, pd *lib.AdminPageData, vhr *lib.VHostRow) error {
			err := vhr.LoadDetails(m)
			if err != nil {
				return err
			}
			return templates.VHostDetails(pd, vhr).Render(m.Ctx(), w)
		},
	)
}

func VHostDeleteHandler(g *shared.GlobalContext) func(w http.ResponseWriter, r *http.Request) {
	return vhostDetails(
		g,
		true,
		func(m shared.MetaContext, w http.ResponseWriter, pd *lib.AdminPageData, vhr *lib.VHostRow) error {
			err := lib.AbortVanityBuild(m, pd.User, vhr)
			if err != nil {
				return err
			}
			// Reload after deletion
			err = pd.LoadVHosts(m)
			if err != nil {
				return err
			}
			return templates.VHostsMain(pd, &lib.VHostAddArgs{}).Render(m.Ctx(), w)
		},
	)
}

func VHostCheckHandler(g *shared.GlobalContext) func(w http.ResponseWriter, r *http.Request) {
	return vhostDetails(
		g,
		true,
		func(m shared.MetaContext, w http.ResponseWriter, pd *lib.AdminPageData, vhr *lib.VHostRow) error {
			err := lib.CheckVanityHost(m, pd.User, vhr)
			if err != nil {
				// VHost check puts the error into a different position, so
				// it's handled differently.
				err = lib.VHostCheckError{Err: err}
				return err
			}
			err = pd.LoadVHosts(m)
			if err != nil {
				return err
			}
			vhr = pd.VHosts.FindVHostRow(vhr.VHostID)
			if vhr == nil {
				return core.NewHttp422Error(core.NotFoundError("vhost"), "no such vhost after refresh")
			}
			err = vhr.LoadDetails(m)
			if err != nil {
				return err
			}
			return templates.VHostDetails(pd, vhr).Render(m.Ctx(), w)
		},
	)
}

func VHostDeleteInviteHandler(g *shared.GlobalContext) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		newBaseHandler(g).ServeWithDataLoad(w, r,
			ServeOpts{CSRFCheck: true, CheckVHostAdmin: true},
			lib.DataLoadOpts{
				Usage:    true,
				Headers:  true,
				VHosts:   true,
				UserPlan: true,
			},
			func(m shared.MetaContext, pd *lib.AdminPageData) error {
				args := lib.NewArgs(r)
				doDebugDelay(m)
				ic, err := args.MultiUseInviteCode(lib.ParamSourceURL, "inviteCode")
				if err != nil {
					return err
				}
				vhr := pd.VHosts.FindVHostRow(pd.User.AdminOfHost.VId)
				if vhr == nil {
					return core.NewHttp422Error(core.NotFoundError("vhost"), "no such vhost")
				}
				err = vhr.LoadDetails(m)
				if err != nil {
					return err
				}
				muic := vhr.Details.FindMultiUseInviteCode(*ic)
				if muic == nil {
					return core.NewHttp422Error(core.NotFoundError("invite code"), "no such invite code")
				}
				m = m.WithHostID(pd.User.AdminOfHost)
				err = shared.DisableMultiUseInviteCode(m, *ic)
				if err != nil {
					return err
				}
				muic.Valid = false
				return templates.VHostInviteCodeRow(pd, vhr, *muic).Render(m.Ctx(), w)
			},
		)
	}
}

func VHostNewInviteHandler(g *shared.GlobalContext) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		newBaseHandler(g).ServeWithDataLoad(w, r,
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
				ic, err := lib.RandomMultiUseInviteCode()
				if err != nil {
					return err
				}
				vhr := pd.VHosts.FindVHostRow(pd.User.AdminOfHost.VId)
				if vhr == nil {
					return core.NewHttp422Error(core.NotFoundError("vhost"), "no such vhost")
				}
				err = shared.InsertMultiuseInviteCode(m, *ic)
				if err != nil {
					return err
				}
				err = vhr.LoadDetails(m)
				if err != nil {
					return err
				}
				muic := vhr.Details.FindMultiUseInviteCode(*ic)
				if muic == nil {
					return core.NewHttp422Error(core.NotFoundError("invite code"), "no such invite code")
				}
				return templates.VHostNewInviteRow(pd, vhr, *muic).Render(m.Ctx(), w)
			},
		)
	}
}

func VHostSetUserViewershipHandler(g *shared.GlobalContext) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		newBaseHandler(g).ServeWithVHostDetails(w, r,
			func(m shared.MetaContext, pd *lib.AdminPageData, vhr *lib.VHostRow) error {
				args := lib.NewArgs(r)
				vm, err := args.ViewershipMode(lib.ParamSourceForm, "mode")
				if err != nil {
					return err
				}
				err = shared.VHostSetUserViewership(m, vm)
				if err != nil {
					return err
				}
				err = vhr.LoadDetails(m)
				if err != nil {
					return err
				}
				return templates.VHostChangeUserViewershipFormInner(pd, vhr).Render(m.Ctx(), w)
			},
		)
	}
}
