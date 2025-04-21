// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/foks-proj/go-foks/client/foks/cmd/simple_ui"
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/mattn/go-isatty"
)

type FileStream struct {
	*os.File
}

func (f FileStream) IsATTY() bool {
	return isatty.IsTerminal(f.Fd())
}

type TerminalUI struct{}

func (t *TerminalUI) Printf(f string, args ...interface{}) { fmt.Printf(f, args...) }
func (t *TerminalUI) OutputStream() io.WriteCloser         { return os.Stdout }
func (t *TerminalUI) ErrorStream() libclient.IOStreamer    { return libclient.WrappedStderr }

func SetUIs(m libclient.MetaContext) {
	uis := simple_ui.Setup()
	uis.Terminal = &TerminalUI{}
	m.G().SetUIs(uis)
}
