// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package libkv

import (
	"errors"
	"fmt"
	"sync"

	"github.com/foks-proj/go-foks/client/libclient"
	"github.com/foks-proj/go-foks/lib/core"
	"github.com/foks-proj/go-foks/lib/kv"
	"github.com/foks-proj/go-foks/proto/lcl"
	proto "github.com/foks-proj/go-foks/proto/lib"
	"github.com/foks-proj/go-foks/proto/rem"
)

type Versioner interface {
	GetVersion() proto.KVVersion
}

type memNode[V any, M any] struct {
	v *V
	m *M
}

type CacheSettings struct {
	UseMem  bool
	UseDisk bool
}

type Cache[
	S libclient.Scoper, // Scoper for DB cache, usually an FQParty
	K comparable, // The DB Key, like a DirID, etc
	V Versioner, // The value to store
	VP interface { // Pointer to the value to store, must implement core.Codecable
		*V
		core.Codecable
	},
	MV any, // A MemVal, something that we only store in memory (like decrypted data)
] struct {
	sync.RWMutex
	typ      lcl.DataType
	s        S
	m        map[K]*memNode[V, MV]
	dbt      libclient.DbType
	settings CacheSettings
}

func NewCache[
	S libclient.Scoper,
	K comparable,
	V Versioner,
	VP interface {
		*V
		core.Codecable
	},
	MV any,
](typ lcl.DataType, s S, settings CacheSettings) *Cache[S, K, V, VP, MV] {
	return &Cache[S, K, V, VP, MV]{
		typ:      typ,
		s:        s,
		dbt:      libclient.DbTypeSoft,
		settings: settings,
	}
}

func (c *Cache[S, K, V, VP, MV]) putMem(k K, v V, mv *MV) {
	c.Lock()
	defer c.Unlock()
	if !c.settings.UseMem {
		return
	}
	if c.m == nil {
		c.m = make(map[K]*memNode[V, MV])
	}
	c.m[k] = &memNode[V, MV]{v: &v, m: mv}
}

func (c *Cache[S, K, V, VP, MV]) clearMem(k K) {
	c.Lock()
	defer c.Unlock()
	if c.m == nil {
		return
	}
	delete(c.m, k)
}

func (c *Cache[S, K, V, VP, MV]) putDb(m MetaContext, k K, v V) error {
	if !c.settings.UseDisk {
		return nil
	}
	return m.DbPut(
		c.dbt,
		libclient.PutArg{
			Scope: c.s,
			Typ:   c.typ,
			Key:   k,
			Val:   (VP)(&v),
		},
	)
}

func (c *Cache[S, K, V, VP, MV]) Put(m MetaContext, k K, v V, memVal *MV) error {
	err := c.putDb(m, k, v)
	if err != nil {
		return err
	}
	c.putMem(k, v, memVal)
	return nil
}

func (c *Cache[S, K, V, VP, MV]) getMem(k K) (*V, *MV) {
	c.RLock()
	defer c.RUnlock()
	if c.m == nil {
		return nil, nil
	}
	v := c.m[k]
	if v == nil {
		return nil, nil
	}
	return v.v, v.m
}

func (c *Cache[S, K, V, VP, MV]) getDb(m MetaContext, k K) (*V, error) {
	if !c.settings.UseDisk {
		return nil, nil
	}
	var ret V
	getSlot := (VP)(&ret)
	_, err := m.DbGet(getSlot, c.dbt, c.s, c.typ, k)
	if err != nil && errors.Is(err, core.RowNotFoundError{}) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Cache[S, K, V, VP, MV]) Get(m MetaContext, k K) (*V, *MV, error) {
	v, memVal := c.getMem(k)
	if v != nil {
		return v, memVal, nil
	}
	ret, err := c.getDb(m, k)
	if err != nil {
		return nil, nil, err
	}
	if ret == nil {
		return nil, nil, nil
	}
	c.putMem(k, *ret, nil)

	return ret, nil, nil
}

func (c *Cache[S, K, V, VP, MV]) ClearBefore(m MetaContext, k K, vers proto.KVVersion) error {
	if vers == 0 {
		return nil
	}
	v, _ := c.getMem(k)
	if v != nil {
		cacheVers := (*v).GetVersion()
		if cacheVers == 0 || cacheVers == vers {
			return nil
		}
		if cacheVers != 0 && cacheVers != vers {
			c.clearMem(k)
		}
	}
	v, err := c.getDb(m, k)
	if err != nil {
		return err
	}
	if v == nil {
		return nil
	}
	err = m.DbDelete(c.dbt, c.s, c.typ, k)
	if err != nil {
		return err
	}
	return nil
}

