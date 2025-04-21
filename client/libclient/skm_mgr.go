// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libclient

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/zalando/go-keyring"
)

// SecretKeyMaterialManager manages secret key material that's read out of the secret
// store. This object exists per-user. It will aid in decrypting, and encrypting SKMs
// as necessary.
type SecretKeyMaterialManager struct {
	sync.RWMutex

	user        proto.FQUser
	role        proto.Role
	did         proto.DeviceID
	selfTok     proto.PermissionToken
	provisional bool

	mtime time.Time
	ctime time.Time

	// Specified once per secret store file, we use this to formulate macOS
	// keychain labels.
	liid proto.LocalInstanceID

	// The current row in the secret store file, may or may not be unlocked
	bun *lcl.StoredSecretKeyBundle

	// For passphrase unlock, we know salt before we know the seed and secret
	salt  *proto.PassphraseSalt
	sv    *proto.StretchVersion
	ppgen *proto.PassphraseGeneration

	// If available, here is the secret seed that's driving this device
	seed *proto.SecretSeed32

	// when testing we write to a different location in the keychain so as not
	// to polute the main keychain. Revisit maybe once we have global configs
	// we are passing around.
	isTest bool

	// Directory where the secret store is (in case we need to store other files
	// alongside).
	secretStoreDir string
}

func NewSecretKeyMaterialManager(u proto.FQUser, role proto.Role) *SecretKeyMaterialManager {
	return &SecretKeyMaterialManager{
		user: u,
		role: role,
	}
}

func (s *SecretKeyMaterialManager) Row() lcl.LabeledSecretKeyBundle {
	return lcl.LabeledSecretKeyBundle{
		Fqur: proto.FQUserAndRole{
			Fqu:  s.user,
			Role: s.role,
		},
		KeyID:       s.did,
		SelfTok:     s.selfTok,
		Provisional: s.provisional,
	}
}

func (s *SecretKeyMaterialManager) DeviceID() proto.DeviceID {
	s.RLock()
	defer s.RUnlock()
	return s.did
}

func (s *SecretKeyMaterialManager) SigingKey() (*proto.Ed25519SecretKey, error) {
	s.RLock()
	defer s.RUnlock()
	if s.seed == nil {
		return nil, nil
	}
	return core.DeviceSigningSecretKey(*s.seed)
}

func (s *SecretKeyMaterialManager) DHKey() (*proto.Curve25519SecretKey, error) {
	s.RLock()
	defer s.RUnlock()
	if s.seed == nil {
		return nil, nil
	}
	return core.DeviceDHSecretKey(*s.seed)
}

func (s *SecretKeyMaterialManager) Seed() *proto.SecretSeed32 {
	s.RLock()
	defer s.RUnlock()
	return s.seed
}

func (s *SecretKeyMaterialManager) SelfViewToken() *proto.PermissionToken {
	s.RLock()
	defer s.RUnlock()
	return &s.selfTok
}

// Load the secret key for this user form the secret store file
func (s *SecretKeyMaterialManager) Load(ctx context.Context, ss *SecretStore, opts SecretStoreGetOpts) error {
	s.Lock()
	defer s.Unlock()
	return s.loadLocked(ctx, ss, opts)
}

func (s *SecretKeyMaterialManager) ClearProvisionalBit(ctx context.Context, ss *SecretStore) error {
	s.Lock()
	defer s.Unlock()

	n, err := ss.ClearProvisionalBits(ctx, s.user, []proto.DeviceID{s.did})
	if err != nil {
		return err
	}
	if n == 0 {
		return core.NotFoundError("no provisional bit to clear")
	}
	s.provisional = false
	return nil
}

