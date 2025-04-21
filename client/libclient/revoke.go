// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

// ClearOnRevoke is called on a device that was recently revoked, It should clear the user out of in-memory
// user contexts, and should also clear the device's devkey from the secret store.
func ClearOnRevoke(
	m MetaContext,
	uc *UserContext,
	uw *UserWrapper,
) (bool, error) {

	if uw == nil {
		tok, err := uc.GetSelfViewToken(m)
		if err != nil {
			return false, err
		}
		uw, err = LoadUser(m,
			(LoadUserArg{LoadMode: LoadModeDeadSelf}).SetFQU(uc.FQU(), *tok),
		)
		if err != nil {
			return false, err
		}
	}

	dk, err := uc.Devkey(m.Ctx())
	if err != nil {
		return false, err
	}
	dkPub, err := dk.EntityID()
	if err != nil {
		return false, err
	}
	dev, err := uw.FindDevice(dkPub)
	if err != nil {
		return false, err
	}
	if dev == nil {
		return false, core.NotFoundError("device wasn't found")
	}
	if dev.Revoked == nil {
		return false, nil
	}

	devs, _ := dkPub.StringErr()

	m.Infow("ClearOnRevoke", "dev", devs)

	pub, err := dk.Publicize(&uc.Info.Fqu.HostID)
	if err != nil {
		return false, err
	}

	err = clearUserFromLocalStores(m, uc.Info.Fqu, pub)
	if err != nil {
		return false, err
	}

	return true, nil
}

func DeleteUserDevkey(m MetaContext, fqu proto.FQUser, role proto.Role, eid proto.EntityID) error {
	did, err := eid.ToDeviceID()
	if err != nil {
		return err
	}
	skm, ss, err := LoadSecretKeyMaterialManagerForUser(m, fqu, role)
	if _, ok := err.(core.KeyNotFoundError); ok {
		return nil
	}
	if err != nil {
		return err
	}

	err = skm.Delete(m.Ctx(), ss, did)

	// It's ok if we don't find the key on this device.
	if err != nil {
		return err
	}

	err = ss.Save(m.Ctx())
	if err != nil {
		return err
	}
	return nil
}
