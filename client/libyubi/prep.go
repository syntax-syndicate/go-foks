// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libyubi

import (
	"context"
	"fmt"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type Prepper struct {
	// input
	Host        proto.HostID
	Role        proto.Role
	Serial      proto.YubiSerial
	Slot        proto.YubiSlot
	PQSlot      proto.YubiSlot
	Disp        *Dispatch
	Pin         proto.YubiPIN
	LockWithPIN bool

	// state
	card    *proto.YubiCardInfo
	mks     proto.ManagementKeyState
	cKey    *KeySuite
	pqKey   *KeySuitePQ
	hKey    *KeySuiteHybrid
	empties map[proto.YubiSlot]bool
	keys    map[proto.YubiSlot]*proto.YubiSlotAndKeyID
}

func (y *Prepper) generateKeyOpts() *GenerateKeyOpts {
	return &GenerateKeyOpts{
		LockWithPIN: y.LockWithPIN,
	}
}

func (y *Prepper) loadCard(ctx context.Context) error {
	card, err := y.Disp.FindCardBySerial(ctx, y.Serial)
	if err != nil {
		return err
	}
	empties := make(map[proto.YubiSlot]bool)
	for _, slot := range card.EmptySlots {
		empties[slot] = true
	}
	keys := make(map[proto.YubiSlot]*proto.YubiSlotAndKeyID)
	for _, key := range card.Keys {
		key := key
		keys[key.Slot] = &key
	}
	y.empties = empties
	y.keys = keys
	y.card = card
	return nil
}

func (y *Prepper) prepPQKey(ctx context.Context) error {
	var ret *KeySuitePQ
	var err error

	if y.empties[y.PQSlot] {
		ret, err = y.Disp.GenerateKeyPQ(ctx, y.card.Id, y.PQSlot,
			y.generateKeyOpts(),
		)
	} else if key, ok := y.keys[y.PQSlot]; ok {
		var pqid *proto.YubiPQKeyID
		pqid, err = core.YubiIDtoYubiPQKeyID(key.Id)
		if err != nil {
			return err
		}
		yid := proto.YubiKeyInfoHybrid{
			Card: y.card.Id,
			PqKey: proto.YubiSlotAndPQKeyID{
				Slot: y.PQSlot,
				Id:   *pqid,
			},
		}
		ret, err = y.Disp.LoadPQ(ctx, yid)
	} else {
		err = core.YubiError(fmt.Sprintf("PQ slot %d not found", y.PQSlot))
	}
	if err != nil {
		return err
	}
	y.pqKey = ret
	return nil
}

func (y *Prepper) prepClassicalKey(ctx context.Context) error {

	var ret *KeySuite
	var err error

	if y.empties[y.Slot] {
		ret, err = y.Disp.GenerateKey(ctx, y.card.Id, y.Slot, y.Role, y.Host,
			y.generateKeyOpts(),
		)
	} else if key, ok := y.keys[y.Slot]; ok {
		yid := proto.YubiKeyInfo{
			Card: y.card.Id,
			Key:  *key,
		}
		ret, err = y.Disp.Load(ctx, yid, y.Role, y.Host)
	} else {
		err = core.YubiError(fmt.Sprintf("slot %d not found", y.Slot))
	}
	if err != nil {
		return err
	}
	y.cKey = ret
	return nil
}

func (y *Prepper) fuse() error {
	y.hKey = y.cKey.Fuse(y.pqKey)
	return nil
}

func (y *Prepper) checkArgs() error {
	if y.Slot == y.PQSlot {
		return core.YubiError("cannot use same slot for primary and PQ keys")
	}
	return nil
}

func (y *Prepper) checkPIN(ctx context.Context) error {
	if y.Pin.IsZero() {
		return nil
	}
	mks, err := y.Disp.InputPIN(ctx, y.card.Id, y.Pin)
	if err != nil {
		return err
	}
	y.mks = mks
	return nil
}

func (y *Prepper) run(ctx context.Context) error {
	var err error

	if err = y.checkArgs(); err != nil {
		return err
	}
	if err = y.loadCard(ctx); err != nil {
		return err
	}
	if err = y.checkPIN(ctx); err != nil {
		return err
	}
	if err = y.prepClassicalKey(ctx); err != nil {
		return err
	}
	if err = y.prepPQKey(ctx); err != nil {
		return err
	}
	if err = y.fuse(); err != nil {
		return err
	}
	return nil
}

func (y *Prepper) Run(ctx context.Context) (*KeySuiteHybrid, error) {
	err := y.run(ctx)
	if err != nil {
		return nil, err
	}
	return y.hKey, nil
}
