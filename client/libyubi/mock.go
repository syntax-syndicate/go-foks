// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libyubi

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/go-piv/piv-go/v2/piv"
)

type ECDSAKeypair struct {
	priv *ecdsa.PrivateKey
	pub  *ecdsa.PublicKey
	pp   piv.PINPolicy
}

func NewECDSAKeyPair(pp piv.PINPolicy) (*ECDSAKeypair, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	pub := priv.Public()
	ecpub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("bad pubkey")
	}
	return &ECDSAKeypair{
		priv: priv,
		pub:  ecpub,
		pp:   pp,
	}, nil
}

type retryLimits struct {
	pin int
	puk int
}

type MockCard struct {
	sync.Mutex
	populateSlots []int
	serial        proto.YubiSerial
	slots         map[int](*ECDSAKeypair)
	closed        bool
	pos           int
	pin           *proto.YubiPIN
	puk           *proto.YubiPUK
	mgmt          *proto.YubiManagementKey
	nBadPINs      int
	nBadPUKs      int
	pinVerified   bool
	retryLimits   *retryLimits
	hasPEMK       bool // has pin-encrypted management key
}

func (c *MockCard) ClearPIN() {
	c.Lock()
	defer c.Unlock()
	c.pinVerified = false
	c.hasPEMK = false
}

func (c *MockCard) getRetryLimitsLocked() *retryLimits {
	if c.retryLimits == nil {
		c.retryLimits = &retryLimits{
			pin: 3,
			puk: 10,
		}
	}
	return c.retryLimits
}

type MockBus struct {
	*BusBase
	cardOrder []string
	cards     map[string](*MockCard)
}

func mockRandomBusSeed() ([]byte, error) {
	var seed [32]byte
	err := core.RandomFill(seed[:])
	if err != nil {
		return nil, err
	}
	return seed[:], nil
}

func NewMockBus() (*MockBus, error) {
	seed, err := mockRandomBusSeed()
	if err != nil {
		return nil, err
	}
	return NewMockBusWithSeed(seed, 2)
}

func genSerial(seed []byte, pos byte) (proto.YubiSerial, error) {
	hm := hmac.New(sha512.New512_256, seed)
	hm.Write([]byte{pos})
	buf := hm.Sum(nil)
	var ret uint64
	err := binary.Read(bytes.NewReader(buf), binary.BigEndian, &ret)
	if err != nil {
		return 0, err
	}
	return proto.YubiSerial(ret), nil
}

func NewMockBusWithSeed(seed MockYubiSeed, cardCount int) (*MockBus, error) {
	ret := &MockBus{
		BusBase:   newBusBase(),
		cards:     make(map[string](*MockCard)),
		cardOrder: make([]string, cardCount),
	}
	if cardCount > 0xff {
		return nil, core.InternalError("too many cards")
	}
	takenSlots := []int{0x82, 0x83, 0x85, 0x86, 0x88, 0x90}
	for i := 0; i < cardCount; i++ {
		serial, err := genSerial(seed, byte(i))
		if err != nil {
			return nil, err
		}
		card, err := newMockYubiKey(takenSlots, serial, i)
		if err != nil {
			return nil, err
		}
		nm := card.Name()
		ret.cards[nm] = card
		ret.cardOrder[i] = nm
	}
	return ret, nil
}

// There are two reasons why we can't deterministically generate yubikey cards.
// First, the Go library really doesn't want you to genreate P256 ECDSA keys deterministically.
// There are special knobs in the library to prevent a determinstic rand.Reader stream, etc.
// The deeper issue is that we want to be able to generate a key at slot 0x82 in one agent
// "process" and access it from another. So we have to keep track of these bits, whether
// or not a slot has been allocated. This is a more fundamental problem. The ugly truth
// is that we do need global state. But if we hide it behind a 64-byte seed, we won't
// have conflicts across tests in practice.
var cardTableMu sync.Mutex
var cardTable map[proto.YubiSerial](*MockCard)

