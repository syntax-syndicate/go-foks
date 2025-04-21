// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package js

import (
	_ "embed"
)

//go:embed htmx.min.js
var HtmxMin string

//go:embed htmx.js
var Htmx string

//go:embed foks.js
var Foks string

//go:embed foks.min.js
var FoksMin string
