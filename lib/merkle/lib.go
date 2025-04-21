// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package merkle

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func HashNode(n *proto.MerkleNode, h *proto.MerkleNodeHash) error {
	return core.PrefixedHashInto(n, (*h)[:])
}

func HashRoot(r *proto.MerkleRoot, h *proto.MerkleRootHash) error {
	return core.PrefixedHashInto(r, (*h)[:])
}

func HashBackPointers(b *proto.MerkleBackPointers, h *proto.MerkleBackPointerHash) error {
	return core.PrefixedHashInto(b, (*h)[:])
}

func ToTreeRoot(m *proto.MerkleRoot) (*proto.TreeRoot, error) {
	var hsh proto.MerkleRootHash
	err := HashRoot(m, &hsh)
	if err != nil {
		return nil, err
	}
	v, err := m.GetV()
	if err != nil {
		return nil, err
	}
	if v != proto.MerkleRootVersion_V1 {
		return nil, core.VersionNotSupportedError("merkle root version from future")
	}
	return &proto.TreeRoot{
		Epno: m.V1().Epno,
		Hash: hsh,
	}, nil

}
