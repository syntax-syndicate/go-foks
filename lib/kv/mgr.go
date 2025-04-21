// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package kv

import (
	"sync"

	proto "github.com/foks-proj/go-foks/proto/lib"
)

type DirKeyCache struct {
	sync.Mutex
	d map[proto.DirID]*DirKeys
}

type Manager struct {
	sync.Mutex
	DirKeys *DirKeyCache
}

func NewDirKeyCache() *DirKeyCache {
	return &DirKeyCache{
		d: make(map[proto.DirID]*DirKeys),
	}
}

func (d *DirKeyCache) Put(id proto.DirID, k *DirKeys) {
	d.Lock()
	defer d.Unlock()
	d.d[id] = k
}

func (d *DirKeyCache) Get(id proto.DirID) *DirKeys {
	d.Lock()
	defer d.Unlock()
	return d.d[id]
}
