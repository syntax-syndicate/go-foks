package agent

import (
	"context"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func (a *AgentConn) LogSend(
	ctx context.Context,
	arg lcl.LogSendSet,
) (lcl.LogSendRes, error) {
	var zed lcl.LogSendRes
	logs := core.Map(arg.Files, func(p proto.LocalFSPath) core.Path {
		return core.ImportPath(p)
	})
	m := a.MetaContext(ctx)
	res, err := libclient.LogSend(m, logs)
	if err != nil {
		return zed, err
	}
	return *res, nil
}

func (a *AgentConn) LogSendList(
	ctx context.Context,
	n uint64,
) (
	lcl.LogSendSet,
	error,
) {
	var zed lcl.LogSendSet
	m := a.MetaContext(ctx)
	logs, err := libclient.ListLatestLogs(m, int(n))
	if err != nil {
		return zed, err
	}
	return lcl.LogSendSet{
		Files: core.Map(logs, func(p core.Path) proto.LocalFSPath { return p.Export() }),
	}, nil
}

var _ lcl.LogSendInterface = (*AgentConn)(nil)
