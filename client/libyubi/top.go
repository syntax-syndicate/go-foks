// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libyubi

import (
	"context"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/go-piv/piv-go/v2/piv"
)

type nextSlotter struct {
	nxt proto.YubiSlot
	nm  proto.YubiCardName
}

type Dispatch struct {
	bus  Bus
	test *nextSlotter
}

func (d *Dispatch) SenseCard() bool {
	return SenseCard()
}

func SenseCard() bool {
	_, err := piv.Cards()
	return err == nil
}

func (d *Dispatch) GetBusType() BusType {
	return d.bus.Type()
}

func (d *Dispatch) Load(
	ctx context.Context,
	i proto.YubiKeyInfo,
	r proto.Role,
	h proto.HostID,
) (
	*KeySuite,
	error,
) {
	return load(ctx, d.bus, i, r, h)
}

func (d *Dispatch) LoadPQ(
	ctx context.Context,
	i proto.YubiKeyInfoHybrid,
) (
	*KeySuitePQ,
	error,
) {
	return loadPQ(ctx, d.bus, i)
}

func (d *Dispatch) LoadHybrid(
	ctx context.Context,
	i proto.YubiKeyInfoHybrid,
	r proto.Role,
	h proto.HostID,
) (
	*KeySuiteHybrid,
	error,
) {
	return loadHybrid(ctx, d.bus, i, r, h)
}

func (d *Dispatch) GetManagementKey(
	ctx context.Context,
	i proto.YubiCardID,
) (
	*proto.YubiManagementKey,
	error,
) {
	return getManagementKey(ctx, d.bus, i)
}

// AccessPQKey uses a preexsiting ECDSA key as a PQ
// KEM seed slot. It should not be used to create new keys, since that would imply
// reusing an ECDSA key as a PQKey, which is dangerous. However, it can be used safely
// in "yubi provision", which accesses a previously-allocated PQKey.
func (d *Dispatch) AccessPQKey(
	ctx context.Context,
	i proto.YubiKeyInfo,
) (
	*KeySuitePQ,
	error,
) {
	return dangerousAccessPQKey(ctx, d.bus, i)
}

func (d *Dispatch) ListCards(
	ctx context.Context,
) ([]proto.YubiCardID, error) {
	return listCards(ctx, d.bus)
}

func (d *Dispatch) Explore(
	ctx context.Context,
	i proto.YubiCardID,
) (*proto.YubiCardInfo, error) {
	return explore(ctx, d.bus, i)
}

type GenerateKeyOpts struct {
	LockWithPIN bool
}

func (o *GenerateKeyOpts) PINPolicy() piv.PINPolicy {
	if o == nil {
		return piv.PINPolicyNever
	}
	if o.LockWithPIN {
		return piv.PINPolicyAlways
	}
	return piv.PINPolicyNever
}

func (d *Dispatch) GenerateKey(
	ctx context.Context,
	i proto.YubiCardID,
	slot proto.YubiSlot,
	r proto.Role,
	h proto.HostID,
	opts *GenerateKeyOpts,
) (
	*KeySuite,
	error,
) {
	return generateKey(ctx, d.bus, i, slot, r, h, opts)
}

func (d *Dispatch) InputPIN(
	ctx context.Context,
	id proto.YubiCardID,
	pin proto.YubiPIN,
) (
	proto.ManagementKeyState,
	error,
) {
	return inputPIN(ctx, d.bus, id, pin)
}

func (d *Dispatch) GenerateKeyPQ(
	ctx context.Context,
	i proto.YubiCardID,
	slot proto.YubiSlot,
	opts *GenerateKeyOpts,
) (
	*KeySuitePQ,
	error,
) {
	return generateKeyPQ(ctx, d.bus, i, slot, opts)
}

