// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cks

import (
	"sync"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type EncKey struct {
	proto.CKSEncKey
}

func (k EncKey) SecretBoxKey() *proto.SecretBoxKey {
	return (*proto.SecretBoxKey)(&k.CKSEncKey)
}

func NewEncKey() (*EncKey, error) {
	var key proto.CKSEncKey
	err := core.RandomFill(key[:])
	if err != nil {
		return nil, err
	}
	return &EncKey{CKSEncKey: key}, nil
}

// Package CKS = "Crypto Key Store"
//
// Concerned with how to store crypto keys in the Database, and sometimes on the filesystem,
// optimizing for simplicity of management, and also so that we can keep some very sensitive
// keys offline.

// KeyID gives the 16-byte key ID of a key, so we know which key to decrypt with in the
// case that keys got rotated.
func (k EncKey) ID() (*proto.CKSKeyID, error) {
	var hsh proto.StdHash
	err := core.PrefixedHashInto(&k.CKSEncKey, hsh[:])
	if err != nil {
		return nil, err
	}
	id16, err := proto.ID16Type_CKSKey.MakeID16(hsh[:16])
	if err != nil {
		return nil, err
	}
	ret, err := id16.ToCKSKeyID()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (k *Keyring) Seal(
	payload core.CryptoPayloader,
) (
	*proto.CKSBox,
	error,
) {
	k.RLock()
	defer k.RUnlock()
	return k.curr.Seal(payload)
}

func (k EncKey) Seal(
	payload core.CryptoPayloader,
) (
	*proto.CKSBox,
	error,
) {
	box, err := core.SealIntoSecretBox(payload, k.SecretBoxKey())
	if err != nil {
		return nil, err
	}
	keyid, err := k.ID()
	if err != nil {
		return nil, err
	}
	ret := proto.CKSBox{
		Key: *keyid,
		Box: *box,
	}
	return &ret, err
}

type Keyring struct {
	sync.RWMutex
	keys map[proto.CKSKeyID]*EncKey
	curr *EncKey
}

func NewKeyring() *Keyring {
	return &Keyring{
		keys: make(map[proto.CKSKeyID]*EncKey),
	}
}

func (k *Keyring) AddCurr(c *EncKey) error {
	k.Lock()
	defer k.Unlock()
	err := k.add(*c)
	if err != nil {
		return err
	}
	k.curr = c
	return nil
}

func (k *Keyring) Add(key EncKey) error {
	k.Lock()
	defer k.Unlock()
	return k.add(key)
}

func (k *Keyring) add(key EncKey) error {
	id, err := key.ID()
	if err != nil {
		return err
	}
	k.keys[*id] = &key
	return nil
}

func (k *Keyring) AddAll(keys []EncKey) error {
	k.Lock()
	defer k.Unlock()
	var lst *EncKey
	for _, key := range keys {
		err := k.add(key)
		if err != nil {
			return err
		}
		lst = &key
	}
	if lst != nil {
		k.curr = lst
	}
	return nil
}

func (k *Keyring) Open(
	obj core.CryptoPayloader,
	box *proto.CKSBox,
) error {
	k.RLock()
	key, ok := k.keys[box.Key]
	k.RUnlock()
	if !ok {
		return core.KeyNotFoundError{Which: "cks"}
	}
	return core.OpenSecretBoxInto(obj, box.Box, key.SecretBoxKey())
}