type RootCache struct {
	*Cache[*proto.FQParty, core.EmptyKey, proto.KVRoot, *proto.KVRoot, struct{}]
}

func (r *RootCache) ClearBefore(m MetaContext, vers proto.KVVersion) error {
	if !r.settings.UseDisk {
		return nil
	}
	return r.Cache.ClearBefore(m, core.EmptyKey{}, vers)
}

type DirSeedPair struct {
	Active     *proto.DirKeySeed
	Encrypting *proto.DirKeySeed
}

type DirCache struct {
	*Cache[*proto.FQParty, proto.DirID, proto.KVDirPair, *proto.KVDirPair, DirSeedPair]
}

func NewRootCache(p proto.FQParty, settings CacheSettings) *RootCache {
	return &RootCache{NewCache[*proto.FQParty, core.EmptyKey, proto.KVRoot, *proto.KVRoot, struct{}](lcl.DataType_KVNSRoot, &p, settings)}
}

func (r *RootCache) Get(m MetaContext) (*proto.KVRoot, error) {
	v, _, err := r.Cache.Get(m, core.EmptyKey{})
	return v, err
}

func (r *RootCache) Put(m MetaContext, v proto.KVRoot) error {
	return r.Cache.Put(m, core.EmptyKey{}, v, nil)
}

func NewDirCache(p proto.FQParty, settings CacheSettings) *DirCache {
	return &DirCache{
		Cache: NewCache[
			*proto.FQParty,
			proto.DirID,
			proto.KVDirPair,
			*proto.KVDirPair,
			DirSeedPair,
		](lcl.DataType_KVDir, &p, settings),
	}
}

type DirWithSeed struct {
	proto.KVDir
	Seed *proto.DirKeySeed
}

type DirPair struct {
	Active     DirWithSeed
	Encrypting *DirWithSeed
	Owner      proto.FQParty
	IsRoot     bool
}

func (d *DirPair) GetVersion() proto.KVVersion {
	if d.Encrypting != nil {
		return d.Encrypting.Version
	}
	return d.Active.Version
}

func (d *DirPair) Id() proto.DirID {
	return d.Active.Id
}

func (d *DirPair) Split() (*proto.KVDirPair, *DirSeedPair) {
	ret := proto.KVDirPair{
		Active: d.Active.KVDir,
	}
	seed := DirSeedPair{
		Active: d.Active.Seed,
	}
	if d.Encrypting != nil {
		ret.Encrypting = &d.Encrypting.KVDir
		seed.Encrypting = d.Encrypting.Seed
	}
	return &ret, &seed
}

func NewDirPairFromSingle(o proto.FQParty, d proto.KVDir, s proto.DirKeySeed) *DirPair {
	return &DirPair{
		Active: DirWithSeed{
			KVDir: d,
			Seed:  &s,
		},
		Owner: o,
	}
}

func NewDirPair(o proto.FQParty, d proto.KVDirPair, s *DirSeedPair) *DirPair {
	ret := &DirPair{
		Active: DirWithSeed{
			KVDir: d.Active,
		},
		Owner: o,
	}
	if s != nil {
		ret.Active.Seed = s.Active
	}
	if d.Encrypting != nil {
		ret.Encrypting = &DirWithSeed{
			KVDir: *d.Encrypting,
		}
		if s != nil {
			ret.Encrypting.Seed = s.Encrypting
		}
	}
	return ret
}

func (d *DirCache) Get(m MetaContext, k proto.DirID) (*DirPair, error) {
	v, memVal, err := d.Cache.Get(m, k)
	if err != nil {
		return nil, err
	}
	m.VisitDir(k, v)
	if v == nil {
		return nil, nil
	}
	return NewDirPair(*d.s, *v, memVal), nil
}

func (d *DirCache) Put(m MetaContext, v *DirPair) error {
	dp, seed := v.Split()
	return d.Cache.Put(m, v.Id(), *dp, seed)
}

func (d *DirCache) PutMem(v *DirPair) {
	dp, seed := v.Split()
	d.Cache.putMem(v.Id(), *dp, seed)
}

type Dirent struct {
	proto.KVDirent
	Nm *proto.KVPathComponent // known upon decryption
}

type DirentCache struct {
	*Cache[*proto.FQParty, proto.HMAC, proto.KVDirent, *proto.KVDirent, proto.KVPathComponent]
}

