// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package oauth2

import (
	"net/http"

	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/server/shared"
	"github.com/foks-proj/go-foks/server/web/lib"
	"github.com/foks-proj/go-foks/server/web/templates"
)

func TinyHandler(g *shared.GlobalContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		lib.NewBaseHandler(g).ServeWithVHost(w, r, func(m shared.MetaContext) error {
			args := lib.NewArgs(r)
			oid, err := args.OAuth2SessionID(lib.ParamSourceURL, "id")
			if err != nil {
				return err
			}
			url, err := shared.OAuth2AuthRedirect(m, *oid)
			if err != nil {
				return err
			}
			return lib.RedirectError{Url: url.String()}
		})
	}
}

func CallbackHandler(g *shared.GlobalContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		lib.NewBaseHandler(g).ServeWithVHost(w, r, func(m shared.MetaContext) error {
			args := lib.NewArgs(r)
			oid, err := args.OAuth2SessionID(lib.ParamSourceQuery, "state")
			if err != nil {
				return err
			}
			codeRaw, err := args.String(lib.ParamSourceQuery, "code")
			if err != nil {
				return err
			}
			code := proto.OAuth2Code(codeRaw)

			err = shared.OAuth2ExchangeCodeForToken(m, &shared.ExchangeArg{
				Oid: *oid, Code: code,
			})
			if err != nil {
				return err
			}
			page := templates.OAuth2Callback()
			pd, err := lib.LoadOAuth2PageData(m, lib.DataLoadOpts{Headers: true, PageTitle: "FOKS OAuth2 Callback"})
			if err != nil {
				return err
			}
			return templates.Layout(page, pd).Render(m.Ctx(), w)
		})
	}
}
