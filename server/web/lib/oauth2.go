// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import "github.com/foks-proj/go-foks/server/shared"

type OAuth2PageData struct {
	Head *HeaderData
}

func (d *OAuth2PageData) GetHead() *HeaderData {
	if d == nil {
		return nil
	}
	return d.Head
}

func (d *OAuth2PageData) GetCSRFToken() *CSRFToken {
	return nil
}

var _ PageDataer = (*OAuth2PageData)(nil)

func (o *OAuth2PageData) load(
	m shared.MetaContext,
	opts DataLoadOpts,
) error {
	if opts.Headers {
		title := opts.PageTitle
		if title == "" {
			title = "FOKS Admin Control Panel"
		}
		o.Head = NewHeaderData(m.Ctx(), title)
	}
	return nil
}

func LoadOAuth2PageData(
	m shared.MetaContext,
	opts DataLoadOpts,
) (
	*OAuth2PageData,
	error,
) {
	ret := OAuth2PageData{}
	err := ret.load(m, opts)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
