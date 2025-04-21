// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import "sync"

type LocktabEntry[K comparable] struct {
	sync.Mutex
	p    *Locktab[K]
	name K

	// protected with p.Mutex !!
	c int
}

func (e *LocktabEntry[K]) inc() {
	e.c++
}

func (e *LocktabEntry[K]) dec() {
	e.c--
}

type Locktab[K comparable] struct {
	sync.Mutex
	m map[K]*LocktabEntry[K]
}

func (l *Locktab[K]) Acquire(k K) *LocktabEntry[K] {
	l.Lock()
	if l.m == nil {
		l.m = make(map[K]*LocktabEntry[K])
	}
	var ret *LocktabEntry[K]
	if ret = l.m[k]; ret == nil {
		ret = &LocktabEntry[K]{
			p:    l,
			name: k,
		}
		l.m[k] = ret
	}
	ret.inc()
	l.Unlock()
	ret.Lock()
	return ret
}

func (e *LocktabEntry[K]) Release() {
	parent := e.p
	e.Unlock()

	parent.Lock()
	e.dec()
	if e.c == 0 {
		delete(parent.m, e.name)
	}
	parent.Unlock()
}
