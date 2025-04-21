// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

type EncryptSeedMacOSVersion int
type EncryptSeedGenericVersion int

const (
	EncryptSeedMacOSVersion0 EncryptSeedMacOSVersion = 0
	EncryptSeedMacOSVersion1 EncryptSeedMacOSVersion = 1
	EncryptSeedMacOSVersion2 EncryptSeedMacOSVersion = 2
)

const (
	// Align with most recent version of EncryptSeedMacOSVersion2
	EncryptSeedGenericVersion2 EncryptSeedGenericVersion = 2
)

type EncryptSeedMacOSOpts struct {
	IsTest bool
	Vers   EncryptSeedMacOSVersion
}
