// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package team

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

func SignBearerTokenChallenge(
	fqu proto.FQUser,
	teamid proto.TeamID,
	role proto.Role,
	gen proto.Generation,
	tok rem.TeamBearerToken,
	key core.EntityPrivate,
) (
	*proto.Signature,
	rem.TeamBearerTokenChallengeBlob,
	error,
) {
	p := rem.TeamBearerTokenChallengePayload{
		User: fqu,
		Team: teamid,
		Role: role,
		Gen:  gen,
		Tok:  tok,
		Tm:   proto.Now(),
	}
	sig, o, err := core.Sign2(key, &p)
	if err != nil {
		return nil, nil, err
	}
	return sig, *o, nil
}

type BearerTokenState string

const (
	BearerTokenStateInert   BearerTokenState = "inert"
	BearerTokenStateActive  BearerTokenState = "active"
	BearerTokenStateExpired BearerTokenState = "expired"
	BearerTokenStateRevoked BearerTokenState = "revoked"
)
