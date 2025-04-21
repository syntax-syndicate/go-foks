// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package lib

func Pluralize(s string, n int) string {
	if n == 1 {
		return s
	}
	return s + "s"
}
