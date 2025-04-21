// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"context"
	"encoding/hex"

	"github.com/foks-proj/go-foks/lib/core"
)

type NoncePackage struct {
	StyleSrcElem string
	ScriptSrc    string
}

func randomHex() (string, error) {
	var buf [16]byte
	err := core.RandomFill(buf[:])
	if err != nil {
		return "", err
	}
	ret := hex.EncodeToString(buf[:])
	return ret, nil
}

func NewNoncePackage() (*NoncePackage, error) {
	var ret NoncePackage
	var err error
	ret.StyleSrcElem, err = randomHex()
	if err != nil {
		return nil, err
	}
	ret.ScriptSrc, err = randomHex()
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

type noncePkgKeyType string

var noncePkgKey noncePkgKeyType = "nonces"

func (n *NoncePackage) AddToCtx(ctx context.Context) context.Context {
	return context.WithValue(ctx, noncePkgKey, *n)
}

func NoncePkgFromContext(ctx context.Context) *NoncePackage {
	raw := ctx.Value(noncePkgKey)
	if raw == nil {
		return nil
	}
	ret, ok := raw.(NoncePackage)
	if !ok {
		return nil
	}
	return &ret
}