func NewDirentCache(p proto.FQParty, settings CacheSettings) *DirentCache {
	return &DirentCache{
		Cache: NewCache[
			*proto.FQParty,
			proto.HMAC,
			proto.KVDirent,
			*proto.KVDirent,
			proto.KVPathComponent,
		](lcl.DataType_KVDirent, &p, settings),
	}
}

func (d *DirentCache) Get(m MetaContext, k proto.HMAC) (*Dirent, error) {
	v, memVal, err := d.Cache.Get(m, k)
	if err != nil {
		return nil, err
	}
	m.VisitDirent(v)
	if v == nil {
		return nil, nil
	}
	return &Dirent{
		KVDirent: *v,
		Nm:       memVal,
	}, nil
}

func (d *DirentCache) Put(m MetaContext, v *Dirent) error {
	return d.Cache.Put(m, v.NameMac, v.KVDirent, v.Nm)
}

func (d *DirentCache) PutMem(v *Dirent) {
	d.Cache.putMem(v.NameMac, v.KVDirent, v.Nm)
}

func (d *DirPair) WriteTo() *DirWithSeed {
	if d.Encrypting != nil {
		return d.Encrypting
	}
	return &d.Active
}

type Symlink struct {
	Raw  proto.KVPath
	Path kv.ParsedPath
}

type SymlinkCache struct {
	*Cache[*proto.FQParty, proto.SymlinkID, proto.SmallFileBox, *proto.SmallFileBox, Symlink]
}

func NewSymlinkCache(p proto.FQParty, settings CacheSettings) *SymlinkCache {
	return &SymlinkCache{
		Cache: NewCache[
			*proto.FQParty,
			proto.SymlinkID,
			proto.SmallFileBox,
			*proto.SmallFileBox,
			Symlink,
		](lcl.DataType_KVSymlink, &p, settings),
	}
}

func (s *SymlinkCache) PutMem(i proto.SymlinkID, b *proto.SmallFileBox, v *Symlink) {
	s.Cache.putMem(i, *b, v)
}

type SmallFileCache struct {
	*Cache[*proto.FQParty, proto.SmallFileID, proto.SmallFileBox, *proto.SmallFileBox, lcl.SmallFileData]
}

func NewSmallFileCache(p proto.FQParty, settings CacheSettings) *SmallFileCache {
	return &SmallFileCache{
		Cache: NewCache[
			*proto.FQParty,
			proto.SmallFileID,
			proto.SmallFileBox,
			*proto.SmallFileBox,
			lcl.SmallFileData,
		](lcl.DataType_KVSymlink, &p, settings),
	}
}

func (s *SmallFileCache) PutMem(i proto.SmallFileID, b *proto.SmallFileBox, v *lcl.SmallFileData) {
	s.Cache.putMem(i, *b, v)
}

type LargeFileMetadataCache struct {
	*Cache[*proto.FQParty, proto.FileID, proto.LargeFileMetadata, *proto.LargeFileMetadata, proto.FileKeySeed]
}

func NewLargeFileMetadataCache(p proto.FQParty, settings CacheSettings) *LargeFileMetadataCache {
	return &LargeFileMetadataCache{
		Cache: NewCache[
			*proto.FQParty,
			proto.FileID,
			proto.LargeFileMetadata,
			*proto.LargeFileMetadata,
			proto.FileKeySeed,
		](lcl.DataType_KVFileHeader, &p, settings),
	}
}

type ChunkIndex struct {
	FileID proto.FileID
	Offset proto.Offset
}

func (c ChunkIndex) DbKey() (proto.DbKey, error) {
	fid, err := c.FileID.KVNodeID().StringErr()
	if err != nil {
		return nil, err
	}
	s := fmt.Sprintf("%s-%d", fid, c.Offset)
	return proto.DbKey(s), nil
}

type LargeFileChunkCache struct {
	*Cache[*proto.FQParty, ChunkIndex, rem.GetEncryptedChunkRes, *rem.GetEncryptedChunkRes, struct{}]
}

func NewLargeFileChunkCache(p proto.FQParty, settings CacheSettings) *LargeFileChunkCache {
	settings.UseMem = false
	return &LargeFileChunkCache{
		Cache: NewCache[
			*proto.FQParty,
			ChunkIndex,
			rem.GetEncryptedChunkRes,
			*rem.GetEncryptedChunkRes,
			struct{},
		](lcl.DataType_KVFileChunk, &p, settings),
	}
}

type GitRefSetCache struct {
	*Cache[*proto.FQParty, proto.DirID, proto.GitRefBoxedSet, *proto.GitRefBoxedSet, gitRefSet]
}

