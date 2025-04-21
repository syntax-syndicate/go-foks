// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package admin

import (
	"net/http"

	"github.com/foks-proj/go-foks/proto/rem"
	"github.com/foks-proj/go-foks/server/shared"
	"github.com/foks-proj/go-foks/server/web/lib"
	"github.com/foks-proj/go-foks/server/web/templates"
)

type IndexHandler struct {
	*BaseHandler
}

func NewIndexHandler(g *shared.GlobalContext) *IndexHandler {
	return &IndexHandler{
		BaseHandler: newBaseHandler(g),
	}
}

func (i IndexHandler) RedirectTo() error {
	return lib.RedirectError{
		Url: i.URL(),
	}
}

func (i IndexHandler) URL() string { return "/" }

func (i *IndexHandler) handleLogin(
	m shared.MetaContext,
	w http.ResponseWriter,
	r *http.Request,
) error {
	cfg, err := m.G().Config().WebConfig(m.Ctx())
	if err != nil {
		return err
	}
	param := cfg.SessionParam()
	s := r.URL.Query().Get(param)
	if s == "" {
		return nil
	}
	ret := lib.RedirectError{
		Url: "/",
	}
	// failure to parse, means no login accepted
	ws, err := rem.WebSessionString(s).Parse()
	if err != nil {
		return ret
	}
	wu, err := lib.LoadUserBySession(m, *ws)
	if err != nil {
		return ret
	}
	if wu == nil {
		return ret
	}
	err = lib.SetSessionCookie(m, w, *ws)
	if err != nil {
		return ret
	}
	return AdminHandler{}.RedirectTo()
}

func (i *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i.Serve(w, r, func(m shared.MetaContext) error {
		err := i.handleLogin(m, w, r)
		if err != nil {
			return err
		}
		_, err = i.LoggedInUser(m, r)
		if err == nil {
			return AdminHandler{}.RedirectTo()
		}
		pd, err := lib.LoadAdminPageData(m, nil, lib.DataLoadOpts{Headers: true, PageTitle: "FOKS"})
		if err != nil {
			return err
		}
		c := templates.Index()
		return templates.Layout(c, pd).Render(r.Context(), w)
	})

}
