// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libyubi

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"strings"
	"sync"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/go-piv/piv-go/v2/piv"
)

func NewRealBus() *RealBus {
	return &RealBus{
		BusBase: newBusBase(),
	}
}

type RealBus struct {
	*BusBase
}

type RealCard struct {
	sync.Mutex
	card   *piv.YubiKey
	serial *proto.YubiSerial
	closed bool
}

var _ Bus = (*RealBus)(nil)
var _ Card = (*RealCard)(nil)

func (h *Handle) Bus() Bus {
	return h.bus
}

func (r *RealBus) Type() BusType {
	return BusTypeReal
}

func (r *RealBus) openCard(ctx context.Context, nm proto.YubiCardName) (Card, error) {
	c, err := piv.Open(string(nm))
	if err != nil {
		return nil, err
	}
	return &RealCard{card: c}, nil
}

func (r *RealBus) Handle(ctx context.Context, nm proto.YubiCardName) (*Handle, error) {
	return r.BusBase.handle(ctx, r, nm)
}

func (c *RealCard) Serial() (proto.YubiSerial, error) {
	c.Lock()
	defer c.Unlock()

	if c.serial != nil {
		return *c.serial, nil
	}

	tmp, err := c.card.Serial()
	if err != nil {
		return 0, err
	}
	ptmp := proto.YubiSerial(tmp)
	c.serial = &ptmp
	return ptmp, nil
}

func (c *RealCard) PrivateKey(slot piv.Slot, pk crypto.PublicKey, auth piv.KeyAuth) (crypto.PrivateKey, error) {
	c.Lock()
	defer c.Unlock()
	priv, err := c.card.PrivateKey(slot, pk, auth)
	if err != nil {
		if strings.Contains(err.Error(), "error 6982") {
			return nil, core.YubiPINRequredError{}
		}
	}
	if err != nil {
		return nil, err
	}
	return priv, nil
}

func (c *RealCard) Attest(slot piv.Slot) (*x509.Certificate, error) {
	c.Lock()
	defer c.Unlock()
	return c.card.Attest(slot)
}

func (c *RealCard) Close() error {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.card.Close()
}

func (c *RealCard) GenerateKey(mgmtKey []byte, slot piv.Slot, key piv.Key) (crypto.PublicKey, error) {
	c.Lock()
	defer c.Unlock()
	return c.card.GenerateKey(mgmtKey, slot, key)
}

// Abstract this out since it's different for a mock yubikey that's running
// raw crypto.ECDSA. Note that we don't access piv.YubiKey here.
func (c *RealCard) SharedKey(priv crypto.PrivateKey, receiver *ecdsa.PublicKey) ([]byte, error) {
	xPrivEcdsa, ok := priv.(*piv.ECDSAPrivateKey)
	if !ok {
		return nil, core.YubiError("cannot access private ECDSA function")
	}
	shared, err := xPrivEcdsa.SharedKey(receiver)
	if err != nil {
		return nil, fixErr(err)
	}
	return shared, nil
}

func (b *RealBus) Cards(ctx context.Context, filter bool) ([]proto.YubiCardName, error) {
	raw, err := piv.Cards()
	if err != nil {
		return nil, core.YubiBusError{Err: err}
	}
	ret := make([]proto.YubiCardName, len(raw))
	for i, r := range raw {
		if !filter || strings.Contains(strings.ToLower(r), "yubikey") {
			ret[i] = proto.YubiCardName(r)
		}
	}
	return ret, nil
}

func (b *RealBus) Slot(s proto.YubiSlot) (piv.Slot, error) {
	ret, ok := piv.RetiredKeyManagementSlot(uint32(s))
	if !ok {
		return ret, core.YubiError("invalid slot")
	}
	return ret, nil
}

func fillDefaultPIN(pin proto.YubiPIN) proto.YubiPIN {
	if pin.IsZero() {
		return proto.YubiPIN(piv.DefaultPIN)
	}
	return pin
}

func fillDefaultPINp(pinp *proto.YubiPIN) proto.YubiPIN {
	var pin proto.YubiPIN
	if pinp != nil {
		pin = *pinp
	}
	return fillDefaultPIN(pin)
}

func fillDefaultPUKp(p *proto.YubiPUK) proto.YubiPUK {
	var puk proto.YubiPUK
	if p != nil {
		puk = *p
	}
	return fillDefaultPUK(puk)
}

func fillDefaultPUK(puk proto.YubiPUK) proto.YubiPUK {
	if puk.IsZero() {
		return proto.YubiPUK(piv.DefaultPUK)
	}
	return puk
}

func fixErr(err error) error {
	if err == nil {
		return nil
	} else if ae, ok := err.(piv.AuthErr); ok {
		return core.YubiAuthError{Retries: ae.Retries}
	} else if strings.Contains(err.Error(), "error 6982") {
		return core.YubiPINRequredError{}
	}
	return err
}

