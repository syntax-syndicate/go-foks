// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

//go:build !windows
// +build !windows

package libclient

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPosix(t *testing.T) {
	hf := NewHomeFinder("tester", nil, nil, nil, "posix", func() RunMode { return RunModeProd }, nil)
	d, err := hf.CacheDir()
	require.NoError(t, err)
	if !strings.Contains(d, ".cache/tester") {
		t.Errorf("Bad Cache dir: %s", d)
	}
	d, err = hf.DataDir()
	require.NoError(t, err)
	if !strings.Contains(d, ".local/share/tester") {
		t.Errorf("Bad Data dir: %s", d)
	}
	d, err = hf.ConfigDir()
	require.NoError(t, err)
	if !strings.Contains(d, ".config/tester") {
		t.Errorf("Bad Config dir: %s", d)
	}
}

func TestDarwinHomeFinder(t *testing.T) {
	for _, osname := range []string{"darwin", "ios"} {
		hf := NewHomeFinder("foks", nil, nil, nil, osname, func() RunMode { return RunModeProd }, nil)
		d, err := hf.ConfigDir()
		require.NoError(t, err)
		if !strings.HasSuffix(d, "Library/Application Support/foks") {
			t.Errorf("Bad config dir: %s", d)
		}
		d, err = hf.CacheDir()
		require.NoError(t, err)
		if !strings.HasSuffix(d, "Library/Caches/foks") {
			t.Errorf("Bad cache dir: %s", d)
		}
		hfInt := NewHomeFinder("foks", func() string { return "home" }, nil, func() string { return "mobilehome" },
			osname, func() RunMode { return RunModeProd }, nil)
		hfDarwin := hfInt.(Darwin)
		hfDarwin.forceIOS = true
		hf = hfDarwin
		d, err = hf.ConfigDir()
		require.NoError(t, err)
		require.True(t, strings.HasSuffix(d, "Library/Application Support/foks"))
		require.True(t, strings.HasPrefix(d, "mobilehome"))
		d, err = hf.DataDir()
		require.NoError(t, err)
		require.True(t, strings.HasSuffix(d, "Library/Application Support/foks"))
		require.False(t, strings.HasPrefix(d, "mobilehome"))
		require.True(t, strings.HasPrefix(d, "home"))
	}
}

func TestDarwinHomeFinderInDev(t *testing.T) {
	devHomeFinder := NewHomeFinder("foks", nil, nil, nil, "darwin", func() RunMode { return RunModeDevel }, nil)
	configDir, err := devHomeFinder.ConfigDir()
	require.NoError(t, err)
	if !strings.HasSuffix(configDir, "Library/Application Support/foks/devel") {
		t.Errorf("Bad config dir: %s", configDir)
	}
	cacheDir, err := devHomeFinder.CacheDir()
	require.NoError(t, err)
	if !strings.HasSuffix(cacheDir, "Library/Caches/foks/devel") {
		t.Errorf("Bad cache dir: %s", cacheDir)
	}
}

func TestPosixRuntimeDir(t *testing.T) {
	var cmdHome string
	env := make(map[string]string)
	ge := func(s string) string { return env[s] }
	hf := NewHomeFinder("tester", func() string { return cmdHome }, nil, nil, "posix", func() RunMode { return RunModeProd }, ge)

	origHomeEnv := os.Getenv("HOME")

	// Custom env, custom cmd, XDG set
	cmdHome = "/footown"
	env["HOME"] = "/yoyo"
	env["XDG_RUNTIME_DIR"] = "/barland"
	d, err := hf.RuntimeDir()
	require.NoError(t, err)
	require.Equal(t, "/footown/.config/tester", d, "expect custom cmd to win")

	// Custom env, no cmd, XDG set
	cmdHome = ""
	env["HOME"] = "/yoyo"
	env["XDG_RUNTIME_DIR"] = "/barland"
	d, err = hf.RuntimeDir()
	require.NoError(t, err)
	require.Equal(t, "/yoyo/.config/tester", d, "expect custom env to win")

	// Standard env, no cmd, XDG set
	cmdHome = ""
	env["HOME"] = origHomeEnv
	env["XDG_RUNTIME_DIR"] = "/barland"
	d, err = hf.RuntimeDir()
	require.NoError(t, err)
	require.Equal(t, "/barland/tester", d, "expect xdg to win")

	// Standard env, no cmd, XDG unset
	cmdHome = ""
	env["HOME"] = origHomeEnv
	delete(env, "XDG_RUNTIME_DIR")
	d, err = hf.RuntimeDir()
	require.NoError(t, err)
	require.Equal(t, path.Join(origHomeEnv, ".config", "tester"), d, "expect home to win")
}