func (m *MockCard) makeKey(slot int, pp piv.PINPolicy) error {
	key, err := NewECDSAKeyPair(pp)
	if err != nil {
		return err
	}
	m.slots[slot] = key
	return nil
}

func newMockYubiKey(slots []int, serial proto.YubiSerial, pos int) (*MockCard, error) {

	cardTableMu.Lock()
	defer cardTableMu.Unlock()
	if cardTable == nil {
		cardTable = make(map[proto.YubiSerial](*MockCard))
	}
	ret := cardTable[serial]
	if ret != nil {
		return ret, nil
	}

	ret = &MockCard{
		populateSlots: slots,
		pos:           pos,
		serial:        serial,
		slots:         make(map[int](*ECDSAKeypair)),
	}
	for _, slot := range slots {
		err := ret.makeKey(slot, piv.PINPolicyNever)
		if err != nil {
			return nil, err
		}
	}
	cardTable[serial] = ret
	return ret, nil
}

func (k *MockCard) Name() string {
	return fmt.Sprintf("Mock YubiKey %d (0x%x)", k.pos, k.serial)
}

func (b *MockBus) Type() BusType {
	return BusTypeMock
}

func (m *MockCard) init() error {
	if m.closed {
		return core.YubiError("mock yubi: used after close")
	}
	return nil
}

func (m *MockCard) Serial() (proto.YubiSerial, error) {
	m.Lock()
	defer m.Unlock()

	return (m.serial & 0xffffffff), nil
}

func (m *MockCard) PrivateKey(slot piv.Slot, pub crypto.PublicKey, auth piv.KeyAuth) (crypto.PrivateKey, error) {
	m.Lock()
	defer m.Unlock()

	err := m.init()
	if err != nil {
		return nil, err
	}
	key := m.slots[int(slot.Key)]
	if key == nil {
		return nil, piv.ErrNotFound
	}
	var needPinCheck bool
	switch key.pp {
	case piv.PINPolicyAlways:
		needPinCheck = true
	case piv.PINPolicyOnce:
		needPinCheck = !m.pinVerified
	}

	if needPinCheck {
		if auth.PIN == "" {
			return nil, core.YubiPINRequredError{}
		}
		err = m.validatePINLocked(proto.YubiPIN(auth.PIN))
		if err != nil {
			return nil, err
		}
	}

	return key.priv, nil
}

func (m *MockCard) Attest(slot piv.Slot) (*x509.Certificate, error) {
	m.Lock()
	defer m.Unlock()

	err := m.init()
	if err != nil {
		return nil, err
	}
	key := m.slots[int(slot.Key)]
	if key == nil {
		return nil, piv.ErrNotFound
	}
	return &x509.Certificate{PublicKey: key.pub}, nil
}

func (m *MockCard) Close() error {
	m.Lock()
	defer m.Unlock()
	m.closed = true
	return nil
}

func (m *MockCard) GenerateKey(mks []byte, slot piv.Slot, desc piv.Key) (crypto.PublicKey, error) {
	m.Lock()
	defer m.Unlock()

	err := m.init()
	if err != nil {
		return nil, err
	}

	mk, err := ImportManagementKey(mks)
	if err != nil {
		return nil, err
	}

	err = m.validateManagementKey(mk)
	if err != nil {
		return nil, err
	}

	ikey := int(slot.Key)
	if m.slots[ikey] != nil {
		return nil, errors.New("key already exists")
	}
	err = m.makeKey(ikey, desc.PINPolicy)
	if err != nil {
		return nil, err
	}
	return m.slots[ikey].pub, nil
}

