// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"github.com/foks-proj/go-foks/lib/core"
	proto "github.com/foks-proj/go-foks/proto/lib"
)

func FindProvisionedDeviceKeys(
	m MetaContext,
) (
	[]proto.DeviceID,
	error,
) {
	au, err := m.ActiveConnectedUser(&ACUOpts{AssertUnlocked: true})
	if err != nil {
		return nil, err
	}
	uw, err := LoadMe(m, au)
	if err != nil {
		return nil, err
	}

	ss := m.G().SecretStore()

	var deviceIds []proto.DeviceID
	for _, di := range uw.Prot().Devices {
		did, err := di.Key.Member.Id.Entity.ToDeviceID()
		if err == nil {
			deviceIds = append(deviceIds, did)
		}
	}

	return ss.FilterDeviceIDs(uw.fqu, deviceIds)
}

func AssertNotAlreadyProvisioned(
	m MetaContext,
) error {
	keys, err := FindProvisionedDeviceKeys(m)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return core.DeviceAlreadyProvisionedError{}
}
