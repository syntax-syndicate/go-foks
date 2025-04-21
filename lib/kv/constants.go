// Copyright (c) 2025 ne43, Inc.
// Licensed under the MIT License. See LICENSE in the project root for details.

package kv

import "golang.org/x/crypto/nacl/secretbox"

const SmallFileSize = 2048
const SmallFileOverhead = 8
const BigFileMaxSize = 1024 * 1024 * 1024 // 1GB
const MaxInputFileChunkSize = 4 * 1024 * 1024
const MaxEncryptedChunkSize = MaxInputFileChunkSize + 5 + secretbox.Overhead // 5 = msgpack overhead
const FilePadding = 512
const MinPaddedChunkSize = 0x20 + 2
const MinPaddedInputSize = 0x20
const MinEncryptedChunkSize = MinPaddedChunkSize + secretbox.Overhead
const MaxPathLength = 256

type PadSpec struct {
	AtOrAbove int
	Overhead  int
}

var PadSpecs = []PadSpec{
	{0x0, 2},
	{0x100, 3},
	{0x10000, 5},
}
