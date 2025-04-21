// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

import (
	"context"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/infra"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type NativeQueueService struct {
	bec *BackendClient
}

func NewNativeQueueService(g *GlobalContext, callerType proto.ServerType) *NativeQueueService {
	return &NativeQueueService{
		bec: NewBackendClient(g, proto.ServerType_Queue, callerType, nil),
	}
}

func (n *NativeQueueService) cli(ctx context.Context) (*infra.QueueClient, error) {
	gcli, err := n.bec.Cli(ctx)
	if err != nil {
		return nil, err
	}
	return &infra.QueueClient{Cli: gcli, ErrorUnwrapper: core.StatusToError}, nil
}
func (n *NativeQueueService) Enqueue(ctx context.Context, arg infra.EnqueueArg) error {
	cli, err := n.cli(ctx)
	if err != nil {
		return err
	}
	return cli.Enqueue(ctx, arg)
}

func (n *NativeQueueService) Dequeue(ctx context.Context, arg infra.DequeueArg) ([]byte, error) {
	cli, err := n.cli(ctx)
	if err != nil {
		return nil, err
	}
	return cli.Dequeue(ctx, arg)
}

var _ QueueServer = (*NativeQueueService)(nil)
