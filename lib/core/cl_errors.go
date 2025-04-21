// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"fmt"

	proto "github.com/foks-proj/go-foks/proto/lib"
)

// User loader errors all are in this file for better organization.

type CLBadTreeLocationError struct {
	Err   error
	Seqno proto.Seqno
}

type CLBadMerklePathError struct {
	Err   error
	Which string
	Seqno proto.Seqno
}

func (b CLBadMerklePathError) Error() string {
	return fmt.Sprintf("bad merkle %s path @ seqno=%d: %s", b.Which, b.Seqno, b.Err.Error())
}

func (b CLBadTreeLocationError) Error() string {
	return fmt.Sprintf("bad tree location @ seqno=%d: %s", b.Seqno, b.Err.Error())
}

type CLBadCountError struct {
	Which    string
	Expected int
	Actual   int
}

func (b CLBadCountError) Error() string {
	return fmt.Sprintf("bad %s count: expected %d, got %d", b.Which, b.Expected, b.Actual)
}

type CLInvalidSeqnoError struct{}

func (i CLInvalidSeqnoError) Error() string {
	return "invalid seqno; all must be >=1"
}

type CLBadSeqnoError struct {
	Which    string
	Expected proto.Seqno
	Actual   proto.Seqno
}

func (b CLBadSeqnoError) Error() string {
	return fmt.Sprintf("bad %s seqno: expected %d, got %d", b.Which, b.Expected, b.Actual)
}

type CLBadPrevError struct {
	Seqno    proto.Seqno
	Expected *proto.LinkHash
	Actual   *proto.LinkHash
}

type CLProvisionError struct {
	Seqno proto.Seqno
	Err   error
}

func (p CLProvisionError) Error() string {
	return fmt.Sprintf("provision error @ seqno=%d: %s", p.Seqno, p.Err.Error())
}

func (b CLBadPrevError) Error() string {
	return fmt.Sprintf("bad prev: at seqno=%d, expected %s, got %s", b.Seqno, b.Expected.DebugString(), b.Actual.DebugString())
}

type CLOpenLinkError struct {
	Err error
	N   int
}

func (o CLOpenLinkError) Error() string {
	return fmt.Sprintf("open link error @%d: %s", o.N, o.Err.Error())
}

type CLBadMerkleLeafValueError struct {
	Which string
	Seqno proto.Seqno
}

func (b CLBadMerkleLeafValueError) Error() string {
	return fmt.Sprintf("bad merkle leaf value of type %s @ seqno=%d", b.Which, b.Seqno)
}

type CLMissingSubchainTreeLocationSeedError struct{}

func (c CLMissingSubchainTreeLocationSeedError) Error() string {
	return "server should have returned a sigchain tree location seed but didn't"
}

type ULEldestError struct {
	Err error
}

func (e ULEldestError) Error() string {
	return fmt.Sprintf("eldest error: %s", e.Err.Error())
}

type ULRevokeError struct {
	Seqno proto.Seqno
	Err   error
}

func (r ULRevokeError) Error() string {
	return fmt.Sprintf("revoke error @ seqno=%d: %s", r.Seqno, r.Err.Error())
}

type ULOpenCommitmentError struct {
	Which string
	Idx   int
	Err   error
}

func (o ULOpenCommitmentError) Error() string {
	return fmt.Sprintf("open %s@%d commitment error: %s", o.Which, o.Idx, o.Err.Error())
}

type CLInvalidSignerError struct {
	Fqe   proto.FQEntity
	Seqno proto.Seqno
	Err   error
}

func (u CLInvalidSignerError) Error() string {
	s, _ := u.Fqe.Entity.StringErr()
	var errstr string
	if u.Err != nil {
		errstr = " (" + u.Err.Error() + ")"
	}
	return fmt.Sprintf("invalid signer @ %d: %s%s", u.Seqno, s, errstr)
}

type CLBadMerkleLookupError struct {
	Err   error
	Seqno proto.Seqno
}

func (c CLBadMerkleLookupError) Error() string {
	return fmt.Sprintf("bad merkle lookup @ seqno=%d: %s", c.Seqno, c.Err.Error())
}

type CLBadMerkleRootHashError struct {
	Epno    proto.MerkleEpno
	Exected proto.MerkleRootHash
	Actual  proto.MerkleRootHash
}

func (c CLBadMerkleRootHashError) Error() string {
	return fmt.Sprintf("bad merkle root hash @ epno=%d", c.Epno)
}

type CLBadMerkleHistoricalRootError struct {
	Err  error
	Epno proto.MerkleEpno
}

func (c CLBadMerkleHistoricalRootError) Error() string {
	return fmt.Sprintf("bad merkle historical root @ epno=%d: %s", c.Epno, c.Err.Error())
}

type CLBadMerkleVerifyPresenceError struct {
	Err  error
	Epno proto.MerkleEpno
	Key  proto.MerkleTreeRFOutput
}

func (c CLBadMerkleVerifyPresenceError) Error() string {
	return fmt.Sprintf("verify presence failed @ epno=%d: %s", c.Epno, c.Err.Error())
}

type CLBadMerkleHistoricalLeafValueError struct {
	Epno proto.MerkleEpno
	Key  proto.MerkleTreeRFOutput
}

func (e CLBadMerkleHistoricalLeafValueError) Error() string {
	return fmt.Sprintf("bad merkle historical leaf value @ epno=%d; hash mismatch", e.Epno)
}

type CLBadLinkNoRoot struct {
	Seqno proto.Seqno
}

func (e CLBadLinkNoRoot) Error() string {
	return fmt.Sprintf("bad link: no root @ seqno=%d", e.Seqno)
}

type CLBadKeySequenceError struct {
	Gen   proto.Generation
	Role  proto.Role
	Which string
}

func (e CLBadKeySequenceError) Error() string {
	rs, _ := e.Role.StringErr()
	return fmt.Sprintf("bad key sequence for %s @ role=%s, gen=%d", e.Which, rs, e.Gen)
}

type CLBoxError struct {
	Gen  proto.Generation
	Role proto.Role
	Desc string
}

func (e CLBoxError) Error() string {
	rs, _ := e.Role.StringErr()
	return fmt.Sprintf("bad box at @ role=%s, gen=%d: %s", rs, e.Gen, e.Desc)
}

type CLIndexRangeError struct {
	Msg   string
	Seqno proto.Seqno
}

func (e CLIndexRangeError) Error() string {
	return fmt.Sprintf("index range error @%d: %s", e.Seqno, e.Msg)
}

// ChainLoaderErrors are relevant for both generic chain loads, and user loads.
// Some error subtypes are only relevant for user loading, and those are prefixed with "UL"
// and not "CL".
type ChainLoaderError struct {
	Err  error
	Race bool // If set, it might be due to a merkle race, in which case, it should be retried
}

func (u ChainLoaderError) Error() string {
	return fmt.Sprintf("chain loader error: %s", u.Err.Error())
}

func (u ChainLoaderError) Unwrap() error {
	return u.Err
}

type UserSettingsError string

func (u UserSettingsError) Error() string {
	return "user settings error: " + string(u)
}
