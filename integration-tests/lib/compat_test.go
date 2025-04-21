// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

import (
	"context"
	"testing"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/stretchr/testify/require"
)

func TestOldClient(t *testing.T) {
	ctx := context.Background()
	cli, closeFn, err := newRegClient(ctx)
	require.NoError(t, err)
	defer closeFn()

	un, err := RandomUsername(8)
	require.NoError(t, err)

	unn, err := core.NormalizeName(proto.NameUtf8(un))
	require.NoError(t, err)
	cli.MakeArgHeader = func() proto.Header {
		return proto.Header{}
	}

	_, err = cli.ReserveUsername(ctx, unn)
	require.Error(t, err)
	require.Equal(t, err, core.VersionNotSupportedError("client not supported: proto header version 0"))
}

func TestNewClientNewProtoVersion(t *testing.T) {
	ctx := context.Background()
	cli, closeFn, err := newRegClient(ctx)
	require.NoError(t, err)
	defer closeFn()

	un, err := RandomUsername(8)
	require.NoError(t, err)

	unn, err := core.NormalizeName(proto.NameUtf8(un))
	require.NoError(t, err)
	cli.MakeArgHeader = func() proto.Header {
		ret := proto.Header{}
		ret.V = proto.HeaderVersion(2)
		return ret
	}

	_, err = cli.ReserveUsername(ctx, unn)
	require.Error(t, err)
	require.Equal(t, err, core.VersionNotSupportedError("client not supported: proto header version 2"))
}

func TestNewClientNewCompatVersion(t *testing.T) {
	ctx := context.Background()
	cli, closeFn, err := newRegClient(ctx)
	require.NoError(t, err)
	defer closeFn()

	un, err := RandomUsername(8)
	require.NoError(t, err)

	unn, err := core.NormalizeName(proto.NameUtf8(un))
	require.NoError(t, err)
	cli.MakeArgHeader = func() proto.Header {
		ret := proto.NewHeaderWithV1(
			proto.HeaderV1{
				Vers: proto.CompatibilityVersion(2),
			},
		)
		return ret
	}
	g := globalTestEnv.G
	var msg string
	var keysAndValues []interface{}
	g.TestWarner = func(m string, k ...interface{}) {
		msg = m
		keysAndValues = k
	}
	defer func() {
		g.TestWarner = nil
	}()

	_, err = cli.ReserveUsername(ctx, unn)
	require.NoError(t, err)

	require.Equal(t, "ClientVersion", msg)
	require.Equal(t, 6, len(keysAndValues))
	require.Equal(t, "versionDiff", keysAndValues[0])
	require.Equal(t, -1, keysAndValues[1])
	require.Equal(t, "clientVersion", keysAndValues[2])
	require.Equal(t, proto.CompatibilityVersion(2), keysAndValues[3])
	require.Equal(t, "serverVersion", keysAndValues[4])
	require.Equal(t, proto.CompatibilityVersion(1), keysAndValues[5])
}

func TestServerCompat(t *testing.T) {

	tew := ForkNewTestEnvWrapper(t)
	defer tew.Shutdown()
	tew.G.TestMakeResHeader = func() proto.Header {
		return proto.Header{}
	}

	ctx := context.Background()
	cli := tew.regCli(t)

	makeUnn := func() proto.Name {
		un, err := RandomUsername(8)
		require.NoError(t, err)

		unn, err := core.NormalizeName(proto.NameUtf8(un))
		require.NoError(t, err)
		return unn
	}

	unn := makeUnn()
	_, err := cli.ReserveUsername(ctx, unn)
	require.Error(t, err)
	require.Equal(t, err, core.VersionNotSupportedError("server not supported: proto header version 0; please ask for upgrade"))

	tew.G.TestMakeResHeader = func() proto.Header {
		return proto.NewHeaderWithV1(
			proto.HeaderV1{},
		)
	}

	// interesting: the RPC actually went through the first time, by design. So we need to ask
	// to reserve a second username or else we'll fail the RPC.
	unn = makeUnn()
	_, err = cli.ReserveUsername(ctx, unn)
	require.Error(t, err)
	require.Equal(t, err, core.VersionNotSupportedError("server is too old: compatibility version 0; please ask for upgrade"))

	tew.G.TestMakeResHeader = func() proto.Header {
		return proto.NewHeaderWithV1(
			proto.HeaderV1{
				Vers: core.CurrentCompatibilityVersion + 1,
			},
		)
	}
	var msg string
	var keysAndValues []interface{}

	tew.G.TestWarner = func(m string, k ...interface{}) {
		msg = m
		keysAndValues = k
	}

	unn = makeUnn()
	_, err = cli.ReserveUsername(ctx, unn)
	require.NoError(t, err)

	require.Equal(t, "ServerVersion", msg)
	require.Equal(t, 6, len(keysAndValues))
	require.Equal(t, "versionDiff", keysAndValues[0])
	require.Equal(t, -1, keysAndValues[1])
	require.Equal(t, "clientVersion", keysAndValues[2])
	require.Equal(t, proto.CompatibilityVersion(1), keysAndValues[3])
	require.Equal(t, "serverVersion", keysAndValues[4])
	require.Equal(t, proto.CompatibilityVersion(2), keysAndValues[5])
}
