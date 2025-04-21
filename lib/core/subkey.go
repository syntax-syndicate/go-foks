// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import proto "github.com/foks-proj/go-foks/proto/lib"

func MakeSubkey(
	parent PrivateSuiter,
	host proto.HostID,
) (
	EntityPrivate,
	*proto.Box,
	error,
) {
	seed := RandomSecretSeed32()
	subkeySeed, err := DeviceSigningSecretKey(seed)
	if err != nil {
		return nil, nil, err
	}
	subkey := NewEntityPrivateEd25519WithSeed(
		proto.EntityType_Subkey,
		*subkeySeed,
	)
	subkeyPub, err := subkey.EntityPublic()
	if err != nil {
		return nil, nil, err
	}
	parentPub, err := parent.Publicize(&host)
	if err != nil {
		return nil, nil, err
	}
	payload := proto.SubkeySeed{
		Parent: parentPub.GetEntityID(),
		Subkey: subkeyPub.GetEntityID(),
		Seed:   seed,
	}
	box, err := SelfBox(parent, &payload, host)
	if err != nil {
		return nil, nil, err
	}
	return subkey, box, nil
}
