// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package agent

import (
	"context"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func (c *AgentConn) PassphraseUnlock(ctx context.Context, pp proto.Passphrase) error {
	m := c.MetaContext(ctx)
	return libclient.PassphraseUnlockCurrentUser(m, pp)
}

func (c *AgentConn) PassphraseSet(ctx context.Context, pp lcl.PassphraseSetArg) error {
	m := c.MetaContext(ctx)
	au, err := m.ActiveConnectedUser(&libclient.ACUOpts{AssertUnlocked: true})
	if err != nil {
		return err
	}
	puk := au.PrivKeys.LatestPuk()
	if puk == nil {
		return core.KeyNotFoundError{Which: "puk"}
	}

	pm := libclient.NewPassphraseManager(au.FQU())

	psi, err := libclient.NewPMELoggedIn(m, au)
	if err != nil {
		return err
	}

	if pp.First {
		err = pm.SetPassphrase(ctx, psi, pp.Passphrase, puk)
	} else {
		err = pm.ChangePassphraseWithPUK(ctx, psi, pp.Passphrase, au.PrivKeys.GetPUKs())
	}
	if err != nil {
		return err
	}

	// If we're no yubikey, no need to encrypt the seed.
	if au.IsOnYubiKey() {
		return nil
	}

	skmm := au.SkmmGetOrMake()

	ss := m.G().SecretStore()

	err = skmm.Load(ctx, ss, libclient.SecretStoreGetOpts{NoProvisional: true})
	if err != nil {
		return err
	}

	err = skmm.SetPassphrase(ctx, pm, ss, pp.First)
	if err != nil {
		return err
	}

	return nil
}
