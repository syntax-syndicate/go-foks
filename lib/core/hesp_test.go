// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHESP(t *testing.T) {
	testConfig(t, KexSeedHESPConfig)
	testConfig(t, BackupSeedHESPConfig)
}

func testConfig(t *testing.T, c *HESPConfig) {

	s := NewHESP(c)
	err := s.Generate()
	require.NoError(t, err)
	zeds := 0
	for _, v := range s.words {
		if v == 0 {
			zeds++
		}
	}
	for _, v := range s.ints {
		if v == 0 {
			zeds++
		}
	}
	require.Less(t, zeds, 3)
	fmt.Printf("%+v\n", s)
	fmt.Printf("%s\n", s.ToString())

	s2 := NewHESP(c)
	err = s2.FromString(s.ToString())
	require.NoError(t, err)
	require.Equal(t, s, s2)
	require.Equal(t, s.ToString(), s2.ToString())
}
