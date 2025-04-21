// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-snowpack-rpc/rpc"
	"go.uber.org/zap"
)

type RpcClientMetaContext struct {
	MetaContext
}

var _ core.RpcClientMetaContexter = RpcClientMetaContext{}

func (m RpcClientMetaContext) Log() *zap.Logger {
	return m.MetaContext.G().Log()
}

func (m RpcClientMetaContext) RPCLogOptions() (rpc.LogOptions, error) {
	return m.MetaContext.G().Cfg().RPCLogOptions()
}

func (m RpcClientMetaContext) Background() core.RpcClientMetaContexter {
	return RpcClientMetaContext{MetaContext: m.MetaContext.Background()}
}

func (m RpcClientMetaContext) WithLogTag(s string) core.RpcClientMetaContexter {
	return RpcClientMetaContext{MetaContext: m.MetaContext.WithLogTag(s)}
}

func NewRpcClient(
	g *GlobalContext,
	addr proto.TCPAddr,
	rootCAs *x509.CertPool,
	clientCert *tls.Certificate,
	opts *core.RpcClientOpts,
) *core.RpcClient {
	return core.NewRpcClient(
		RpcClientMetaContext{MetaContext: NewMetaContextBackground(g)},
		addr,
		rootCAs,
		func(context.Context) (*tls.Certificate, error) { return clientCert, nil },
		opts,
	)
}

func NewRpcTypedClient[
	T ~struct {
		Cli            rpc.GenericClient
		ErrorUnwrapper U
		MakeArgHeader  A
		CheckResHeader R
	},
	U ~func(proto.Status) error,
	A ~func() lcl.Header,
	R ~func(context.Context, lcl.Header) error,
](
	m MetaContext,
	gcli rpc.GenericClient,
) T {
	var errio IOStreamer
	if m.G().UIs().Terminal != nil {
		errio = m.G().UIs().Terminal.ErrorStream()
	} else {
		errio = WrappedStderr
	}
	return T{
		Cli:            gcli,
		ErrorUnwrapper: core.StatusToError,
		MakeArgHeader:  MakeProtoHeader,
		CheckResHeader: MakeCheckProtoResHeader(errio),
	}

}
