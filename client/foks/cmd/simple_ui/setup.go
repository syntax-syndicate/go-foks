// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package simple_ui

import "github.com/foks-proj/go-foks/client/libclient"

func Setup() libclient.UIs {
	return libclient.UIs{
		Signup:     &SignupUI{},
		Assist:     &AssistUI{},
		Passphrase: &PassphraseUI{},
		Backup:     &BackupUI{},
		SSOLogin:   &SSOLoginUI{},
		PIN:        &PINUI{},
		LogSend:    &LogSendUI{},
	}
}