func (m *MockCard) SharedKey(priv crypto.PrivateKey, pub *ecdsa.PublicKey) ([]byte, error) {
	m.Lock()
	defer m.Unlock()

	privEcdsa, ok := priv.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("bad privkey")
	}
	x, err := privEcdsa.ECDH()
	if err != nil {
		return nil, err
	}
	y, err := pub.ECDH()
	if err != nil {
		return nil, err
	}
	shared, err := x.ECDH(y)
	if err != nil {
		return nil, err
	}
	return shared, nil
}

func (m *MockBus) init() error {
	return nil
}

func (m *MockBus) openCard(ctx context.Context, name proto.YubiCardName) (Card, error) {
	m.Lock()
	defer m.Unlock()

	err := m.init()
	if err != nil {
		return nil, err
	}
	card, ok := m.cards[string(name)]
	if !ok {
		return nil, errors.New("no such card")
	}
	card.closed = false
	return card, nil
}

func (m *MockBus) Slot(s proto.YubiSlot) (piv.Slot, error) {
	m.Lock()
	defer m.Unlock()

	err := m.init()
	if err != nil {
		return piv.Slot{}, err
	}
	return piv.Slot{Key: uint32(s)}, nil
}

func (m *MockBus) Cards(ctx context.Context, filter bool) ([]proto.YubiCardName, error) {
	m.Lock()
	defer m.Unlock()

	err := m.init()
	if err != nil {
		return nil, err
	}
	ret := make([]proto.YubiCardName, len(m.cardOrder))
	for i, name := range m.cardOrder {
		ret[i] = proto.YubiCardName(name)
	}
	return ret, nil
}

func (m *MockBus) Handle(ctx context.Context, nm proto.YubiCardName) (*Handle, error) {
	return m.BusBase.handle(ctx, m, nm)
}

func (c *MockCard) validatePINLocked(p proto.YubiPIN) error {
	existing := fillDefaultPINp(c.pin)
	p = fillDefaultPIN(p)

	retries := c.getRetryLimitsLocked()

	triesLeft := func() int {
		maxRetries := retries.pin
		return maxRetries - c.nBadPINs
	}
	if triesLeft() <= 0 {
		return core.YubiAuthError{Retries: triesLeft()}
	}
	if !existing.Eq(p) {
		c.nBadPINs++
		return core.YubiAuthError{Retries: triesLeft()}
	}
	c.nBadPINs = 0
	c.pinVerified = true
	return nil
}

func (c *MockCard) ValidatePIN(p proto.YubiPIN) error {
	c.Lock()
	defer c.Unlock()
	return c.validatePINLocked(p)
}

func (c *MockCard) SetPIN(old, new proto.YubiPIN) error {
	c.Lock()
	defer c.Unlock()
	err := c.validatePINLocked(old)
	if err != nil {
		return err
	}
	c.pin = &new
	c.pinVerified = false
	return nil
}

func (c *MockCard) SetPUK(old, new proto.YubiPUK) error {
	c.Lock()
	defer c.Unlock()
	err := c.validatePUKLocked(old)
	if err != nil {
		return err
	}
	c.puk = &new
	c.pinVerified = false
	return nil
}

func (c *MockCard) validatePUKLocked(puk proto.YubiPUK) error {
	existing := fillDefaultPUKp(c.puk)
	puk = fillDefaultPUK(puk)

	retries := c.getRetryLimitsLocked()

	triesLeft := func() int {
		maxRetries := retries.puk
		return maxRetries - c.nBadPUKs
	}

	if triesLeft() <= 0 {
		return core.YubiAuthError{Retries: triesLeft()}
	}
	if !existing.Eq(puk) {
		c.nBadPUKs++
		return core.YubiAuthError{Retries: triesLeft()}
	}
	c.nBadPUKs = 0
	return nil
}

func (c *MockCard) ValidatePUK(puk proto.YubiPUK) error {
	c.Lock()
	defer c.Unlock()
	return c.validatePUKLocked(puk)
}