func (d *Dispatch) GenerateKeyHybrid(
	ctx context.Context,
	i proto.YubiCardID,
	slot proto.YubiSlot,
	kemSlot proto.YubiSlot,
	r proto.Role,
	h proto.HostID,
	opts *GenerateKeyOpts,
) (
	*KeySuiteHybrid,
	error,
) {
	return generateKeyHybrid(ctx, d.bus, i, slot, kemSlot, r, h, opts)
}

func (d *Dispatch) FindCardIDBySerial(
	ctx context.Context,
	serial proto.YubiSerial,
) (*proto.YubiCardID, error) {
	return findCardIDBySerial(ctx, d.bus, serial)
}

func (d *Dispatch) FindCardBySerial(
	ctx context.Context,
	serial proto.YubiSerial,
) (*proto.YubiCardInfo, error) {
	return findCardBySerial(ctx, d.bus, serial)
}

func (d *Dispatch) ValidatePIN(
	ctx context.Context,
	id proto.YubiCardID,
	pin proto.YubiPIN,
	doUnlock bool,
) error {
	return validatePIN(ctx, d.bus, id, pin, doUnlock)
}

func (d *Dispatch) SetPUK(
	ctx context.Context,
	id proto.YubiCardID,
	old proto.YubiPUK,
	new proto.YubiPUK,
) error {
	return setPUK(ctx, d.bus, id, old, new)
}

func (d *Dispatch) ValidatePUK(
	ctx context.Context,
	id proto.YubiCardID,
	puk proto.YubiPUK,
) error {
	return validatePUK(ctx, d.bus, id, puk)
}

func (d *Dispatch) SetPIN(
	ctx context.Context,
	id proto.YubiCardID,
	old proto.YubiPIN,
	new proto.YubiPIN,
) error {
	return setPIN(ctx, d.bus, id, old, new)
}

func (d *Dispatch) ResetPINandPUK(
	ctx context.Context,
	id proto.YubiCardID,
	mk proto.YubiManagementKey,
	pin proto.YubiPIN,
	puk proto.YubiPUK,
) error {
	return resetPINandPUK(ctx, d.bus, id, mk, pin, puk)
}

func (d *Dispatch) HasDefaultManagementKey(
	ctx context.Context,
	id proto.YubiCardID,
) (bool, error) {
	return hasDefaultManagementKey(ctx, d.bus, id)
}

func (d *Dispatch) SetOrGetManagementKey(
	ctx context.Context,
	id proto.YubiCardID,
	pin proto.YubiPIN,
) (
	*proto.YubiManagementKey,
	bool,
	error,
) {
	return setOrGetManagementKey(ctx, d.bus, id, pin)
}

func (d *Dispatch) SetManagementKey(
	ctx context.Context,
	id proto.YubiCardID,
	old *proto.YubiManagementKey,
	new proto.YubiManagementKey,
) error {
	return setManagementKey(ctx, d.bus, id, old, new)
}

func (d *Dispatch) ClearSecrets() {
	d.bus.ClearSecrets()
}

var realDispatch *Dispatch

type MockYubiSeed []byte

// There is only one real bus, so return the same one every time.
// Important for keeping track of allocated test slots.
func allocRealYubi() (*Dispatch, error) {
	if realDispatch == nil {
		realDispatch = &Dispatch{bus: NewRealBus()}
	}
	return realDispatch, nil
}

func AllocDispatch(
	seed MockYubiSeed,
) (
	*Dispatch,
	error,
) {
	if len(seed) == 0 {
		return allocRealYubi()
	}
	bus, err := NewMockBusWithSeed(seed, 2)
	if err != nil {
		return nil, err
	}
	return &Dispatch{bus: bus}, nil
}

func NewMockYubiSeed() (MockYubiSeed, error) {
	return mockRandomBusSeed()
}

func (s MockYubiSeed) String() string {
	return core.B62Encode(s)
}

func IsDefaultPIN(pin proto.YubiPIN) bool {
	return pin.String() == piv.DefaultPIN
}
