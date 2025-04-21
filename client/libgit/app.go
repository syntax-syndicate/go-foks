// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libgit

import (
	"sync"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/client/libkv"
	"github.com/foks-proj/go-foks/lib/core"
)

// A git remote,
//
//	like foks://ne43.pub/bob/pickles
//	 or  foks://zodod.com/t:vassar_alumni/photos.git
type RemoteStr string

var AppID = libclient.AppID("git")

type App struct {
	sync.Mutex
	parent *libclient.UserContext
	kvApp  *libkv.App
}

func (a *App) Cleanup(m libclient.MetaContext) error { return nil }
func (a *App) ID() libclient.AppID                   { return AppID }

func NewApp(u *libclient.UserContext) *App {
	return &App{
		parent: u,
	}
}

func GetApp(u *libclient.UserContext) (*App, error) {
	ret := libclient.GetApp(u, AppID, NewApp)
	if ret == nil {
		return nil, core.InternalError("failed to get git app")
	}
	if ret.kvApp != nil {
		return ret, nil
	}
	kvApp, err := libkv.GetApp(u)
	if err != nil {
		return nil, err
	}
	ret.kvApp = kvApp
	return ret, nil
}

var _ libclient.App = &App{}