func (c *RealCard) ValidatePIN(pin proto.YubiPIN) error {
	c.Lock()
	defer c.Unlock()

	err := c.card.VerifyPIN(fillDefaultPIN(pin).String())
	return fixErr(err)
}

func (c *RealCard) SetPIN(old, new proto.YubiPIN) error {
	c.Lock()
	defer c.Unlock()
	err := c.card.SetPIN(fillDefaultPIN(old).String(), new.String())
	return fixErr(err)
}

func (c *RealCard) SetPUK(old, new proto.YubiPUK) error {
	c.Lock()
	defer c.Unlock()
	err := c.card.SetPUK(fillDefaultPUK(old).String(), new.String())
	return fixErr(err)
}

func (c *RealCard) ValidatePUK(puk proto.YubiPUK) error {
	c.Lock()
	defer c.Unlock()
	puk = fillDefaultPUK(puk)
	err := c.card.SetPUK(puk.String(), puk.String())
	return fixErr(err)
}

func (c *RealCard) HasDefaultManagementKey() (bool, error) {
	c.Lock()
	defer c.Unlock()
	return c.hasDefaultManagmentKeyLocked()
}

func (c *RealCard) hasDefaultManagmentKeyLocked() (bool, error) {
	defKey := piv.DefaultManagementKey
	defMd := piv.Metadata{
		ManagementKey: &defKey,
	}
	err := c.card.SetMetadata(defKey, &defMd)
	if err == nil {
		return true, nil
	}

	// Very unfortunate we can't get better errors, but alas.
	if strings.Contains(err.Error(), "challenge failed") ||
		strings.Contains(err.Error(), "authentication failed") ||
		(strings.Contains(err.Error(), "auth challenge") &&
			strings.Contains(err.Error(), "error 6982")) {
		return false, nil
	}

	return false, err
}

func isDefaultManagementKey(k *proto.YubiManagementKey) bool {
	if k == nil {
		return false
	}
	defKey := piv.DefaultManagementKey
	return bytes.Equal(k[:], defKey[:])
}

func (c *RealCard) GetManagementKey(pin proto.YubiPIN) (*proto.YubiManagementKey, error) {
	c.Lock()
	defer c.Unlock()
	return c.getManagementKeyLocked(pin)
}

func (c *RealCard) getManagementKeyLocked(pin proto.YubiPIN) (*proto.YubiManagementKey, error) {
	hasDef, err := c.hasDefaultManagmentKeyLocked()
	if err != nil {
		return nil, err
	}
	if hasDef {
		return nil, core.YubiDefaultManagementKeyError{}
	}
	md, err := c.card.Metadata(pin.String())
	if err != nil {
		return nil, err
	}

	if md.ManagementKey == nil {
		return nil, core.KeyNotFoundError{Which: "management key"}
	}
	var ret proto.YubiManagementKey
	copy(ret[:], (*md.ManagementKey)[:])
	return &ret, nil
}

func (c *RealCard) SetManagementKey(
	oldp *proto.YubiManagementKey,
	new proto.YubiManagementKey,
) error {
	c.Lock()
	defer c.Unlock()
	return c.setManagementKeyLocked(oldp, new)
}

func (c *RealCard) setManagementKeyLocked(
	oldp *proto.YubiManagementKey,
	new proto.YubiManagementKey,
) error {

	var old []byte
	if oldp != nil {
		old = oldp.Bytes()
	} else {
		old = piv.DefaultManagementKey
	}

	tmp := new.Bytes()

	err := c.card.SetManagementKey(old, tmp)
	if err != nil {
		return err
	}

	err = c.card.SetMetadata(tmp, &piv.Metadata{
		ManagementKey: &tmp,
	})
	if err != nil {
		return err
	}
	return nil
}

type internalCarder interface {
	setManagementKeyLocked(oldp *proto.YubiManagementKey, new proto.YubiManagementKey) error
	getManagementKeyLocked(pin proto.YubiPIN) (*proto.YubiManagementKey, error)
}

func (c *RealCard) SetOrGetManagementKey(
	pin proto.YubiPIN,
) (*proto.YubiManagementKey, bool, error) {
	c.Lock()
	defer c.Unlock()
	return setOrGetManagementKeyLocked(c, pin)
}

func setOrGetManagementKeyLocked(
	c internalCarder, pin proto.YubiPIN,
) (*proto.YubiManagementKey, bool, error) {

	curr, err := c.getManagementKeyLocked(pin)
	isdef := errors.Is(err, core.YubiDefaultManagementKeyError{})
	if isdef {
		err = nil
	}
	if err != nil {
		return nil, false, err
	}
	if !isdef {
		return curr, false, nil
	}

	var new proto.YubiManagementKey
	err = core.RandomFill(new[:])
	if err != nil {
		return nil, false, err
	}
	err = c.setManagementKeyLocked(nil, new)
	if err != nil {
		return nil, false, err
	}
	return &new, true, nil
}

var _ Card = (*RealCard)(nil)