func (s *SecretKeyMaterialManager) Delete(ctx context.Context, ss *SecretStore, did proto.DeviceID) error {
	if len(did) == 0 {
		return core.InternalError("need device ID for delete as a safety precaution")
	}

	s.Lock()
	defer s.Unlock()
	err := s.loadLocked(ctx, ss, SecretStoreGetOpts{ByDeviceID: true})
	if err != nil {
		return err
	}
	if !s.did.Eq(did) {
		return core.KeyMismatchError{}
	}

	err = ss.Delete(s.user, s.role, did)
	if err != nil {
		return err
	}

	err = s.deleteStorageByType(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *SecretKeyMaterialManager) IsProvisional() bool {
	s.RLock()
	defer s.RUnlock()
	return s.provisional
}

func (s *SecretKeyMaterialManager) loadLocked(ctx context.Context, ss *SecretStore, opts SecretStoreGetOpts) error {
	if s.bun != nil {
		return nil
	}
	row, err := ss.Get(SecretStoreGetArgs{
		Fqu:      s.user,
		Role:     s.role,
		DeviceID: s.did,
		Opts:     opts,
	})
	if err != nil {
		return err
	}
	if row == nil {
		return core.KeyNotFoundError{Which: "secret"}
	}
	s.bun = &row.Bundle
	s.secretStoreDir = ss.Dir()
	s.did = row.KeyID
	s.selfTok = row.SelfTok
	s.provisional = row.Provisional
	s.mtime = row.Mtime.Import()
	s.ctime = row.Ctime.Import()
	tmp, err := ss.LocalInstanceID()
	if err != nil {
		return err
	}
	s.liid = *tmp
	return nil
}

func (s *SecretKeyMaterialManager) Bundle() *lcl.StoredSecretKeyBundle {
	s.Lock()
	defer s.Unlock()
	return s.bun
}

// Unbundle a generic versioned SecretKeyBundle into the one version we can
// currently understand, and send up an error if that didn't work.
func (s *SecretKeyMaterialManager) unbundle(b lcl.SecretKeyBundle) error {
	v, err := b.GetV()
	if err != nil {
		return err
	}
	if v != lcl.SecretKeyBundleVersion_V1 {
		return core.VersionNotSupportedError(fmt.Sprintf("cannot handle SecretKeyBundleVersion %d", v))
	}
	tmp := b.V1()
	s.seed = &tmp
	return nil
}

func (s *SecretKeyMaterialManager) Salt() *proto.PassphraseSalt {
	s.RLock()
	defer s.RUnlock()
	return s.salt
}

func (s *SecretKeyMaterialManager) StretchVersion() *proto.StretchVersion {
	s.RLock()
	defer s.RUnlock()
	return s.sv
}

func (s *SecretKeyMaterialManager) DeviceKeyPrivateSuiter(
	ctx context.Context,
) (core.PrivateSuiter, error) {
	err := s.ReadySeed(ctx)
	if err != nil {
		return nil, err
	}
	return core.NewPrivateSuite25519(proto.EntityType_Device, s.role, *s.seed, s.user.HostID)
}

func (s *SecretKeyMaterialManager) ReadySeed(ctx context.Context) error {
	_, err := s.ReadySeedReturnType(ctx)
	return err
}

func (s *SecretKeyMaterialManager) ReadySeedReturnType(ctx context.Context) (proto.SecretKeyStorageType, error) {
	s.Lock()
	defer s.Unlock()
	return s.readySeedLocked(ctx)
}

func (s *SecretKeyMaterialManager) ClearKeychain(enc lcl.KeychainEncryptedSecretBundle) error {
	lbl, err := s.SecretKeyLabel()
	if err != nil {
		return err
	}
	err = keyring.Delete(enc.Service, lbl)
	if err != nil {
		return err
	}
	return nil
}

func SecretKeyLabelV2(
	fqu proto.FQUser,
	liid proto.LocalInstanceID,
	did proto.DeviceID,
) (string, error) {
	return lcl.SecretKeyKeychainLabelV2{
		Fqu:  fqu,
		Liid: liid,
		Did:  did,
	}.StringErr()
}

func (s *SecretKeyMaterialManager) SecretKeyLabel() (string, error) {
	return SecretKeyLabelV2(s.user, s.liid, s.did)
}

func (s *SecretKeyMaterialManager) deleteStorageByType(ctx context.Context) error {
	if s.bun == nil {
		return core.NotFoundError("no key material loaded for this user")
	}

	typ, err := s.bun.GetT()
	if err != nil {
		return err
	}
	switch typ {
	case proto.SecretKeyStorageType_ENC_MACOS_KEYCHAIN:
		return s.ClearMacOSKeychain(s.bun.EncMacosKeychain())
	case proto.SecretKeyStorageType_ENC_NOISE_FILE:
		return ClearNoiseFile(ctx, s.secretStoreDir, s.bun.EncNoiseFile().Filename)
	case proto.SecretKeyStorageType_ENC_KEYCHAIN:
		return s.ClearKeychain(s.bun.EncKeychain())
	}
	return nil
}

type EncryptSeedGenericOpts struct {
	IsTest bool
}

func EncryptSeedWithGenericKeychain(
	ctx context.Context,
	ss proto.SecretSeed32,
	liid proto.LocalInstanceID,
	fqu proto.FQUser,
	did proto.DeviceID,
	opts EncryptSeedGenericOpts,
) (*lcl.StoredSecretKeyBundle, error) {
	lbl, err := SecretKeyLabelV2(fqu, liid, did)
	if err != nil {
		return nil, err
	}

	var key proto.SecretBoxKey
	err = core.RandomFill(key[:])
	if err != nil {
		return nil, err
	}
	skb := lcl.NewSecretKeyBundleWithV1(ss)
	newBox, err := core.SealIntoSecretBox(&skb, &key)
	if err != nil {
		return nil, err
	}
	key62 := core.B62Encode(key[:])
	bun := lcl.KeychainEncryptedSecretBundle{
		Service:   KeychainServiceName(opts.IsTest),
		SecretBox: *newBox,
	}
	err = keyring.Set(bun.Service, lbl, key62)
	if err != nil {
		return nil, err
	}

	ret := lcl.NewStoredSecretKeyBundleWithEncKeychain(bun)
	return &ret, nil
}

func (s *SecretKeyMaterialManager) unlockGenericKeychain(
	ctx context.Context,
	enc lcl.KeychainEncryptedSecretBundle,
) error {
	lbl, err := s.SecretKeyLabel()
	if err != nil {
		return err
	}
	skey, err := keyring.Get(enc.Service, lbl)
	if err != nil {
		return err
	}
	if len(skey) == 0 {
		return core.KeychainError("keychain item not found")
	}
	key, err := core.B62Decode(skey)
	if err != nil {
		return err
	}
	var boxKey proto.SecretBoxKey
	if len(key) != len(boxKey) {
		return core.KeychainError("keychain item has unexpected length")
	}
	copy(boxKey[:], key)
	var ret lcl.SecretKeyBundle
	err = core.OpenSecretBoxInto(&ret, enc.SecretBox, &boxKey)
	if err != nil {
		return err
	}
	err = s.unbundle(ret)
	if err != nil {
		return err
	}
	return nil
}

func (s *SecretKeyMaterialManager) readySeedLocked(
	ctx context.Context,
) (
	proto.SecretKeyStorageType,
	error,
) {
	var typ proto.SecretKeyStorageType

	if s.bun == nil {
		return typ, core.NotFoundError("no key material loaded for this user")
	}

	typ, err := s.bun.GetT()
	if err != nil {
		return typ, err
	}

	if s.seed != nil {
		return typ, nil
	}

	switch typ {
	case proto.SecretKeyStorageType_PLAINTEXT:
		return typ, s.unbundle(s.bun.Plaintext())
	case proto.SecretKeyStorageType_ENC_MACOS_KEYCHAIN:
		return typ, s.unlockMacOS(ctx, s.bun.EncMacosKeychain())
	case proto.SecretKeyStorageType_ENC_PASSPHRASE:
		bun := s.bun.EncPassphrase()
		s.salt = &bun.Salt
		s.sv = &bun.StretchVersion
		s.ppgen = &bun.Ppgen
		return typ, core.PassphraseLockedError{}
	case proto.SecretKeyStorageType_ENC_NOISE_FILE:
		return typ, s.unlockNoise(ctx, s.bun.EncNoiseFile())
	case proto.SecretKeyStorageType_ENC_KEYCHAIN:
		return typ, s.unlockGenericKeychain(ctx, s.bun.EncKeychain())
	default:
		return typ, core.NotImplementedError{}
	}
}

func (s *SecretKeyMaterialManager) UnlockSeed(
	ctx context.Context,
	pme PassphraseManagerEngine,
	puks []core.SharedPrivateSuiter,
) (proto.SecretKeyStorageType, error) {
	s.Lock()
	defer s.Unlock()
	typ, err := s.readySeedLocked(ctx)
	if err == nil {
		return typ, err
	}
	if !errors.Is(err, core.PassphraseLockedError{}) {
		return typ, err
	}
	pm := NewPassphraseManager(s.user)
	err = pm.UnlockWithPUKs(ctx, pme, puks)
	if err != nil {
		return typ, err
	}
	bundle := s.bun.EncPassphrase()
	err = s.decryptInner(pm, bundle)
	if err != nil {
		return typ, err
	}
	return typ, nil
}

func (s *SecretKeyMaterialManager) SetPassphrase(
	ctx context.Context,
	ppm *PassphraseManager,
	ss *SecretStore,
	first bool,
) error {
	s.Lock()
	defer s.Unlock()
	typ, err := s.readySeedLocked(ctx)
	if typ != proto.SecretKeyStorageType_ENC_PASSPHRASE && !first {
		return nil
	}
	if err != nil && errors.Is(err, core.PassphraseLockedError{}) && typ == proto.SecretKeyStorageType_ENC_PASSPHRASE {
		bundle := s.bun.EncPassphrase()
		err = s.decryptInner(ppm, bundle)
	}
	if err != nil {
		return err
	}
	err = s.encryptWithPassphraseLocked(ctx, ppm, ss)
	if err != nil {
		return err
	}
	return nil
}

// ReeencryptIfPassphraseLocked will reencrypt the secret key material if it's currently encrypted
// with a passphrase. This should be used with the *PUK* updates on this machine, forcing a reencryption
// to the newest SKWK.
func (s *SecretKeyMaterialManager) ReencryptIfPassphraseLocked(
	ctx context.Context,
	ppm *PassphraseManager,
	ss *SecretStore,
) error {
	s.Lock()
	defer s.Unlock()
	err := s.loadLocked(ctx, ss, SecretStoreGetOpts{NoProvisional: true})
	if _, ok := err.(core.NotFoundError); ok {
		return nil
	}
	if err != nil {
		return err
	}
	typ, err := s.bun.GetT()
	if err != nil {
		return err
	}
	if typ != proto.SecretKeyStorageType_ENC_PASSPHRASE {
		return nil
	}
	bundle := s.bun.EncPassphrase()
	err = s.decryptInner(ppm, bundle)
	if err != nil {
		return err
	}

	err = s.encryptWithPassphraseLocked(ctx, ppm, ss)
	if err != nil {
		return err
	}
	return nil
}

func (s *SecretKeyMaterialManager) decryptInner(
	ppm *PassphraseManager,
	bundle lcl.PassphraseEncryptedSecretKeyBundle,
) error {

	ourKey, err := ppm.GetSKMWK(bundle.Ppgen)
	if err != nil {
		return err
	}
	var ret lcl.SecretKeyBundle
	err = core.OpenSecretBoxInto(&ret, bundle.SecretBox, (*proto.SecretBoxKey)(ourKey))
	if err != nil {
		return err
	}
	err = s.unbundle(ret)
	if err != nil {
		return err
	}
	return nil
}

func (s *SecretKeyMaterialManager) DecryptWithPassphrase(
	ctx context.Context,
	ppm *PassphraseManager,
	ss *SecretStore,
) error {

	s.Lock()
	defer s.Unlock()

	if s.bun == nil {
		return core.NotFoundError("no secret key material loaded from file")
	}
	typ, err := s.bun.GetT()
	if err != nil {
		return err
	}
	if typ != proto.SecretKeyStorageType_ENC_PASSPHRASE {
		return core.SecretKeyStorageTypeError{Actual: typ}
	}

	bundle := s.bun.EncPassphrase()

	err = s.decryptInner(ppm, bundle)
	if err != nil {
		return err
	}
	return nil
}

// To be called by a logged-in user with unlocked PUKs and authenticated access
// to the user server. It will load the user settings chain (which requires a login),
// refresh the server's list of encrypted SKWKs (if necessary), and reencrypt
// the passphrase on disk to reflect a potentially refresh passphrase.
func (s *SecretKeyMaterialManager) RefreshUserSettingsMaybeUpdatePassphraseEncryption(
	ctx context.Context,
	ppm *PassphraseManager,
	pme PassphraseManagerEngine,
	ppGen proto.PassphraseGeneration,
	puks []core.SharedPrivateSuiter,
	ss *SecretStore,
) error {

	err := ppm.RefreshWithUserSettings(ctx, pme, puks)
	if err != nil {
		return err
	}

	return s.MaybeUpdatePassphraseEncryption(ctx, ppm, ss, ppGen)
}

func (s *SecretKeyMaterialManager) MaybeUpdatePassphraseEncryption(
	ctx context.Context,
	ppm *PassphraseManager,
	ss *SecretStore,
	ppGen proto.PassphraseGeneration,
) error {
	s.Lock()
	defer s.Unlock()

	// If we're fully encrypted at the latest password, we're ok to return out of
	// here. Otherwise, we need to keep going and reencrypt our secret for the
	// newest passphrase
	if ppm.IsLatest(ppGen) {
		return nil
	}

	err := s.encryptWithPassphraseLocked(ctx, ppm, ss)
	if err != nil {
		return err
	}

	return nil
}

func EncryptWithPPE(
	ctx context.Context,
	ss proto.SecretSeed32,
	ppe *lcl.KexPPE,
) (*lcl.StoredSecretKeyBundle, error) {
	return encryptWithSKMWK(
		ctx,
		ss,
		&ppe.Skwk,
		ppe.PpGen,
		ppe.Salt,
		ppe.Sv,
	)
}

func EncryptWithPassphrase(
	ctx context.Context,
	ss proto.SecretSeed32,
	pm *PassphraseManager,
) (*lcl.StoredSecretKeyBundle, error) {

	latestKey, latestGen, salt, sv := pm.GetLatest()
	if latestKey == nil || salt == nil {
		return nil, core.InternalError("no passphrase found for encryption")
	}

	return encryptWithSKMWK(
		ctx,
		ss,
		latestKey,
		latestGen,
		*salt,
		sv,
	)
}

func EncryptWithPUKs(
	ctx context.Context,
	fqu proto.FQUser,
	ss proto.SecretSeed32,
	puks []core.SharedPrivateSuiter,
	pme PassphraseManagerEngine,
) (
	*lcl.StoredSecretKeyBundle,
	error,
) {
	pm := NewPassphraseManager(fqu)
	err := pm.RefreshWithUserSettings(ctx, pme, puks)
	if err != nil {
		return nil, err
	}
	return EncryptWithPassphrase(ctx, ss, pm)
}

func encryptWithSKMWK(
	ctx context.Context,
	ss proto.SecretSeed32,
	latestKey *lcl.SKMWK,
	latestGen proto.PassphraseGeneration,
	salt proto.PassphraseSalt,
	sv proto.StretchVersion,
) (
	*lcl.StoredSecretKeyBundle,
	error,
) {

	newBundle := lcl.PassphraseEncryptedSecretKeyBundle{
		Ppgen:          latestGen,
		Salt:           salt,
		StretchVersion: sv,
	}

	skb := lcl.NewSecretKeyBundleWithV1(ss)
	newBox, err := core.SealIntoSecretBox(&skb, (*proto.SecretBoxKey)(latestKey))
	if err != nil {
		return nil, err
	}
	newBundle.SecretBox = *newBox

	tmp := lcl.NewStoredSecretKeyBundleWithEncPassphrase(newBundle)

	return &tmp, nil
}

func (s *SecretKeyMaterialManager) encryptWithPassphraseLocked(
	ctx context.Context,
	ppm *PassphraseManager,
	ss *SecretStore,
) error {
	tmp, err := EncryptWithPassphrase(ctx, *s.seed, ppm)
	if err != nil {
		return err
	}
	row := lcl.LabeledSecretKeyBundle{
		Fqur: proto.FQUserAndRole{
			Fqu:  s.user,
			Role: s.role,
		},
		KeyID:   s.did,
		SelfTok: s.selfTok,
		Bundle:  *tmp,
	}
	err = ss.UpdateAndSave(ctx, row)
	if err != nil {
		return err
	}
	s.bun = tmp
	return nil
}

func (s *SecretKeyMaterialManager) EncryptWithPassphrase(
	ctx context.Context,
	ppm *PassphraseManager,
	ss *SecretStore,
) error {
	s.Lock()
	defer s.Unlock()
	return s.encryptWithPassphraseLocked(ctx, ppm, ss)
}

func (s *SecretKeyMaterialManager) ClearSeed() {
	s.Lock()
	defer s.Unlock()
	s.seed = nil
}

func (s *SecretKeyMaterialManager) Clear() {
	s.Lock()
	defer s.Unlock()
	s.bun = nil
	s.seed = nil
}

func (s *SecretKeyMaterialManager) TestingOnlyDeleteMacOSKeychainItem(
	ctx context.Context,
	liid proto.LocalInstanceID,
) error {
	s.Lock()
	defer s.Unlock()
	lab := lcl.SecretKeyKeychainLabelV2{
		Fqu:  s.user,
		Liid: liid,
		Did:  s.did,
	}
	uns, err := lab.StringErr()
	if err != nil {
		return err
	}
	bun := lcl.MacOSKeychainEncryptedSecretBundle{
		Account: uns,
		Service: MacOSServiceName(true),
	}
	return s.ClearMacOSKeychain(bun)
}

func (s *SecretKeyMaterialManager) UnlockToPlaintext(
	ctx context.Context,
	ss *SecretStore,
) error {
	s.Lock()
	defer s.Unlock()
	if s.seed == nil {
		return core.InternalError("need an unlocked secret manager first")
	}
	bun := lcl.NewStoredSecretKeyBundleWithPlaintext(lcl.NewSecretKeyBundleWithV1(*s.seed))
	row := lcl.LabeledSecretKeyBundle{
		Fqur: proto.FQUserAndRole{
			Fqu:  s.user,
			Role: s.role,
		},
		KeyID:   s.did,
		SelfTok: s.selfTok,
		Bundle:  bun,
	}
	err := ss.UpdateAndSave(ctx, row)
	if err != nil {
		return err
	}
	tmp := s.bun
	s.bun = &bun

	// Now potentially clear the keychain, depending on previous encryption type
	// and also platform

	if tmp == nil {
		return nil
	}

	typ, err := tmp.GetT()
	if err != nil {
		return err
	}

	switch typ {
	case proto.SecretKeyStorageType_ENC_MACOS_KEYCHAIN:
		err = s.ClearMacOSKeychain(tmp.EncMacosKeychain())
		if err != nil {
			switch err.(type) {
			// Platform error we get if not on Mac; that's OK
			case core.PlatformError:
			default:
				return err
			}
		}
		return nil
	case proto.SecretKeyStorageType_ENC_KEYCHAIN:
		err = s.ClearKeychain(tmp.EncKeychain())
		if err != nil {
			return err
		}
		return nil
	default:
		return nil
	}
}

func LoadSecretKeyMaterialManagerForUser(
	m MetaContext,
	fqu proto.FQUser,
	role proto.Role,
) (
	*SecretKeyMaterialManager,
	*SecretStore,
	error,
) {
	ss := m.G().SecretStore()
	au := m.G().ActiveUser()

	if au != nil && au.FQU().Eq(fqu) {
		eq, err := role.Eq(au.Role())
		if err != nil {
			return nil, nil, err
		}
		if eq {
			ret, err := au.LoadSkkm(m)
			if err != nil {
				return nil, nil, err
			}
			return ret, ss, nil
		}
	}
	skkm := NewSecretKeyMaterialManager(fqu, role)
	err := skkm.Load(m.Ctx(), m.G().SecretStore(), SecretStoreGetOpts{NoProvisional: true})
	if err != nil {
		return nil, nil, err
	}
	return skkm, ss, nil
}
