// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"slices"

	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

type StaleKeys struct {
	Keys map[core.RoleKey]bool
}

func NewStaleKeys() *StaleKeys {
	return &StaleKeys{
		Keys: make(map[core.RoleKey]bool),
	}
}

func (s *StaleKeys) Export() []proto.Role {
	var ret []proto.Role
	tmp := make([]core.RoleKey, 0, len(s.Keys))
	for k := range s.Keys {
		tmp = append(tmp, k)
	}
	slices.SortFunc(tmp, func(a, b core.RoleKey) int {
		return a.Cmp(b)
	})
	for _, k := range tmp {
		ret = append(ret, k.Export())
	}
	return ret
}

func (s *StaleKeys) Refresh(r core.RoleKey) {
	delete(s.Keys, r)
}

func (s *StaleKeys) MarkStale(r core.RoleKey) {
	s.Keys[r] = true
}

func (s *StaleKeys) Import(roles []proto.Role) error {
	for _, r := range roles {
		rk, err := core.ImportRole(r)
		if err != nil {
			return err
		}
		s.Keys[*rk] = true
	}
	return nil
}

func (s *StaleKeys) IsEmpty() bool {
	return len(s.Keys) == 0
}
