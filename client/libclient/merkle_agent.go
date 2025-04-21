// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"context"
	"crypto/x509"
	"errors"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/lib/merkle"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type MerkleAgentLocalStorage struct {
	g      *GlobalContext
	hostID proto.HostID
}

func NewMerkleAgentLocalStorage(g *GlobalContext, hostID proto.HostID) *MerkleAgentLocalStorage {
	return &MerkleAgentLocalStorage{g: g, hostID: hostID}
}

func NewMerkleAgent(g *GlobalContext, hid proto.HostID, addr proto.TCPAddr, cp *x509.CertPool) *merkle.Agent {
	opts := core.NewRpcClientOpts().WithName("merkle_agent")
	opts.ConfigConnHook = core.MakeConfigConnHook(proto.ServerType_MerkleQuery, hid, g)
	storage := NewMerkleAgentLocalStorage(g, hid)
	cli := NewRpcClient(g, addr, cp, nil, opts)
	return merkle.NewAgent(g, storage, cli, hid)
}

type merkleEpno struct {
	proto.MerkleEpno
}

func (m merkleEpno) DbKey() (proto.DbKey, error) {
	b, err := core.EncodeToBytes(&m.MerkleEpno)
	if err != nil {
		return nil, err
	}
	return proto.DbKey(b), nil
}

var _ core.DbKeyer = merkleEpno{}

func (a *MerkleAgentLocalStorage) GetRootHashFromCache(ctx context.Context, ep proto.MerkleEpno) (*proto.MerkleRootHash, error) {
	m := NewMetaContext(ctx, a.g)
	var hsh proto.MerkleRootHash
	_, err := m.DbGet(&hsh, DbTypeSoft, &a.hostID, lcl.DataType_MerkleRootHashByEpno, merkleEpno{ep})
	if errors.Is(err, core.RowNotFoundError{}) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &hsh, nil
}

func (a *MerkleAgentLocalStorage) GetRootFromCache(ctx context.Context, ep proto.MerkleEpno) (*proto.MerkleRoot, error) {
	hsh, err := a.GetRootHashFromCache(ctx, ep)
	if err != nil {
		return nil, err
	}
	if hsh == nil {
		return nil, nil
	}
	return a.getRootWithHash(ctx, hsh)
}

func (a *MerkleAgentLocalStorage) getRootWithHash(ctx context.Context, hsh *proto.MerkleRootHash) (*proto.MerkleRoot, error) {
	m := NewMetaContext(ctx, a.g)
	var root proto.MerkleRoot
	_, err := m.DbGet(&root, DbTypeSoft, &a.hostID, lcl.DataType_MerkleRootByHash, hsh)
	if errors.Is(err, core.RowNotFoundError{}) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &root, nil
}

func (a *MerkleAgentLocalStorage) GetLatestRootFromCache(ctx context.Context) (*proto.MerkleRoot, error) {
	m := NewMetaContext(ctx, a.g)
	c, _, err := m.DbGetCounter(DbTypeHard, &a.hostID, lcl.DataType_MerkleLatestEpno)
	if errors.Is(err, core.RowNotFoundError{}) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a.GetRootFromCache(ctx, proto.MerkleEpno(c))
}

func (a *MerkleAgentLocalStorage) Store(
	ctx context.Context,
	ep proto.MerkleEpno,
	h *proto.MerkleRootHash,
	r *proto.MerkleRoot,
	latest bool,
) error {
	m := NewMetaContext(ctx, a.g)

	if h == nil {
		return core.InternalError("hash must be non-nil")
	}
	if ep == 0 && r == nil {
		return core.InternalError("nothing to store")
	}
	if latest && ep == 0 {
		return core.InternalError("cannot store latest without epno")
	}

	if latest {
		tmp := int64(ep)
		row := PutArg{
			Scope:   &a.hostID,
			Typ:     lcl.DataType_MerkleLatestEpno,
			Counter: &tmp,
		}
		err := m.DbPut(DbTypeHard, row)
		if err != nil {
			return err
		}
	}

	var rows []PutArg
	if ep > 0 {
		rows = append(rows, PutArg{
			Scope: &a.hostID,
			Typ:   lcl.DataType_MerkleRootHashByEpno,
			Key:   merkleEpno{ep},
			Val:   h,
		})
	}

	if r != nil {
		rows = append(rows, PutArg{
			Scope: &a.hostID,
			Typ:   lcl.DataType_MerkleRootByHash,
			Key:   h,
			Val:   r,
		})
	}

	err := m.DbPutTx(DbTypeSoft, rows)
	if err != nil {
		return err
	}
	return nil
}