func fillDefaultManagementKey(mk proto.YubiManagementKey) proto.YubiManagementKey {
	if mk.IsZero() {
		return proto.YubiManagementKey(piv.DefaultManagementKey)
	}
	return mk
}

func fillDefaultManagementKeyP(mk *proto.YubiManagementKey) proto.YubiManagementKey {
	if mk == nil {
		var tmp proto.YubiManagementKey
		mk = &tmp
	}
	return fillDefaultManagementKey(*mk)
}

func ImportManagementKey(b []byte) (*proto.YubiManagementKey, error) {
	var ret proto.YubiManagementKey
	if len(b) == 0 {
		return nil, nil
	}
	if len(b) != len(ret) {
		return nil, core.BadArgsError("management key must be 24 bytes")
	}
	copy(ret[:], b)
	return &ret, nil
}

func (c *MockCard) validateManagementKey(k *proto.YubiManagementKey) error {
	a := fillDefaultManagementKeyP(c.mgmt)
	b := fillDefaultManagementKeyP(k)

	if !a.Eq(b) {
		return core.YubiAuthError{Retries: 0}
	}
	return nil
}

func (c *MockCard) SetManagementKey(old *proto.YubiManagementKey, key proto.YubiManagementKey) error {
	c.Lock()
	defer c.Unlock()

	return c.setManagementKeyLocked(old, key)
}

func (c *MockCard) setManagementKeyLocked(old *proto.YubiManagementKey, key proto.YubiManagementKey) error {
	err := c.validateManagementKey(old)
	if err != nil {
		return err
	}
	c.mgmt = &key
	c.pinVerified = false

	// look at real.go -- in realCard.setManagementKey, we're both setting the key
	// and updating the metadata to write the management key back to the card.
	// so this notion of PEMK is related only to whether or not we know the PIN
	// need a different mock interface to model some other application aside from FOKS setting
	// the management key
	c.hasPEMK = (c.pin != nil)
	return nil
}

func (c *MockCard) GetManagementKey(pin proto.YubiPIN) (*proto.YubiManagementKey, error) {
	c.Lock()
	defer c.Unlock()
	return c.getManagementKeyLocked(pin)
}

func (c *MockCard) getManagementKeyLocked(pin proto.YubiPIN) (*proto.YubiManagementKey, error) {
	if c.mgmt == nil || isDefaultManagementKey(c.mgmt) {
		return nil, core.YubiDefaultManagementKeyError{}
	}
	err := c.validatePINLocked(pin)
	if err != nil {
		return nil, err
	}
	if !c.hasPEMK {
		return nil, core.KeyNotFoundError{Which: "management key"}
	}
	return c.mgmt, nil
}

func (c *MockCard) SetOrGetManagementKey(pin proto.YubiPIN) (*proto.YubiManagementKey, bool, error) {
	c.Lock()
	defer c.Unlock()
	return setOrGetManagementKeyLocked(c, pin)
}

func (c *MockCard) HasDefaultManagementKey() (bool, error) {
	c.Lock()
	defer c.Unlock()
	if c.mgmt == nil {
		return true, nil
	}
	if isDefaultManagementKey(c.mgmt) {
		return true, nil
	}
	return false, nil
}

func (c *MockCard) SetRetries(
	mk proto.YubiManagementKey,
	pin int,
	puk int,
) error {
	c.Lock()
	defer c.Unlock()
	err := c.validateManagementKey(&mk)
	if err != nil {
		return err
	}
	def := proto.YubiPIN(piv.DefaultPIN)
	defPuk := proto.YubiPUK(piv.DefaultPUK)
	c.pin = &def
	c.puk = &defPuk
	c.nBadPINs = 0
	c.nBadPUKs = 0
	c.retryLimits = &retryLimits{
		pin: pin,
		puk: puk,
	}
	c.pinVerified = false
	c.hasPEMK = true
	return nil
}

var _ Bus = (*MockBus)(nil)
var _ Card = (*MockCard)(nil)
