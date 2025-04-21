// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package shared

import (
	"context"
	"errors"
	"time"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/infra"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type QueueServer interface {
	Enqueue(ctx context.Context, arg infra.EnqueueArg) error
	Dequeue(ctx context.Context, arg infra.DequeueArg) ([]byte, error)
}

func makeLaneID(id proto.KexSessionID, seqno proto.KexSeqNo, actor rem.KexActorType) *infra.QueueLaneID {
	var laneID infra.QueueLaneID
	copy(laneID[:], id[:])
	l := len(laneID)
	laneID[l-1] = byte(seqno)
	laneID[l-2] = byte(actor)

	return &laneID
}

func KexPoke(ctx context.Context, s QueueServer, id proto.KexSessionID, sender proto.EntityID, seqno proto.KexSeqNo, actor rem.KexActorType) error {
	laneID := makeLaneID(id, seqno, actor)
	msg := rem.QueueMsg{
		Seqno:   seqno,
		Seneder: sender,
	}
	msgRaw, err := core.EncodeToBytes(&msg)
	if err != nil {
		return err
	}
	arg := infra.EnqueueArg{
		QueueId: infra.QueueID_Kex,
		LaneId:  *laneID,
		Msg:     msgRaw,
	}

	err = s.Enqueue(ctx, arg)
	if err != nil {
		return err
	}
	return nil
}

func KexWait(
	ctx context.Context,
	s QueueServer,
	id proto.KexSessionID,
	seqno proto.KexSeqNo,
	dur time.Duration,
	actor rem.KexActorType,
) (
	proto.EntityID,
	proto.KexSeqNo,
	error,
) {
	laneID := makeLaneID(id, seqno, actor)
	arg := infra.DequeueArg{
		QueueId: infra.QueueID_Kex,
		LaneId:  *laneID,
		Wait:    proto.ExportDurationMilli(dur),
	}
	msgRaw, err := s.Dequeue(ctx, arg)
	if err != nil {
		return proto.EntityID{}, 0, err
	}
	var msg rem.QueueMsg
	err = core.DecodeFromBytes(&msg, msgRaw)
	if err != nil {
		return proto.EntityID{}, 0, err
	}
	return msg.Seneder, msg.Seqno, nil
}

func oauth2SessionIDToLaneID(id proto.OAuth2SessionID) *infra.QueueLaneID {
	var laneID infra.QueueLaneID
	copy(laneID[:], id[:]) // will leave 2 bytes empty, but that's ok
	return &laneID
}

func OAuth2Poke(
	ctx context.Context,
	s QueueServer,
	id proto.OAuth2SessionID,
	msg proto.OAuth2TokenSet,
) error {
	laneID := oauth2SessionIDToLaneID(id)
	msgRaw, err := core.EncodeToBytes(&msg)
	if err != nil {
		return err
	}
	arg := infra.EnqueueArg{
		QueueId: infra.QueueID_OAuth2,
		LaneId:  *laneID,
		Msg:     msgRaw,
	}
	err = s.Enqueue(ctx, arg)
	if err != nil {
		return err
	}
	return nil
}

func OAuth2Wait(
	ctx context.Context,
	s QueueServer,
	id proto.OAuth2SessionID,
	dur time.Duration,
) (*proto.OAuth2TokenSet, error) {
	laneID := oauth2SessionIDToLaneID(id)
	arg := infra.DequeueArg{
		QueueId: infra.QueueID_OAuth2,
		LaneId:  *laneID,
		Wait:    proto.ExportDurationMilli(dur),
	}
	msgRaw, err := s.Dequeue(ctx, arg)
	if errors.Is(err, core.TimeoutError{}) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var msg proto.OAuth2TokenSet
	err = core.DecodeFromBytes(&msg, msgRaw)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

type NullQueueServer struct{}

func (n *NullQueueServer) Enqueue(ctx context.Context, arg infra.EnqueueArg) error {
	return EmptyConfigError{}
}
func (n *NullQueueServer) Dequeue(ctx context.Context, arg infra.DequeueArg) ([]byte, error) {
	return nil, EmptyConfigError{}
}

var _ QueueServer = (*NullQueueServer)(nil)
