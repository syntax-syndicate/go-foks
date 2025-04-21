// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package team

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

// The KeySchedule is the list of changes incoming to a team. Our check here is to make sure that it
// matches the KeyBoxes that came along with the schedule.
func (s KeySchedule) MatchBoxes(ptks proto.SharedKeyBoxSet, host proto.HostID) error {

	var i int
	nxt := func() *proto.SharedKeyBox {
		if i >= len(ptks.Boxes) {
			return nil
		}
		ret := &ptks.Boxes[i]
		i++
		return ret

	}

	for _, item := range s.Items {
		rx := item.Role.Export()
		for _, m := range item.Members {
			skb := nxt()
			if skb == nil {
				return core.TeamError("ran out of shared key boxes")
			}
			err := rx.AssertEq(skb.Role, core.TeamError("shared key box role mismatch"))
			if err != nil {
				return err
			}
			if skb.Gen != item.Gen {
				return core.TeamError("shared key box gen mismatch")
			}
			if skb.Targ.Host != nil && !skb.Targ.Host.Eq(m.Fqe.Host) {
				return core.TeamError("mismatched Host in MatchBoxes")
			}
			if !skb.Targ.Eid.Eq(m.Fqe.Entity.Unfix()) {
				return core.TeamError("mismatched EntityID in MatchBoxes")
			}
			err = skb.Targ.Role.AssertEq(m.SrcRole.Export(), core.TeamError("shared key box srcRole mismatch"))
			if err != nil {
				return err
			}
		}
	}

	if nxt() != nil {
		return core.TeamError("too many shared key boxes")
	}

	return nil
}

func (s KeySchedule) MatchPublicKeys(sk []proto.SharedKey) error {

	var i int
	nxt := func() *proto.SharedKey {
		if i >= len(sk) {
			return nil
		}
		ret := &sk[i]
		i++
		return ret
	}

	for _, item := range s.Items {
		if !item.NewKeyGen {
			continue
		}
		sk := nxt()
		if sk == nil {
			return core.TeamError("ran out of shared keys")
		}
		if sk.Gen != item.Gen {
			return core.TeamError("shared key gen mismatch")
		}
		err := sk.Role.AssertEq(item.Role.Export(), core.TeamError("shared key role mismatch"))
		if err != nil {
			return err
		}
	}
	if nxt() != nil {
		return core.TeamError("too many shared keys")
	}
	return nil
}

func (s KeySchedule) MatchSeedChain(ch []proto.SeedChainBox) error {

	var i int
	nxt := func() *proto.SeedChainBox {
		if i >= len(ch) {
			return nil
		}
		ret := &ch[i]
		i++
		return ret
	}

	for _, item := range s.Items {
		if !item.NewKeyGen || item.Gen.IsFirst() {
			continue
		}
		skb := nxt()
		if skb == nil {
			return core.TeamError("ran out of seed chain boxes")
		}
		if skb.Gen != item.Gen-1 {
			return core.TeamError("seed chain box gen mismatch")
		}
		err := skb.Role.AssertEq(item.Role.Export(), core.TeamError("seed chain box role mismatch"))
		if err != nil {
			return err
		}
	}
	if nxt() != nil {
		return core.TeamError("too many seed chain boxes")
	}
	return nil
}
