// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/foks-proj/go-foks/server/web/static/css"
	"github.com/foks-proj/go-foks/server/web/static/img"
	"github.com/foks-proj/go-foks/server/web/static/js"
)

type StaticFSFile struct {
	*bytes.Reader
	nm string
}

func NewStaticFSFile(s string) *StaticFSFile {
	return &StaticFSFile{
		Reader: bytes.NewReader([]byte(s)),
		nm:     s,
	}
}

func (f *StaticFSFile) Close() error               { return nil }
func (f *StaticFSFile) Stat() (fs.FileInfo, error) { return f, nil }

type startupTime struct {
	sync.Mutex
	tm time.Time
}

func (s *startupTime) Time() time.Time {
	s.Lock()
	defer s.Unlock()
	if s.tm.IsZero() {
		s.tm = time.Now()
	}
	return s.tm
}

var upSince startupTime

func (f *StaticFSFile) IsDir() bool        { return false }
func (f *StaticFSFile) ModTime() time.Time { return upSince.Time() }
func (f *StaticFSFile) Mode() fs.FileMode  { return fs.FileMode(0o644) }
func (f *StaticFSFile) Name() string       { return f.nm }
func (f *StaticFSFile) Size() int64        { return int64(f.Reader.Len()) }
func (f *StaticFSFile) Sys() any           { return nil }

var _ fs.File = (*StaticFSFile)(nil)
var _ fs.FileInfo = (*StaticFSFile)(nil)

type StaticFS struct {
	sync.Mutex
	hashes  []string
	lookup  map[string]string
	allData []string
}

func (s *StaticFS) init() {
	files := []struct {
		n string
		d string
	}{
		{n: "css/style.css", d: css.Style},
		{n: "css/style.min.css", d: css.StyleMin},
		{n: "js/htmx.js", d: js.Htmx},
		{n: "js/htmx.min.js", d: js.HtmxMin},
		{n: "js/foks.js", d: js.Foks},
		{n: "js/foks.min.js", d: js.FoksMin},
		{n: "img/bars-scale-fade.svg", d: img.BarsScaleFadeSVG},
	}
	s.lookup = make(map[string]string)

	for _, f := range files {
		s.lookup[f.n] = f.d
		s.allData = append(s.allData, f.d)
	}
}

func NewStaticFS() *StaticFS {
	var ret StaticFS
	ret.init()
	return &ret
}

func (f *StaticFS) Open(s string) (fs.File, error) {
	dat, ok := f.lookup[s]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return NewStaticFSFile(dat), nil
}

func (f *StaticFS) AllFiles() []string {
	return f.allData
}

func hashFile(dat string) string {
	hsh := sha256.Sum256([]byte(dat))
	return "'sha256-" + base64.StdEncoding.EncodeToString(hsh[:]) + "'"
}

func (f *StaticFS) AllHashes() []string {
	f.Lock()
	defer f.Unlock()
	if len(f.hashes) > 0 {
		return f.hashes
	}
	all := f.AllFiles()
	ret := make([]string, len(all))
	for i, f := range all {
		ret[i] = hashFile(f)
	}
	f.hashes = ret

	return f.hashes
}

var _ fs.FS = (*StaticFS)(nil)

var staticFS *StaticFS

func GetStaticFS() *StaticFS {
	if staticFS == nil {
		staticFS = NewStaticFS()
	}
	return staticFS
}

func setStaticContentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the file extension of the requested file
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		switch ext {
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		default:
		}
		h.ServeHTTP(w, r)
	})
}
