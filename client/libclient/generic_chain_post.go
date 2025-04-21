// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func MakeGenericLink(
	m MetaContext,
	au *UserContext,
	payload proto.GenericLinkPayload,
) (
	*core.MakeLinkRes,
	error,
) {
	usl := NewUserSettingsLoader(au)
	chain, err := usl.Run(m)
	if err != nil {
		return nil, err
	}

	root, err := usl.LatestTreeRoot(m)
	if err != nil {
		return nil, err
	}
	fqe := au.FQU().ToFQEntity()

	dev, err := au.Devkey(m.Ctx())
	if err != nil {
		return nil, err
	}

	seqno := proto.ChainEldestSeqno
	var prev *proto.LinkHash
	if chain != nil {
		seqno = chain.Tail.Base.Seqno + 1
		prev = &chain.LastHash
	}

	return core.MakeGenericLink(
		fqe.Entity,
		fqe.Host,
		dev,
		payload,
		seqno,
		prev,
		*root,
	)
}
