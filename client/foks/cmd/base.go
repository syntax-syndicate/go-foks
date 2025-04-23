// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cmd

import (
	"context"

	"github.com/foks-proj/go-foks/client/agent"
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-snowpack-rpc/rpc"
)

func quickStart[
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
	m libclient.MetaContext,
	opts *agent.StartupOpts,
) (
	T,
	func(),
	error,
) {
	if opts == nil {
		opts = &agent.StartupOpts{}
	}
	err := agent.Startup(m, agent.StartupOpts(*opts))
	if err != nil {
		return T{}, nil, err
	}
	gcli, cleanFn, err := m.G().ConnectToAgentCli(m.Ctx())
	if err != nil {
		return T{}, nil, err
	}
	return newClient[T](m, gcli), cleanFn, nil
}

func quickStartLambda[
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
	m libclient.MetaContext,
	opts *agent.StartupOpts,
	fn func(T) error,
) error {
	cli, clean, err := quickStart[T](m, opts)
	if err != nil {
		return err
	}
	defer clean()
	err = fn(cli)
	return err
}

func withClient[
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
	m libclient.MetaContext,
	fn func(T) error,
) error {
	gcli, cleanFn, err := m.G().ConnectToAgentCli(m.Ctx())
	if err != nil {
		return err
	}
	defer cleanFn()
	return fn(newClient[T, U, A, R](m, gcli))
}

func newClient[
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
	m libclient.MetaContext,
	gcli rpc.GenericClient,
) T {
	return libclient.NewRpcTypedClient[T](m, gcli)
}
