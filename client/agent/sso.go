// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import proto "github.com/foks-proj/go-foks/proto/lib"

type SSOLoginSession struct {
	SessionBase
	id     proto.UISessionID
	oauth2 *proto.OAuth2Session
	ssoCfg *proto.SSOConfig
}

func (l *SSOLoginSession) Init(id proto.UISessionID) {
	l.SessionBase.Init()
	l.id = id
}
