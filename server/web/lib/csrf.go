// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"time"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/server/shared"
)

type CSRFToken string

func (t CSRFToken) String() string {
	return string(t)
}

func CreateCSRFToken(
	m shared.MetaContext,
	uid proto.UID,
) (
	CSRFToken,
	error,
) {
	id, key, err := shared.LookupLatestChallengeKey(m, shared.HmacKeyCSRFProtect)
	if err != nil {
		return "", err
	}
	now := m.Now()
	payload := proto.CSRFPayload{
		Uid:   uid,
		Etime: proto.ExportTime(now.Add(time.Duration(24*30) * time.Hour)),
	}
	res, err := core.Hmac(&payload, key)
	if err != nil {
		return "", err
	}
	tok1 := proto.CSRFTokenV1{
		KeyID: *id,
		Hmac:  *res,
		Etime: payload.Etime,
	}
	tok := proto.NewCSRFTokenWithV1(tok1)
	b, err := core.EncodeToBytes(&tok)
	if err != nil {
		return "", err
	}
	return CSRFToken(core.B62Encode(b)), nil
}

func CheckCSRFToken(
	m shared.MetaContext,
	uid proto.UID,
	tok CSRFToken,
) error {
	b, err := core.B62Decode(tok.String())
	if err != nil {
		return err
	}
	var obj proto.CSRFToken
	err = core.DecodeFromBytes(&obj, b)
	if err != nil {
		return err
	}
	vers, err := obj.GetV()
	if err != nil {
		return err
	}
	if vers != proto.CSRFTokenVersion_V1 {
		return core.VersionNotSupportedError("csrf token from future")
	}
	v1 := obj.V1()
	db, err := m.Db(shared.DbTypeUsers)
	if err != nil {
		return err
	}
	defer db.Release()
	key, err := shared.LookupHMACKeyByID(m, db, v1.KeyID, shared.HmacKeyCSRFProtect)
	if err != nil {
		return err
	}

	payload := proto.CSRFPayload{
		Uid:   uid,
		Etime: v1.Etime,
	}

	computed, err := core.Hmac(&payload, key)
	if err != nil {
		return err
	}
	if !computed.Eq(v1.Hmac) {
		return core.VerifyError("bad hmac")
	}
	tm := v1.Etime.Import()
	if m.Now().After(tm) {
		return core.ExpiredError{}
	}
	return nil
}
