// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import "sync"

type AppID string

type App interface {
	Cleanup(m MetaContext) error
	ID() AppID
}

type Apps struct {
	sync.Mutex
	apps map[AppID]App
}

func (a *Apps) Get(id AppID, u *UserContext, f func(u *UserContext) App) App {
	a.Lock()
	defer a.Unlock()
	if a.apps == nil {
		a.apps = make(map[AppID]App)
	}
	app := a.apps[id]
	if app == nil {
		app = f(u)
		a.apps[id] = app
	}
	return app
}

func GetApp[
	T any,
	PT interface {
		*T
		App
	},
](u *UserContext, id AppID, f func(u *UserContext) PT) PT {
	apps := u.Apps()
	tmp := apps.Get(id, u, func(u *UserContext) App {
		ret := f(u)
		return ret
	})
	ret, ok := tmp.(PT)
	if !ok {
		return nil
	}
	return ret
}

func NewApps() *Apps {
	return &Apps{}
}
