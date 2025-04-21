// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"crypto/hmac"
	"fmt"

	"golang.org/x/crypto/argon2"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

var PassphraseStreamLen = 64

type PassphraseStream []byte

type StretchedPassphrase struct {
	salt   proto.PassphraseSalt
	ppgen  proto.PassphraseGeneration
	svers  proto.StretchVersion
	stream PassphraseStream
	skey   core.PrivateSuiter
	pubkey core.PublicSuiter
}

type StretchOpts struct {
	IsTest bool
}

type stretchFn func(raw proto.Passphrase, salt proto.PassphraseSalt, len uint32) PassphraseStream

var stretchers = map[proto.StretchVersion]stretchFn{

	// For testing so that we don't take too long on our tests. Please never use in
	// production.
	proto.StretchVersion_TEST: func(raw proto.Passphrase, salt proto.PassphraseSalt, len uint32) PassphraseStream {
		return argon2.IDKey([]byte(raw), salt[:], 1, 512, 1, len)
	},

	// For production, V1. Likely we'll never update.
	proto.StretchVersion_V1: func(raw proto.Passphrase, salt proto.PassphraseSalt, len uint32) PassphraseStream {
		return argon2.IDKey([]byte(raw), salt[:], 3, 64*1024, 2, len)
	},
}

func NewStretchedPassphrase(
	opts StretchOpts,
	raw proto.Passphrase,
	salt proto.PassphraseSalt,
	ppg proto.PassphraseGeneration,
	sv proto.StretchVersion,
) (*StretchedPassphrase, error) {

	fn := stretchers[sv]
	if fn == nil {
		return nil, core.VersionNotSupportedError(fmt.Sprintf("cannot support passphrase stretch version %d", sv))
	}
	if sv == proto.StretchVersion_TEST && !opts.IsTest {
		return nil, core.TestingOnlyError{}
	}
	b := fn(raw, salt, uint32(PassphraseStreamLen))
	return &StretchedPassphrase{
		salt:   salt,
		ppgen:  ppg,
		svers:  sv,
		stream: ((PassphraseStream)(b)),
	}, nil
}

func (s *StretchedPassphrase) SecretBoxKey() proto.SecretBoxKey {
	var ret proto.SecretBoxKey
	copy(ret[:], s.stream[32:64])
	return ret
}

func (s *StretchedPassphrase) SecretKeySuite() (core.PrivateSuiter, error) {
	if s.skey != nil {
		return s.skey, nil
	}
	seed := proto.SecretSeed32(s.stream[0:32])
	var emptyHost proto.HostID
	ret, err := core.NewPrivateSuite25519(
		proto.EntityType_PassphraseKey,
		proto.NewRoleDefault(proto.RoleType_NONE),
		seed,
		emptyHost,
	)
	if err != nil {
		return nil, err
	}
	s.skey = ret
	return ret, nil
}

func (s *StretchedPassphrase) PublicKeySuite() (core.PublicSuiter, error) {
	if s.pubkey != nil {
		return s.pubkey, nil
	}
	priv, err := s.SecretKeySuite()
	if err != nil {
		return nil, err
	}
	pub, err := priv.Publicize(nil)
	if err != nil {
		return nil, err
	}
	s.pubkey = pub
	return pub, nil
}

// testEq tests that the two passphrases are the same. We likely will only
// wind up using this on test, since in a distributed setting, the server
// doesn't get to see stretched passphrase
func (s *StretchedPassphrase) Eq(s2 *StretchedPassphrase) bool {
	return hmac.Equal(s.stream, s2.stream)
}
