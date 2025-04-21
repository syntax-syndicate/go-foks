// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package css

import (
	_ "embed"
)

//go:embed style.min.css
var StyleMin string

//go:embed style.css
var Style string
