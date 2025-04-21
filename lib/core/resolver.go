// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"

	proto "github.com/foks-proj/go-foks/proto/lib"
)

type CNameResolver interface {
	// If no resolution is possible, we'll just return "".
	// but for testing can be used to mock out differnt hostnames that all
	// resolve to localhost.
	Resolve(n proto.Hostname) proto.Hostname // Recursive
	ResolveSingle(n proto.Hostname) proto.Hostname
	Add(n proto.Hostname, m proto.Hostname)
	Remove(n proto.Hostname)
}

type PassthroughCNameResolver struct{}

func (PassthroughCNameResolver) Resolve(n proto.Hostname) proto.Hostname       { return "" }
func (PassthroughCNameResolver) ResolveSingle(n proto.Hostname) proto.Hostname { return "" }
func (PassthroughCNameResolver) Add(n proto.Hostname, m proto.Hostname)        {}
func (PassthroughCNameResolver) Remove(n proto.Hostname)                       {}

var _ CNameResolver = PassthroughCNameResolver{}

type SimpleCNameResolver struct {
	sync.Mutex
	globs   []DNSAlias
	entries map[proto.Hostname]proto.Hostname
}

func (c *SimpleCNameResolver) Add(x, y proto.Hostname) {
	c.Lock()
	defer c.Unlock()
	c.entries[x] = y
}

func (c *SimpleCNameResolver) AddObj(x DNSAlias) {
	c.Lock()
	defer c.Unlock()
	c.addObjLocked(x)
}

func (c *SimpleCNameResolver) addObjLocked(x DNSAlias) {
	if x.From.Glob != nil {
		c.globs = append(c.globs, x)
	} else {
		c.entries[x.From.Hostname] = x.To
	}
}

func (s *SimpleCNameResolver) WithObjs(x []DNSAlias) *SimpleCNameResolver {
	s.Lock()
	defer s.Unlock()
	for _, o := range x {
		s.addObjLocked(o)
	}
	return s
}

func (c *SimpleCNameResolver) ResolveSingle(x proto.Hostname) proto.Hostname {
	c.Lock()
	defer c.Unlock()
	return c.resolveSingleWithLock(x)
}

func (c *SimpleCNameResolver) resolveSingleWithLock(x proto.Hostname) proto.Hostname {
	xn := x.Normalize()
	for _, g := range c.globs {
		if g.From.Glob.MatchString(string(xn)) {
			return g.To
		}
	}

	y, ok := c.entries[xn]
	if ok {
		return y
	}
	return ""
}

func (c *SimpleCNameResolver) Resolve(x proto.Hostname) proto.Hostname {
	c.Lock()
	defer c.Unlock()

	// Recursively re-resolve until we can't any further
	ret := x.Normalize()
	seen := make(map[proto.Hostname]bool)
	for {
		y := c.resolveSingleWithLock(ret)
		if y == "" {
			return ret
		}
		if seen[y] {
			return y
		}
		seen[y] = true
		ret = y
	}
}

func (c *SimpleCNameResolver) Remove(x proto.Hostname) {
	c.Lock()
	defer c.Unlock()
	delete(c.entries, x)
}

var _ CNameResolver = (*SimpleCNameResolver)(nil)

func NewSimpleCNameResolver() *SimpleCNameResolver {
	return &SimpleCNameResolver{
		entries: make(map[proto.Hostname]proto.Hostname),
	}
}

type HostnameOrGlob struct {
	Hostname proto.Hostname
	Glob     *regexp.Regexp
}

type DNSAlias struct {
	From HostnameOrGlob
	To   proto.Hostname
}

func (h *HostnameOrGlob) Parse(s string) error {
	if !isGlob(s) {
		h.Hostname = proto.Hostname(s)
		return nil
	}
	r, err := GlobToRegexp(s)
	if err != nil {
		return err
	}
	h.Glob = r
	return nil
}

func (c *DNSAlias) UnmarshalJSON(b []byte) error {
	type typ struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	var tmp typ
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}
	var from HostnameOrGlob
	err = from.Parse(tmp.From)
	if err != nil {
		return err
	}
	c.From = from
	c.To = proto.Hostname(tmp.To)
	return nil
}

func ParseCNameAliases(v []string) ([]DNSAlias, error) {
	var ret []DNSAlias
	for _, s := range v {
		pair := strings.Split(s, "=")
		if len(pair) != 2 {
			continue
		}
		var from HostnameOrGlob
		err := from.Parse(pair[0])
		if err != nil {
			return nil, err
		}
		ret = append(ret, DNSAlias{
			From: from,
			To:   proto.Hostname(pair[1]),
		})
	}
	return ret, nil
}

func isGlob(s string) bool {
	return strings.Contains(s, "*")
}