func NewGitRefSetCache(p proto.FQParty, settings CacheSettings) *GitRefSetCache {
	return &GitRefSetCache{
		Cache: NewCache[
			*proto.FQParty,
			proto.DirID,
			proto.GitRefBoxedSet,
			*proto.GitRefBoxedSet,
			gitRefSet,
		](lcl.DataType_KVGitRefSet, &p, settings),
	}
}

type DirentCacheAccess struct {
	Version proto.KVVersion
	Hmac    proto.HMAC
}

type DirCacheAccess struct {
	Version proto.KVVersion
	Dirents map[proto.DirentID]DirentCacheAccess
}

type DirDirentPair struct {
	Dir    proto.DirID
	Dirent proto.DirentID
}

type CacheAccess struct {
	Dir  map[proto.DirID]DirCacheAccess
	Root proto.KVVersion
	kvp  *KVParty
}

func (C *CacheAccess) clear() {
	C.Dir = nil
	C.Root = 0
	C.kvp = nil
}

func NewCacheAccess() *CacheAccess {
	return &CacheAccess{}
}

type MetaContext struct {
	libclient.MetaContext
	cacheAccess *CacheAccess
	au          *libclient.UserContext
}

func (m MetaContext) Base() libclient.MetaContext {
	return m.MetaContext
}

func NewMetaContext(m libclient.MetaContext) MetaContext {
	return MetaContext{
		MetaContext: m,
		cacheAccess: NewCacheAccess(),
	}
}

func (m MetaContext) SetActiveUser(u *libclient.UserContext) MetaContext {
	m.au = u
	return m
}

func (m MetaContext) ActiveUser() (*libclient.UserContext, error) {
	if m.au != nil {
		return m.au, nil
	}
	ret := m.G().ActiveUser()
	if ret != nil {
		return ret, nil
	}
	return nil, core.NoActiveUserError{}
}

func (c *CacheAccess) setKVParty(p *KVParty) error {
	if c.kvp != nil && !c.kvp.Eq(p) {
		return core.InternalError("FQParty changed in request")
	}
	if c.kvp != nil {
		return nil
	}
	c.kvp = p
	return nil
}

func (m MetaContext) VisitDir(d proto.DirID, p *proto.KVDirPair) {
	if m.cacheAccess == nil {
		return
	}
	m.cacheAccess.VisitDir(d, p)
}

func (c *CacheAccess) VisitDir(d proto.DirID, p *proto.KVDirPair) {
	if p == nil {
		return
	}
	v := p.GetVersion()
	if c.Dir == nil {
		c.Dir = make(map[proto.DirID]DirCacheAccess)
	}
	dcc, ok := c.Dir[d]
	if ok {
		dcc.Version = max(v, dcc.Version)
	} else {
		dcc = DirCacheAccess{
			Version: v,
		}
		c.Dir[d] = dcc
	}
}

func max(a, b proto.KVVersion) proto.KVVersion {
	if a > b {
		return a
	}
	return b
}

func (m *MetaContext) InitCacheContext(p *KVParty) error {
	if p == nil {
		return core.InternalError("no party")
	}
	m.cacheAccess = NewCacheAccess()
	return m.cacheAccess.setKVParty(p)
}

func (m MetaContext) VisitRoot(v *proto.KVRoot) {
	if m.cacheAccess == nil {
		return
	}
	m.cacheAccess.VisitRoot(v)
}

func (c *CacheAccess) VisitRoot(r *proto.KVRoot) {
	if r == nil {
		return
	}
	c.Root = max(r.Vers, c.Root)
}

func (m MetaContext) VisitDirent(d *proto.KVDirent) {
	if m.cacheAccess == nil {
		return
	}
	m.cacheAccess.VisitDirent(d)
}

func (c *CacheAccess) VisitDirent(d *proto.KVDirent) {
	if d == nil {
		return
	}
	if c.Dir == nil {
		c.Dir = make(map[proto.DirID]DirCacheAccess)
	}
	dir, ok := c.Dir[d.ParentDir]
	if !ok {
		dir = DirCacheAccess{
			Version: d.Version,
		}
		c.Dir[d.ParentDir] = dir
	} else {
		dir.Version = max(d.DirVersion, dir.Version)
	}
	if dir.Dirents == nil {
		dir.Dirents = make(map[proto.DirentID]DirentCacheAccess)
	}
	dir.Dirents[d.Id] = DirentCacheAccess{
		Version: d.Version,
		Hmac:    d.NameMac,
	}
	c.Dir[d.ParentDir] = dir
}
