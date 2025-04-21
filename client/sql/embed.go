// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package sql

import (
	_ "embed"
)

//go:embed hard.sql
var HardSQL string

//go:embed soft.sql
var SoftSQL string
