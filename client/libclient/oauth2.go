// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"time"

	"github.com/foks-proj/go-foks/lib/sso"
)

type OAuth2GlobalContext struct {
	g              *GlobalContext
	refreshInteral time.Duration
	requestTimeout time.Duration
}

func (o *OAuth2GlobalContext) Now() time.Time                     { return o.g.Now() }
func (o *OAuth2GlobalContext) ConfigSet() *sso.OAuth2IdPConfigSet { return o.g.oauth2ConfigSet }
func (o *OAuth2GlobalContext) RefreshInterval() time.Duration     { return o.refreshInteral }
func (o *OAuth2GlobalContext) RequestTimeout() time.Duration      { return o.requestTimeout }

var _ sso.OAuth2GlobalContext = (*OAuth2GlobalContext)(nil)

func (g *GlobalContext) OAuth2GlobalContext() (*OAuth2GlobalContext, error) {
	ri, err := g.Cfg().GetOAuth2RefreshInterval()
	if err != nil {
		return nil, err
	}
	rt, err := g.Cfg().GetOAuth2RequestTimeout()
	if err != nil {
		return nil, err
	}
	return &OAuth2GlobalContext{
		g:              g,
		refreshInteral: ri,
		requestTimeout: rt,
	}, nil
}
