// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libgit

import (
	"github.com/foks-proj/go-foks/client/libclient"
)

type DbgLogger struct {
	mctx libclient.MetaContext
}

func NewDbgLogger(mctx libclient.MetaContext) *DbgLogger {
	return &DbgLogger{mctx: mctx}
}

func (l *DbgLogger) Log(s string) {
	l.mctx.Infow("git helper", "msg", s)
}
