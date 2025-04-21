// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package simple_ui

import (
	"fmt"
	"os"

	"github.com/foks-proj/go-foks/client/foks/cmd/common_ui"
	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"golang.org/x/term"
)

type PassphraseUI struct {
}

func (p *PassphraseUI) GetPassphrase(
	m libclient.MetaContext,
	ui proto.UserInfo,
	flags libclient.GetPassphraseFlags,
) (
	*proto.Passphrase,
	error,
) {
	opts := common_ui.FormatUserInfoOpts{
		Avatar: false,
		Active: false,
	}
	s, err := common_ui.FormatUserInfoAsPromptItem(ui, &opts)
	if err != nil {
		return nil, err
	}
	var prompt string
	if flags.IsConfirm {
		prompt = "Reenter to confirm"
	} else {
		new := ""
		prefix := "E"
		if flags.IsNew {
			new = "new "
		}
		if flags.IsRetry {
			prefix = "Passphrases didn't match; try again; e"
		}
		prompt = fmt.Sprintf("%snter %spassphrase for %s", prefix, new, s)
	}

	fmt.Printf("%s: ", prompt)
	pp, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	fmt.Printf("\n")
	if len(pp) == 0 {
		return nil, core.CanceledInputError{}
	}
	ret := proto.Passphrase(pp)
	return &ret, nil
}

var _ libclient.PassphraseUIer = &PassphraseUI{}

type PINUI struct{}

func (p *PINUI) GetPIN(
	m libclient.MetaContext,
) (
	*proto.YubiPIN,
	error,
) {
	fmt.Printf("Enter YubiKey PIN: ")
	pin, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	fmt.Printf("\n")
	ret := proto.YubiPIN(pin)
	return &ret, nil
}

var _ libclient.PINUIer = &PINUI{}
