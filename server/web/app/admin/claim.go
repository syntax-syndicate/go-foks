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

func ClaimHandler(
	g *shared.GlobalContext,
	doAdd bool,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		opts := ServeOpts{CSRFCheck: true}
		newBaseHandler(g).ServeLoggedIn(w, r, opts,
			func(m shared.MetaContext, u *lib.User) error {
				tid, err := lib.NewArgs(r).TeamID(lib.ParamSourceURL, "tid")
				if err != nil {
					return err
				}
				m = m.WithHostID(&u.Host.HostID)
				err = shared.ChangeQuotaMasterRetry(m, doAdd, u.Uid, *tid)
				if err != nil {
					return err
				}
				urs, err := lib.LoadUsageData(m, u.Name)
				if err != nil {
					return err
				}
				findRes := urs.FindRow(tid.ToPartyID())
				if findRes == nil {
					return core.NotFoundError("team row")
				}
				err = templates.Claim(*findRes, urs).Render(m.Ctx(), w)
				if err != nil {
					return err
				}
				doDebugDelay(m)
				return nil
			},
		)
	}
}
