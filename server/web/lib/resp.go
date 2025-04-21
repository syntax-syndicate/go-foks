// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"context"
)

type HeaderData struct {
	Title string
	Nonce NoncePackage
}

func NewHeaderData(ctx context.Context, t string) *HeaderData {
	ret := &HeaderData{Title: t}
	ret.loadFromContext(ctx)
	return ret
}

func (r *HeaderData) loadFromContext(ctx context.Context) {
	np := NoncePkgFromContext(ctx)
	if np != nil {
		r.Nonce = *np
	}
}
