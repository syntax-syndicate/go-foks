// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import "time"

type NagState struct {
	Shown      time.Time
	Refreshed  time.Time
	NumDevices uint64
	Cleared    bool
}

func (u *UserContext) NagState() NagState {
	u.Lock()
	defer u.Unlock()
	return u.nagState
}

func (u *UserContext) SetNagState(n NagState) {
	u.Lock()
	defer u.Unlock()
	u.nagState = n
}

func (u *UserContext) ShowNag(m MetaContext) {
	u.Lock()
	defer u.Unlock()
	u.nagState.Shown = m.G().Now()
}
