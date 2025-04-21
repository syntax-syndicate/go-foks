// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package ui

// nkw = new key wizard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/foks-proj/go-foks/client/libclient"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func RunNewKeyWizard(m libclient.MetaContext) error {
	return RunModelForSessionType(m, proto.UISessionType_NewKeyWizard)
}

func newStateNKWHome() stateHome {
	return stateHome{
		oked:   false,
		letsDo: "Let's make a new key",
		nxt:    statePickUser{},
	}
}

const (
	newDeviceKey menuVal = 1
	newBackupKey menuVal = 2
	newYubiKey   menuVal = 3
)

type nkwState struct {
	mode menuVal
}

func newStateNKWPickKeyGenus() state {
	return statePickFromMenu{
		title: "Why type of key do you want to make?",
		initHook: func(mctx libclient.MetaContext, mdl model) ([]menuOpt, error) {
			uinfo, err := mdl.cli.LoadStateFromActiveUser(mctx.Ctx(), mdl.sessId)
			if err != nil {
				return nil, err
			}
			yd, err := mdl.ycli.ListAllLocalYubiDevices(mctx.Ctx(), mdl.sessId)
			if err != nil {
				return nil, err
			}

			var foundDevKey bool

			if uinfo.KeyGenus == proto.KeyGenus_Device {
				foundDevKey = true
			} else {
				alleu, err := mdl.usercli.GetExistingUsers(mctx.Ctx())
				if err != nil {
					return nil, err
				}
				for _, u := range alleu {
					if u.KeyGenus == proto.KeyGenus_Device && u.Fqu.Eq(uinfo.Fqu) {
						foundDevKey = true
						break
					}
				}
			}

			var opts []menuOpt

			if !foundDevKey {
				opts = append(opts, menuOpt{
					label: fmt.Sprintf("💻 A new %s on this device",
						BoldStyle.Render("permanent key")),
					val: newDeviceKey,
				})
			}

			opts = append(opts, menuOpt{
				label: fmt.Sprintf("💾 A new %s (to write down on paper)",
					BoldStyle.Render("backup key")),
				val: newBackupKey,
			})
			if len(yd) > 0 {
				opts = append(opts, menuOpt{
					label: fmt.Sprintf("🔑 A new key on my %s",
						BoldStyle.Render("YubiKey")),
					val: newYubiKey,
				})
			}
			return opts, nil
		},
		postHook: func(mctx libclient.MetaContext, s statePickFromMenu, mdl model) error {
			mdl.nkw.mode = s.choice
			return nil
		},
		nextHook: func(s statePickFromMenu, mdl model) state {
			switch s.choice {
			case newDeviceKey:
				return newStatePickDeviceName()
			case newBackupKey:
				return newStateFinishNKWNewBackupKey()
			case newYubiKey:
				return newStatePickYubiDeviceNew()
			}
			return nil
		},
	}
}

type stateNKWNewDeviceKey struct {
}

func (s stateNKWNewDeviceKey) init(mctx libclient.MetaContext, mdl model) (state, tea.Cmd, error) {
	return s, nil, nil
}

func (s stateNKWNewDeviceKey) next(mctx libclient.MetaContext, mdl model) (state, tea.Cmd, error) {
	return nil, nil, nil
}

func (s stateNKWNewDeviceKey) view() string {
	return "New device key"
}

func (s stateNKWNewDeviceKey) update(mctx libclient.MetaContext, mdl model, msg tea.Msg) (state, tea.Cmd, error) {
	return s, nil, nil
}

func (s stateNKWNewDeviceKey) failure() failure { return nil }
func (s stateNKWNewDeviceKey) summary() summary { return nil }

var _ state = stateNKWNewDeviceKey{}
