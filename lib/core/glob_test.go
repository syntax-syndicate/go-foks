// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobToRegexp(t *testing.T) {
	glb := "*.ne43.io"
	r, err := GlobToRegexp(glb)
	require.NoError(t, err)
	require.True(t, r.MatchString("foo-44949.ne43.io"))
	require.False(t, r.MatchString("foo-44949.ne43.io."))
	require.False(t, r.MatchString("foo-44949.ne43-io"))
}
