// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package ui

import (
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
)

type menuVal int

const (
	menuValNone menuVal = -1
)

func (v menuVal) isChosen() bool { return v != menuValNone }

type menuOpt struct {
	val   menuVal
	label string
}

func (s menuOpt) FilterValue() string { return "" }

type statePickFromMenu struct {
	choice  menuVal
	opts    []menuOpt
	spinner spinner.Model
	loading bool
	title   string
	picker  list.Model

	nextHook func(s statePickFromMenu, mdl model) state
	postHook func(mctx libclient.MetaContext, s statePickFromMenu, mdl model) error
	initHook func(mctx libclient.MetaContext, mdl model) ([]menuOpt, error)
	succHook func(s statePickFromMenu) summary
}

func (s statePickFromMenu) summary() summary {
	if s.succHook != nil {
		return s.succHook(s)
	}
	return nil
}
func (s statePickFromMenu) failure() failure { return nil }

type statePickFromMenuInitRes struct {
	opts []menuOpt
	err  error
}

func (s statePickFromMenu) init(mctx libclient.MetaContext, mdl model) (state, tea.Cmd, error) {
	s.spinner = spinner.New()
	s.spinner.Spinner = spinner.Dot
	s.loading = true
	s.choice = menuValNone
	return s, tea.Batch(
		s.spinner.Tick,
		func() tea.Msg {
			debugSpinners(mctx)
			opts, err := s.initHook(mctx, mdl)
			return statePickFromMenuInitRes{opts: opts, err: err}
		},
	), nil
}

func (s statePickFromMenu) next(mctx libclient.MetaContext, mdl model) (state, tea.Cmd, error) {
	if s.loading {
		return nil, nil, nil
	}
	if !s.choice.isChosen() {
		return nil, nil, nil
	}
	nxt := s.nextHook(s, mdl)
	return nxt, nil, nil
}

func (s statePickFromMenu) update(mctx libclient.MetaContext, mdl model, msg tea.Msg) (state, tea.Cmd, error) {

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd, nil

	case statePickFromMenuInitRes:
		if msg.err != nil {
			return s, nil, msg.err
		}
		s.loading = false
		items := core.Map(msg.opts, func(o menuOpt) list.Item { return o })
		s.opts = msg.opts
		l := list.New(items, menuOptDelegate{}, defaultWidth, listHeight)
		l.Title = s.title
		styleList(&l)
		s.picker = l
		return s, nil, nil
	}

	s.picker, cmd = s.picker.Update(msg)

	if !isEnter(msg) {
		return s, cmd, nil
	}

	choice, ok := s.picker.SelectedItem().(menuOpt)
	if !ok {
		return s, cmd, nil
	}
	s.choice = choice.val

	if s.postHook != nil {
		err := s.postHook(mctx, s, mdl)
		if err != nil {
			return s, cmd, err
		}
	}

	return s, cmd, nil
}

func (s statePickFromMenu) view() string {
	if s.loading {
		return "\n\n  " + s.spinner.View() + " Loading..."
	}
	return "\n" + s.picker.View()
}

type menuOptDelegate struct {
	baseItemDelegate
}

func (d menuOptDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(menuOpt)
	if !ok {
		return
	}
	d.renderString(w, m, index, i.label)
}

var _ state = &statePickFromMenu{}
