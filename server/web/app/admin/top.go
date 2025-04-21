// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package admin

import (
	"net/http"

	"github.com/foks-proj/go-foks/server/shared"
	"github.com/foks-proj/go-foks/server/web/lib"
	"github.com/foks-proj/go-foks/server/web/templates"
)

type AdminHandler struct{}

func (a AdminHandler) URL() string       { return "/admin" }
func (a AdminHandler) RedirectTo() error { return lib.RedirectError{Url: a.URL()} }

func MainPanelHandler(g *shared.GlobalContext) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		opts := ServeOpts{}
		lopts := lib.DataLoadOpts{
			Usage:  true,
			VHosts: true,
			Which: lib.PageSelector{
				PageType: lib.PageTypeUsage,
			},
		}
		newBaseHandler(g).ServeWithDataLoad(w, r, opts, lopts, func(m shared.MetaContext, pd *lib.AdminPageData) error {
			return templates.AdminMain(pd).Render(m.Ctx(), w)
		})
	}
}

func renderTop(m shared.MetaContext, pd *lib.AdminPageData, w http.ResponseWriter) error {
	c := templates.Admin(pd)
	return templates.Layout(c, pd).Render(m.Ctx(), w)
}

func TopHandler(g *shared.GlobalContext) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		newBaseHandler(g).ServeWithDataLoad(w, r,
			ServeOpts{CSRFSend: true},
			lib.DataLoadOpts{
				Usage:    true,
				Headers:  true,
				VHosts:   true,
				UserPlan: true,
				Which: lib.PageSelector{
					PageType: lib.PageTypeUsage,
				},
			},
			func(m shared.MetaContext, pd *lib.AdminPageData) error {
				return renderTop(m, pd, w)
			},
		)
	}
}
