// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package core

func RandomBase36String(n int) (string, error) {
	b := make([]byte, n)
	err := RandomFill(b)
	if err != nil {
		return "", err
	}
	return Base36Encoding.EncodeToString(b), nil
}
